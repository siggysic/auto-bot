package backtesthelps

import (
	"encoding/csv"
	"fmt"
	"os"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/saniales/golang-crypto-trading-bot/channel"
	"github.com/saniales/golang-crypto-trading-bot/exchanges"
	"github.com/saniales/golang-crypto-trading-bot/fastfisheatslowfish"
	"github.com/saniales/golang-crypto-trading-bot/logger"
	"github.com/saniales/golang-crypto-trading-bot/mongo"
)

type PositionBackTest struct {
	Date   string
	Prices []string
}

type ChannelBackTest struct {
}

func NewChannelBackTest() channel.Channel {
	return &ChannelBackTest{}
}

func (*ChannelBackTest) InitChannel() error {
	return nil
}
func (*ChannelBackTest) Send(msg string) error {
	return nil
}
func (*ChannelBackTest) Close() error {
	return nil
}

type MongoBackTest struct {
	Mactive fastfisheatslowfish.MActiveType
}

func (*MongoBackTest) FindAndUpdateAction(active mongo.Actives) error {
	return nil
}
func (m *MongoBackTest) FindOneActivesWithSymbol(symbol string) (*mongo.Actives, error) {
	active := m.Mactive[symbol]
	return &mongo.Actives{
		Symbol:     active.Symbol,
		Amount:     active.Amount,
		Side:       active.Side,
		Round:      active.Round,
		Price:      active.Price,
		HighestROE: active.HighestROE,
		UpdatedAt:  active.UpdatedAt,
	}, nil
}
func (*MongoBackTest) CreateLog(logs mongo.Logs) error {
	return nil
}

type LoggerBackTest struct {
	csvwriter *csv.Writer
}

func NewLoggerBackTest(csvwriter *csv.Writer) logger.ILogger {
	return &LoggerBackTest{csvwriter: csvwriter}
}

func (svc *LoggerBackTest) LogPrice(lg *logger.LoggerData) {
	position := &futures.AccountPosition{}
	if lg.Position != nil {
		position = lg.Position
	}
	rows := []string{
		lg.BotSymbol,
		fmt.Sprintf("%f", lg.ROE),
		position.UnrealizedProfit,
		fmt.Sprintf("%d", lg.SelectedRound.Round),
		string(lg.SelectedRound.TargetType),
		fmt.Sprintf("%f", lg.SelectedRound.Target),
		position.EntryPrice,
		position.InitialMargin,
		position.Leverage,
		position.PositionAmt,
		string(position.PositionSide),
	}
	_ = svc.csvwriter.Write(rows)
	svc.csvwriter.Flush()
}

func ReadFESFileCSV() map[string]exchanges.BinanceFutureHistoryBackTest {
	csvFile, err := os.Open("/Users/nattapol.srikitpipat/Workspaces/go/src/golang-crypto-trading-bot/backtest/csv/btc-data-1m.csv")
	if err != nil {
		panic(err)
	}
	defer csvFile.Close()

	csvReader := csv.NewReader(csvFile)
	csvReader.FieldsPerRecord = -1
	csvLines, err := csvReader.ReadAll()
	if err != nil {
		panic(err)
	}

	prices := []string{}
	dates := map[string]bool{}
	for _, line := range csvLines {
		dates[line[0]] = true
		prices = append(prices, line...)
	}

	symbols := map[string]exchanges.BinanceFutureHistoryBackTest{}
	symbols["BTCUSDT"] = exchanges.BinanceFutureHistoryBackTest{
		Prices:         prices,
		Dates:          dates,
		Index:          0,
		IsPriceDecease: true,
	}

	return symbols
}

func ReadEMAFileCSV() map[string]exchanges.BinanceFutureHistoryBackTest {
	csvFile, err := os.Open("/Users/nattapol.srikitpipat/Workspaces/go/src/golang-crypto-trading-bot/backtest/csv/btc-data-4h.csv")
	if err != nil {
		panic(err)
	}
	defer csvFile.Close()

	csvReader := csv.NewReader(csvFile)
	csvReader.FieldsPerRecord = -1
	csvLines, err := csvReader.ReadAll()
	if err != nil {
		panic(err)
	}

	prices := []string{}
	dates := map[string]bool{}
	for _, line := range csvLines {
		dates[line[0]] = true
		prices = append(prices, line[1:]...)
	}

	symbols := map[string]exchanges.BinanceFutureHistoryBackTest{}
	symbols["BTCUSDT"] = exchanges.BinanceFutureHistoryBackTest{
		Prices:         prices,
		Dates:          dates,
		Index:          2561,
		IsPriceDecease: false,
	}

	return symbols
}
