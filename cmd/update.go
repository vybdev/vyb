package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vybdev/vyb/workspace/project"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Updates the vyb project metadata.",
	Run:   Update,
}

func Update(_ *cobra.Command, _ []string) {
	// for now, `vyb update` only works when executed on the root of the project
	err := project.Update(".")
	if err != nil {
		fmt.Printf("Error creating metadata: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Project metadata updated successfully.")
}
