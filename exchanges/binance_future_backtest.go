package exchanges

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/saniales/golang-crypto-trading-bot/environment"
	"github.com/shopspring/decimal"
)

type BinanceFutureBackTestTransaction struct {
	date     string
	price    string
	action   string
	side     string
	leverage int
	pnl      float64
	amount   float64
	roe      float64
}

type BinanceFutureHistoryBackTest struct {
	transactions   []BinanceFutureBackTestTransaction
	leverage       int
	Prices         []string
	IsPriceDecease bool
	Dates          map[string]bool
	Index          int

	buyPrices []BinanceFuturePriceBackTest

	currentPrice string
	currentDate  string
}

type BinanceFuturePriceBackTest struct {
	price    string
	leverage int
	amount   float64
	position string
}

type BinanceFutureBackTestWrapper struct {
	symbol map[string]BinanceFutureHistoryBackTest
}

// NewBinanceFutureWrapper creates a generic wrapper of the binance API.
func NewBinanceFutureBackTestWrapper(symbols map[string]BinanceFutureHistoryBackTest) ExchangeWrapper {
	return &BinanceFutureBackTestWrapper{
		symbol: symbols,
	}
}

func (svc *BinanceFutureBackTestWrapper) Name() string {
	return "binance_future_backtest"
}
func (svc *BinanceFutureBackTestWrapper) GetCandles(market *environment.Market, interval string, startTime, endTime int64, limit int) ([]*futures.Kline, error) {
	defer func() {
		if r := recover(); r != nil {
			backtestRecovery(svc)
		}
	}()
	symbol := MarketNameFor(market, svc)
	lines := []*futures.Kline{}
	history := svc.symbol[symbol]
	end := history.Index
	start := end - limit
	if history.Index >= len(history.Prices) {
		panic("index out of bound")
	}
	transform := func(prices ...string) []*futures.Kline {
		ps := []*futures.Kline{}
		for _, price := range prices {
			ps = append(ps, &futures.Kline{
				OpenTime: time.Now().UnixMilli(),
				Close:    price,
			})
		}
		return ps
	}
	lines = append(lines, transform(history.Prices[start:end+1]...)...)
	history.currentPrice = history.Prices[history.Index]
	history.Index = history.Index + 1
	svc.symbol[symbol] = history
	return lines, nil
}

