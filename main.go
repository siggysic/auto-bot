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

package main

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/saniales/golang-crypto-trading-bot/channel"
	bot "github.com/saniales/golang-crypto-trading-bot/cmd"
	"github.com/saniales/golang-crypto-trading-bot/fastfisheatslowfish"
	"github.com/saniales/golang-crypto-trading-bot/logger"
	"github.com/saniales/golang-crypto-trading-bot/mongo"
	"github.com/saniales/golang-crypto-trading-bot/strategies"
	"github.com/spf13/viper"
)

func init() {
	err := newConfigReader()
	if err != nil {
		panic(err)
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	token := viper.GetString("discord_token")
	channelId := viper.GetString("discord_channel_id")
	discordNotiGapMinute := viper.GetInt64("discord_noti_gap_minute")
	mongoURI := viper.GetString("mongo_uri")
	discordNotiTime := time.Now()
	interval := 10 * time.Second

	// Discord channel
	discordBot := channel.NewDiscord(token, channelId)
	err := discordBot.InitChannel()
	if err != nil {
		panic(err)
	}

	// Mongo
	mongoDB, err := initMongoDB(mongoURI)
	if err != nil {
		panic(err)
	}
	mongoRepo := mongo.NewMongoRepository(mongoDB)
	mactives := make(map[string]mongo.Actives)
	logger := logger.NewGenericLogger()

	fastFishEatAll := fastfisheatslowfish.New(discordBot, discordNotiGapMinute, discordNotiTime, interval, mongoDB, mongoRepo, mactives, logger)

	strategies.AddCustomStrategy(fastFishEatAll.Running())
	bot.Execute()
}

func newConfigReader() error {
	viper.AddConfigPath(".")
	viper.SetConfigName(".bot_config")

	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

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
