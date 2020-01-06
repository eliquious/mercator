package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/adshao/go-binance"
	"github.com/gookit/color"
	"github.com/spf13/cobra"
)

// NewBinanceExchangeScope creates a new scope for the Binance crypto exchange
func NewBinanceExchangeScope(env *Environment, apiKey string, apiSecret string) (Scope, error) {
	if apiKey == "" || apiSecret == "" {
		return nil, errors.New("Binance scope requires env variables: BINANCE_API_KEY and BINANCE_API_SECRET")
	}

	client := binance.NewClient(apiKey, apiSecret)

	exch := client.NewExchangeInfoService()
	resp, err := exch.Do(context.Background())
	if err != nil {
		return nil, errors.New("failed to list symbols")
	}

	scope := &binanceScope{prefix: "binance", description: "Access exchange info", client: client, symbols: resp.Symbols}
	rootCommand := &cobra.Command{Use: scope.prefix, Short: scope.description}

	scope.addRateLimitCommand(env, rootCommand)
	scope.addServerTimeCommand(env, rootCommand)
	scope.addAccountCommands(env, rootCommand)
	scope.addPriceCommands(env, rootCommand)
	scope.addDepthCommand(env, rootCommand)
	scope.addCalcSharesCommand(env, rootCommand)
	scope.addRiskCommand(env, rootCommand)
	scope.addCconvertCommand(env, rootCommand)

	addExitCommand(env, rootCommand)
	addQuitCommand(env, rootCommand)

	scope.command = rootCommand
	return scope, nil
}

type binanceScope struct {
	prefix      string
	description string
	client      *binance.Client
	symbols     []binance.Symbol
	command     *cobra.Command
}

func (s *binanceScope) GetScopeMeta() ScopeMeta {
	return ScopeMeta{s.prefix, s.description}
}

func (s *binanceScope) GetCommand() *cobra.Command {
	return s.command
}

