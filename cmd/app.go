package cmd

import (
	"os"
	"os/exec"

	"github.com/pkg/errors"
)

func validateGit() error {
	c := exec.Command("git", "status")
	c.Stdin = os.Stdin
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	err := c.Run()
	if err != nil {
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
