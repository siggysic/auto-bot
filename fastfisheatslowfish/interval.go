// Copyright © 2017 Alessandro Sanino <saninoale@gmail.com>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package fastfisheatslowfish

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"strconv"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/bwmarrin/discordgo"
	"github.com/juju/errors"
	. "github.com/logrusorgru/aurora"
	"github.com/saniales/golang-crypto-trading-bot/consts"
	"github.com/saniales/golang-crypto-trading-bot/environment"
	"github.com/saniales/golang-crypto-trading-bot/exchanges"
	"github.com/saniales/golang-crypto-trading-bot/mongo"
	"github.com/saniales/golang-crypto-trading-bot/strategies"
)

type MActiveType map[string]Actives

var discordBot *discordgo.Session
var discordBotToken string
var discordChannelId string
var discordNotiGapMinute int64
var discordNotiTime time.Time
var interval = 10 * time.Second

var mongoDB *mongo.Mongo
var mongoRepo *MongoRepository
var mactives MActiveType = make(map[string]Actives)
var isSendNoti = false

// FastFishEatSlowFish sends messages to a specified discord channel.
var FastFishEatSlowFish = strategies.IntervalStrategy{
	Model: strategies.StrategyModel{
		Name: "FastFishEatSlowFish",
		Setup: func(exchanges []exchanges.ExchangeWrapper, markets []*environment.Market) error {
			log.Println("========== Setup ==========")
			initDiscordBot(markets)

			for _, ex := range exchanges {

				ex.SetHedgeMode(true)

				for ind, market := range markets {
					if len(market.Rounds) == 0 {
						continue
					}
					var firstRound environment.CoinRounds
					if len(market.Rounds) > 0 {
						firstRound = market.Rounds[0]
					}
					positions, err := ex.GetBalances()
					if err != nil {
						return err
					}
					// Setup mongo
					if ind == 0 {
						mongoURI := market.CustomStorageM[consts.MongoURIStorage]
						mongoDB, err = initMongoDB(mongoURI)
						if err != nil {
							return err
						}
						mongoRepo = NewMongoRepository(mongoDB)
					}

					botSymbol := toSymbol(market.BaseCurrency, market.MarketCurrency)
					position := positionMatchers(botSymbol, firstRound.PositionType, positions)

					log.Println("market name :", market.Name)
					log.Println("isPosNil(position) :", isPosNil(position))
					log.Printf("position : %+v\n", position)
					if isPosNil(position) {
						if len(market.Rounds) > 0 {
							firstRound := market.Rounds[0]
							err = initialAction(ex, market, position, firstRound, botSymbol, 0)
							if err != nil {
								log.Println(err)
								return err
							}
						} else {
							return errors.New("Config round is empty.")
						}
					} else {
						act, err := mongoRepo.FindOneActivesWithSymbol(botSymbol)
						if err != nil {
							log.Println(err)
							return err
						}

						// Save in mem
						mactives.Save(botSymbol, act.Amount, act.Side, act.Price, act.Round, act.HighestROE)
					}
				}
			}

			log.Println("========== Setup ==========")

			return nil
		},
		// ========
		// OnUpdate
		// ========
		OnUpdate: func(exchanges []exchanges.ExchangeWrapper, markets []*environment.Market) error {
			scheduleReports := []string{}
			for _, ex := range exchanges {

				positions, err := ex.GetBalances()
				if err != nil {
					log.Println(err)
					continue
				}
				for _, market := range markets {
					if len(market.Rounds) == 0 {
						continue
					}

					var firstRound environment.CoinRounds
					if len(market.Rounds) > 0 {
						firstRound = market.Rounds[0]
					}

					botSymbol := toSymbol(market.BaseCurrency, market.MarketCurrency)
					position := positionMatchers(botSymbol, firstRound.PositionType, positions)
					var roe float64

					if isPosNil(position) {
						err = initialAction(ex, market, position, firstRound, botSymbol, 0)
						if err != nil {
							log.Println(err)
							continue
						}
					} else {
						activeMem := mactives.Get(botSymbol)

						if len(market.Rounds) > 0 {
							var selectedRound *environment.CoinRounds
							var nextRound *environment.CoinRounds
							maxRound := len(market.Rounds)
							for ind, round := range market.Rounds {
								if round.Round == activeMem.Round {
									selectedRound = &round
									if ind+1 < maxRound {
										nextRound = &market.Rounds[ind+1]
									}
									break
								}
							}
							if selectedRound == nil {
								log.Println("selectedRound == nil")
								continue
							}

							position = positionMatchers(botSymbol, selectedRound.PositionType, positions)
							roe = calROE(position.UnrealizedProfit, position.InitialMargin)

							// Can close position
							logRound(botSymbol, selectedRound, position, roe)
							if roe >= selectedRound.Target {
								if selectedRound.TargetType == environment.TrailingStopPercent && selectedRound.TrailingStopPercent != nil {
									if activeMem.HighestROE != nil {
										highestROE := *activeMem.HighestROE
										diffTrailingROE := roe - highestROE
										log.Println("activeMem.HighestROE : ", *activeMem.HighestROE)
										log.Println("diffTrailingROE : ", diffTrailingROE)
										if diffTrailingROE > 0 {
											// Save in mem
											mactives.SaveROE(botSymbol, &roe)

											log.Printf("update diffTrailingROE > 0 with %+v\n", mactives.Get(botSymbol))
											// Save in mongo
											err = mongoRepo.FindAndUpdateAction(mactives.Get(botSymbol))
											if err != nil {
												log.Println(err)
												continue
											}
										} else if diffTrailingROE < 0 {
											{
												// calCurrentRoe := roe + diffTrailingROE
												log.Println("selectedRound.TrailingStopPercent : ", *selectedRound.TrailingStopPercent)
												if -diffTrailingROE >= *selectedRound.TrailingStopPercent || roe <= selectedRound.Target {
													// Close posiiton
													var closeOrder *futures.CreateOrderResponse
													amt, err := strconv.ParseFloat(position.PositionAmt, 64)
													if err != nil {
														log.Println(err)
														continue
													}

													closeOrder, err = closePosition(ex, market, position, botSymbol, *selectedRound, amt, roe)
													if err != nil {
														log.Println(err)
														continue
													}

													// Push message
													report := reportClosePosition(closeOrder.Symbol, closeOrder.ClientOrderID, closeOrder.OrderID, string(closeOrder.Type), string(closeOrder.Status), closeOrder.Price, position.UnrealizedProfit, roe, selectedRound.Round)
													_, err = discordBot.ChannelMessageSend(discordChannelId, report)
													if err != nil {
														log.Println(err)
														continue
													}

													// Initial action
													err = initialAction(ex, market, position, firstRound, botSymbol, 0)
													if err != nil {
														log.Println(err)
														continue
													}
												}
											}
										}
									} else if activeMem.HighestROE == nil || roe > *activeMem.HighestROE {
										// Save in mem
										mactives.SaveROE(botSymbol, &roe)

										log.Printf("update activeMem.HighestROE == nil || roe > *activeMem.HighestROE with %+v\n", mactives.Get(botSymbol))

										// Save in mongo
										err = mongoRepo.FindAndUpdateAction(mactives.Get(botSymbol))
										if err != nil {
											log.Println(err)
											continue
										}
									}
								} else if selectedRound.TargetType == environment.FixedPercentTarget || selectedRound.TrailingStopPercent == nil {
									if roe >= selectedRound.Target {
										// Close posiiton
										var closeOrder *futures.CreateOrderResponse
										amt, err := strconv.ParseFloat(position.PositionAmt, 64)
										if err != nil {
											log.Println(err)
											continue
										}

										closeOrder, err = closePosition(ex, market, position, botSymbol, *selectedRound, amt, roe)
										if err != nil {
											log.Println(err)
											continue
										}

										// Push message
										report := reportClosePosition(closeOrder.Symbol, closeOrder.ClientOrderID, closeOrder.OrderID, string(closeOrder.Type), string(closeOrder.Status), closeOrder.Price, position.UnrealizedProfit, roe, selectedRound.Round)
										_, err = discordBot.ChannelMessageSend(discordChannelId, report)
										if err != nil {
											log.Println(err)
											continue
										}

										// Initial action
										err = initialAction(ex, market, position, firstRound, botSymbol, 0)
										if err != nil {
											log.Println(err)
											continue
										}
									}
								}
							}

							// Can open poisiton
							if nextRound == nil {
								log.Println("Reached max round")
								continue
							}
							if roe <= -nextRound.Signal {
								err = initialAction(ex, market, position, *nextRound, botSymbol, roe)
								if err != nil {
									log.Println(err)
									continue
								}
							}
						}
					}

					defer func() {
						// Ticker noti current position accounts
						if isSendNoti {
							// Push message
							report := reportCurrentPositions(botSymbol, position.PositionAmt, position.UnrealizedProfit, roe, position.Leverage, string(position.PositionSide))
							scheduleReports = append(scheduleReports, report)
						}
					}()
				}
			}
			// Ticker noti current position accounts
			isSendNoti = beforeExitUpdate(discordNotiTime)
			if len(scheduleReports) > 0 {
				discordNotiTime = initUpdate(discordNotiTime, discordNotiGapMinute)

				for _, report := range scheduleReports {
					_, err := discordBot.ChannelMessageSend(discordChannelId, report)
					if err != nil {
						log.Println(err)
					}
				}

				scheduleReports = []string{}
			}
			return nil
		},
		OnError: func(err error) {
			pc, fn, line, _ := runtime.Caller(1)
			log.Printf("[error] %s in %s[%s:%d] %v", err, runtime.FuncForPC(pc).Name(), fn, line, err)
			discordBot.Close()
			mongoDB.Disconnect()
		},
		TearDown: func([]exchanges.ExchangeWrapper, []*environment.Market) error {
			err := discordBot.Close()
			if err != nil {
				return err
			}
			mongoDB.Disconnect()
			return nil
		},
	},
	Interval: interval,
}

