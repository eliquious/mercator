package main

import "github.com/spf13/cobra"

// Scope is an interface for environment scopes.
type Scope interface {
	GetScopeMeta() ScopeMeta
	GetCommand() *cobra.Command
}

// ScopeMeta contains the scope prefix and description
type ScopeMeta struct {
	Prefix      string
	Description string
}

// CommandFunc creates a new cobra.Command
type CommandFunc func(*Environment) *cobra.Command
