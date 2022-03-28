package ema

import (
	"errors"
	"fmt"
	"log"
	"runtime"
	"strconv"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/cinar/indicator"
	"github.com/saniales/golang-crypto-trading-bot/channel"
	"github.com/saniales/golang-crypto-trading-bot/environment"
	"github.com/saniales/golang-crypto-trading-bot/exchanges"
	"github.com/saniales/golang-crypto-trading-bot/logger"
	"github.com/saniales/golang-crypto-trading-bot/mongo"
	"github.com/saniales/golang-crypto-trading-bot/strategies"
)

type MActiveType map[string]mongo.Actives

type EMA struct {
	discordBot           channel.Channel
	discordNotiGapMinute int64
	discordNotiTime      time.Time
	interval             time.Duration
	mongoDB              *mongo.Mongo
	mongoRepo            mongo.IMongoRepository
	mactives             MActiveType
	isSendNoti           bool
	scheduleReports      []string
	logger               logger.ILogger
}

func New(discordBot channel.Channel, discordNotiGapMinute int64, discordNotiTime time.Time,
	interval time.Duration, mongoDB *mongo.Mongo, mongoRepo mongo.IMongoRepository, mactives MActiveType,
	logger logger.ILogger) *EMA {
	return &EMA{
		discordBot:           discordBot,
		discordNotiGapMinute: discordNotiGapMinute,
		discordNotiTime:      discordNotiTime,
		interval:             interval,
		mongoDB:              mongoDB,
		mongoRepo:            mongoRepo,
		mactives:             mactives,
		isSendNoti:           false,
		scheduleReports:      []string{},
		logger:               logger,
	}
}

