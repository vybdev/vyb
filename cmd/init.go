package cmd

import (
    "fmt"
    "os"

    "github.com/AlecAivazis/survey/v2"
    "github.com/spf13/cobra"
    "github.com/vybdev/vyb/config"
    "github.com/vybdev/vyb/workspace/project"
    "gopkg.in/yaml.v3"
)

var initCmd = &cobra.Command{
    Use:   "init",
    Short: "Initializes a vyb project. Must be executed from the project's root directory.",
    Run:   Init,
}

// TODO(vyb): instead of duplicating this information here, add a function to get a list of supported providers in the llm package
// supportedProviders lists the providers a user can choose from.  The list
// will grow as more integrations land; for now only "openai" is available.
var supportedProviders = []string{"openai"}

// Init is the cobra handler for `vyb init`.
func Init(_ *cobra.Command, _ []string) {
    // ---------------------------------------------------------------------
    // 1. Generate metadata (this also creates the .vyb directory).
    // ---------------------------------------------------------------------
    if err := project.Create("."); err != nil {
        fmt.Printf("Error creating metadata: %v\n", err)
        os.Exit(1)
    }

    // ---------------------------------------------------------------------
    // 2. Ensure .vyb/config.yaml exists – prompt the user when missing.
    // ---------------------------------------------------------------------
    cfgPath := ".vyb/config.yaml"
    if _, err := os.Stat(cfgPath); err == nil {
        // Configuration already present – nothing else to do.
        fmt.Println("Project metadata created successfully (existing config preserved).")
        return
    } else if !os.IsNotExist(err) {
        fmt.Printf("Error checking %s: %v\n", cfgPath, err)
        os.Exit(1)
    }

    provider := chooseProvider()
    cfg := &config.Config{Provider: provider}

    data, err := marshalConfig(cfg)
    if err != nil {
        fmt.Printf("Error marshalling config.yaml: %v\n", err)
        os.Exit(1)
    }

    if err := os.WriteFile(cfgPath, data, 0644); err != nil {
        fmt.Printf("Error writing %s: %v\n", cfgPath, err)
        os.Exit(1)
    }

    fmt.Println("Project metadata and configuration created successfully.")
}

// chooseProvider interacts with the user to pick a provider.  When the
// session is not interactive or the prompt fails, it returns the default
// provider.
func chooseProvider() string {
    var selection string
    prompt := &survey.Select{
        Message: "Select LLM provider:",
        Options: supportedProviders,
        Default: config.Default().Provider,
    }
    // Ignore prompt errors (non-tty, etc.) and fall back to default.
    if err := survey.AskOne(prompt, &selection); err != nil || selection == "" {
        return config.Default().Provider
    }
    return selection
}

// marshalConfig converts Config to YAML while guaranteeing a trailing
// newline (cosmetic only).
func marshalConfig(cfg *config.Config) ([]byte, error) {
    if cfg == nil {
        return nil, fmt.Errorf("config must not be nil")
    }
    b, err := yaml.Marshal(cfg)
    if err != nil {
        return nil, err
    }
    // Ensure the file ends with a newline for POSIX friendliness.
    if len(b) == 0 || b[len(b)-1] != '\n' {
        b = append(b, '\n')
    }
    return b, nil
}
