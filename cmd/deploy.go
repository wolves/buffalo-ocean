package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"github.com/fatih/structs"
	"github.com/gobuffalo/makr"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	emoji "gopkg.in/kyokomi/emoji.v1"
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
	Init        bool
	Key         string
	Service     string
	Tag         string
}

var deploy = Deploy{}
var projectName string
var serverName string

func init() {
	deployCmd.Flags().StringVarP(&deploy.AppName, "app-name", "a", "", "The name for the application")
	deployCmd.Flags().StringVarP(&deploy.Branch, "branch", "b", "master", "Branch to use for deployment")
	deployCmd.Flags().StringVarP(&deploy.Environment, "environment", "e", "production", "Setting for the GO_ENV variable")
	deployCmd.Flags().BoolVar(&deploy.Init, "init", false, "Initialize the server along with deployment. Run this the first time.")
	deployCmd.Flags().StringVarP(&deploy.Key, "key", "k", "", "API Key for the service you are deploying to")
	deployCmd.Flags().StringVarP(&deploy.Service, "service", "s", "digitalocean", "Service for deploying to")
	deployCmd.Flags().StringVarP(&deploy.Tag, "tag", "t", "", "Tag to use for deployment. Overrides banch.")
	oceanCmd.AddCommand(deployCmd)
}

func (d Deploy) Run() error {

	projectName = d.AppName
	serverName = fmt.Sprintf("%s-%s", projectName, d.Environment)

	if d.Init {
		if err := provisionProcess(d); err != nil {
			return errors.WithStack(err)
		}
	} else {
		if err := deployProcess(d); err != nil {
			return errors.WithStack(err)
		}
	}
	// return runMigrations(serverName)
	return nil
}

func provisionProcess(d Deploy) error {
	green := color.New(color.FgGreen).SprintFunc()
	color.Blue("\n==> Provisioning server: %v.\n", green(serverName))
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
	g.Add(makr.Func{
		Runner: func(root string, data makr.Data) error {
			return cloneProject()
		},
	})
	g.Add(makr.Func{
		Runner: func(root string, data makr.Data) error {
			return setupProject(data)
		},
	})
	g.Add(makr.Func{
		Runner: func(root string, data makr.Data) error {
			return displayServerInfo()
		},
	})

	return g.Run(".", structs.Map(d))
}

func deployProcess(d Deploy) error {
	// TODO: Code a better way for checking if setup has been completed
	isSetup := validateProjectIsSetup()

	if !isSetup {
		provisionProcess(d)
	}

	color.Blue("\n========= DEPLOYING =========")
	g := makr.New()
	g.Add(makr.Func{
		Runner: func(root string, data makr.Data) error {
			return updateProject(data)
		},
	})
	g.Add(makr.Func{
		Runner: func(root string, data makr.Data) error {
			return deployProject(data)
		},
	})
	g.Add(makr.Func{
		Runner: func(root string, data makr.Data) error {
			return displayServerInfo()
		},
	})

	return g.Run(".", structs.Map(d))

}

func createCloudServer(d makr.Data) error {
	green := color.New(color.FgGreen).SprintFunc()

	color.Blue("\n==> Creating docker machine: %s\n", green(serverName))

	var k string
	if d["Key"] != "" {
		k = d["Key"].(string)
	} else {
		fmt.Println("Enter your write enabled Digital Ocean API KEY or create one with the link below.")
		fmt.Println("https://cloud.digitalocean.com/settings/api/tokens/new")
		k = requestUserInput("Please enter your DigitalOcean Token:")
	}

	driver := "--driver=digitalocean"
	accessToken := fmt.Sprintf("--digitalocean-access-token=%s", k)
	serverSize := "--digitalocean-size=s-1vcpu-1gb"

	cmd := exec.Command("docker-machine", "create", serverName, driver, accessToken, serverSize)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err := cmd.Run(); err != nil {
		errors.WithStack(err)
	}

	return nil
}

func createSwapFile() error {
	color.Blue("\n==> Creating Swapfile")
	cmds := []string{"dd if=/dev/zero of=/swapfile bs=2k count=1024k"}
	cmds = append(cmds, "mkswap /swapfile")
	cmds = append(cmds, "chmod 600 /swapfile")
	cmds = append(cmds, "swapon /swapfile")

	remoteCmd(strings.Join(cmds[:], " && "))
	remoteCmd("bash -c \"echo '/swapfile       none    swap    sw      0       0 ' >> /etc/fstab\"")

	return nil
}

