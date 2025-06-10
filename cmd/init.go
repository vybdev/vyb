package cmd

import (
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/vybdev/vyb/config"
	"github.com/vybdev/vyb/llm"
	"github.com/vybdev/vyb/workspace/project"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initializes a vyb project. Must be executed from the project's root directory.",
	Run:   Init,
}

// Init is the cobra handler for `vyb init`.
func Init(_ *cobra.Command, _ []string) {
	// ---------------------------------------------------------------------
	// 1. Ask the user which provider should be configured.
	// ---------------------------------------------------------------------
	provider := chooseProvider()

	// ---------------------------------------------------------------------
	// 2. Generate project configuration and update annotations
	// ---------------------------------------------------------------------
	if err := project.Create(".", provider); err != nil {
		fmt.Printf("Error initializing project: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Project initialized successfully.")
}

// chooseProvider interacts with the user to pick a provider.  When the
// session is not interactive or the prompt fails, it returns the default
// provider.
func chooseProvider() string {
	providers := llm.SupportedProviders()

	var selection string
	prompt := &survey.Select{
		Message: "Select LLM provider:",
		Options: providers,
		Default: config.Default().Provider,
	}
	// Ignore prompt errors (non-tty, etc.) and fall back to default.
	if err := survey.AskOne(prompt, &selection); err != nil || selection == "" {
		return config.Default().Provider
	}
	return selection
}
