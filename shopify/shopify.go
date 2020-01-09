package shopify

import (
	"fmt"
	"math"

	"github.com/eliquious/console"
)

// NewShopifyScope creates a new Shopify scope for the CLI.
func NewShopifyScope() (*console.Scope, error) {
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
			console.PrintInfo("CPM", "$%.2f", costPerMille)

			clickThroughRate := env.Configuration.GetFloat64("shopify.ctr")
			console.PrintInfo("CTR", "%.2f%%", clickThroughRate*100)

			conversionRate := env.Configuration.GetFloat64("shopify.conv")
			console.PrintInfo("Conversion Rate", "%.2f%%", conversionRate*100)
			fmt.Println()

			productCost, err := cmd.Flags().GetFloat64("cost")
			if err != nil {
				return err
			}

			productTotal, err := cmd.Flags().GetFloat64("price")
			if err != nil {
				return err
			}

			earningsGoal, err := cmd.Flags().GetFloat64("goal")
			if err != nil {
				return err
			}

			revenuePerSale := productTotal - productCost
			console.PrintInfo("Gross Earnings Goal", "$%0.2f", earningsGoal)
			console.PrintInfo("Product Total", "$%0.2f", productTotal)
			console.PrintInfo("Product Cost", "$%0.2f", productCost)
			console.PrintInfo("Revenue per Sale", "$%0.2f", revenuePerSale)
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

			console.PrintInfo("Required Gross Sales", "%.0f", sales)
			console.PrintInfo("Required Visitors", "%.0f", visitors)
			console.PrintInfo("Required Ad Impressions", "%.0f", adImpressions)
			fmt.Println()

			console.PrintInfo("Gross", "$%.2f", gross)
			console.PrintInfo("Total Product Cost", "$%.2f", productExpenses)
			console.PrintInfo("Required Marketing Budget", "$%.2f", marketingBudget)
			console.PrintInfo("Net Revenue", "$%.2f", revenue)
			fmt.Println()

			console.PrintInfo("Profit/Marketing Ratio", "%.4f", revenue/marketingBudget)
			console.PrintInfo("Profit/Expenses Ratio", "%.4f", revenue/(marketingBudget+productExpenses))
			console.PrintInfo("Marketing Cost per Visitor", "$%.2f", marketingBudget/visitors)
			console.PrintInfo("Marketing Cost per Purchase", "$%.2f", costPerPurchase)
			console.PrintInfo("Profit per Sale", "$%.2f", revenuePerSale-costPerPurchase)
			fmt.Println()
			return nil
		},
	}
	revenueCommand.Flags().Float64("cost", 1, "Product cost")
	revenueCommand.Flags().Float64("price", 1, "Product sale price")
	revenueCommand.Flags().Float64("goal", 1000, "Sales goal")
	shopify.AddCommand(revenueCommand)
	return shopify, nil
}
