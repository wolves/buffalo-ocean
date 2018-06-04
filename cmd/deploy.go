package cmd

import (
	"fmt"

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
		return deploy.runDeploy()
	},
}

var deploy = Project{}
var projectName string
var serverName string

func init() {
	deployCmd.Flags().StringVarP(&deploy.AppName, "app-name", "a", "", "The name for the application")
	deployCmd.Flags().StringVarP(&deploy.Branch, "branch", "b", "master", "Branch to use for deployment")
	deployCmd.Flags().StringVarP(&deploy.Environment, "environment", "e", "production", "Setting for the GO_ENV variable")
	deployCmd.Flags().StringVarP(&deploy.Tag, "tag", "t", "", "Tag to use for deployment. Overrides banch.")
	deployCmd.Flags().BoolVar(&deploy.SkipSSL, "skip-ssl", false, "Skip the SSL setup step")
	oceanCmd.AddCommand(deployCmd)
}

func (p Project) runDeploy() error {
	projectName = p.AppName
	serverName = fmt.Sprintf("%s-%s", projectName, p.Environment)

	if msg, ok := validateMachine("machineInstalled", serverName); !ok {
		return errors.New(msg)
	}

	if err := deployProcess(p); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func deployProcess(d Project) error {

	green := color.New(color.FgGreen).SprintFunc()
	color.Blue("\n==> DEPLOYING TO SERVER: %v.\n", green(serverName))

	g := makr.New()
	g.Add(makr.Func{
		Runner: func(root string, data makr.Data) error {
			if msg, ok := validateMachine("isStopped", serverName); ok {
				return errors.New(msg)
			}
			return nil
		},
	})
	g.Add(makr.Func{
		Runner: func(root string, data makr.Data) error {
			if msg, ok := validateMachine("isSetup", serverName); !ok {
				return errors.New(msg)
			}
			return nil
		},
	})
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

	var webContainerPort string
	if !deploy.SkipSSL {
		webContainerPort = "3000"
	} else {
		webContainerPort = "80"
	}

	cmds := []string{"docker container stop buffaloweb"}
	cmds = append(cmds, "docker container rm buffaloweb")
	cmds = append(cmds, "docker build -t buffaloimage -f buffaloproject/Dockerfile buffaloproject")
	cmds = append(cmds, fmt.Sprintf("docker container run -it --name buffaloweb -v /root/buffaloproject:/app -p %s:3000 --network=buffalonet -e GO_ENV=%s -e %s -d buffaloimage", webContainerPort, buffaloEnv, dbURL))

	for _, cmd := range cmds {
		if err := remoteCmd(cmd); err != nil {
			return errors.WithStack(err)
		}
	}

	if _, err := emoji.Printf("\n========= :beers: %s :beers: =========\n", magenta("DEPLOYMENT COMPLETE")); err != nil {
		return errors.WithStack(err)
	}
	return nil
}
