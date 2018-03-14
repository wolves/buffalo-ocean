package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"github.com/pkg/errors"
)

func requestUserInput(msg string) string {
	reader := bufio.NewReader(os.Stdin)
	color.Yellow("\n%s", msg)
	key, _ := reader.ReadString('\n')
	return strings.TrimSpace(key)
}

func remoteCmd(cmd string) error {
	c := exec.Command("docker-machine", "ssh", serverName, cmd)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin

	if err := c.Run(); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func copyFileToMachine(file, dir string) error {
	d := fmt.Sprintf("%s:%s", serverName, dir)
	c := exec.Command("docker-machine", "scp", file, d)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin

	if err := c.Run(); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func copyFileToRemoteProject(file string) error {
	p := fmt.Sprintf("%s:~/buffaloproject/", serverName)
	c := exec.Command("docker-machine", "scp", file, p)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin

	if err := c.Run(); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func validateGit() error {
	c := exec.Command("git", "status")
	b, err := c.CombinedOutput()
	if err != nil {
		fmt.Println(string(b))
		return errors.Wrap(err, "Must be a valid Git application")
	}

	return nil
}

func displayServerInfo() error {
	ip, _ := exec.Command("docker-machine", "ip", serverName).Output()
	fmt.Printf("\nssh root@%s -i ~/.docker/machine/machines/%s/id_rsa", strings.TrimSpace(string(ip)), serverName)
	fmt.Printf("\nopen http://%s\n", ip)

	return nil
}
