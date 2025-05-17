package cmd

import (
	"fmt"
	"github.com/dangazineu/vyb/workspace/project"
	"github.com/spf13/cobra"
	"os"
)

var forceRoot bool

var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Removes all vyb metadata from the given project. Must be executed at the project root directory, or include the --forceRoot flag.",
	Run:   Remove,
}

func init() {
	removeCmd.Flags().BoolVarP(&forceRoot, "force-root", "f", false, "Ignores any previous root references, and deletes all the project configuration within the current working directory and its sub-directories.")
}

func Remove(_ *cobra.Command, _ []string) {
	err := project.Remove(".", !forceRoot)
	if err != nil {
		fmt.Printf("Error removing project configuration: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Project configuration removed successfully.")
}
