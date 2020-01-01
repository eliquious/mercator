package main

import (
	"os"

	"github.com/gookit/color"
	"github.com/spf13/cobra"
)

// NewRootScope creates a new root scope for the mercator CLI.
func NewRootScope(env *Environment) Scope {
	scope := &rootScope{
		Prefix:      "mercator",
		Description: "A command line tool for managing monetary assets",
	}

	rootCommand := &cobra.Command{
		Use:   scope.Prefix,
		Short: scope.Description,
	}
	useCommand := &cobra.Command{
		Use:   "use",
		Short: "Use changes the scope for the environment",
		Run: func(cmd *cobra.Command, args []string) {
			color.Error.Println("unknown scope")
		},
	}
	binanceScopeCommand := &cobra.Command{
		Use:   "binance",
		Short: "Access Binance exchange information",
		Run: func(cmd *cobra.Command, args []string) {
			apiKey := os.Getenv("BINANCE_API_KEY")
			apiSecret := os.Getenv("BINANCE_API_SECRET")

			scope, err := NewBinanceExchangeScope(env, apiKey, apiSecret)
			if err != nil {
				color.Error.Println("Binance scope requires env variables: BINANCE_API_KEY and BINANCE_API_SECRET")
				return
			}
			env.Push(scope)
		},
	}
	useCommand.AddCommand(binanceScopeCommand)
	rootCommand.AddCommand(useCommand)

	// addHelpCommand(rootCommand)
	addExitCommand(env, rootCommand)
	addQuitCommand(env, rootCommand)

	rootCommand.SetUsageTemplate(`Usage:{{if .Runnable}}
	{{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
	{{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}
	  
  Aliases:
	{{.NameAndAliases}}{{end}}{{if .HasExample}}
	  
  Examples:
  {{.Example}}{{end}}{{if .HasAvailableSubCommands}}
	  
  Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
	{{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}
	  
  Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}` + "\n")

	scope.Command = rootCommand
	return scope
}

func addHelpCommand(cmd *cobra.Command) {
	helpCommand := &cobra.Command{
		Use:   "help",
		Short: "Prints command usage",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Usage()
		},
	}
	cmd.AddCommand(helpCommand)
}

func addExitCommand(env *Environment, cmd *cobra.Command) {
	exitCommand := &cobra.Command{
		Use:   "exit",
		Short: "Exits the current scope. Exits CLI if at top-level scope.",
		Run: func(cmd *cobra.Command, args []string) {
			env.Pop()
		},
	}
	cmd.AddCommand(exitCommand)
}

func addQuitCommand(env *Environment, cmd *cobra.Command) {
	quitCommand := &cobra.Command{
		Use:   "quit",
		Short: "Fully exits the Mercator CLI regardless of scope",
		Run: func(cmd *cobra.Command, args []string) {
			os.Exit(0)
		},
	}
	cmd.AddCommand(quitCommand)
}

// rootScope contains a list of relevant commands for the default scope.
type rootScope struct {
	Prefix      string
	Description string
	Command     *cobra.Command
}

func (s *rootScope) GetScopeMeta() ScopeMeta {
	return ScopeMeta{s.Prefix, s.Description}
}

func (s *rootScope) GetCommand() *cobra.Command {
	return s.Command
}
