package callback

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog"

	"butcherbackup/internal/cli"
)

const defaultPollInterval = 200 * time.Millisecond

type Command struct{ log zerolog.Logger }

func New(log zerolog.Logger) *Command { return &Command{log: log} }

func (*Command) Name() string { return "callback" }
func (*Command) Description() string {
	return "Publish a notification file and wait until it is removed (for use as a dar slice hook)"
}

func (c *Command) Run(args []string) error {
	fs := cli.NewFlagSet(c, "[--basename <name>] [--number <n>] [--number-padded <n>] [--extension <ext>] [--context <context>] [--timeout <duration>] [--poll-interval <duration>] <notification-file>")
	basename := fs.String("basename", "", "archive base name (dar %b)")
	number := fs.String("number", "", "slice number (dar %n)")
	numberPadded := fs.String("number-padded", "", "zero-padded slice number (dar %N)")
	extension := fs.String("extension", "", "slice extension (dar %e)")
	dContext := fs.String("context", "", "dar context: init, operation or last_slice (dar %c)")
	timeout := fs.Duration("timeout", 0, "give up waiting after this long (default: wait indefinitely)")
	pollInterval := fs.Duration("poll-interval", defaultPollInterval, "how often to check whether the file was removed")

	if err := cli.Parse(fs, c.log, args); err != nil {
		return err
	}
	usage := func(format string, a ...any) error {
		err := fmt.Errorf(format, a...)
		c.log.Error().Msgf("error: %v", err)
		fs.Usage()
		return &cli.UsageError{Err: err}
	}

	switch n := fs.NArg(); {
	case n == 0:
		return usage("missing notification file")
	case n > 1:
		return usage("unexpected arguments after the notification file: %s (options must come before it)", strings.Join(fs.Args()[1:], " "))
	}

	notification := Notification{
		Basename:     *basename,
		NumberPadded: *numberPadded,
		Extension:    *extension,
		Context:      *dContext,
	}
	if *number != "" {
		n, err := strconv.Atoi(*number)
		if err != nil || n < 0 {
			return usage("invalid value %q for --number: not a non-negative integer", *number)
		}
		notification.Number = &n
	}
	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
	if set["timeout"] && *timeout <= 0 {
		return usage("invalid value %s for --timeout: must be a positive duration", *timeout)
	}
	if *pollInterval <= 0 {
		return usage("invalid value %s for --poll-interval: must be a positive duration", *pollInterval)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return publishAndWait(ctx, fs.Arg(0), notification, *pollInterval, *timeout)
}

// publishAndWait creates the notification file and waits until it is removed.
// The file is left in place when waiting fails.
func publishAndWait(ctx context.Context, path string, n Notification, interval, timeout time.Duration) error {
	if err := createNotification(path, n); err != nil {
		return err
	}
	switch err := waitUntilGone(ctx, path, interval, timeout); {
	case err == nil:
		return nil
	case errors.Is(err, errTimeout):
		return fmt.Errorf("timed out after %s waiting for notification file %q to be removed", timeout, path)
	case errors.Is(err, context.Canceled):
		return fmt.Errorf("interrupted while waiting for notification file %q to be removed", path)
	default:
		return err
	}
}
