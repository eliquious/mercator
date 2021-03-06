package binance

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	binance "github.com/adshao/go-binance/v2"
	"github.com/eliquious/console"
	"github.com/eliquious/console/colors"
	"github.com/gookit/color"
	"github.com/olekukonko/tablewriter"
)

// NewBinanceExchangeScope creates a new scope for the Binance crypto exchange
func NewBinanceExchangeScope() (*console.Scope, error) {
	apiKey := os.Getenv("BINANCE_API_KEY")
	apiSecret := os.Getenv("BINANCE_API_SECRET")

	if apiKey == "" || apiSecret == "" {
		return nil, errors.New("Binance scope requires env variables: BINANCE_API_KEY and BINANCE_API_SECRET")
	}

	proxyUsername := os.Getenv("PROXY_USER")
	proxyPassword := os.Getenv("PROXY_PASS")
	if proxyPassword == "" || proxyUsername == "" {
		return nil, errors.New("Binance scope requires env variables: PROXY_USER and PROXY_PASS")
	}

	client := binance.NewClient(apiKey, apiSecret)
	client.HTTPClient = &http.Client{
		Transport: ProxyRoundTripper(nil, proxyUsername, proxyPassword),
	}

	exch := client.NewExchangeInfoService()
	resp, err := exch.Do(context.Background())
	if err != nil {
		log.Println(err)
		return nil, errors.New("failed to list symbols")
	}

	scope := console.NewScope("binance", "Access Binance exchange information")

	addRateLimitCommand(scope, client)
	addServerTimeCommand(scope, client)
	addPriceCommands(scope, client, resp.Symbols)
	addAccountCommands(scope, client, resp.Symbols)
	addDepthCommand(scope, client, resp.Symbols)
	addCalcSharesCommand(scope, client, resp.Symbols)
	addRiskCommand(scope, client, resp.Symbols)
	addCurrentValueCommand(scope, client, resp.Symbols)
	addHistoricalMarketTrades(scope, client, resp.Symbols)
	addRecentMarketTrades(scope, client, resp.Symbols)
	addAssetDetail(scope, client, resp.Symbols)
	addSymbolDetail(scope, client, resp.Symbols)
	addFutureValueCommand(scope, client, resp.Symbols)
	return scope, nil
}

type binanceScope struct {
	prefix      string
	description string
	client      *binance.Client
	symbols     []binance.Symbol
	// command     *cobra.Command
}

func addRateLimitCommand(scope *console.Scope, client *binance.Client) {
	rateCommand := &console.Command{
		Use:   "rate-limits",
		Short: "API limits for the exchange",
		Run: func(env *console.Environment, cmd *console.Command, args []string) error {
			exchange := client.NewExchangeInfoService()
			info, err := exchange.Do(context.Background())
			if err != nil {
				return err
			}

			for index := 0; index < len(info.RateLimits); index++ {
				limit := info.RateLimits[index]

				fmt.Printf("%s: %s\n  %s: %d\n  %s: %s\n\n",
					color.Green.Render("Interval"),
					limit.Interval,
					color.Green.Render("Limit"),
					limit.Limit,
					color.Green.Render("Type"),
					limit.RateLimitType,
				)
			}
			return nil
		},
	}
	scope.AddCommand(rateCommand)
}

func addServerTimeCommand(scope *console.Scope, client *binance.Client) {
	timeCommand := &console.Command{
		Use:   "server-time",
		Short: "Server time and timezone",
		Run: func(env *console.Environment, cmd *console.Command, args []string) error {
			exchange := client.NewExchangeInfoService()
			info, err := exchange.Do(context.Background())
			if err != nil {
				return err
			}

			fmt.Printf("%s: %s\n%s: %s\n",
				color.Green.Render("Server Time"),
				time.Unix(0, info.ServerTime*1e6).UTC().Format(time.RFC3339),
				color.Green.Render("Timezone"),
				info.Timezone,
			)
			return nil
		},
	}
	scope.AddCommand(timeCommand)
}

