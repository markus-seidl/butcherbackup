package dar

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
)

// Result is the outcome of running a command.
type Result struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

// Runner executes a program; replace it in tests to avoid needing dar installed.
type Runner interface {
	Run(ctx context.Context, program string, args []string) (Result, error)
}

// ExitError is returned when dar exits with a non-zero status.
type ExitError struct {
	Code   int
	Stderr string
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("dar exited with status %d: %s", e.Code, e.Stderr)
}

// ExecRunner runs the program with os/exec.
type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, program string, args []string) (Result, error) {
	cmd := exec.CommandContext(ctx, program, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	res := Result{Stdout: stdout.String(), Stderr: stderr.String()}
	var exitErr *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &exitErr):
		res.ExitCode = exitErr.ExitCode()
	default:
		res.ExitCode = -1
		return res, err
	}
	return res, nil
}

// Run executes the command with runner (ExecRunner when nil). A non-zero exit
// status returns the Result together with an *ExitError.
func (c *Command) Run(ctx context.Context, runner Runner) (Result, error) {
	if runner == nil {
		runner = ExecRunner{}
	}
	res, err := runner.Run(ctx, c.program, c.Args())
	if err != nil {
		return res, err
	}
	if res.ExitCode != 0 {
		return res, &ExitError{Code: res.ExitCode, Stderr: res.Stderr}
	}
	return res, nil
}
