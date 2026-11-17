package dar

import (
	"context"
	"errors"
	"slices"
	"testing"
)

type fakeRunner struct {
	program string
	args    []string
	result  Result
	err     error
}

func (f *fakeRunner) Run(_ context.Context, program string, args []string) (Result, error) {
	f.program, f.args = program, args
	return f.result, f.err
}

func TestRunPassesProgramAndArgs(t *testing.T) {
	cmd := mustBuild(t, NewBuilder().List("/backup/test").Format(FormatSlicing))
	fake := &fakeRunner{result: Result{Stdout: "listing"}}

	res, err := cmd.Run(context.Background(), fake)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if fake.program != "dar" {
		t.Errorf("program = %q, want dar", fake.program)
	}
	if want := []string{"-l", "/backup/test", "-Tslicing"}; !slices.Equal(fake.args, want) {
		t.Errorf("args = %q, want %q", fake.args, want)
	}
	if res.Stdout != "listing" || res.ExitCode != 0 {
		t.Errorf("result = %+v", res)
	}
}

func TestRunUsesCustomBinary(t *testing.T) {
	cmd := mustBuild(t, NewBuilder().Create("a").Binary("/opt/dar/bin/dar"))
	fake := &fakeRunner{}
	if _, err := cmd.Run(context.Background(), fake); err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if fake.program != "/opt/dar/bin/dar" {
		t.Errorf("program = %q", fake.program)
	}
}

func TestRunNonZeroExit(t *testing.T) {
	cmd := mustBuild(t, NewBuilder().Extract("a"))
	fake := &fakeRunner{result: Result{ExitCode: 5, Stderr: "slice missing"}}

	res, err := cmd.Run(context.Background(), fake)
	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("error = %v, want *ExitError", err)
	}
	if exitErr.Code != 5 || exitErr.Stderr != "slice missing" {
		t.Errorf("ExitError = %+v", exitErr)
	}
	if res.ExitCode != 5 || res.Stderr != "slice missing" {
		t.Errorf("result = %+v", res)
	}
}

func TestRunRunnerError(t *testing.T) {
	boom := errors.New("cannot start")
	_, err := mustBuild(t, NewBuilder().Create("a")).Run(context.Background(), &fakeRunner{err: boom})
	if !errors.Is(err, boom) {
		t.Errorf("error = %v, want %v", err, boom)
	}
}

func TestExecRunner(t *testing.T) {
	res, err := ExecRunner{}.Run(context.Background(), "sh", []string{"-c", "echo out; echo err >&2; exit 3"})
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if res.ExitCode != 3 || res.Stdout != "out\n" || res.Stderr != "err\n" {
		t.Errorf("result = %+v", res)
	}
}

func TestExecRunnerMissingProgram(t *testing.T) {
	res, err := ExecRunner{}.Run(context.Background(), "/nonexistent/dar", nil)
	if err == nil {
		t.Fatal("Run() succeeded, want error")
	}
	if res.ExitCode != -1 {
		t.Errorf("ExitCode = %d, want -1", res.ExitCode)
	}
}

func TestRunDefaultsToExecRunner(t *testing.T) {
	cmd := mustBuild(t, NewBuilder().Create("a").Binary("/nonexistent/dar"))
	if _, err := cmd.Run(context.Background(), nil); err == nil {
		t.Error("Run(nil runner) succeeded, want error from missing binary")
	}
}