func addPriceCommands(scope *console.Scope, client *binance.Client, symbols []binance.Symbol) {
	priceCommand := &console.Command{
		Use:              "symbol-price",
		Short:            "Get the current price for the given symbols",
		EagerSuggestions: true,
		Suggestions: func(env *console.Environment, args []string) []string {
			return getSymbolList(symbols)
		},
		Run: func(env *console.Environment, cmd *console.Command, args []string) error {
			currentPrices, err := getCurrentPrices(client)
			if err != nil {
				return err
			}

			for _, arg := range args {
				price, ok := currentPrices[arg]
				if !ok {
					fmt.Printf("%s:  %s\n", color.LightGreen.Render(arg), color.Red.Render("unknown symbol"))
					continue
				}
				fmt.Printf("%s:  %s\n", color.LightGreen.Render(arg), price)
			}
			return nil
		},
	}
	scope.AddCommand(priceCommand)

	assetPricesCommand := &console.Command{
		Use:              "asset-price",
		Short:            "Get the all current prices for an asset",
		EagerSuggestions: true,
		Suggestions: func(env *console.Environment, args []string) []string {
			return getBaseAssetList(symbols)
		},
		Run: func(env *console.Environment, cmd *console.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("one asset must be given")
			}

			currentPrices, err := getCurrentPrices(client)
			if err != nil {
				return err
			}

			baseAssetMap := getBaseAssetMap(symbols)
			symbols, ok := baseAssetMap[args[0]]
			if !ok {
				return errors.New("unknown symbol: " + args[0])
			}

			for _, symbol := range symbols {
				price, ok := currentPrices[symbol.Symbol]
				if !ok {
					fmt.Printf("%s:  %s\n", color.LightGreen.Render(symbol.Symbol), color.LightRed.Render("unknown price"))
					continue
				}
				fmt.Printf("%s:  %s\n", color.LightGreen.Render(symbol.Symbol), price)
			}
			return nil
		},
	}
	scope.AddCommand(assetPricesCommand)

	comparePriceCommand := &console.Command{
		Use:   "compare",
		Short: "Compare price from one asset to another through an intermediary market",
		Long: `This converts the price from two markets and compares the price to the direct market price. For example, if
you want to know if the ETHUSDT price matches the ETHBTC/BTCUSDT price you can use this command.

    compare ETHUSDT ETHBTC BTCUSDT
		`,
		EagerSuggestions: true,
		Suggestions: func(env *console.Environment, args []string) []string {
			return getSymbolList(symbols)
		},
		Run: func(env *console.Environment, cmd *console.Command, args []string) error {
			if len(args) != 3 {
				return errors.New("three symbols must be given")
			}

			currentPrices, err := getCurrentPrices(client)
			if err != nil {
				return err
			}

			marketPrice, ok := currentPrices[args[0]]
			if !ok {
				return errors.New("unknown symbol: " + args[0])
			}
			fmt.Printf("%s:  %s\n", color.LightGreen.Render(args[1]), marketPrice)

			mp, err := strconv.ParseFloat(marketPrice, 64)
			if err != nil {
				return fmt.Errorf("could not convert price: %s %s", args[0], marketPrice)
			}

			p1, ok := currentPrices[args[1]]
			if !ok {
				// fmt.Printf("%s:  %s\n", color.LightGreen.Render(args[1]), color.Red.Render("unknown symbol"))
				return errors.New("unknown symbol: " + args[1])
			}
			fmt.Printf("%s:  %s\n", color.LightGreen.Render(args[1]), p1)

			p2, ok := currentPrices[args[2]]
			if !ok {
				// fmt.Printf("%s:  %s\n", color.LightGreen.Render(args[2]), color.Red.Render("unknown symbol"))
				return errors.New("unknown symbol: " + args[2])
			}
			fmt.Printf("%s:  %s\n", color.LightGreen.Render(args[2]), p2)

			c1, err := strconv.ParseFloat(p1, 64)
			if err != nil {
				// color.Error.Println("could not convert price: ", args[1], p1)
				return fmt.Errorf("could not convert price: %s %s", args[1], p1)
			}

			c2, err := strconv.ParseFloat(p2, 64)
			if err != nil {
				return fmt.Errorf("could not convert price: %s %s", args[2], p2)
			}

			if c2 <= 0 {
				color.Error.Println()
				return fmt.Errorf(args[2] + " has has went to 0")
			}
			fmt.Printf("\nConverted Price: %0.8f\n", c1*c2)

			diff := math.Abs(c1*c2 - mp)
			fmt.Printf("Difference:      %0.8f (%0.2f%%)\n", diff, diff/mp*100)

			fmt.Println("\nSuggestion:")
			if diff/mp*100 < 1.0 {
				fmt.Println("There's no opportunity here as the price difference is less than 1.0%%.")
			} else if (c1 * c2) < mp {
				fmt.Printf("Buy %s at %s (%0.8f) and sell %s at %s for a gain of %0.2f%%\n", args[1], p1, c1*c2, args[0], marketPrice, diff/mp*100)
			} else {
				fmt.Printf("Buy %s at %s and sell %s at %s (%0.8f) for a gain of %0.2f%%\n", args[0], marketPrice, args[1], p1, c1*c2, diff/mp*100)
			}
			return nil
		},
	}
	scope.AddCommand(comparePriceCommand)
}