// EMA sends messages to a specified discord channel.
func (svc *EMA) Running() strategies.IntervalStrategy {
	return strategies.IntervalStrategy{
		Model: strategies.StrategyModel{
			Name: "EMA",
			Setup: func(exchanges []exchanges.ExchangeWrapper, markets []*environment.Market) error {
				// discordBot := svc.discordBot
				mongoRepo := svc.mongoRepo
				mactives := svc.mactives
				log.Println("========== Setup ==========")
				// err := discordBot.Send(logger.ReportInitialBot(markets))
				// if err != nil {
				// 	return err
				// }

				for _, ex := range exchanges {
					for _, market := range markets {
						if len(market.Rounds) == 0 {
							return errors.New("Config round is empty.")
						}

						positions, err := ex.GetBalances()
						if err != nil {
							return err
						}
						botSymbol := toSymbol(market.BaseCurrency, market.MarketCurrency)
						position := positionMatchers(botSymbol, positions)

						if !isPosNil(position) {
							act, err := mongoRepo.FindOneActivesWithSymbol(botSymbol)
							if err != nil {
								log.Println(err)
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
				mactives := svc.mactives
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
						round := market.Rounds[0]
						botSymbol := toSymbol(market.BaseCurrency, market.MarketCurrency)
						var roe float64

						// duration := svc.timeframe(round.Interval)
						// endDate := time.Now()
						// startDate := endDate.Add(-(time.Duration(round.SlowEMA) * duration))

						// start := startDate.UnixNano() / 1000000
						// end := endDate.UnixNano() / 1000000

						limit := 300

						candles, err := ex.GetCandles(market, round.Interval, 0, 0, limit)
						if err != nil {
							log.Println(err)
							continue
						}

						prices := []float64{}
						for _, candle := range candles {
							price, _ := strconv.ParseFloat(candle.Close, 64)
							prices = append(prices, price)
						}
						slowValue := indicator.Ema(round.SlowEMA, prices)
						fastValue := indicator.Ema(round.FastEMA, prices)
						slowEMA := slowValue[len(slowValue)-1]
						fastEMA := fastValue[len(fastValue)-1]

						currentTrend := svc.getCurrentTrend(fastEMA, slowEMA)

						fmt.Println(slowEMA)
						fmt.Println(fastEMA)
						fmt.Println(currentTrend)

						svc.logger.LogPrice(&logger.LoggerData{
							BotSymbol: botSymbol, SelectedRound: &round, SlowEMA: slowEMA, FastEMA: fastEMA, Trend: string(currentTrend),
						})

						position := positionMatchers(botSymbol, positions)
						if isPosNil(position) {
							var zero *float64

							actives, ok := mactives.Get(botSymbol)
							if !ok {
								// Save in mem
								mactives.Save(botSymbol, "0", string(currentTrend), "0", round.Round, zero)

								continue
							}

							if actives.Side != string(currentTrend) {
								err = svc.openPosition(ex, market, currentTrend, round, botSymbol, 0)
								if err != nil {
									log.Println(err)
									continue
								}
							}
						} else {
							activeMem, _ := mactives.Get(botSymbol)
							if activeMem.Side != string(currentTrend) {
								_, err = svc.closePosition(ex, market, position, botSymbol, round, round.Amount, roe)
								if err != nil {
									log.Println(err)
									continue
								}

								err = svc.openPosition(ex, market, currentTrend, round, botSymbol, 0)
								if err != nil {
									log.Println(err)
									continue
								}
							}
						}
					}
				}
				return nil
			},
			OnError: func(err error) {
				discordBot := svc.discordBot
				mongoDB := svc.mongoDB
				pc, fn, line, _ := runtime.Caller(1)
				log.Printf("[error] %s in %s[%s:%d] %v", err, runtime.FuncForPC(pc).Name(), fn, line, err)
				discordBot.Close()
				mongoDB.Disconnect()
			},
			TearDown: func([]exchanges.ExchangeWrapper, []*environment.Market) error {
				discordBot := svc.discordBot
				mongoDB := svc.mongoDB
				err := discordBot.Close()
				if err != nil {
					return err
				}
				mongoDB.Disconnect()
				return nil
			},
		},
		Interval: svc.interval,
	}
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

func positionMatchers(symbol string, positions *futures.Account) *futures.AccountPosition {
	var pos *futures.AccountPosition
	for _, position := range positions.Positions {
		if position.Symbol == symbol && position.InitialMargin != "0" {
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

func (svc *EMA) openPosition(ex exchanges.ExchangeWrapper, market *environment.Market, positionType environment.PositionType, round environment.CoinRounds, botSymbol string, roe float64) error {
	err := ex.SetLeverage(market, round.Leverage)
	if err != nil {
		return err
	}

	clientOrderId := ""
	var orderId int64
	order, err := ex.BuyMarket(market, round.Amount, futures.PositionSideType(positionType))
	if err != nil {
		log.Println(err)
		return err
	}

	clientOrderId = order.ClientOrderID
	orderId = order.OrderID

	// Push message
	report := logger.ReportBuyMarket(order.Symbol, order.ClientOrderID, order.OrderID, string(order.Type), round.Leverage, string(order.Side), string(order.Status), fmt.Sprint(round.Amount))
	err = svc.discordBot.Send(report)
	if err != nil {
		return err
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

	for _, pos := range positionRisk {
		if pos.PositionSide == string(positionType) {
			price = pos.EntryPrice
			up = pos.UnRealizedProfit
			ps = pos.PositionSide
			amt = pos.PositionAmt
			lv = pos.Leverage
			side = pos.PositionSide
			break
		}
	}

	// Save in mem
	svc.mactives.Save(botSymbol, amt, side, price, round.Round, nil)

	actives, _ := svc.mactives.Get(botSymbol)
	log.Printf("update environment.BuyPosition with %+v\n", actives)
	// Save in mongo
	err = svc.mongoRepo.FindAndUpdateAction(actives)
	if err != nil {
		log.Println(err)
	}

	// Save log event
	logs := mongo.Logs{
		ClientOrderID: clientOrderId, OrderID: int64(orderId), Symbol: botSymbol, Amount: amt,
		Leverage: lv, Side: "BUY", Position: ps, Round: round.Round,
		Price: price, Profilt: up, ROE: roe, CreatedAt: time.Now(),
	}
	err = svc.mongoRepo.CreateLog(logs)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func (svc *EMA) closePosition(
	ex exchanges.ExchangeWrapper, market *environment.Market, position *futures.AccountPosition,
	botSymbol string, selectedRound environment.CoinRounds, amt float64, roe float64) (*futures.CreateOrderResponse, error) {

	closeOrder, err := ex.SellMarket(market, amt, position.PositionSide)
	if err != nil {
		return nil, err
	}

	entryPrice, err := strconv.ParseFloat(position.EntryPrice, 64)
	if err != nil {
		return nil, err
	}
	diffPrice := entryPrice * roe
	exitPrice := entryPrice + diffPrice

	// Save log event
	logs := mongo.Logs{
		ClientOrderID: closeOrder.ClientOrderID, OrderID: closeOrder.OrderID, Symbol: botSymbol, Amount: fmt.Sprintf("%f", amt),
		Leverage: position.Leverage, Side: "SELL", Position: string(position.PositionSide), Round: selectedRound.Round,
		Price: fmt.Sprintf("%f", exitPrice), Profilt: position.UnrealizedProfit, ROE: roe, CreatedAt: time.Now(),
	}
	err = svc.mongoRepo.CreateLog(logs)
	if err != nil {
		return nil, err
	}

	// Push message
	report := logger.ReportClosePosition(closeOrder.Symbol, closeOrder.ClientOrderID, closeOrder.OrderID, string(closeOrder.Type), string(closeOrder.Status), closeOrder.Price, position.UnrealizedProfit, roe, selectedRound.Round)
	err = svc.discordBot.Send(report)
	if err != nil {
		log.Println(err)
	}

	return closeOrder, nil
}

func (svc *EMA) timeframe(period string) (duration time.Duration) {

	switch period {
	case "1m":
		duration = time.Minute
	case "3m":
		duration = 3 * time.Minute
	case "5m":
		duration = 5 * time.Minute
	case "15m":
		duration = 15 * time.Minute
	case "30m":
		duration = 30 * time.Minute
	case "1h":
		duration = time.Hour
	case "2h":
		duration = 2 * time.Hour
	case "4h":
		duration = 4 * time.Hour
	case "8h":
		duration = 8 * time.Hour
	case "12h":
		duration = 12 * time.Hour
	case "1d":
		duration = 24 * time.Hour
	case "3d":
		duration = 3 * 24 * time.Hour
	case "1w":
		duration = 7 * 24 * time.Hour
	case "1M":
		duration = 30 * 24 * time.Hour
	default:
		duration = 24 * time.Hour
	}

	return
}

func (svc *EMA) getCurrentTrend(fastValue, slowValue float64) environment.PositionType {

	var currentTrend environment.PositionType = ""
	if fastValue > slowValue {
		currentTrend = environment.BuyPosition
	} else if fastValue < slowValue {
		currentTrend = environment.SellPosiiton
	}
	return currentTrend
}

func (m MActiveType) Save(symbol, amount, side, price string, round int, hroe *float64) {
	m[symbol] = mongo.Actives{
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

func (m MActiveType) Get(symbol string) (mongo.Actives, bool) {
	act, ok := m[symbol]
	return act, ok
}
