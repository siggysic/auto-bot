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

package environment

import (
	"github.com/shopspring/decimal"
)

type PositionType string
type SignalType string
type TargetType string

const (
	BuyPosition  PositionType = "LONG"
	SellPosiiton PositionType = "SHORT"

	CoinPercentSignal SignalType = "COIN_PERCENT"

	FixedPercentTarget  TargetType = "FIXED_PERCENT"
	TrailingStopPercent TargetType = "TRAILING_STOP_PERCENT"
)

// ExchangeConfig Represents a configuration for an API Connection to an exchange.
//
//     Can be used to generate an ExchangeWrapper.
type ExchangeConfig struct {
	ExchangeName     string                     `yaml:"exchange"`          // Represents the exchange name.
	PublicKey        string                     `yaml:"public_key"`        // Represents the public key used to connect to Exchange API.
	SecretKey        string                     `yaml:"secret_key"`        // Represents the secret key used to connect to Exchange API.
	DepositAddresses map[string]string          `yaml:"deposit_addresses"` // Represents the bindings between coins and deposit address on the exchange.
	FakeBalances     map[string]decimal.Decimal `yaml:"fake_balances"`     // Used only in simulation mode, fake starting balance [coin:balance].
}

// StrategyConfig contains where a strategy will be applied in the specified exchange.
type StrategyConfig struct {
	Strategy             string         `yaml:"strategy"` // Represents the applied strategy name: must be unique in the system.
	DiscordToken         string         `yaml:"discord_token"`
	DiscordChannelId     string         `yaml:"discord_channel_id"`
	DiscordNotiGapMinute int            `yaml:"discord_noti_gap_minute"`
	Markets              []MarketConfig `yaml:"markets"` // Represents the exchanges where the strategy is applied.
}

// MarketConfig contains all market configuration data.
type MarketConfig struct {
	Name      string                   `yaml:"market"`   // Represents the market where the strategy is applied.
	Exchanges []ExchangeBindingsConfig `yaml:"bindings"` // Represents the list of markets where the strategy is applied, along with extra-data regarding binded exchanges.
}

// ExchangeBindingsConfig represents the binding of market names between bot notation and exchange ticker.
type ExchangeBindingsConfig struct {
	Name       string       `yaml:"exchange"`    // Represents the name of the exchange.
	MarketName string       `yaml:"market_name"` // Represents the name of the market as seen from the exchange.
	MarginType string       `yaml:"margin_type"`
	OrderType  string       `yaml:"order_type"`
	Rounds     []CoinRounds `yaml:"rounds"`
}

type CoinRounds struct {
	Round               int          `yaml:"round"`
	Amount              float64      `yaml:"amount"`
	PositionType        PositionType `yaml:"position_type"`
	Leverage            int          `yaml:"leverage"`
	Signal              float64      `yaml:"signal"`
	SignalType          SignalType   `yaml:"signal_type"`
	Target              float64      `yaml:"target"`
	TargetType          TargetType   `yaml:"target_type"`
	TrailingStopPercent *float64     `yaml:"trailing_acceptable_percent"`

	Interval string `yaml:"interval"`
	FastEMA  int    `yaml:"fast_ema"`
	SlowEMA  int    `yaml:"slow_ema"`
}

// BotConfig contains all config data of the bot, which can be also loaded from config file.
type BotConfig struct {
	MongoURI         string           `yaml:"mongo_uri"`
	SimulationModeOn bool             `yaml:"simulation_mode"`  // if true, do not create real orders and do not get real balance
	ExchangeConfigs  []ExchangeConfig `yaml:"exchange_configs"` // Represents the current exchange configuration.
	Strategies       []StrategyConfig `yaml:"strategies"`       // Represents the current strategies adopted by the bot.
}