func getCurrentPrices(client *binance.Client) (map[string]string, error) {
	exchange := client.NewListPricesService()
	resp, err := exchange.Do(context.Background())
	if err != nil {
		color.Error.Println(err.Error())
		return nil, err
	}

	currentPrices := make(map[string]string)
	for _, price := range resp {
		currentPrices[price.Symbol] = price.Price
	}
	return currentPrices, nil
}

func addAccountCommands(scope *console.Scope, client *binance.Client, symbols []binance.Symbol) {
	accountInfoCommand := &console.Command{
		Use:   "account-info",
		Short: "Show user account info",
		Run: func(env *console.Environment, cmd *console.Command, args []string) error {
			exchange := client.NewGetAccountService()
			resp, err := exchange.Do(context.Background())
			if err != nil {
				return err
			}

			fmt.Println("\nCommissions:")
			fmt.Printf("- %s:  %d\n", color.LightGreen.Render("Maker Commission"), resp.MakerCommission)
			fmt.Printf("- %s:  %d\n", color.LightGreen.Render("Taker Commission"), resp.TakerCommission)
			fmt.Printf("- %s:  %d\n", color.LightGreen.Render("Buyer Commission"), resp.BuyerCommission)
			fmt.Printf("- %s: %d\n", color.LightGreen.Render("Seller Commission"), resp.SellerCommission)
			fmt.Println("\nPermissions:")
			fmt.Printf("- %s:    %v\n", color.LightGreen.Render("Can Trade"), resp.CanTrade)
			fmt.Printf("- %s:  %v\n", color.LightGreen.Render("Can Deposit"), resp.CanDeposit)
			fmt.Printf("- %s: %v\n", color.LightGreen.Render("Can Withdraw"), resp.CanWithdraw)
			return nil
		},
	}
	scope.AddCommand(accountInfoCommand)

	accountBalanceCommand := &console.Command{
		Use:   "account-balance",
		Short: "Show user account balances",
		Run: func(env *console.Environment, cmd *console.Command, args []string) error {
			exchange := client.NewGetAccountService()
			resp, err := exchange.Do(context.Background())
			if err != nil {
				return err
			}

			balances := resp.Balances
			sort.Sort(OrderedBy(balances, byTotalBalance))

			color.LightWhite.Println("\nAccount Balance(s):")
			for index := 0; index < len(resp.Balances); index++ {
				balance := resp.Balances[index]

				f1, _ := strconv.ParseFloat(balance.Free, 64)
				l1, _ := strconv.ParseFloat(balance.Locked, 64)

				if f1 > 0 || l1 > 0 {
					fmt.Printf("%s:\n", color.LightGreen.Render(balance.Asset))
					fmt.Printf("  %s:     %s\n", color.LightYellow.Render("Free"), balance.Free)
					fmt.Printf("  %s:   %s\n", color.LightYellow.Render("Locked"), balance.Locked)
					fmt.Printf("  %s:    %0.8f\n", color.LightYellow.Render("Total"), f1+l1)
				}
			}
			return nil
		},
	}
	scope.AddCommand(accountBalanceCommand)

	var symbol string
	var limit int
	accountTradesCommand := &console.Command{
		Use:           "account-trades",
		Short:         "Show user account trades",
		RequiredFlags: []string{"symbol"},
		Suggestions: func(env *console.Environment, args []string) []string {
			if contains(args, "--symbol") && len(args) > 2 {
				return getSymbolList(symbols)
			}
			return []string{}
		},
		Run: func(env *console.Environment, cmd *console.Command, args []string) error {
			exchange := client.NewListTradesService()

			// filter by symbol if set
			if cmd.Flags().Changed("symbol") {
				exchange = exchange.Symbol(symbol)
			}

			// add query limit
			if cmd.Flags().Changed("limit") {
				exchange = exchange.Limit(limit)
			}

			trades, err := exchange.Do(context.Background())
			if err != nil {
				return err
			}

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"ID", "Timestamp", "Price", "Quantity", "Side"})

			// ID, timestamp,price,qty
			for index := 0; index < len(trades); index++ {
				trade := trades[index]

				var side string
				if trade.IsBuyer {
					side = color.Green.Render("BUY")
				} else {
					side = color.Red.Render("SELL")
				}

				row := []string{
					strconv.Itoa(int(trade.ID)),
					time.Unix(0, trade.Time*1e6).Local().Format("2006-01-02T15:04:05"),
					trade.Price,
					trade.Quantity,
					side,
				}
				table.Append(row)
			}
			table.Render() // Send output

			return nil
		},
	}
	accountTradesCommand.Flags().StringVar(&symbol, "symbol", "", "Filter trades by this symbol")
	accountTradesCommand.Flags().IntVar(&limit, "limit", 50, "Number of results to return")
	scope.AddCommand(accountTradesCommand)
}