func (svc *BinanceFutureBackTestWrapper) GetMarketSummary(market *environment.Market) (*environment.MarketSummary, error) {
	return nil, nil
}
func (svc *BinanceFutureBackTestWrapper) GetOrderBook(market *environment.Market) (*environment.OrderBook, error) {
	return nil, nil
}
func (svc *BinanceFutureBackTestWrapper) BuyLimit(market *environment.Market, amount float64, limit float64) (string, error) {
	return "", nil
}
func (svc *BinanceFutureBackTestWrapper) SellLimit(market *environment.Market, amount float64, limit float64) (string, error) {
	return "", nil
}
func (svc *BinanceFutureBackTestWrapper) BuyMarket(market *environment.Market, amount float64, side futures.PositionSideType) (*futures.CreateOrderResponse, error) {
	symbol := MarketNameFor(market, svc)
	history := svc.symbol[symbol]
	history.buyPrices = append(history.buyPrices, BinanceFuturePriceBackTest{
		price:    history.currentPrice,
		leverage: history.leverage,
		amount:   amount,
		position: string(side),
	})
	entryPriceAvg, amt, upnl, _ := cal(history)
	initialMargin := amount * entryPriceAvg * (float64(1) / float64(history.leverage))
	history.transactions = append(history.transactions, BinanceFutureBackTestTransaction{
		price:    fmt.Sprintf("%s", history.currentPrice),
		action:   "BUY",
		side:     string(side),
		leverage: history.leverage,
		pnl:      upnl,
		amount:   amt,
		roe:      (upnl / initialMargin) * 100,
		date:     history.currentDate,
	})
	svc.symbol[symbol] = history

	return &futures.CreateOrderResponse{
		Symbol:        symbol,
		OrderID:       1234,
		ClientOrderID: "test",
		Price:         "0",
		Status:        "NEW",
		Type:          "MARKET",
		Side:          "BUY",
		PositionSide:  side,
	}, nil
}
func (svc *BinanceFutureBackTestWrapper) SellMarket(market *environment.Market, amount float64, side futures.PositionSideType) (*futures.CreateOrderResponse, error) {
	symbol := MarketNameFor(market, svc)
	history := svc.symbol[symbol]
	entryPriceAvg, amt, upnl, _ := cal(history)
	initialMargin := amount * entryPriceAvg * (float64(1) / float64(history.leverage))
	history.buyPrices = []BinanceFuturePriceBackTest{}
	history.transactions = append(history.transactions, BinanceFutureBackTestTransaction{
		price:    fmt.Sprintf("%s", history.currentPrice),
		action:   "SELL",
		side:     string(side),
		leverage: history.leverage,
		pnl:      upnl,
		amount:   amt,
		roe:      (upnl / initialMargin) * 100,
		date:     history.currentDate,
	})
	svc.symbol[symbol] = history

	return &futures.CreateOrderResponse{
		Symbol:        symbol,
		OrderID:       1234,
		ClientOrderID: "test",
		Price:         "0",
		Status:        "NEW",
		Type:          "MARKET",
		Side:          "SELL",
		PositionSide:  side,
	}, nil
}
func (svc *BinanceFutureBackTestWrapper) ClosePosition(market *environment.Market, side environment.PositionType) (*futures.CreateOrderResponse, error) {
	return nil, nil
}
func (svc *BinanceFutureBackTestWrapper) CalculateTradingFees(market *environment.Market, amount float64, limit float64, orderType TradeType) float64 {
	return 0
}
func (svc *BinanceFutureBackTestWrapper) CalculateWithdrawFees(market *environment.Market, amount float64) float64 {
	return 0
}
func (svc *BinanceFutureBackTestWrapper) GetOrder(symbol string, orderId int64) (*futures.Order, error) {
	return nil, nil
}
func (svc *BinanceFutureBackTestWrapper) GetBalances() (*futures.Account, error) {
	defer func() {
		if r := recover(); r != nil {
			backtestRecovery(svc)
		}
	}()
	pos := []*futures.AccountPosition{}
	for key, his := range svc.symbol {
		ok := his.Dates[his.Prices[0]]
		if ok {
			his.currentDate = his.Prices[0]
			his.Prices = his.Prices[1:]
		}
		his.currentPrice = his.Prices[0]
		if his.IsPriceDecease {
			his.Prices = his.Prices[1:]
		}
		svc.symbol[key] = his

		entryPriceAvg, amount, upnl, position := cal(his)
		if entryPriceAvg == 0 && amount == 0 && upnl == 0 && position == "" {
			continue
		}
		initialMargin := amount * entryPriceAvg * (float64(1) / float64(his.leverage))
		accpos := &futures.AccountPosition{
			Leverage:         fmt.Sprintf("%d", his.leverage),
			InitialMargin:    fmt.Sprintf("%f", initialMargin),
			Symbol:           key,
			UnrealizedProfit: fmt.Sprintf("%f", upnl),
			EntryPrice:       fmt.Sprintf("%f", entryPriceAvg),
			PositionSide:     futures.PositionSideType(position),
			PositionAmt:      fmt.Sprintf("%f", amount),
		}

		if entryPriceAvg == 0 && upnl == 0 {
			accpos.InitialMargin = "0"
		}
		pos = append(pos, accpos)
	}
	return &futures.Account{Positions: pos}, nil
}
func (svc *BinanceFutureBackTestWrapper) GetBalance(symbol string) (*decimal.Decimal, error) {
	return nil, nil
}
func (svc *BinanceFutureBackTestWrapper) GetDepositAddress(coinTicker string) (string, bool) {
	return "", false
}
func (svc *BinanceFutureBackTestWrapper) GetPositionRisk(market *environment.Market) ([]*futures.PositionRisk, error) {
	symbol := MarketNameFor(market, svc)
	history := svc.symbol[symbol]
	entryPriceAvg, amount, upnl, position := cal(history)
	return []*futures.PositionRisk{
		{
			EntryPrice:       fmt.Sprintf("%f", entryPriceAvg),
			Leverage:         fmt.Sprintf("%d", history.leverage),
			PositionAmt:      fmt.Sprintf("%f", amount),
			Symbol:           symbol,
			UnRealizedProfit: fmt.Sprintf("%f", upnl),
			PositionSide:     position,
		},
	}, nil
}
func (svc *BinanceFutureBackTestWrapper) FeedConnect(markets []*environment.Market) error {
	return nil
}
func (svc *BinanceFutureBackTestWrapper) Withdraw(destinationAddress string, coinTicker string, amount float64) error {
	return nil
}
func (svc *BinanceFutureBackTestWrapper) String() string {
	return svc.Name()
}
func (svc *BinanceFutureBackTestWrapper) SetLeverage(market *environment.Market, leverage int) error {
	symbol := MarketNameFor(market, svc)
	history := svc.symbol[symbol]
	history.leverage = leverage
	svc.symbol[symbol] = history
	return nil
}
func (svc *BinanceFutureBackTestWrapper) SetHedgeMode(hedge bool) error {
	return nil
}
func (svc *BinanceFutureBackTestWrapper) GetBacktest() map[string]BinanceFutureHistoryBackTest {
	return svc.symbol
}

