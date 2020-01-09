package main

import (
	"github.com/eliquious/console"
	"github.com/eliquious/mercator/binance"
	"github.com/eliquious/mercator/shopify"
	"github.com/gookit/color"
)

func main() {
	c := console.New("mercator", console.WithTitleScreen(printASCII))

	shopify, err := shopify.NewShopifyScope()
	if err != nil {
		color.Error.Println(err)
		return
	}
	c.AddScope(shopify)

	binance, err := binance.NewBinanceExchangeScope()
	if err != nil {
		color.Error.Println(err)
		return
	}
	c.AddScope(binance)
	c.Run()
}
