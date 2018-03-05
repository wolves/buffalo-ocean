package cmd

import (
	"os/exec"
	"strings"

	"github.com/fatih/color"
)

func validateMachine(t string, n string) (string, bool) {
	var msg string
	var rsp bool

	switch t {
	case "machineInstalled":
		rsp = validateDockerMachineInstalled()
		msg = color.RedString("\nDocker is not installed. https://docs.docker.com/install/")
	case "isUnique":
		rsp = !validateMachineNameUnique(n)
		msg = color.RedString("\nA Docker machine with that name already exists")
	case "isStopped":
		rsp = validateMachineIsStopped(n)
		msg = color.RedString("\nIt appears your Docker Machine with name \"%s\" is currently stopped.", n)
	case "isSetup":
		rsp = validateMachineProjectIsSetup(n)
		msg = color.RedString("\nThe containers on the Docker Machine named \"%s\" do not appear to be setup yet or are not running. Either restart containers before deploying or run deploy with the \"--init\" flag.", n)
	default:
		rsp = false
		msg = color.RedString("\nNot a valid Docker Machine check")
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
