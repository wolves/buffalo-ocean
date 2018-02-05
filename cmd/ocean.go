package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// oceanCmd represents the ocean command
var oceanCmd = &cobra.Command{
	Use:     "ocean",
	Aliases: []string{"o"},
	Short:   "Tools for deploying Buffalo to DigitalOcean",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("ocean called")
	},
}

func init() {
	rootCmd.AddCommand(oceanCmd)
}
