package main

import (
	"fmt"

	"github.com/gookit/color"
)

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
