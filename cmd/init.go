package cmd

import (
	"fmt"
	"os"

	"github.com/dangazineu/vyb/workspace/project"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initializes a vyb project. Must be executed from the project's root directory.",
	Run:   Init,
}

func Init(_ *cobra.Command, _ []string) {
	err := project.Create(".")
	if err != nil {
		fmt.Printf("Error creating metadata: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Project metadata created successfully.")
}