func addDepthCommand(scope *console.Scope, client *binance.Client, symbols []binance.Symbol) {
	accountBalanceCommand := &console.Command{
		Use:              "depth",
		Short:            "Show symbol depth",
		EagerSuggestions: true,
		ValidateArgs:     console.ExactArgs(1),
		Suggestions: func(env *console.Environment, args []string) []string {
			return getSymbolList(symbols)
		},
		Run: func(env *console.Environment, cmd *console.Command, args []string) error {
			exchange := client.NewDepthService()

			resp, err := exchange.Symbol(strings.ToUpper(args[0])).Limit(10).Do(context.Background())
			if err != nil {
				return err
			}

			fmt.Println("\n      ", args[0], "Order Book")
			fmt.Println("------------------------------")
			for index := len(resp.Asks) - 1; index >= 0; index-- {
				ask := resp.Asks[index]
				quant, err := strconv.ParseFloat(ask.Quantity, 64)
				if err != nil {
					return err
				}
				fmt.Printf(" % 12s %s\n", color.Magenta.Render(ask.Price), padLeft(fmt.Sprintf("%0.4f", quant), " ", 15))
			}
			// fmt.Println("------------ -----------------")
			fmt.Println()
			for _, bid := range resp.Bids {
				quant, err := strconv.ParseFloat(bid.Quantity, 64)
				if err != nil {
					return err
				}
				fmt.Printf(" % 12s %s\n", color.Cyan.Render(bid.Price), padLeft(fmt.Sprintf("%0.4f", quant), " ", 15))
			}
			fmt.Println("------------ -----------------")
			fmt.Println()
			return nil
		},
	}
	scope.AddCommand(accountBalanceCommand)
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func addCalcSharesCommand(scope *console.Scope, client *binance.Client, symbols []binance.Symbol) {
	var inv, price float64
	command := &console.Command{
		Use:              "shares",
		Short:            "Calculate shares if bought at a certain price",
		EagerSuggestions: false,
		Suggestions: func(env *console.Environment, args []string) []string {
			if contains(args, "--inv") && len(args) > 2 {
				return getSymbolList(symbols)
			}
			return []string{}
		},
		Run: func(env *console.Environment, cmd *console.Command, args []string) error {
			if price == 0 && len(args) == 0 {
				return errors.New("either price or symbol is required")
			}

			// required flags
			if !cmd.Flags().Changed("inv") {
				return errors.New("investment amount is required")
			}

			if len(args) > 0 {
				prices, err := getCurrentPrices(client)
				if err != nil {
					return errors.New("either price or symbol is required")
				}

				marketPrice, ok := prices[strings.ToUpper(args[0])]
				if !ok {
					return errors.New("unknown symbol")
				}

				price, err = strconv.ParseFloat(marketPrice, 64)
				if err != nil {
					return errors.New("could not parse current price")
				}
			}

			if len(args) > 1 {
				color.Warn.Printf("more than one symbol provided. using %s", args[0])
			}

			if price == 0 {
				return errors.New("current price is 0.0")
			} else if price < 0 {
				return errors.New("price must be positive")
			}

			//
			if len(args) > 0 {
				info, err := getSymbolInfo(symbols, strings.ToUpper(args[0]))
				if err != nil {
					return err
				}

				fmt.Printf("%s: %s %s buys %s %s at %s\n",
					color.LightGreen.Render("Shares"),
					formatBasePrice(info, inv),
					color.LightBlue.Render(info.QuoteAsset),
					formatBasePrice(info, inv/price),
					color.LightBlue.Render(info.BaseAsset),
					formatQuotePrice(info, price),
				)
			} else {
				fmt.Printf("%s: %.8f at %.8f\n", color.Green.Render("Shares"), inv/price, price)
			}
			return nil
		},
	}
	command.Flags().Float64VarP(&inv, "inv", "i", 0, "Investment amount")
	command.Flags().Float64VarP(&price, "price", "p", 1, "Buy price")
	scope.AddCommand(command)
}

func addRiskCommand(scope *console.Scope, client *binance.Client, symbols []binance.Symbol) {
	var inv, entry, stop, ratio float64
	command := &console.Command{
		Use:          "risk",
		Short:        "Calculate risk if bought and sold at certain prices",
		ValidateArgs: console.ExactArgs(1),
		Suggestions: func(env *console.Environment, args []string) []string {
			return getSymbolList(symbols)
		},
		Run: func(env *console.Environment, cmd *console.Command, args []string) error {
			if entry <= 0 {
				return errors.New("entry price is required")
			}
			if stop <= 0 {
				return errors.New("stop price is required")
			} else if stop >= entry {
				return errors.New("stop price must be less than entry price")
			}
			if ratio <= 0 {
				return errors.New("risk/reward ratio must be greater than 0")
			}

			//
			info, err := getSymbolInfo(symbols, strings.ToUpper(args[0]))
			if err != nil {
				color.Error.Println(err.Error())
				return err
			}

			shares := inv / entry
			fmt.Printf("%s: %s %s buys %s %s at %s\n",
				color.Green.Render("Shares"),
				formatBasePrice(info, inv),
				color.LightBlue.Render(info.QuoteAsset),
				formatBasePrice(info, shares),
				color.LightBlue.Render(info.BaseAsset),
				formatQuotePrice(info, entry),
			)
			fmt.Printf("%s: %s %s\n",
				color.Green.Render("Risk"),
				formatBasePrice(info, shares*(entry-stop)),
				color.LightBlue.Render(info.QuoteAsset),
			)
			fmt.Printf("%s: %s %s if sold at %s %s\n",
				color.Green.Render("Earnings"),
				formatBasePrice(info, shares*(entry-stop)*ratio),
				color.LightBlue.Render(info.QuoteAsset),
				formatBasePrice(info, entry+(entry-stop)*ratio),
				color.LightBlue.Render(info.BaseAsset),
			)
			return nil
		},
		RequiredFlags: []string{"inv", "entry", "stop"},
	}
	command.Flags().Float64Var(&inv, "inv", 0, "Investment amount")
	command.Flags().Float64Var(&entry, "entry", 1, "Entry price")
	command.Flags().Float64Var(&stop, "stop", 1, "Stop price")
	command.Flags().Float64Var(&ratio, "ratio", 2, "Risk/reward ratio")
	scope.AddCommand(command)
}

func addCurrentValueCommand(scope *console.Scope, client *binance.Client, symbols []binance.Symbol) {
	var amount float64
	command := &console.Command{
		Use:           "current-value",
		Short:         "Get the current value of an asset for the given symbols",
		ValidateArgs:  console.MinimumArgs(1),
		RequiredFlags: []string{"amount"},
		Suggestions: func(env *console.Environment, args []string) []string {
			if contains(args, "--amount") && len(args) > 2 {
				return getSymbolList(symbols)
			}
			return []string{}
		},
		Run: func(env *console.Environment, cmd *console.Command, args []string) error {

			prices, err := getCurrentPrices(client)
			if err != nil {
				return errors.New("failed to get current prices")
			}

			for _, arg := range args {
				arg = strings.ToUpper(arg)
				marketPrice, ok := prices[strings.ToUpper(args[0])]
				if !ok {
					color.Error.Println("unknown symbol")
					continue
				}

				price, err := strconv.ParseFloat(marketPrice, 64)
				if err != nil {
					color.Error.Println("could not parse current price of ", arg)
					continue
				}

				info, err := getSymbolInfo(symbols, strings.ToUpper(args[0]))
				if err != nil {
					color.Error.Println(err.Error())
					continue
				}

				fmt.Printf("%s: %s\n",
					color.LightGreen.Render(arg),
					formatBasePrice(info, amount*price),
				)
			}
			return nil
		},
	}
	command.Flags().Float64Var(&amount, "amount", 1, "Amount of asset")
	scope.AddCommand(command)
}

func addFutureValueCommand(scope *console.Scope, client *binance.Client, symbols []binance.Symbol) {
	var amount, price float64
	command := &console.Command{
		Use:              "future-value",
		Short:            "Calculate value of shares if sold at a future price",
		EagerSuggestions: false,
		RequiredFlags:    []string{"amount", "price"},
		Run: func(env *console.Environment, cmd *console.Command, args []string) error {

			// required flags
			if !cmd.Flags().Changed("amount") {
				return errors.New("amount of shares is required")
			}

			if !cmd.Flags().Changed("price") {
				return errors.New("price is required")
			}

			if price == 0 {
				return errors.New("current price is 0.0")
			} else if price < 0 {
				return errors.New("price must be positive")
			}

			fmt.Printf("The %s shares would be valued at %s if sold at %s\n", formatShares(amount), formatValue(price*amount), formatPrice(price))
			return nil
		},
	}
	command.Flags().Float64VarP(&amount, "amount", "a", 0, "Number of shares")
	command.Flags().Float64VarP(&price, "price", "p", 1, "Buy price")
	scope.AddCommand(command)
}

func addHistoricalMarketTrades(scope *console.Scope, client *binance.Client, symbols []binance.Symbol) {
	var symbol string
	var limit int
	command := &console.Command{
		Use:           "historical-market-trades",
		Short:         "List the historical market trades",
		RequiredFlags: []string{"symbol"},
		Suggestions: func(env *console.Environment, args []string) []string {
			if contains(args, "--symbol") && len(args) > 2 {
				return getSymbolList(symbols)
			}
			return []string{}
		},
		Run: func(env *console.Environment, cmd *console.Command, args []string) error {
			exchange := client.NewHistoricalTradesService()

			// filter by symbol if set
			if cmd.Flags().Changed("symbol") {
				exchange = exchange.Symbol(symbol)
			}

			// add query limit
			if cmd.Flags().Changed("limit") {
				exchange = exchange.Limit(limit)
			}

			trades, err := exchange.Do(context.Background())
			if err != nil {
				return err
			}

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"ID", "Timestamp", "Price", "Quantity"})

			// ID, timestamp,price,qty
			for index := 0; index < len(trades); index++ {
				trade := trades[index]

				row := []string{
					strconv.Itoa(int(trade.ID)),
					time.Unix(0, trade.Time*1e6).Local().Format("2006-01-02T15:04:05"),
					trade.Price,
					trade.Quantity,
				}
				table.Append(row)

				// fmt.Printf("%s: %s\n  %s: %d\n  %s: %s\n\n",
				// 	color.Green.Render("Interval"),
				// 	limit.Interval,
				// 	color.Green.Render("Limit"),
				// 	limit.Limit,
				// 	color.Green.Render("Type"),
				// 	limit.RateLimitType,
				// )
			}
			table.Render() // Send output

			return nil
		},
	}
	command.Flags().StringVar(&symbol, "symbol", "", "Filter trades by this symbol")
	command.Flags().IntVar(&limit, "limit", 50, "Number of results to return")
	scope.AddCommand(command)
}