func initUpdate(t time.Time, addMin int64) time.Time {
	return addTimeNoti(t, addMin)
}

func beforeExitUpdate(notiTime time.Time) (isNotiTime bool) {
	now := time.Now()

	isNotiTime = now.After(notiTime)
	return
}

func addTimeNoti(t time.Time, addMin int64) time.Time {
	return t.Add(time.Duration(addMin) * time.Minute)
}

func toSymbol(base string, market string) string {
	return fmt.Sprintf("%s%s", base, market)
}

func positionMatchers(symbol string, positionType environment.PositionType, positions *futures.Account) *futures.AccountPosition {
	var pos *futures.AccountPosition
	for _, position := range positions.Positions {
		if position.Symbol == symbol && string(position.PositionSide) == string(positionType) {
			pos = position
			break
		}
	}
	return pos
}

func calROE(upnl string, initMargin string) float64 {
	unrealizedPNL, err := strconv.ParseFloat(upnl, 64)
	if err != nil {
		log.Println(err)
	}
	initialMargin, err := strconv.ParseFloat(initMargin, 64)
	if err != nil {
		log.Println(err)
	}

	if initialMargin == 0 {
		return 0
	}

	return (unrealizedPNL / initialMargin) * 100
}

func isPosNil(position *futures.AccountPosition) bool {
	if position == nil {
		return true
	}

	return position.InitialMargin == "0"
}

