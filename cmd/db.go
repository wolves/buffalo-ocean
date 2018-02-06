package cmd

import (
	"context"
	"os"
	"os/exec"
	"time"

	"github.com/fatih/color"
)

func runMigrations(s string) error {
	color.Blue("\n==> Running migrations")
	if _, err := os.Stat("./database.yml"); err == nil {
		serverName := s
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		c := exec.CommandContext(ctx, "docker-machine", "ssh", serverName, "/bin/app", "migrate")
		c.Stdin = os.Stdin
		c.Stderr = os.Stderr
		c.Stdout = os.Stdout
		return c.Run()
	}
	return nil
}
