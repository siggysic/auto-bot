package logger

import (
	"fmt"
	"log"

	. "github.com/logrusorgru/aurora"
)

type GenericLogger struct {
}

func NewGenericLogger() *GenericLogger {
	return &GenericLogger{}
}

func (svc *GenericLogger) LogPrice(lg *LoggerData) {
	position := lg.Position
	selectedRound := lg.SelectedRound
	botSymbol := lg.BotSymbol
	roe := lg.ROE
	roeColor := Green(lg.ROE)
	unrealizedProfColor := Green(position.UnrealizedProfit)
	if roe < 0 {
		roeColor = Red(roe)
		unrealizedProfColor = Red(position.UnrealizedProfit)
	}
	header := fmt.Sprintf("\n==== [%s] Round %d ====", botSymbol, selectedRound.Round)
	body := fmt.Sprintf("\t [%s] Target: %f %% Current: %f %%\n", selectedRound.TargetType, Magenta(selectedRound.Target), roeColor)
	body = body + fmt.Sprintf("%s\n", Cyan("---- Position ----"))
	body = body + fmt.Sprintf("\t Entry price: %s[%s] Leverage: %s Position: %s[%s] Unrealized profit: %s USDT",
		Brown(position.EntryPrice), Brown(position.InitialMargin), Yellow(position.Leverage), Yellow(position.PositionAmt), Yellow(position.PositionSide), unrealizedProfColor)
	log.Printf("%s\n%s\n", Cyan(header), body)
}
