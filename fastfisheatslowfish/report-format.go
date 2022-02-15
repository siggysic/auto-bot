package fastfisheatslowfish

import (
	"fmt"

	"github.com/saniales/golang-crypto-trading-bot/environment"
)

func reportInitialBot(markets []*environment.Market) string {
	coins := ""
	for ind, market := range markets {
		coins += fmt.Sprintf("%d. %s\n", ind+1, market.Name)
	}

	return fmt.Sprintf(`
		Bot start watchs:
			%s
	`, coins)
}

// func reportPriceMarket(cur string, high, low, vol, ask, bid, las decimal.Decimal) string {
// 	return fmt.Sprintf(`
// 		📈 [%s] in Binance 📈
// 			high	-> %f
// 			low		-> %f
// 			vol		-> %f
// 			ask		-> %f
// 			bid		-> %f
// 			last	-> %f
// 	`, cur, high.BigFloat(), low.BigFloat(), vol.BigFloat(), ask.BigFloat(), bid.BigFloat(), las.BigFloat())
// }

func reportBuyMarket(cur string, cOrderId string, orderId int64, orderType string, leverage int, side string, orderStatus string, amount string) string {
	return actionFormat("LONG", cur, cOrderId, orderId, orderType, leverage, side, orderStatus, amount)
}

func reportSellMarket(cur string, cOrderId string, orderId int64, orderType string, leverage int, side string, orderStatus string, amount string) string {
	return actionFormat("SHORT", cur, cOrderId, orderId, orderType, leverage, side, orderStatus, amount)
}

func reportClosePosition(cur string, cOrderId string, orderId int64, orderType string, orderStatus string, price string, profit string, roe float64, round int) string {
	return closeActionFormat(cur, cOrderId, orderId, orderType, orderStatus, price, profit, roe, round)
}

func reportCurrentPositions(cur string, amount string, profit string, roe float64, lerverage string, side string) string {
	return fmt.Sprintf(`
		⏰ [POSITION] %s (%s) ⏰
			Amount			-> %s
			Profit			-> %s x%s (%f%%)
	`, cur, side, amount, profit, lerverage, roe)
}

func actionFormat(action string, cur string, cOrderId string, orderId int64, orderType string, leverage int, side string, orderStatus string, amount string) string {
	return fmt.Sprintf(`
		💸 [%s] %s success!! 💸
			Client order id	-> %s
			Order id		-> %d
			Order type		-> %s
			Order status	-> %s
			Amount			-> %s
			Leverage		-> %d
			Side			-> %s
	`, action, cur, cOrderId, orderId, orderType, orderStatus, amount, leverage, side)
}

func closeActionFormat(cur string, cOrderId string, orderId int64, side string, orderStatus string, price string, profit string, roe float64, round int) string {
	return fmt.Sprintf(`
		💰 [CLOSE POSITION] %s success!! 💰
			Client order id	-> %s
			Order id		-> %d
			Order status	-> %s
			Profit			-> %s USDT
			ROE				-> %f %%
			Price			-> %s
			Side			-> %s
			Round			-> %d
	`, cur, cOrderId, orderId, orderStatus, profit, roe, price, side, round)
}
