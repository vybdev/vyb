package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/vybdev/vyb/cmd/template"
	"github.com/vybdev/vyb/config"
	"github.com/vybdev/vyb/logging"
	"os"
)

var logLevel string
var debugLogging bool

var rootCmd = &cobra.Command{
	Use:   "vyb",
	Short: "vyb is a CLI tool that uses AI to help you iteratively develop applications faster",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		cfg, err := config.Load(".")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if logLevel == "" {
			logLevel = cfg.Logging.Level
		}

		if logLevel == "" {
			logLevel = "info"
		}

		if err := logging.Init(logLevel); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
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
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "", "log level (e.g. debug, info, warn, error, fatal, panic)")
	rootCmd.PersistentFlags().BoolVar(&debugLogging, "debug", false, "enable request/response debug logging")
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