func (svc BinanceFutureHistoryBackTest) Transactions() []BinanceFutureBackTestTransaction {
	return svc.transactions
}
func (svc BinanceFutureBackTestTransaction) Date() string {
	return svc.date
}
func (svc BinanceFutureBackTestTransaction) Price() string {
	return svc.price
}
func (svc BinanceFutureBackTestTransaction) Amount() float64 {
	return svc.amount
}
func (svc BinanceFutureBackTestTransaction) Leverage() int {
	return svc.leverage
}
func (svc BinanceFutureBackTestTransaction) Action() string {
	return svc.action
}
func (svc BinanceFutureBackTestTransaction) Side() string {
	return svc.side
}
func (svc BinanceFutureBackTestTransaction) PNL() float64 {
	return svc.pnl
}
func (svc BinanceFutureBackTestTransaction) ROE() float64 {
	return svc.roe
}

func cal(history BinanceFutureHistoryBackTest) (entryPriceAvg, amount, upnl float64, position string) {
	var totalPrices float64 = 0
	currentPrice, _ := strconv.ParseFloat(history.currentPrice, 64)
	for _, price := range history.buyPrices {
		pric, _ := strconv.ParseFloat(price.price, 64)
		totalPrices += pric
		amount += price.amount
		position = price.position
	}
	if len(history.buyPrices) > 0 {
		entryPriceAvg = totalPrices / float64(len(history.buyPrices))
		if position == "LONG" {
			upnl = (currentPrice - entryPriceAvg) * amount
		} else {
			upnl = (entryPriceAvg - currentPrice) * amount
		}
	}

	return entryPriceAvg, amount, upnl, position
}

func backtestRecovery(svc *BinanceFutureBackTestWrapper) {
	backtest := svc.GetBacktest()
	for key, his := range backtest {
		csvFile, err := os.Create(fmt.Sprintf(`/Users/nattapol.srikitpipat/Workspaces/go/src/golang-crypto-trading-bot/backtest/csv/%s.csv`, key))
		if err != nil {
			log.Fatalf("failed creating file: %s", err)
		}

		csvwriter := csv.NewWriter(csvFile)

		rows := []string{"DATE", "PRICE", "AMOUNT", "LEVERAGE", "ACTION", "SIDE", "PROFIT (USDT)", "ROE(%)"}
		_ = csvwriter.Write(rows)
		for _, tran := range his.Transactions() {
			rows := []string{
				tran.Date(), tran.Price(), fmt.Sprintf("%f", tran.Amount()), fmt.Sprintf("%d", tran.Leverage()), tran.Action(),
				tran.Side(), fmt.Sprintf("%f", tran.PNL()), fmt.Sprintf("%f", tran.ROE())}
			_ = csvwriter.Write(rows)
		}
		csvwriter.Flush()
		csvFile.Close()
	}
	panic("Bye")
}
