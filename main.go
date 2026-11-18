package main

import (
	"os"

	"butcherbackup/internal/cli"
	"butcherbackup/internal/commands/backup"
	"butcherbackup/internal/commands/callback"
	"butcherbackup/internal/commands/restore"
	"butcherbackup/internal/logging"
)

func main() {
	log := logging.New(os.Stderr)
	p := cli.NewParser("butcherbackup", log)
	p.Register(backup.New(log))
	p.Register(restore.New(log))
	p.Register(callback.New(log))
	os.Exit(p.Run(os.Args[1:]))
}
