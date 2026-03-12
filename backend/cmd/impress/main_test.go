package main

import (
	"testing"
)

func TestRootCmd(t *testing.T) {
	cmd := rootCmd()
	if cmd.Use != "impress" {
		t.Errorf("expected root command name 'impress', got %q", cmd.Use)
	}
	// Verify subcommands are registered
	subNames := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subNames[sub.Name()] = true
	}
	expected := []string{"init", "serve", "migrate", "seed"}
	for _, name := range expected {
		if !subNames[name] {
			t.Errorf("expected subcommand %q to be registered", name)
		}
	}
}
