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
var projectName string

func init() {
	setupCmd.Flags().StringVarP(&setup.AppName, "app-name", "a", "", "The name for the application")
	setupCmd.Flags().StringVarP(&setup.Branch, "branch", "b", "master", "Branch to use for deployment")
	setupCmd.Flags().StringVarP(&setup.Environment, "environment", "e", "production", "Setting for the GO_ENV variable")
	setupCmd.Flags().StringVarP(&setup.Key, "key", "k", "", "API Key for the service you are deploying to")
	setupCmd.Flags().StringVarP(&setup.Service, "service", "s", "digitalocean", "Service for deploying to")
	oceanCmd.AddCommand(setupCmd)
}

type Setup struct {
	AppName     string
	Branch      string
	Environment string
	Key         string
	Service     string
}

func (s Setup) Run() error {
	green := color.New(color.FgGreen).SprintFunc()

	projectName = s.AppName
	serverName = fmt.Sprintf("%s-%s", projectName, s.Environment)

	fmt.Printf("==> Provisioning server: %v.\n", green(serverName))
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
	g.Add(makr.Func{
		Runner: func(root string, data makr.Data) error {
			return createDeployKeys()
		},
	})

	return g.Run(".", structs.Map(s))
}

func requestKey() string {
	reader := bufio.NewReader(os.Stdin)
	color.Yellow("Please enter your DigitalOcean Token:")
	key, _ := reader.ReadString('\n')
	return strings.TrimSpace(key)
}

func createCloudServer(d makr.Data) error {
	green := color.New(color.FgGreen).SprintFunc()

	fmt.Printf("==> Deploying: %s\n", green(serverName))
	fmt.Printf("==> Creating docker machine: %s\n", serverName)

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

	cmd.Run()
	// fmt.Printf("CMD: %v\n", cmd)
	fmt.Println("==> Server creation completed!")
	return nil
}

func createSwapFile() error {
	fmt.Println("==> Creating Swapfile")
	cmds := []string{"dd if=/dev/zero of=/swapfile bs=2k count=1024k"}
	cmds = append(cmds, "mkswap /swapfile")
	cmds = append(cmds, "chmod 600 /swapfile")
	cmds = append(cmds, "swapon /swapfile")

	remoteCmd(strings.Join(cmds[:], " && "))
	remoteCmd("bash -c \"echo '/swapfile       none    swap    sw      0       0 ' >> /etc/fstab\"")

	return nil
}

func createDeployKeys() error {
	fmt.Println("==> Creating Deploy Key")
	cmd := fmt.Sprintf("bash -c \"echo | ssh-keygen -q -N '' -t rsa -b 4096 -C 'deploy@%s'\"", projectName)
	remoteCmd(cmd)

	fmt.Println("\nPlease add this to your project's deploy keys on Github or Gitlab:")
	remoteCmd("tail .ssh/id_rsa.pub")

	return nil
}

func remoteCmd(cmd string) error {
	fmt.Println("DEBUG: remoteCmd:", cmd)
	c := exec.Command("docker-machine", "ssh", serverName, cmd)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin

	c.Run()

	return nil
}