func addRecentMarketTrades(scope *console.Scope, client *binance.Client, symbols []binance.Symbol) {
	var symbol string
	var limit int
	command := &console.Command{
		Use:           "recent-market-trades",
		Short:         "List the recent market trades",
		RequiredFlags: []string{"symbol"},
		Suggestions: func(env *console.Environment, args []string) []string {
			if contains(args, "--symbol") && len(args) > 2 {
				return getSymbolList(symbols)
			}
			return []string{}
		},
		Run: func(env *console.Environment, cmd *console.Command, args []string) error {
			exchange := client.NewRecentTradesService()

			// filter by symbol if set
			if cmd.Flags().Changed("symbol") {
				exchange = exchange.Symbol(symbol)
			}

			// add query limit
			if cmd.Flags().Changed("limit") {
				exchange = exchange.Limit(limit)
			}

			trades, err := exchange.Do(context.Background())
			if err != nil {
				return err
			}

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"ID", "Timestamp", "Price", "Quantity"})

			// ID, timestamp,price,qty
			for index := 0; index < len(trades); index++ {
				trade := trades[index]

				row := []string{
					strconv.Itoa(int(trade.ID)),
					time.Unix(0, trade.Time*1e6).Local().Format("2006-01-02T15:04:05"),
					trade.Price,
					trade.Quantity,
				}
				table.Append(row)
			}
			table.Render() // Send output

			return nil
		},
	}
	command.Flags().StringVar(&symbol, "symbol", "", "Filter trades by this symbol")
	command.Flags().IntVar(&limit, "limit", 50, "Number of results to return")
	scope.AddCommand(command)
}