func initDiscordBot(markets []*environment.Market) error {
	// Create a new Discord session using the provided bot token.
	var err error
	now := time.Now()
	discordBotToken = markets[0].CustomStorageM[consts.DiscordTokenStorage]
	discordChannelId = markets[0].CustomStorageM[consts.DiscordChannelIdStorage]
	discordNotiGapMinute, _ = strconv.ParseInt(markets[0].CustomStorageM[consts.DiscordNotiGapMinuteStorage], 10, 0)
	discordNotiTime = addTimeNoti(now, discordNotiGapMinute)
	discordBot, err = discordgo.New("Bot " + discordBotToken)
	if err != nil {
		return err
	}

	go func() {
		err = discordBot.Open()
		if err != nil {
			return
		}
	}()

	//sleep some time
	time.Sleep(time.Second * 5)
	if err != nil {
		return err
	}
	_, err = discordBot.ChannelMessageSend(discordChannelId, reportInitialBot(markets))
	if err != nil {
		return err
	}

	return nil
}

func initMongoDB(mongoURI string) (*mongo.Mongo, error) {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

	mong := mongo.NewMongo(ctx, mongoURI)
	err := mong.Connect()
	if err != nil {
		return nil, err
	}
	return mong, nil
}

func initialAction(ex exchanges.ExchangeWrapper, market *environment.Market, position *futures.AccountPosition, round environment.CoinRounds, botSymbol string, roe float64) error {
	err := ex.SetLeverage(market, round.Leverage)
	if err != nil {
		return err
	}

	clientOrderId := ""
	var orderId int64
	if round.PositionType == environment.BuyPosition {
		order, err := ex.BuyMarket(market, round.Amount)
		if err != nil {
			log.Println(err)
			return err
		}

		clientOrderId = order.ClientOrderID
		orderId = order.OrderID

		// Push message
		report := reportBuyMarket(order.Symbol, order.ClientOrderID, order.OrderID, string(order.Type), round.Leverage, string(order.Side), string(order.Status), fmt.Sprint(round.Amount))
		_, err = discordBot.ChannelMessageSend(discordChannelId, report)
		if err != nil {
			return err
		}

	} else if round.PositionType == environment.SellPosiiton {
		order, err := ex.SellMarket(market, round.Amount)
		if err != nil {
			log.Println(err)
			return err
		}

		clientOrderId = order.ClientOrderID
		orderId = order.OrderID

		// Push message
		report := reportSellMarket(order.Symbol, order.ClientOrderID, order.OrderID, string(order.Type), round.Leverage, string(order.Side), string(order.Status), fmt.Sprint(round.Amount))
		_, err = discordBot.ChannelMessageSend(discordChannelId, report)
		if err != nil {
			return err
		}
	}

	price := ""
	up := ""
	ps := ""
	lv := ""
	amt := ""
	side := ""
	positionRisk, err := ex.GetPositionRisk(market)
	if err != nil {
		log.Println(err)
		return err
	}
	if len(positionRisk) > 0 {
		price = positionRisk[0].EntryPrice
		up = positionRisk[0].UnRealizedProfit
		ps = positionRisk[0].PositionSide
		amt = positionRisk[0].PositionAmt
		lv = positionRisk[0].Leverage
		side = positionRisk[0].PositionSide
	}

	// Save in mem
	mactives.Save(botSymbol, amt, side, price, round.Round, nil)

	log.Printf("update environment.BuyPosition with %+v\n", mactives.Get(botSymbol))
	// Save in mongo
	err = mongoRepo.FindAndUpdateAction(mactives.Get(botSymbol))
	if err != nil {
		log.Println(err)
	}

	// Save log event
	logs := Logs{
		ClientOrderID: clientOrderId, OrderID: int64(orderId), Symbol: botSymbol, Amount: amt,
		Leverage: lv, Side: "BUY", Position: ps, Round: round.Round,
		Price: price, Profilt: up, ROE: roe, CreatedAt: time.Now(),
	}
	err = mongoRepo.CreateLog(logs)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func closePosition(
	ex exchanges.ExchangeWrapper, market *environment.Market, position *futures.AccountPosition,
	botSymbol string, selectedRound environment.CoinRounds, amt float64, roe float64) (closeOrder *futures.CreateOrderResponse, err error) {

	if selectedRound.PositionType == environment.BuyPosition {
		closeOrder, err = ex.SellMarket(market, amt)
		if err != nil {
			return nil, err
		}

	} else {
		closeOrder, err = ex.BuyMarket(market, amt)
		if err != nil {
			return nil, err
		}
	}

	if closeOrder != nil {
		entryPrice, err := strconv.ParseFloat(position.EntryPrice, 64)
		if err != nil {
			return nil, err
		}
		diffPrice := entryPrice * roe
		exitPrice := entryPrice + diffPrice

		// Save log event
		logs := Logs{
			ClientOrderID: closeOrder.ClientOrderID, OrderID: closeOrder.OrderID, Symbol: botSymbol, Amount: fmt.Sprintf("%f", amt),
			Leverage: position.Leverage, Side: "SELL", Position: string(position.PositionSide), Round: selectedRound.Round,
			Price: fmt.Sprintf("%f", exitPrice), Profilt: position.UnrealizedProfit, ROE: roe, CreatedAt: time.Now(),
		}
		err = mongoRepo.CreateLog(logs)
		if err != nil {
			return nil, err
		}
	}

	return closeOrder, nil
}

func logRound(botSymbol string, selectedRound *environment.CoinRounds, position *futures.AccountPosition, roe float64) {
	roeColor := Green(roe)
	unrealizedProfColor := Green(position.UnrealizedProfit)
	if roe < 0 {
		roeColor = Red(roe)
		unrealizedProfColor = Red(position.UnrealizedProfit)
	}
	header := fmt.Sprintf("\n==== [%s] Round %d ====", botSymbol, selectedRound.Round)
	body := fmt.Sprintf("\t [%s] Target: %f %% Current: %f %%\n", selectedRound.TargetType, Magenta(selectedRound.Target), roeColor)
	body = body + fmt.Sprintf("%s\n", Cyan("---- Position ----"))
	body = body + fmt.Sprintf("\t Entry price: %s Leverage: %s Position: %s[%s] Unrealized profit: %s USDT",
		Brown(position.EntryPrice), Yellow(position.Leverage), Yellow(position.PositionAmt), Yellow(position.PositionSide), unrealizedProfColor)
	log.Printf("%s\n%s\n", Cyan(header), body)
}

func (m MActiveType) Save(symbol, amount, side, price string, round int, hroe *float64) {
	m[symbol] = Actives{
		Symbol:     symbol,
		Amount:     amount,
		Side:       side,
		Round:      round,
		Price:      price,
		HighestROE: hroe,
	}
}

func (m MActiveType) SaveROE(symbol string, hroe *float64) {
	if active, ok := m[symbol]; ok {
		active.HighestROE = hroe
		m[symbol] = active
	}
}

func (m MActiveType) Get(symbol string) Actives {
	return m[symbol]
}

// // Watch5Sec prints out the info of the market every 5 seconds.
// var Watch5Sec = strategies.IntervalStrategy{
// 	Model: strategies.StrategyModel{
// 		Name: "Watch5Sec",
// 		Setup: func(wrappers []exchanges.ExchangeWrapper, markets []*environment.Market) error {
// 			log.Println("Watch5Sec starting")
// 			return nil
// 		},
// 		OnUpdate: func(wrappers []exchanges.ExchangeWrapper, markets []*environment.Market) error {
// 			_, err := wrappers[0].GetMarketSummary(markets[0])
// 			if err != nil {
// 				return err
// 			}
// 			logrus.Info(markets)
// 			logrus.Info(wrappers)
// 			return nil
// 		},
// 		OnError: func(err error) {
// 			log.Println(err)
// 		},
// 		TearDown: func(wrappers []exchanges.ExchangeWrapper, markets []*environment.Market) error {
// 			log.Println("Watch5Sec exited")
// 			return nil
// 		},
// 	},
// 	Interval: time.Second * 5,
// }

// var slackBot *slacker.Slacker

// // SlackIntegrationExample send messages to Slack as a strategy.
// // RTM not supported (and usually not requested when trading, this is an automated slackBot).
// var SlackIntegrationExample = strategies.IntervalStrategy{
// 	Model: strategies.StrategyModel{
// 		Name: "SlackIntegrationExample",
// 		Setup: func([]exchanges.ExchangeWrapper, []*environment.Market) error {
// 			// connect slack token
// 			slackBot = slacker.NewClient("YOUR-TOKEN-HERE")
// 			slackBot.Init(func() {
// 				log.Println("Slack BOT Connected")
// 			})
// 			slackBot.Err(func(err string) {
// 				log.Println("Error during slack slackBot connection: ", err)
// 			})
// 			go func() {
// 				err := slackBot.Listen(context.Background())
// 				if err != nil {
// 					log.Fatal(err)
// 				}
// 			}()
// 			return nil
// 		},
// 		OnUpdate: func([]exchanges.ExchangeWrapper, []*environment.Market) error {
// 			//if updates has requirements
// 			_, _, err := slackBot.Client().PostMessage("DESIRED-CHANNEL", slack.MsgOptionText("OMG something happening!!!!!", true))
// 			return err
// 		},
// 		OnError: func(err error) {
// 			logrus.Errorf("I Got an error %s", err)
// 		},
// 	},
// 	Interval: time.Second * 10,
// }

// var telegramBot *tb.Bot

// // TelegramIntegrationExample send messages to Telegram as a strategy.
// var TelegramIntegrationExample = strategies.IntervalStrategy{
// 	Model: strategies.StrategyModel{
// 		Name: "TelegramIntegrationExample",
// 		Setup: func([]exchanges.ExchangeWrapper, []*environment.Market) error {
// 			telegramBot, err := tb.NewBot(tb.Settings{
// 				Token:  "YOUR-TELEGRAM-TOKEN",
// 				Poller: &tb.LongPoller{Timeout: 10 * time.Second},
// 			})

// 			if err != nil {
// 				return err
// 			}

// 			telegramBot.Start()
// 			return nil
// 		},
// 		OnUpdate: func([]exchanges.ExchangeWrapper, []*environment.Market) error {
// 			telegramBot.Send(&tb.User{
// 				Username: "YOUR-USERNAME-GROUP-OR-USER",
// 			}, "OMG SOMETHING HAPPENING!!!!!", tb.SendOptions{})

// 			/*
// 				// Optionally it can have options
// 				telegramBot.Send(tb.User{
// 					Username: "YOUR-JOINED-GROUP-USERNAME",
// 				}, "OMG SOMETHING HAPPENING!!!!!", tb.SendOptions{})
// 			*/
// 			return nil
// 		},
// 		OnError: func(err error) {
// 			logrus.Errorf("I Got an error %s", err)
// 			telegramBot.Stop()
// 		},
// 		TearDown: func([]exchanges.ExchangeWrapper, []*environment.Market) error {
// 			telegramBot.Stop()
// 			return nil
// 		},
// 	},
// }
