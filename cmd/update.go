package cmd

import (
	"github.com/spf13/cobra"
	"github.com/vybdev/vyb/logging"
	"github.com/vybdev/vyb/workspace/project"
	"os"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update the project's metadata",
	Long: `This command updates the project's metadata.
It will regenerate all annotations for the current project, preserving any
existing ones that are still valid.`,
	Run: Update,
}

func Update(_ *cobra.Command, _ []string) {
	// for now, `vyb update` only works when executed on the root of the project
	err := project.Update(".")
	if err != nil {
		logging.Log.Fatalf("Error creating metadata: %v\n", err)
		os.Exit(1)
	}
	logging.Log.Info("Project metadata updated successfully.")
}
