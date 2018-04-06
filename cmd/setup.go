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

// setupCmd represents the setup command
var setupCmd = &cobra.Command{
	Use:     "setup",
	Aliases: []string{"s"},
	Short:   "A brief description of your command",

	RunE: func(cmd *cobra.Command, args []string) error {
		return setup.runSetup()
	},
}

var setup = Project{}

func init() {
	setupCmd.Flags().StringVarP(&setup.AppName, "app-name", "a", "", "The name for the application")
	setupCmd.Flags().StringVarP(&setup.Key, "key", "k", "", "API Key for the service you are deploying to")
	setupCmd.Flags().StringVarP(&setup.Branch, "branch", "b", "master", "Branch to use for deployment")
	setupCmd.Flags().StringVarP(&setup.Environment, "environment", "e", "production", "Setting for the GO_ENV variable")
	setupCmd.Flags().StringVarP(&setup.Tag, "tag", "t", "", "Tag to use for deployment. Overrides branch.")
	setupCmd.Flags().BoolVar(&setup.SkipVars, "skip-envs", false, "Skip the environment variable settup step")
	setupCmd.Flags().BoolVar(&setup.SkipSSL, "skip-ssl", false, "Skip the SSL setup step")
	oceanCmd.AddCommand(setupCmd)
}

