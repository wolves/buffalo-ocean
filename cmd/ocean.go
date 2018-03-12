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

type Project struct {
	AppName     string
	Branch      string
	Environment string
	SkipVars    bool
	Key         string
	Tag         string
}

func init() {
	rootCmd.AddCommand(oceanCmd)
}
