package callback

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "butcherbackup")
	cmd := exec.Command("go", "build", "-o", bin, "butcherbackup")
	cmd.Dir = filepath.Join("..", "..", "..")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}
	return bin
}

func TestSignalStopsWaitingAndKeepsFile(t *testing.T) {
	bin := buildBinary(t)
	for _, sig := range []syscall.Signal{syscall.SIGTERM, syscall.SIGINT} {
		t.Run(sig.String(), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "notify.json")
			cmd := exec.Command(bin, "callback", "--timeout", "1m", "--poll-interval", "10ms", path)
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() { done <- cmd.Wait() }()
			t.Cleanup(func() { cmd.Process.Kill() })

			deadline := time.Now().Add(5 * time.Second)
			for {
				if _, err := os.Stat(path); err == nil {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("notification file never appeared")
				}
				time.Sleep(5 * time.Millisecond)
			}
			cmd.Process.Signal(sig)

			select {
			case err := <-done:
				var exitErr *exec.ExitError
				if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
					t.Errorf("wait error = %v, want exit status 1", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("command did not stop after the signal")
			}
			if _, err := os.Stat(path); err != nil {
				t.Errorf("notification file is gone after the signal: %v", err)
			}
		})
	}
}