func (s *binanceScope) addRateLimitCommand(env *Environment, cmd *cobra.Command) {
	rateCommand := &cobra.Command{
		Use:   "rate-limits",
		Short: "API limits for the exchange",
		Run: func(cmd *cobra.Command, args []string) {
			exchange := s.client.NewExchangeInfoService()
			info, err := exchange.Do(context.Background())
			if err != nil {
				color.Error.Println(err.Error())
				return
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
		},
	}
	cmd.AddCommand(rateCommand)
}

func (s *binanceScope) addServerTimeCommand(env *Environment, cmd *cobra.Command) {
	timeCommand := &cobra.Command{
		Use:   "server-time",
		Short: "Server time and timezone",
		Run: func(cmd *cobra.Command, args []string) {
			exchange := s.client.NewExchangeInfoService()
			info, err := exchange.Do(context.Background())
			if err != nil {
				color.Error.Println(err.Error())
				return
			}

			fmt.Printf("%s: %s\n%s: %s\n",
				color.Green.Render("Server Time"),
				time.Unix(0, info.ServerTime*1e6).UTC().Format(time.RFC3339),
				color.Green.Render("Timezone"),
				info.Timezone,
			)
		},
	}
	cmd.AddCommand(timeCommand)
}

func (s *binanceScope) addPriceCommands(env *Environment, cmd *cobra.Command) {
	priceCommand := &cobra.Command{
		Use:       "symbol-price",
		Short:     "Get the current price for the given symbols",
		Args:      cobra.MinimumNArgs(1),
		ValidArgs: s.getSymbolList(),
		Run: func(cmd *cobra.Command, args []string) {
			currentPrices, err := s.getCurrentPrices()
			if err != nil {
				color.Error.Println(err.Error())
				return
			}

			for _, arg := range args {
				price, ok := currentPrices[arg]
				if !ok {
					fmt.Printf("%s:  %s\n", color.LightGreen.Render(arg), color.Red.Render("unknown symbol"))
					continue
				}
				fmt.Printf("%s:  %s\n", color.LightGreen.Render(arg), price)
			}
		},
	}
	cmd.AddCommand(priceCommand)

	assetPricesCommand := &cobra.Command{
		Use:       "asset-price",
		Short:     "Get the all current prices for an asset",
		Args:      cobra.MaximumNArgs(1),
		ValidArgs: s.getBaseAssetList(),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				color.Error.Println("one asset must be given")
				return
			}

			currentPrices, err := s.getCurrentPrices()
			if err != nil {
				color.Error.Println(err.Error())
				return
			}

			baseAssetMap := s.getBaseAssetMap()
			symbols, ok := baseAssetMap[args[0]]
			if !ok {
				color.Error.Printf("unknown symbol: %s", args[0])
				return
			}

			for _, symbol := range symbols {
				price, ok := currentPrices[symbol.Symbol]
				if !ok {
					fmt.Printf("%s:  %s\n", color.LightGreen.Render(symbol.Symbol), color.LightRed.Render("unknown price"))
					continue
				}
				fmt.Printf("%s:  %s\n", color.LightGreen.Render(symbol.Symbol), price)
			}
		},
	}
	cmd.AddCommand(assetPricesCommand)

	comparePriceCommand := &cobra.Command{
		Use:   "compare",
		Short: "Compare price from one asset to another through an intermediary market",
		Long: `This converts the price from two markets and compares the price to the direct market price. For example, if 
you want to know if the ETHUSDT price matches the ETHBTC/BTCUSDT price you can use this command.

    compare ETHUSDT ETHBTC BTCUSDT
		`,
		Args:      cobra.ExactArgs(3),
		ValidArgs: s.getSymbolList(),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				color.Error.Println("three symbols must be given")
				return
			}

			currentPrices, err := s.getCurrentPrices()
			if err != nil {
				color.Error.Println(err.Error())
				return
			}

			marketPrice, ok := currentPrices[args[0]]
			if !ok {
				fmt.Printf("%s:  %s\n", color.LightGreen.Render(args[0]), color.Red.Render("unknown symbol"))
				return
			}
			fmt.Printf("%s:  %s\n", color.LightGreen.Render(args[1]), marketPrice)

			mp, err := strconv.ParseFloat(marketPrice, 64)
			if err != nil {
				color.Error.Println("could not convert price: ", args[1], marketPrice)
				return
			}

			p1, ok := currentPrices[args[1]]
			if !ok {
				fmt.Printf("%s:  %s\n", color.LightGreen.Render(args[1]), color.Red.Render("unknown symbol"))
				return
			}
			fmt.Printf("%s:  %s\n", color.LightGreen.Render(args[1]), p1)

			p2, ok := currentPrices[args[2]]
			if !ok {
				fmt.Printf("%s:  %s\n", color.LightGreen.Render(args[2]), color.Red.Render("unknown symbol"))
				return
			}
			fmt.Printf("%s:  %s\n", color.LightGreen.Render(args[2]), p2)

			c1, err := strconv.ParseFloat(p1, 64)
			if err != nil {
				color.Error.Println("could not convert price: ", args[1], p1)
				return
			}

			c2, err := strconv.ParseFloat(p2, 64)
			if err != nil {
				color.Error.Println("could not convert price: ", args[2], p2)
				return
			}

			if c2 <= 0 {
				color.Error.Println(args[2], "has has went to 0")
				return
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
		},
	}
	cmd.AddCommand(comparePriceCommand)
}

