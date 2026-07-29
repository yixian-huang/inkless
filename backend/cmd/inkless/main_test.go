package main

import (
	"testing"
)

func TestRootCmd(t *testing.T) {
	cmd := rootCmd()
	if cmd.Use != "inkless" {
		t.Errorf("expected root command name 'inkless', got %q", cmd.Use)
	}
	// Verify subcommands are registered
	subNames := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subNames[sub.Name()] = true
	}
	expected := []string{"init", "serve", "migrate", "seed", "site", "articles", "pages"}
	for _, name := range expected {
		if !subNames[name] {
			t.Errorf("expected subcommand %q to be registered", name)
		}
	}
}
