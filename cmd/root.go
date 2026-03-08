package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "linear",
	Short: "CLI for interacting with the Linear API",
	Long: `A command-line tool for interacting with Linear's GraphQL API.

Manage issues, projects, teams, and cycles from the terminal.
Designed to be used by both humans and AI agents.

Authentication:
  Set LINEAR_API_KEY as an environment variable or in a .env file.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(loadEnv)
	rootCmd.PersistentFlags().StringP("output", "o", "text", "Output format: text, json")
}

func loadEnv() {
	// Walk up directories looking for .env
	dir, _ := os.Getwd()
	for {
		envFile := filepath.Join(dir, ".env")
		if _, err := os.Stat(envFile); err == nil {
			godotenv.Load(envFile)
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
}