func (s *binanceScope) getCurrentPrices() (map[string]string, error) {
	exchange := s.client.NewListPricesService()
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

func (s *binanceScope) addAccountCommands(env *Environment, cmd *cobra.Command) {
	accountInfoCommand := &cobra.Command{
		Use:   "account-info",
		Short: "Show user account info",
		Run: func(cmd *cobra.Command, args []string) {
			exchange := s.client.NewGetAccountService()
			resp, err := exchange.Do(context.Background())
			if err != nil {
				color.Error.Println(err.Error())
				return
			}

			fmt.Println("\nCommissions:")
			fmt.Printf("- %s:  %d\n", color.LightGreen.Render("Maker Commission"), resp.MakerCommission)
			fmt.Printf("- %s:  %d\n", color.LightGreen.Render("Taker Commission"), resp.TakerCommission)
			fmt.Printf("- %s:  %d\n", color.LightGreen.Render("Buyer Commission"), resp.BuyerCommission)
			fmt.Printf("- %s: %d\n", color.LightGreen.Render("Seller Commission"), resp.SellerCommission)
			fmt.Println("\nPermissions:")
			fmt.Printf("- %s: %v\n", color.LightGreen.Render("Can Trade"), resp.CanTrade)
			fmt.Printf("- %s: %v\n", color.LightGreen.Render("Can Trade"), resp.CanDeposit)
			fmt.Printf("- %s: %v\n", color.LightGreen.Render("Can Trade"), resp.CanWithdraw)
		},
	}
	cmd.AddCommand(accountInfoCommand)

	accountBalanceCommand := &cobra.Command{
		Use:   "account-balance",
		Short: "Show user account balances",
		Run: func(cmd *cobra.Command, args []string) {
			exchange := s.client.NewGetAccountService()
			resp, err := exchange.Do(context.Background())
			if err != nil {
				color.Error.Println(err.Error())
				return
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
		},
	}
	cmd.AddCommand(accountBalanceCommand)
}

func (s *binanceScope) addDepthCommand(env *Environment, cmd *cobra.Command) {
	accountBalanceCommand := &cobra.Command{
		Use:       "depth",
		Short:     "Show symbol depth",
		Args:      cobra.ExactArgs(1),
		ValidArgs: s.getSymbolList(),
		Run: func(cmd *cobra.Command, args []string) {
			exchange := s.client.NewDepthService()
			resp, err := exchange.Symbol(strings.ToUpper(args[0])).Limit(10).Do(context.Background())
			if err != nil {
				color.Error.Println(err.Error())
				return
			}

			fmt.Println("\n      ", args[0], "Order Book")
			fmt.Println("------------------------------")
			for index := len(resp.Asks) - 1; index >= 0; index-- {
				ask := resp.Asks[index]
				quant, err := strconv.ParseFloat(ask.Quantity, 64)
				if err != nil {
					color.Error.Println(err.Error())
					return
				}
				fmt.Printf(" % 12s %s\n", color.Magenta.Render(ask.Price), padLeft(fmt.Sprintf("%0.4f", quant), " ", 15))
			}
			// fmt.Println("------------ -----------------")
			fmt.Println()
			for _, bid := range resp.Bids {
				quant, err := strconv.ParseFloat(bid.Quantity, 64)
				if err != nil {
					color.Error.Println(err.Error())
					return
				}
				fmt.Printf(" % 12s %s\n", color.Cyan.Render(bid.Price), padLeft(fmt.Sprintf("%0.4f", quant), " ", 15))
			}
			fmt.Println("------------ -----------------")
			fmt.Println()
		},
	}
	cmd.AddCommand(accountBalanceCommand)
}

func (s *binanceScope) addCalcSharesCommand(env *Environment, cmd *cobra.Command) {
	var inv, price float64
	// var symbol string
	command := &cobra.Command{
		Use:   "shares",
		Short: "Calculate shares if bought at a certain price",
		Run: func(cmd *cobra.Command, args []string) {
			if price == 0 && len(args) == 0 {
				color.Error.Println("either price or symbol is required")
				return
			}

			if len(args) > 0 {
				prices, err := s.getCurrentPrices()
				if err != nil {
					color.Error.Println("either price or symbol is required")
					return
				}

				marketPrice, ok := prices[strings.ToUpper(args[0])]
				if !ok {
					color.Error.Println("unknown symbol")
					return
				}

				price, err = strconv.ParseFloat(marketPrice, 64)
				if err != nil {
					color.Error.Println("could not parse current price")
					return
				}
			}

			if len(args) > 1 {
				color.Warn.Printf("more than one symbol provided. using %s", args[0])
			}

			if price == 0 {
				color.Error.Println("current price is 0.0")
				return
			} else if price < 0 {
				color.Error.Println("price must be positive")
				return
			}

			//
			if len(args) > 0 {
				info, err := s.getSymbolInfo(strings.ToUpper(args[0]))
				if err != nil {
					color.Error.Println(err.Error())
					return
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
		},
	}
	command.Flags().Float64VarP(&inv, "inv", "i", 0, "Investment amount")
	command.Flags().Float64VarP(&price, "price", "p", 1, "Buy price")
	command.MarkFlagRequired("inv")
	command.ValidArgs = s.getSymbolList()
	cmd.AddCommand(command)
}

func (s *binanceScope) addRiskCommand(env *Environment, cmd *cobra.Command) {
	var inv, entry, stop, ratio float64
	command := &cobra.Command{
		Use:   "risk",
		Short: "Calculate risk if bought and sold at certain prices",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if entry <= 0 {
				color.Error.Println("entry price is required")
				return
			}
			if stop <= 0 {
				color.Error.Println("stop price is required")
				return
			} else if stop >= entry {
				color.Error.Println("stop price must be less than entry price")
				return
			}
			if ratio <= 0 {
				color.Error.Println("risk/reward ratio must be greater than 0")
				return
			}

			//
			info, err := s.getSymbolInfo(strings.ToUpper(args[0]))
			if err != nil {
				color.Error.Println(err.Error())
				return
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
		},
	}
	command.Flags().Float64Var(&inv, "inv", 0, "Investment amount")
	command.Flags().Float64Var(&entry, "entry", 1, "Entry price")
	command.Flags().Float64Var(&stop, "stop", 1, "Stop price")
	command.Flags().Float64Var(&ratio, "ratio", 2, "Risk/reward ratio")
	command.MarkFlagRequired("inv")
	command.MarkFlagRequired("entry")
	command.MarkFlagRequired("stop")
	command.ValidArgs = s.getSymbolList()
	cmd.AddCommand(command)
}

func (s *binanceScope) addCconvertCommand(env *Environment, cmd *cobra.Command) {
	var amount float64
	command := &cobra.Command{
		Use:   "convert",
		Short: "Convert returns the value of an asset for the given symbols",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {

			prices, err := s.getCurrentPrices()
			if err != nil {
				color.Error.Println("failed toget current prices")
				return
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

				info, err := s.getSymbolInfo(strings.ToUpper(args[0]))
				if err != nil {
					color.Error.Println(err.Error())
					continue
				}

				fmt.Printf("%s: %s\n",
					color.LightGreen.Render(arg),
					formatBasePrice(info, amount*price),
				)
			}
		},
	}
	command.Flags().Float64Var(&amount, "amount", 1, "Amount of asset")
	command.MarkFlagRequired("amount")
	command.ValidArgs = s.getSymbolList()
	cmd.AddCommand(command)
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

func (s *binanceScope) getSymbolList() []string {
	symbols := make([]string, len(s.symbols))
	for index := 0; index < len(s.symbols); index++ {
		symbol := s.symbols[index]
		symbols = append(symbols, symbol.Symbol)
	}
	return symbols
}

func (s *binanceScope) getBaseAssetList() []string {
	assets := make([]string, len(s.symbols))
	for k := range s.getBaseAssetMap() {
		assets = append(assets, k)
	}
	return assets
}

func (s *binanceScope) getSymbolMap() map[string]binance.Symbol {
	symbolMap := make(map[string]binance.Symbol, len(s.symbols))
	for index := 0; index < len(s.symbols); index++ {
		symbol := s.symbols[index]
		symbolMap[symbol.Symbol] = symbol
	}
	return symbolMap
}

func (s *binanceScope) getSymbolInfo(symbol string) (binance.Symbol, error) {
	symbolMap := s.getSymbolMap()
	info, ok := symbolMap[symbol]
	if !ok {
		return info, errors.New("unknown symbol: " + symbol)
	}
	return info, nil
}

func (s *binanceScope) getBaseAssetMap() map[string][]binance.Symbol {
	baseAssetMap := make(map[string][]binance.Symbol, len(s.symbols))
	for index := 0; index < len(s.symbols); index++ {
		symbol := s.symbols[index]
		baseAssetMap[symbol.BaseAsset] = append(baseAssetMap[symbol.BaseAsset], symbol)
	}
	return baseAssetMap
}

func (s *binanceScope) getQuoteAssetMap() map[string][]binance.Symbol {
	quoteAssetMap := make(map[string][]binance.Symbol, 16)
	for index := 0; index < len(s.symbols); index++ {
		symbol := s.symbols[index]
		quoteAssetMap[symbol.QuoteAsset] = append(quoteAssetMap[symbol.QuoteAsset], symbol)
	}
	return quoteAssetMap
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