func addAssetDetail(scope *console.Scope, client *binance.Client, symbols []binance.Symbol) {
	var asset string
	command := &console.Command{
		Use:           "asset-detail",
		Short:         "Returns the asset details",
		RequiredFlags: []string{"asset"},
		Suggestions: func(env *console.Environment, args []string) []string {
			if contains(args, "--asset") && len(args) > 2 {
				return getBaseAssetList(symbols)
			}
			return []string{}
		},
		EagerSuggestions: false,
		Run: func(env *console.Environment, cmd *console.Command, args []string) error {
			exchange := client.NewGetAssetDetailService()

			resp, err := exchange.Do(context.Background())
			if err != nil {
				return err
			}

			if details, ok := resp[asset]; ok {

				var canDeposit, canWithdraw string
				if details.DepositStatus {
					canDeposit = colors.Green("true")
				} else {
					canDeposit = colors.Red("false")
				}
				if details.WithdrawStatus {
					canWithdraw = colors.Green("true")
				} else {
					canWithdraw = colors.Red("false")
				}

				fmt.Printf("Deposit Status: %v\nDeposit Tip: %s\nWithdraw Status: %v\nMinimum Withdraw Amount: %f\nWithdraw Fee: %f\n",
					canDeposit,
					details.DepositTip,
					canWithdraw,
					details.MinWithdrawAmount,
					details.WithdrawFee,
				)
			} else {
				return fmt.Errorf("unknown asset: %s", asset)
			}

			return nil
		},
	}
	command.Flags().StringVar(&asset, "asset", "", "Get the details for this asset")
	scope.AddCommand(command)
}

