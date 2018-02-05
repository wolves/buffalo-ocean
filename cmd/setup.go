package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"github.com/fatih/structs"
	"github.com/gobuffalo/makr"
	"github.com/spf13/cobra"
	emoji "gopkg.in/kyokomi/emoji.v1"
)

// setupCmd represents the setup command
var setupCmd = &cobra.Command{
	Use:     "setup",
	Aliases: []string{"s"},
	Short:   "A brief description of your command",
	RunE: func(cmd *cobra.Command, args []string) error {
		return setup.Run()
	},
}

var setup = Setup{}
var serverName string

func init() {
	setupCmd.Flags().StringVarP(&setup.Branch, "branch", "b", "master", "Branch to use for deployment")
	setupCmd.Flags().StringVarP(&setup.Environment, "environment", "e", "production", "Setting for the GO_ENV variable")
	setupCmd.Flags().StringVarP(&setup.Key, "key", "k", "", "API Key for the service you are deploying to")
	setupCmd.Flags().StringVarP(&setup.Service, "service", "s", "digitalocean", "Service for deploying to")
	oceanCmd.AddCommand(setupCmd)
}

type Setup struct {
	Branch      string
	Environment string
	Key         string
	Service     string
}

func (s Setup) Run() error {
	green := color.New(color.FgGreen).SprintFunc()

	serverName = "Test-App"
	fmt.Printf("Provisioning server: %v.\n", green(serverName))
	g := makr.New()
	g.Add(makr.Func{
		Runner: func(root string, data makr.Data) error {
			return validateGit()
		},
	})
	g.Add(makr.Func{
		Runner: func(root string, data makr.Data) error {
			return validateDockerMachine()
		},
	})

	g.Add(makr.Func{
		Runner: func(root string, data makr.Data) error {
			return createCloudServer(data)
		},
	})

	g.Add(makr.Func{
		Runner: func(root string, data makr.Data) error {
			return createSwapFile()
		},
	})

	return g.Run(".", structs.Map(s))
}

func requestKey() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Please enter your DigitalOcean Token:")
	key, _ := reader.ReadString('\n')
	return strings.TrimSpace(key)
}

func createCloudServer(d makr.Data) error {
	green := color.New(color.FgGreen).SprintFunc()

	fmt.Printf("Deploying: %s\n", green(serverName))
	fmt.Printf("Creating docker machine: %s\n", serverName)

	// Check is key has been set. Yes: Set it to variable and call create / No: fire user prompt to input key
	var k string
	if d["Key"] != "" {
		k = d["Key"].(string)
	} else {
		fmt.Println("Enter your write enabled Digital Ocean API KEY or create one with the link below.")
		fmt.Println("https://cloud.digitalocean.com/settings/api/tokens/new")
		k = requestKey()
	}

	driver := "--driver=digitalocean"
	accessToken := fmt.Sprintf("--digitalocean-access-token=%s", k)
	serverSize := "--digitalocean-size=s-1vcpu-1gb"

	cmd := exec.Command("docker-machine", "create", serverName, driver, accessToken, serverSize)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	// cmd.Run()
	fmt.Printf("CMD: %v\n", cmd)
	emoji.Println(":beers: Server creation completed!")
	return nil
}

func createSwapFile() error {

	return nil
}
