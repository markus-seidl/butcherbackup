package main

import (
	"os"

	"butcherbackup/internal/cli"
	"butcherbackup/internal/commands/backup"
	"butcherbackup/internal/commands/restore"
)

func main() {
	p := cli.NewParser("butcherbackup")
	p.Register(backup.New())
	p.Register(restore.New())
	os.Exit(p.Run(os.Args[1:]))
}