func addSymbolDetail(scope *console.Scope, client *binance.Client, symbols []binance.Symbol) {
	var symbol string
	command := &console.Command{
		Use:           "symbol-detail",
		Short:         "Returns the symbol details",
		RequiredFlags: []string{"symbol"},
		Suggestions: func(env *console.Environment, args []string) []string {
			if contains(args, "--symbol") && len(args) > 2 {
				return getSymbolList(symbols)
			}
			return []string{}
		},
		EagerSuggestions: false,
		Run: func(env *console.Environment, cmd *console.Command, args []string) error {
			s := getSymbolMap(symbols)

			if details, ok := s[symbol]; ok {
				// Status                 string                   `json:"status"`
				// BaseAsset              string                   `json:"baseAsset"`
				// BaseAssetPrecision     int                      `json:"baseAssetPrecision"`
				// QuoteAsset             string                   `json:"quoteAsset"`
				// QuotePrecision         int                      `json:"quotePrecision"`
				// OrderTypes             []string                 `json:"orderTypes"`
				// IcebergAllowed         bool                     `json:"icebergAllowed"`
				// OcoAllowed             bool                     `json:"ocoAllowed"`
				// IsSpotTradingAllowed   bool                     `json:"isSpotTradingAllowed"`
				// IsMarginTradingAllowed bool                     `json:"isMarginTradingAllowed"`

				fmt.Printf("Symbol Status: %v\nBase Asset: %s\nBase Asset Precision: %d\nQuote Asset: %s\nQuote Precision: %d\nIceberg Allowed: %s\nOCO Orders Allowed: %s\nSpot Trading: %s\nMargin Trading: %s\n",
					details.Status,
					details.BaseAsset,
					details.BaseAssetPrecision,
					details.QuoteAsset,
					details.QuotePrecision,
					formatBoolean(details.IcebergAllowed),
					formatBoolean(details.OcoAllowed),
					formatBoolean(details.IsSpotTradingAllowed),
					formatBoolean(details.IsMarginTradingAllowed),
				)

				fmt.Printf("\nSupported Order Types:\n%s\n\n", strings.Join(details.OrderTypes, "\n"))
			} else {
				return fmt.Errorf("unknown symbol: %s", symbol)
			}

			return nil
		},
	}
	command.Flags().StringVar(&symbol, "symbol", "", "Get the details for the symbol")
	scope.AddCommand(command)
}

