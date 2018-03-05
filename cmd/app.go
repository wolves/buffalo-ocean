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

func validateMachine(t string, n string) (string, bool) {
	var msg string
	var rsp bool

	switch t {
	case "machineInstalled":
		rsp = validateDockerMachineInstalled()
		msg = color.RedString("Docker is not installed. https://docs.docker.com/install/")
	case "isUnique":
		rsp = !validateMachineNameUnique(n)
		msg = color.RedString("A Docker machine with that name already exists")
	case "isStopped":
		rsp = validateMachineIsStopped(n)
		msg = color.RedString("It appears your Docker Machine with name \"%s\" is currently stopped.", n)
	case "isSetup":
		rsp = validateMachineProjectIsSetup(n)
		msg = color.RedString("The containers on the Docker Machine named \"%s\" do not appear to be setup yet or are not running. Either restart containers before deploying or run \"setup\" instead of \"deploy\".", n)
	default:
		rsp = false
		msg = color.RedString("Not a valid Docker Machine check")
	}

	return msg, rsp
}

func validateDockerMachineInstalled() bool {
	_, err := exec.LookPath("docker-machine")
	return err == nil
}

func validateMachineIsStopped(n string) bool {
	out, _ := exec.Command("docker-machine", "status", n).Output()
	return strings.Contains(string(out), "Stopped")
}

func validateMachineNameUnique(n string) bool {
	out, _ := exec.Command("docker-machine", "ls").Output()
	return strings.Contains(string(out), n)
}

func validateMachineProjectIsSetup(n string) bool {
	out, _ := exec.Command("docker-machine", "ssh", n, "docker ps").Output()
	return 3 == strings.Count(string(out), "\n")
}
