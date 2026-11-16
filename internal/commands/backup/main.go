package backup

import (
	"errors"

	"butcherbackup/internal/cli"
)

type Command struct{}

func New() *Command { return &Command{} }

func (*Command) Name() string        { return "backup" }
func (*Command) Description() string { return "Create a backup of a source directory" }

func (c *Command) Run(args []string) error {
	fs := cli.NewFlagSet(c, "--source <path> --destination <path> [--exclude <pattern>]...")
	source := fs.String("source", "", "directory to back up (required)")
	destination := fs.String("destination", "", "where to store the backup (required)")
	var exclude cli.StringList
	fs.Var(&exclude, "exclude", "pattern to exclude, may be repeated")

	if err := cli.Parse(fs, args); err != nil {
		return err
	}
	if err := cli.RequireFlags(fs, "source", "destination"); err != nil {
		return err
	}
	_, _, _ = source, destination, exclude

	return errors.New("backup is not implemented yet")
}
