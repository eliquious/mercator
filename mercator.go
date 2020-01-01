package main

import (
	"fmt"

	"github.com/c-bata/go-prompt"
	"github.com/gookit/color"
)

func main() {
	env := NewEnvironment()

	printASCII()
	p := prompt.New(
		env.ExecutorFunc,
		env.CompletorFunc,
		prompt.OptionTitle("mercator"),
		prompt.OptionPrefix("mercator> "),
		prompt.OptionLivePrefix(env.ChangeLivePrefix),
		prompt.OptionMaxSuggestion(12),

		// Text colors
		prompt.OptionScrollbarThumbColor(prompt.Red),
		prompt.OptionScrollbarBGColor(prompt.White),
		prompt.OptionPrefixTextColor(prompt.Red),
		prompt.OptionInputTextColor(prompt.White),
		prompt.OptionDescriptionBGColor(prompt.LightGray),
		prompt.OptionDescriptionTextColor(prompt.DarkGray),
		prompt.OptionSuggestionBGColor(prompt.DarkGray),
		prompt.OptionSuggestionTextColor(prompt.White),
		prompt.OptionSelectedSuggestionBGColor(prompt.LightGray),
		prompt.OptionSelectedSuggestionTextColor(prompt.DarkGray),
		prompt.OptionSelectedDescriptionBGColor(prompt.DarkGray),
		prompt.OptionSelectedDescriptionTextColor(prompt.LightGray),

		// Key bindings for meta key
		prompt.OptionSwitchKeyBindMode(prompt.EmacsKeyBind),
		prompt.OptionAddASCIICodeBind(prompt.ASCIICodeBind{
			ASCIICode: []byte{0x1b, 0x7f},
			Fn:        prompt.DeleteWord,
		}),
		prompt.OptionAddASCIICodeBind(prompt.ASCIICodeBind{
			ASCIICode: []byte{27, 27, 91, 68},
			Fn:        prompt.GoLeftWord,
		}),
		prompt.OptionAddASCIICodeBind(prompt.ASCIICodeBind{
			ASCIICode: []byte{0x1b, 102},
			Fn:        prompt.GoRightWord,
		}),
	)
	p.Run()
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