func createDeployKeys() error {
	color.Blue("\n==> Creating Deploy Key")
	cmd := fmt.Sprintf("bash -c \"echo | ssh-keygen -q -N '' -t rsa -b 4096 -C 'deploy@%s'\"", projectName)
	remoteCmd(cmd)

	color.Yellow("\nPlease add this to your project's deploy keys on Github or Gitlab:")
	remoteCmd("tail .ssh/id_rsa.pub")
	fmt.Println("")

	return nil
}

func cloneProject() error {
	remoteCmd("apt-get install git")

	color.Blue("\n==> Cloning Project")
	r := requestUserInput("Please enter the repo to deploy from (Example: git@github.com:username/project.git):")

	remoteCmd("ssh-keyscan github.com >> ~/.ssh/known_hosts")
	remoteCmd(fmt.Sprintf("bash -c \"yes yes | git clone %s buffaloproject\"", r))

	// TODO: Check for database.yml file
	remoteCmd("bash -c \"cp buffaloproject/database.yml.example buffaloproject/database.yml\"")

	return nil
}

func setupProject(d makr.Data) error {
	magenta := color.New(color.FgMagenta).SprintFunc()
	color.Blue("\n==> Setting Up Project. This may take a few minutes.")

	buffaloEnv := d["Environment"].(string)

	// Copy over .env file from local project to buffaloproject
	env := "./.env"
	if _, err := os.Stat(env); err == nil {
		copyFileToRemoteProject(env)
	} else {
		fmt.Println("FILE DOESN'T EXIST")
		return errors.WithStack(err)
	}

	if err := remoteCmd("docker network create --driver bridge buffalonet"); err != nil {
		return errors.WithStack(err)
	}
	remoteCmd("docker build -t buffaloimage -f buffaloproject/Dockerfile buffaloproject")

	remoteCmd(fmt.Sprintf("docker container run -it --name buffalodb -v /root/db_volume:/var/lib/postgresql/data --network=buffalonet -e POSTGRES_USER=admin -e POSTGRES_PASSWORD=password -e POSTGRES_DB=buffalo_%s -d postgres", buffaloEnv))

	dbURL := fmt.Sprintf("DATABASE_URL=postgres://admin:password@buffalodb:5432/buffalo_%s?sslmode=disable", buffaloEnv)
	remoteCmd(fmt.Sprintf("docker container run -it --name buffaloweb -v /root/buffaloproject:/app -p 80:3000 --network=buffalonet -e GO_ENV=%s -e %s -d buffaloimage", buffaloEnv, dbURL))

	emoji.Printf("\n========= :beers: %s :beers: =========\n", magenta("INITIAL SERVER SETUP & DEPLOYMENT COMPLETE"))
	return nil
}

func updateProject(d makr.Data) error {
	color.Blue("\n==> Updating Project")
	co := fmt.Sprint(d["Branch"].(string))

	if d["Tag"] != "" {
		co = fmt.Sprintf("tags/%s", d["Tag"].(string))
	}

	if err := remoteCmd(fmt.Sprintf("bash -c \"cd buffaloproject && git pull && git checkout %s\"", co)); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func deployProject(d makr.Data) error {
	magenta := color.New(color.FgMagenta).SprintFunc()
	color.Blue("\n==> Deploying Project")
	buffaloEnv := d["Environment"].(string)
	dbURL := fmt.Sprintf("DATABASE_URL=postgres://admin:password@buffalodb:5432/buffalo_%s?sslmode=disable", buffaloEnv)

	cmds := []string{"docker container stop buffaloweb"}
	cmds = append(cmds, "docker container rm buffaloweb")
	cmds = append(cmds, "docker build -t buffaloimage -f buffaloproject/Dockerfile buffaloproject")
	cmds = append(cmds, fmt.Sprintf("docker container run -it --name buffaloweb -v /root/buffaloproject:/app -p 80:3000 --network=buffalonet -e GO_ENV=%s -e %s -d buffaloimage", buffaloEnv, dbURL))

	for _, cmd := range cmds {
		if err := remoteCmd(cmd); err != nil {
			return errors.WithStack(err)
		}
	}

	emoji.Printf("\n========= :beers: %s :beers: =========\n", magenta("DEPLOYMENT COMPLETE"))
	return nil
}

func displayServerInfo() error {
	ip, _ := exec.Command("docker-machine", "ip", serverName).Output()
	fmt.Printf("\nssh root@%s -i ~/.docker/machine/machines/%s/id_rsa", strings.TrimSpace(string(ip)), serverName)
	fmt.Printf("\nopen http://%s\n", ip)

	return nil
}
