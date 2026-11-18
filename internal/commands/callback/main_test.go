package callback

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"butcherbackup/internal/cli"
)

func newCommand() *Command { return New(zerolog.Nop()) }

// appRemoves plays the app: it waits for the file, reads it and deletes it.
func appRemoves(t *testing.T, path string) <-chan map[string]any {
	t.Helper()
	out := make(chan map[string]any, 1)
	go func() {
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			data, err := os.ReadFile(path)
			if err == nil {
				var v map[string]any
				if json.Unmarshal(data, &v) != nil {
					v = map[string]any{"invalid": string(data)}
				}
				os.Remove(path)
				out <- v
				return
			}
			time.Sleep(2 * time.Millisecond)
		}
		out <- nil
	}()
	return out
}

func TestRunSucceedsWhenFileIsRemoved(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notify.json")
	seen := appRemoves(t, path)
	err := newCommand().Run([]string{
		"--basename", "backup", "--number", "3", "--number-padded", "0003",
		"--extension", "dar", "--context", "operation", "--poll-interval", "10ms", path,
	})
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	got := <-seen
	want := map[string]any{"basename": "backup", "number": float64(3), "number_padded": "0003", "extension": "dar", "context": "operation"}
	if len(got) != len(want) {
		t.Fatalf("notification = %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %v, want %v", k, got[k], v)
		}
	}
}

func TestRunContextValuesPassThrough(t *testing.T) {
	for _, ctxValue := range []string{"init", "operation", "last_slice"} {
		path := filepath.Join(t.TempDir(), "notify.json")
		seen := appRemoves(t, path)
		if err := newCommand().Run([]string{"--context", ctxValue, "--poll-interval", "10ms", path}); err != nil {
			t.Fatal(err)
		}
		if got := <-seen; got["context"] != ctxValue {
			t.Errorf("context = %v, want %s", got["context"], ctxValue)
		}
	}
}

func TestRunOnlyGivenValuesAreWritten(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notify.json")
	seen := appRemoves(t, path)
	if err := newCommand().Run([]string{"--number", "2", "--poll-interval", "10ms", path}); err != nil {
		t.Fatal(err)
	}
	if got := <-seen; len(got) != 1 || got["number"] != float64(2) {
		t.Errorf("notification = %v, want only number 2", got)
	}
}

func TestRunIsQuietOnSuccess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notify.json")
	seen := appRemoves(t, path)

	oldOut, oldErr := os.Stdout, os.Stderr
	outR, outW, _ := os.Pipe()
	errR, errW, _ := os.Pipe()
	os.Stdout, os.Stderr = outW, errW
	err := newCommand().Run([]string{"--number", "1", "--poll-interval", "10ms", path})
	os.Stdout, os.Stderr = oldOut, oldErr
	outW.Close()
	errW.Close()
	stdout, _ := io.ReadAll(outR)
	stderr, _ := io.ReadAll(errR)

	if err != nil {
		t.Fatal(err)
	}
	<-seen
	if len(stdout) != 0 || len(stderr) != 0 {
		t.Errorf("output on success: stdout %q stderr %q", stdout, stderr)
	}
}

func TestRunUsageErrors(t *testing.T) {
	file := filepath.Join(t.TempDir(), "notify.json")
	tests := []struct {
		name string
		args []string
	}{
		{"no arguments", nil},
		{"options but no file", []string{"--number", "1"}},
		{"extra argument", []string{file, "extra"}},
		{"option after the file", []string{file, "--number", "1"}},
		{"unknown option", []string{"--bogus", file}},
		{"number not an integer", []string{"--number", "x", file}},
		{"number negative", []string{"--number", "-1", file}},
		{"number with fraction", []string{"--number", "1.5", file}},
		{"timeout invalid", []string{"--timeout", "soon", file}},
		{"timeout zero", []string{"--timeout", "0s", file}},
		{"timeout negative", []string{"--timeout", "-1s", file}},
		{"poll interval invalid", []string{"--poll-interval", "x", file}},
		{"poll interval zero", []string{"--poll-interval", "0s", file}},
		{"poll interval negative", []string{"--poll-interval", "-5ms", file}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := newCommand().Run(tt.args)
			var usageErr *cli.UsageError
			if !errors.As(err, &usageErr) {
				t.Fatalf("error = %v, want a usage error", err)
			}
			if _, statErr := os.Stat(file); statErr == nil {
				t.Error("notification file was created despite the usage error")
			}
		})
	}
}

func TestRunHelp(t *testing.T) {
	if err := newCommand().Run([]string{"-h"}); err == nil || !strings.Contains(err.Error(), "help requested") {
		t.Errorf("error = %v, want flag.ErrHelp", err)
	}
}

func TestRunExistingFileIsAFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notify.json")
	if err := os.WriteFile(path, []byte("previous"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := newCommand().Run([]string{"--poll-interval", "10ms", path})
	var usageErr *cli.UsageError
	if err == nil || errors.As(err, &usageErr) || !strings.Contains(err.Error(), path) {
		t.Fatalf("error = %v, want a plain failure naming %s", err, path)
	}
	if data, _ := os.ReadFile(path); string(data) != "previous" {
		t.Errorf("existing file changed to %q", data)
	}
}

func TestRunMissingDirectoryIsAFailure(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "missing")
	err := newCommand().Run([]string{filepath.Join(dir, "notify.json")})
	var usageErr *cli.UsageError
	if err == nil || errors.As(err, &usageErr) || !strings.Contains(err.Error(), dir) {
		t.Errorf("error = %v, want a plain failure naming %s", err, dir)
	}
}

func TestRunTimeout(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notify.json")
	start := time.Now()
	err := newCommand().Run([]string{"--timeout", "150ms", "--poll-interval", "10ms", path})
	var usageErr *cli.UsageError
	if err == nil || errors.As(err, &usageErr) || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("error = %v, want a timeout failure", err)
	}
	if elapsed := time.Since(start); elapsed < 140*time.Millisecond || elapsed > 2*time.Second {
		t.Errorf("returned after %v, want about 150ms", elapsed)
	}
	if _, statErr := os.Stat(path); statErr != nil {
		t.Errorf("notification file is gone after the timeout: %v", statErr)
	}
}

func TestCommandMetadata(t *testing.T) {
	c := newCommand()
	if c.Name() != "callback" || c.Description() == "" {
		t.Errorf("Name() = %q, Description() = %q", c.Name(), c.Description())
	}
}
