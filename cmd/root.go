package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/vybdev/vyb/cmd/template"
	"os"
)

var rootCmd = &cobra.Command{
	Use:   "vyb",
	Short: "vyb is a CLI tool that uses AI to help you iteratively develop applications faster",
	Run: func(cmd *cobra.Command, args []string) {
		// If no subcommand is provided, print usage.
		fmt.Println(cmd.UsageString())
	},
}

// Execute executes the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	err := template.Register(rootCmd)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(versionCmd)
}
