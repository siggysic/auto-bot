package logger

import (
	"fmt"

	. "github.com/logrusorgru/aurora"
	"github.com/saniales/golang-crypto-trading-bot/environment"
)

type EMALogger struct {
	slowEMA float64
	fastEMA float64
}

func NewEMALogger() *EMALogger {
	return &EMALogger{}
}

func (svc *EMALogger) LogPrice(lg *LoggerData) {
	botSymbol := lg.BotSymbol
	round := lg.SelectedRound
	currentTrend := lg.Trend
	displayCurrentTread := Red(currentTrend)
	slowEMA := lg.SlowEMA
	fastEMA := lg.FastEMA
	if environment.PositionType(currentTrend) == environment.BuyPosition {
		displayCurrentTread = Green(currentTrend)
	}
	fmt.Println(fmt.Sprintf(`
							`+"\n"+`====== NOTIFICATION [%s] ======`+
		"\n  "+`Slow EMA : %d Slow EMA Value : %f`+
		"\n  "+`Fast EMA : %d Fast EMA Value : %f`+
		"\n  "+`Tread : %s
						`, Cyan(botSymbol), Yellow(round.SlowEMA), Yellow(slowEMA), Magenta(round.FastEMA), Magenta(fastEMA), displayCurrentTread))
}
