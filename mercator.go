package main

import (
	"fmt"
	"math"

	"github.com/spf13/pflag"

	"github.com/eliquious/console"
	"github.com/gookit/color"
)

func main() {

	c := console.New("mercator", console.WithTitleScreen(printASCII))

	shopify := console.NewScope("shopify", "Utilities for managing shopify account")
	shopify.InitializeFunc = func(env *console.Environment) {
		env.Configuration.SetDefault("shopify.cpm", 6.2)
		env.Configuration.SetDefault("shopify.ctr", 0.0259)
		env.Configuration.SetDefault("shopify.conv", 0.03)
	}

	revenueCommand := &console.Command{
		Use:   "revenue",
		Short: "Calculates estimated revenue based on projections",
		Run: func(env *console.Environment, cmd *console.Command, args []string) error {
			costPerMille := env.Configuration.GetFloat64("shopify.cpm")
			PrintInfo("CPM", "$%.2f", costPerMille)

			clickThroughRate := env.Configuration.GetFloat64("shopify.ctr")
			PrintInfo("CTR", "%.2f%%", clickThroughRate*100)

			conversionRate := env.Configuration.GetFloat64("shopify.conv")
			PrintInfo("Conversion Rate", "%.2f%%", conversionRate*100)
			fmt.Println()

			productCost, err := cmd.Flags.GetFloat64("cost")
			if err != nil {
				return err
			}

			productTotal, err := cmd.Flags.GetFloat64("price")
			if err != nil {
				return err
			}

			earningsGoal, err := cmd.Flags.GetFloat64("goal")
			if err != nil {
				return err
			}

			revenuePerSale := productTotal - productCost
			PrintInfo("Gross Earnings Goal", "$%0.2f", earningsGoal)
			PrintInfo("Product Total", "$%0.2f", productTotal)
			PrintInfo("Product Cost", "$%0.2f", productCost)
			PrintInfo("Revenue per Sale", "$%0.2f", revenuePerSale)
			fmt.Println()

			// Sales per month
			var sales = math.Floor(earningsGoal/productTotal + 1)

			// Required Visitors
			var visitors = sales / conversionRate
			var adImpressions = visitors / clickThroughRate

			// Product Cost + Earnings
			var gross = sales * productTotal
			var productExpenses = sales * productCost
			var marketingBudget = adImpressions / 1000.0 * costPerMille
			var revenue = gross - productExpenses - marketingBudget
			var costPerPurchase = marketingBudget / float64(sales)

			PrintInfo("Required Gross Sales", "%.0f", sales)
			PrintInfo("Required Visitors", "%.0f", visitors)
			PrintInfo("Required Ad Impressions", "%.0f", adImpressions)
			fmt.Println()

			PrintInfo("Gross", "$%.2f", gross)
			PrintInfo("Total Product Cost", "$%.2f", productExpenses)
			PrintInfo("Required Marketing Budget", "$%.2f", marketingBudget)
			PrintInfo("Net Revenue", "$%.2f", revenue)
			fmt.Println()

			PrintInfo("Profit/Marketing Ratio", "%.4f", revenue/marketingBudget)
			PrintInfo("Profit/Expenses Ratio", "%.4f", revenue/(marketingBudget+productExpenses))
			PrintInfo("Marketing Cost per Visitor", "$%.2f", marketingBudget/visitors)
			PrintInfo("Marketing Cost per Purchase", "$%.2f", costPerPurchase)
			PrintInfo("Profit per Sale", "$%.2f", revenuePerSale-costPerPurchase)
			fmt.Println()
			return nil
		},
		Flags: pflag.NewFlagSet("revenue", pflag.ContinueOnError),
	}
	revenueCommand.Flags.Float64("cost", 1, "Product cost")
	revenueCommand.Flags.Float64("price", 1, "Product sale price")
	revenueCommand.Flags.Float64("goal", 1000, "Sales goal")
	shopify.AddCommand(revenueCommand)
	c.AddScope(shopify)
	c.Run()
}

// PrintInfo prints info with a green label
func PrintInfo(label string, format string, value ...interface{}) {
	fmt.Printf("%s: %s\n", color.LightGreen.Render(label), fmt.Sprintf(format, value...))
}

func printASCII() {
	fmt.Println()
	fmt.Println(`                                                        888                   `)
	fmt.Println(`                                                        888                   `)
	fmt.Println(`                                                        888                   `)
	fmt.Println(`        88888b.d88b.   .d88b.  888d888 .d8888b  8888b.  888888 .d88b.  888d888`)
	fmt.Println(`        888 "888 "88b d8P  Y8b 888P"  d88P"        "88b 888   d88""88b 888P"  `)
	fmt.Println(`        888  888  888 88888888 888    888      .d888888 888   888  888 888    `)
	fmt.Println(`        888  888  888 Y8b.     888    Y88b.    888  888 Y88b. Y88..88P 888    `)
	fmt.Println(`        888  888  888  "Y8888  888     "Y8888P "Y888888  "Y888 "Y88P"  888    `)
	fmt.Println()
	color.FgGray.Println(`                      a personal CLI for financial things`)
	color.Green.Println(`                                by @eliquious`)
	fmt.Println()
}