func (p Project) runSetup() error {
	projectName = p.AppName
	serverName = fmt.Sprintf("%s-%s", projectName, p.Environment)

	if msg, ok := validateMachine("machineInstalled", serverName); !ok {
		return errors.New(msg)
	}

	if err := provisionProcess(p); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func provisionProcess(p Project) error {
	green := color.New(color.FgGreen).SprintFunc()
	color.Blue("\n==> PROVISIONING SERVER: %v.\n", green(serverName))

	g := makr.New()
	g.Add(makr.Func{
		Runner: func(root string, data makr.Data) error {
			return validateGit()
		},
	})
	g.Add(makr.Func{
		Runner: func(root string, data makr.Data) error {
			if msg, ok := validateMachine("isUnique", serverName); !ok {
				return errors.New(msg)
			}
			return nil
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
	if !setup.SkipVars {
		g.Add(makr.Func{
			Runner: func(root string, data makr.Data) error {
				return setupEnvVars()
			},
		})
	}
	g.Add(makr.Func{
		Runner: func(root string, data makr.Data) error {
			return setupProject(data)
		},
	})
	if !setup.SkipVars {
		g.Add(makr.Func{
			Runner: func(root string, data makr.Data) error {
				return cleanupEnvListFile()
			},
		})
	}
	g.Add(makr.Func{
		Runner: func(root string, data makr.Data) error {
			return displayServerInfo()
		},
	})

	return g.Run(".", structs.Map(p))
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
	cmds = append(cmds, "bash -c \"echo '/swapfile       none    swap    sw      0       0 ' >> /etc/fstab\"")

	if err := remoteCmd(strings.Join(cmds[:], " && ")); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func createDeployKeys() error {
	color.Blue("\n==> Creating Deploy Key")
	cmd := fmt.Sprintf("bash -c \"echo | ssh-keygen -q -N '' -t rsa -b 4096 -C 'deploy@%s'\"", projectName)

	if err := remoteCmd(cmd); err != nil {
		return errors.WithStack(err)
	}

	color.Yellow("\n\nPlease add this to your project's deploy keys on Github or Gitlab:")
	remoteCmd("tail .ssh/id_rsa.pub")
	fmt.Println("")

	return nil
}

func cloneProject() error {
	// TODO: Check docker-machine if it has git to determine if this is needed
	remoteCmd("apt-get install git")

	r := requestUserInput("Please enter the repo to deploy from (Example: git@github.com:username/project.git):")

	color.Blue("\n==> Cloning Project")

	if err := remoteCmd("ssh-keyscan github.com >> ~/.ssh/known_hosts"); err != nil {
		return errors.WithStack(err)
	}

	if setup.Tag != "" {
		r = fmt.Sprintf("-b %s %s", setup.Tag, r)
	} else {
		if setup.Branch != "master" {
			r = fmt.Sprintf("-b %s %s", setup.Branch, r)
		}
	}

	if err := remoteCmd(fmt.Sprintf("bash -c \"yes yes | git clone %s buffaloproject\"", r)); err != nil {
		return errors.WithStack(err)
	}

	// TODO: Check for database.yml file and check if database.yml.example exists
	if _, err := os.Stat("./database.yml"); err == nil {
		remoteCmd("bash -c \"cp buffaloproject/database.yml.example buffaloproject/database.yml\"")
	}

	return nil
}

func setupProject(d makr.Data) error {
	green := color.New(color.FgGreen).SprintFunc()
	magenta := color.New(color.FgMagenta).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()
	color.Blue("\n==> Setting Up Project. (This may take a few minutes)")

	buffaloEnv := d["Environment"].(string)

	color.Blue("\n==> CREATING: %s", green("Docker Network"))
	remoteCmd("docker network create --driver bridge buffalonet")
	color.Blue("\n==> CREATING: %s", green("Docker Database Container"))
	remoteCmd(fmt.Sprintf("docker container run -it --name buffalodb -v /root/db_volume:/var/lib/postgresql/data --network=buffalonet -e POSTGRES_USER=admin -e POSTGRES_PASSWORD=password -e POSTGRES_DB=buffalo_%s -d postgres", buffaloEnv))
	color.Blue("\n==> CREATING: %s", green("Docker Image"))
	remoteCmd("docker build -t buffaloimage -f buffaloproject/Dockerfile buffaloproject")

	dbURL := fmt.Sprintf("DATABASE_URL=postgres://admin:password@buffalodb:5432/buffalo_%s?sslmode=disable", buffaloEnv)
	color.Blue("\n==> CREATING: %s", green("Docker Web Container"))

	var webContainerCmd string
	var webContainerPort string

	if !setup.SkipSSL {
		webContainerPort = "3000"
	} else {
		webContainerPort = "80"
	}
	if !setup.SkipVars {
		webContainerCmd = fmt.Sprintf("docker container run -it --name buffaloweb -v /root/buffaloproject:/app -p %s:3000 --network=buffalonet --env-file /root/buffaloproject/env.list -e GO_ENV=%s -e %s -d buffaloimage", webContainerPort, buffaloEnv, dbURL)
	} else {
		webContainerCmd = fmt.Sprintf("docker container run -it --name buffaloweb -v /root/buffaloproject:/app -p %s:3000 --network=buffalonet -e GO_ENV=%s -e %s -d buffaloimage", webContainerPort, buffaloEnv, dbURL)
	}
	if err := remoteCmd(webContainerCmd); err != nil {
		return errors.WithStack(err)
	}

	if !setup.SkipSSL {
		if err := setupCaddy(); err != nil {
			return errors.WithStack(err)
		}
	}

	emoji.Printf("\n%s :beers: %s :beers: %s\n", blue("========="), magenta("INITIAL SERVER SETUP & DEPLOYMENT COMPLETE"), blue("========="))
	return nil
}

func setupCaddy() error {
	green := color.New(color.FgGreen).SprintFunc()
	color.Blue("\n==> CREATING: %s", green("Docker Caddy Container"))

	s := `
	IMPORTANT:: Before proceeding with SSL setup be sure to go to
	https://cloud.digitalocean.com/networking/domains and ensure that
	the domain you will be using for SSL is pointing to your newly created machine.
	Once you have done this press ENTER to continue.
	`
	_ = requestUserInput(s)
	d := requestUserInput("Enter your site domain for SSL (Example: mydomain.com):")
	e := requestUserInput("Enter your email for SSL:")

	c := fmt.Sprintf("%s {\n\ttls %s\n\tproxy / http://127.0.0.1:3000 {\n\t\ttransparent\n\t\twebsocket\n\t}\n}\n", d, e)
	f, err := os.Create("./Caddyfile")
	if err != nil {
		return errors.WithStack(err)
	}
	defer f.Close()
	if _, err := f.WriteString(c); err != nil {
		return errors.WithStack(err)
	}

	if err := remoteCmd("sudo mkdir /etc/caddy/"); err != nil {
		return errors.WithStack(err)
	}

	if err := copyFileToMachine("./Caddyfile", "/etc/caddy/"); err != nil {
		return errors.WithStack(err)
	}

	cmds := []string{"curl https://getcaddy.com | bash -s personal"}
	cmds = append(cmds, "sudo chown root:root /usr/local/bin/caddy")
	cmds = append(cmds, "sudo chmod 755 /usr/local/bin/caddy")
	cmds = append(cmds, "sudo setcap 'cap_net_bind_service=+ep' /usr/local/bin/caddy")

	cmds = append(cmds, "sudo chown -R root:www-data /etc/caddy")
	cmds = append(cmds, "sudo mkdir /etc/ssl/caddy")
	cmds = append(cmds, "sudo chown -R root:www-data /etc/ssl/caddy")
	cmds = append(cmds, "sudo chmod 0770 /etc/ssl/caddy")

	cmds = append(cmds, "sudo chown www-data:www-data /etc/caddy/Caddyfile")
	cmds = append(cmds, "sudo chmod 444 /etc/caddy/Caddyfile")

	cmds = append(cmds, "wget https://raw.githubusercontent.com/mholt/caddy/master/dist/init/linux-systemd/caddy.service")
	cmds = append(cmds, "sudo cp caddy.service /etc/systemd/system/")
	cmds = append(cmds, "sudo chown root:root /etc/systemd/system/caddy.service")
	cmds = append(cmds, "sudo chmod 644 /etc/systemd/system/caddy.service")
	cmds = append(cmds, "sudo systemctl daemon-reload")
	cmds = append(cmds, "sudo systemctl start caddy.service")

	if err := remoteCmd(strings.Join(cmds[:], " && ")); err != nil {
		return errors.WithStack(err)
	}

	if _, err := os.Stat("./Caddyfile"); err == nil {
		if err := os.Remove("./Caddyfile"); err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

func setupEnvVars() error {
	ev := requestUserInput("Enter the ENV variables for your project with a space between each: (eg. SAMPLE=test FOO=bar)")
	e := strings.Split(ev, " ")

	f, err := os.Create("./env.list")
	if err != nil {
		return errors.WithStack(err)
	}
	defer f.Close()

	for _, s := range e {
		sn := fmt.Sprintf("%s\n", s)
		if _, err := f.WriteString(sn); err != nil {
			return errors.WithStack(err)
		}
	}

	if err := copyFileToRemoteProject("./env.list"); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func cleanupEnvListFile() error {
	if _, err := os.Stat("./env.list"); err == nil {
		if err := os.Remove("./env.list"); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}
