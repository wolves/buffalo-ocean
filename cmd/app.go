package cmd

import (
	"fmt"
	"os/exec"

	"github.com/pkg/errors"
)

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
