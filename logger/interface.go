package logger

import (
	"github.com/adshao/go-binance/v2/futures"
	"github.com/saniales/golang-crypto-trading-bot/environment"
)

type ILogger interface {
	LogPrice(log *LoggerData)
}

type LoggerData struct {
	BotSymbol     string
	SelectedRound *environment.CoinRounds
	Position      *futures.AccountPosition
	ROE           float64

	SlowEMA float64
	FastEMA float64
	Trend   string
}
