package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/saniales/golang-crypto-trading-bot/backtest/backtesthelps"
	bot "github.com/saniales/golang-crypto-trading-bot/cmd"
	"github.com/saniales/golang-crypto-trading-bot/ema"
	"github.com/saniales/golang-crypto-trading-bot/mongo"
	"github.com/saniales/golang-crypto-trading-bot/strategies"
)

func main() {
	csvFile, err := os.Create(fmt.Sprintf(`/Users/nattapol.srikitpipat/Workspaces/go/src/golang-crypto-trading-bot/backtest/csv/coin-trans.csv`))
	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}
	defer csvFile.Close()

	csvwriter := csv.NewWriter(csvFile)
	logger := backtesthelps.NewLoggerBackTest(csvwriter)

	mactives := make(map[string]mongo.Actives)
	mongoRepo := &backtesthelps.MongoBackTest{Mactive: mactives}

	emas := ema.New(backtesthelps.NewChannelBackTest(), 10, time.Now(), 0, nil, mongoRepo, mactives, logger)

	strategies.AddCustomStrategy(emas.Running())
	bot.Execute()
}
