package restore

import (
	"errors"

	"butcherbackup/internal/cli"
)

type Command struct{}

func New() *Command { return &Command{} }

func (*Command) Name() string        { return "restore" }
func (*Command) Description() string { return "Restore files from a backup" }

func (c *Command) Run(args []string) error {
	fs := cli.NewFlagSet(c, "--backup <path> --target <path> [file]...")
	backup := fs.String("backup", "", "backup location to restore from (required)")
	target := fs.String("target", "", "directory to restore into (required)")

	if err := cli.Parse(fs, args); err != nil {
		return err
	}
	if err := cli.RequireFlags(fs, "backup", "target"); err != nil {
		return err
	}
	files := fs.Args()
	_, _, _ = backup, target, files

	return errors.New("restore is not implemented yet")
}
