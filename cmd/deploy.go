package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
	Use:     "deploy",
	Aliases: []string{"d"},
	Short:   "Deploy to DigitalOcean using docker",

	RunE: func(cmd *cobra.Command, args []string) error {
		return deploy.Run()
	},
}

type Deploy struct {
	AppName     string
	Branch      string
	Environment string
}

var deploy = Deploy{}

func init() {
	deployCmd.Flags().StringVarP(&setup.AppName, "app-name", "a", "", "The name for the application")
	deployCmd.Flags().StringVarP(&setup.Branch, "branch", "b", "master", "Branch to use for deployment")
	deployCmd.Flags().StringVarP(&setup.Environment, "environment", "e", "production", "Setting for the GO_ENV variable")
	oceanCmd.AddCommand(deployCmd)
}

func (d Deploy) Run() error {
	color.Blue("\n==> Deploying app")

	serverName := fmt.Sprintf("%s-%s", d.AppName, d.Environment)
	return runMigrations(serverName)
}
