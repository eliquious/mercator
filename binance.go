package main

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
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
	scope := &binanceScope{prefix: "binance", description: "Access exchange info", client: client}
	rootCommand := &cobra.Command{Use: scope.prefix, Short: scope.description}
	rootCommand.Run = func(cmd *cobra.Command, args []string) {
		// env.Push(s)
	}

	rateCommand := &cobra.Command{
		Use:   "limits",
		Short: "API limits for the exchange",
		Run: func(cmd *cobra.Command, args []string) {
			exchange := client.NewExchangeInfoService()
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
	rootCommand.AddCommand(rateCommand)

	timeCommand := &cobra.Command{
		Use:   "time",
		Short: "Server time and timezone",
		Run: func(cmd *cobra.Command, args []string) {
			exchange := client.NewExchangeInfoService()
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
	rootCommand.AddCommand(timeCommand)

	// exchangeCommand := setupExchangeCommand(context.Background(), scope.client)
	// rootCommand.AddCommand(exchangeCommand)

	// accountCommand := setupAccountCommand(context.Background(), scope.client)
	// rootCommand.AddCommand(accountCommand)

	// addHelpCommand(rootCommand)
	addExitCommand(env, rootCommand)
	addQuitCommand(env, rootCommand)

	addAccountCommands(env, rootCommand, client)

	scope.command = rootCommand
	return scope, nil
}

type binanceScope struct {
	prefix      string
	description string
	client      *binance.Client
	command     *cobra.Command
}

func (s *binanceScope) GetScopeMeta() ScopeMeta {
	return ScopeMeta{s.prefix, s.description}
}

func (s *binanceScope) GetCommand() *cobra.Command {
	return s.command
}

func addAccountCommands(env *Environment, cmd *cobra.Command, client *binance.Client) {
	accountInfoCommand := &cobra.Command{
		Use:   "account-info",
		Short: "Show user account info",
		Run: func(cmd *cobra.Command, args []string) {
			exchange := client.NewGetAccountService()
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
			exchange := client.NewGetAccountService()
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
