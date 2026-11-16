package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/rs/zerolog"
)

const (
	ExitOK      = 0
	ExitFailure = 1
	ExitUsage   = 2
)

type Command interface {
	Name() string
	Description() string
	Run(args []string) error
}

// UsageError marks an error caused by invalid command line usage. The message
// has already been printed to the user when it is returned.
type UsageError struct{ Err error }

func (e *UsageError) Error() string { return e.Err.Error() }
func (e *UsageError) Unwrap() error { return e.Err }

type Parser struct {
	name     string
	commands []Command
	out      io.Writer
	errOut   io.Writer
	log      zerolog.Logger
}

func NewParser(name string, log zerolog.Logger) *Parser {
	return &Parser{name: name, out: os.Stdout, errOut: os.Stderr, log: log}
}

func (p *Parser) Register(cmd Command) {
	p.commands = append(p.commands, cmd)
}

func (p *Parser) find(name string) Command {
	for _, c := range p.commands {
		if c.Name() == name {
			return c
		}
	}
	return nil
}

func (p *Parser) usage(w io.Writer) {
	fmt.Fprintf(w, "Usage: %s <command> [arguments]\n\nCommands:\n", p.name)
	width := 0
	for _, c := range p.commands {
		width = max(width, len(c.Name()))
	}
	for _, c := range p.commands {
		fmt.Fprintf(w, "  %-*s  %s\n", width, c.Name(), c.Description())
	}
	fmt.Fprintf(w, "\nRun '%s <command> -h' for help on a command.\n", p.name)
}

// Run dispatches args (without the program name) and returns the exit code.
func (p *Parser) Run(args []string) int {
	if len(args) == 0 {
		p.log.Error().Msg("error: no command given")
		p.usage(p.errOut)
		return ExitUsage
	}
	switch args[0] {
	case "-h", "--help", "help":
		p.usage(p.out)
		return ExitOK
	}
	cmd := p.find(args[0])
	if cmd == nil {
		p.log.Error().Msgf("error: unknown command %q", args[0])
		p.usage(p.errOut)
		return ExitUsage
	}
	err := cmd.Run(args[1:])
	var usageErr *UsageError
	switch {
	case err == nil, errors.Is(err, flag.ErrHelp):
		return ExitOK
	case errors.As(err, &usageErr):
		return ExitUsage
	default:
		p.log.Error().Msgf("error: %v", err)
		return ExitFailure
	}
}

// NewFlagSet creates a flag set for a command with usage text that names the command.
func NewFlagSet(cmd Command, synopsis string) *flag.FlagSet {
	fs := flag.NewFlagSet(cmd.Name(), flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() {
		out := os.Stderr
		fmt.Fprintf(out, "%s\n\nUsage: %s %s\n\nArguments:\n", cmd.Description(), cmd.Name(), synopsis)
		fs.PrintDefaults()
	}
	return fs
}

// Parse parses args; flag.ErrHelp is returned as is, other failures are logged
// and become UsageError. The flag package's own error output is suppressed.
func Parse(fs *flag.FlagSet, log zerolog.Logger, args []string) error {
	usage := fs.Usage
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	err := fs.Parse(args)
	fs.SetOutput(os.Stderr)
	fs.Usage = usage
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.Usage()
			return err
		}
		log.Error().Msgf("error: %v", err)
		fs.Usage()
		return &UsageError{Err: err}
	}
	return nil
}

// RequireFlags returns a UsageError naming the first required flag that was not set.
func RequireFlags(fs *flag.FlagSet, log zerolog.Logger, names ...string) error {
	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
	for _, n := range names {
		if !set[n] {
			log.Error().Msgf("error: missing required argument --%s", n)
			fs.Usage()
			return &UsageError{Err: fmt.Errorf("missing required argument --%s", n)}
		}
	}
	return nil
}

// StringList is a flag.Value for repeatable string arguments.
type StringList []string

func (s *StringList) String() string     { return strings.Join(*s, ",") }
func (s *StringList) Set(v string) error { *s = append(*s, v); return nil }
