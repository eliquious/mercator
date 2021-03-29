package main

import (
	"github.com/eliquious/console"
	"github.com/eliquious/console/ext/js"
	"github.com/eliquious/mercator/binance"
	"github.com/eliquious/mercator/shopify"
	"github.com/gookit/color"
)

func main() {
	c := console.New("mercator", console.WithTitleScreen(printASCII))

	// add shopify scope
	shopify, err := shopify.NewShopifyScope()
	if err != nil {
		color.Error.Println(err)
		return
	}
	c.AddScope(shopify)

	// add binance scope
	binance, err := binance.NewBinanceExchangeScope()
	if err != nil {
		color.Error.Println(err)
		return
	}
	c.AddScope(binance)

	// add global JS interpreter
	c.AddCommand(js.EvalCommand())

	// start console
	c.Run()
}
