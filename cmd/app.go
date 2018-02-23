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
	color.Yellow(msg)
	key, _ := reader.ReadString('\n')
	return strings.TrimSpace(key)
}

func remoteCmd(cmd string) error {
	fmt.Println("DEBUG: remoteCmd:", cmd)
	c := exec.Command("docker-machine", "ssh", serverName, cmd)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin

	if err := c.Run(); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func copyFileToRemoteProject(file string) error {
	fmt.Println("DEBUG: Copy File:", file)
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

func validateDockerMachine() error {
	if _, err := exec.LookPath("docker-machine"); err != nil {
		return errors.New("Docker is not installed. https://docs.docker.com/install/")
	}
	return nil
}

func validateProjectIsSetup() bool {
	cmd := "docker ps"
	out, _ := exec.Command("docker-machine", "ssh", serverName, cmd).Output()

	lc := strings.Count(string(out), "\n")
	return lc == 3
}
