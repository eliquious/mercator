package main

import (
	"os"
	"strings"

	"github.com/c-bata/go-prompt"
	"github.com/gookit/color"
	"github.com/kballard/go-shellquote"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// NewEnvironment creates a new environment with a root scope.
func NewEnvironment() *Environment {
	env := &Environment{ScopeStack: make([]Scope, 0)}

	rootScope := NewRootScope(env)
	env.Push(rootScope)
	return env
}

// Environment manages the various cmd scopes
type Environment struct {
	ScopeStack []Scope
}

// ChangeLivePrefix allows for a dynamic prompt prefix
func (env *Environment) ChangeLivePrefix() (string, bool) {
	scopes := []string{}
	for index := 0; index < len(env.ScopeStack); index++ {
		scopes = append(scopes, env.ScopeStack[index].GetScopeMeta().Prefix)
	}
	return strings.Join(scopes, ":") + "> ", true
}

// Push adds a scope to the environment
func (env *Environment) Push(scope Scope) {
	env.ScopeStack = append(env.ScopeStack, scope)
}

// Len returns the number of scopes. Should always be at least 1.
func (env *Environment) Len() int {
	return len(env.ScopeStack)
}

// Pop removes a scope from the environment. Should never remove the root scope.
func (env *Environment) Pop() Scope {
	if len(env.ScopeStack) <= 1 {
		os.Exit(0)
		return nil
	}
	scope := env.CurrentScope()
	env.ScopeStack = env.ScopeStack[:env.Len()-1]
	return scope
}

// CurrentScope gets the current scope from the environment
func (env *Environment) CurrentScope() Scope {
	return env.ScopeStack[env.Len()-1]
}

// ExecutorFunc executes the input.
func (env *Environment) ExecutorFunc(input string) {
	if input == "" {
		return
	}

	// Parse the input
	args, err := shellquote.Split(input)
	if err != nil {
		color.Warn.Println(err.Error())
		return
	}

	// Get the current scope
	scope := env.CurrentScope()
	if scope == nil {
		color.Warn.Println("current scope is nil")
		return
	}

	// Execute the command
	cmd := scope.GetCommand()
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		color.Warn.Println(err.Error())
		return
	}
}

// CompletorFunc gets the Completer from the current scope.
func (env *Environment) CompletorFunc(doc prompt.Document) []prompt.Suggest {
	line := strings.TrimSpace(doc.CurrentLine())
	if strings.TrimSpace(line) == "" {
		return []prompt.Suggest{}
	}

	// Get suggestions from current scope
	scope := env.CurrentScope()
	suggestions := getSuggestions(line, scope.GetCommand().Commands(), doc.GetWordBeforeCursor())
	return prompt.FilterFuzzy(suggestions, doc.GetWordBeforeCursor(), true)
}

func getSuggestions(line string, commands []*cobra.Command, prevWord string) []prompt.Suggest {
	rootCompletions := []prompt.Suggest{}
	for _, cmd := range commands {
		if strings.HasPrefix(line, cmd.Use) {
			suggestions := getSuggestions(line, cmd.Commands(), prevWord)

			cmd.Flags().VisitAll(func(flag *pflag.Flag) {
				suggestions = append(suggestions, prompt.Suggest{Text: "--" + flag.Name, Description: flag.Usage})
			})

			if len(cmd.ValidArgs) > 0 && len(prevWord) > 0 {
				for _, arg := range cmd.ValidArgs {
					suggestions = append(suggestions, prompt.Suggest{Text: arg, Description: ""})
				}
			}
			return suggestions
		}
		rootCompletions = append(rootCompletions, prompt.Suggest{Text: cmd.Use, Description: cmd.Short})
	}
	return rootCompletions
}