func formatQuotePrice(symbol binance.Symbol, price float64) string {
	return fmt.Sprintf(getSymbolFormat(symbol.QuotePrecision), price)
}

func formatBasePrice(symbol binance.Symbol, price float64) string {
	return fmt.Sprintf(getSymbolFormat(symbol.BaseAssetPrecision), price)
}

func getSymbolFormat(precision int) string {
	return fmt.Sprintf("%%.%df", precision)
}

func getSymbolList(s []binance.Symbol) []string {
	symbols := make([]string, 0, len(s))
	for index := 0; index < len(s); index++ {
		symbol := s[index]
		symbols = append(symbols, symbol.Symbol)
	}
	sort.Strings(symbols)
	return symbols
}

func getBaseAssetList(symbols []binance.Symbol) []string {
	assets := make([]string, 0, len(symbols))
	for k := range getBaseAssetMap(symbols) {
		if len(strings.TrimSpace(k)) > 0 {
			assets = append(assets, k)
		}
	}
	sort.Strings(assets)
	return assets
}

func getSymbolMap(symbols []binance.Symbol) map[string]binance.Symbol {
	symbolMap := make(map[string]binance.Symbol, len(symbols))
	for index := 0; index < len(symbols); index++ {
		symbol := symbols[index]
		symbolMap[symbol.Symbol] = symbol
	}
	return symbolMap
}

func getSymbolInfo(symbols []binance.Symbol, symbol string) (binance.Symbol, error) {
	symbolMap := getSymbolMap(symbols)
	info, ok := symbolMap[symbol]
	if !ok {
		return info, errors.New("unknown symbol: " + symbol)
	}
	return info, nil
}

func getBaseAssetMap(symbols []binance.Symbol) map[string][]binance.Symbol {
	baseAssetMap := make(map[string][]binance.Symbol, len(symbols))
	for index := 0; index < len(symbols); index++ {
		symbol := symbols[index]
		if len(strings.TrimSpace(symbol.BaseAsset)) > 0 {
			baseAssetMap[symbol.BaseAsset] = append(baseAssetMap[symbol.BaseAsset], symbol)
		}
	}
	return baseAssetMap
}

func getQuoteAssetMap(symbols []binance.Symbol) map[string][]binance.Symbol {
	quoteAssetMap := make(map[string][]binance.Symbol, 16)
	for index := 0; index < len(symbols); index++ {
		symbol := symbols[index]
		if len(strings.TrimSpace(symbol.QuoteAsset)) > 0 {
			quoteAssetMap[symbol.QuoteAsset] = append(quoteAssetMap[symbol.QuoteAsset], symbol)
		}
	}
	return quoteAssetMap
}

func formatBoolean(val bool) string {
	if val {
		return colors.Green("true")
	}
	return colors.Red("false")
}

func formatValue(val float64) string {
	return colors.Green(fmt.Sprintf("%.8f", val))
}

func formatPrice(val float64) string {
	return colors.LightBlue(fmt.Sprintf("%.8f", val))
}

func formatShares(val float64) string {
	return colors.Yellow(fmt.Sprintf("%.8f", val))
}

func padLeft(str, pad string, length int) string {
	for {
		str = pad + str
		if len(str) > length {
			return str[0:length]
		}
	}
}

func padRight(str, pad string, length int) string {
	for {
		str += pad
		if len(str) > length {
			return str[0:length]
		}
	}
}
