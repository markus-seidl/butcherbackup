package callback

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "notify.json")
	if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func removeAfter(path string, d time.Duration) {
	go func() {
		time.Sleep(d)
		os.Remove(path)
	}()
}

func TestWaitReturnsAfterRemoval(t *testing.T) {
	path := newFile(t)
	removeAfter(path, 60*time.Millisecond)
	start := time.Now()
	if err := waitUntilGone(context.Background(), path, 10*time.Millisecond, 0); err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(start)
	if elapsed < 50*time.Millisecond {
		t.Errorf("returned after %v, before the file was removed", elapsed)
	}
	if elapsed > 500*time.Millisecond {
		t.Errorf("returned after %v, want within a few poll intervals of the removal", elapsed)
	}
}

func TestWaitReturnsImmediatelyWhenAlreadyGone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gone.json")
	start := time.Now()
	if err := waitUntilGone(context.Background(), path, time.Hour, 0); err != nil {
		t.Fatal(err)
	}
	if time.Since(start) > time.Second {
		t.Error("did not return immediately for a missing file with a long poll interval")
	}
}

func TestWaitIgnoresRewrittenContent(t *testing.T) {
	path := newFile(t)
	go func() {
		for i := 0; i < 5; i++ {
			time.Sleep(20 * time.Millisecond)
			os.WriteFile(path, []byte(`{"changed":true}`), 0o644)
		}
		time.Sleep(20 * time.Millisecond)
		os.Remove(path)
	}()
	start := time.Now()
	if err := waitUntilGone(context.Background(), path, 5*time.Millisecond, 0); err != nil {
		t.Fatal(err)
	}
	if time.Since(start) < 100*time.Millisecond {
		t.Errorf("returned after %v, while the file was still being rewritten", time.Since(start))
	}
}

func TestWaitTimeoutLeavesFile(t *testing.T) {
	path := newFile(t)
	start := time.Now()
	err := waitUntilGone(context.Background(), path, 10*time.Millisecond, 80*time.Millisecond)
	if !errors.Is(err, errTimeout) {
		t.Fatalf("error = %v, want errTimeout", err)
	}
	if elapsed := time.Since(start); elapsed < 70*time.Millisecond || elapsed > time.Second {
		t.Errorf("timed out after %v, want about 80ms", elapsed)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file was removed: %v", err)
	}
}

func TestWaitCancellationLeavesFile(t *testing.T) {
	path := newFile(t)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(40 * time.Millisecond)
		cancel()
	}()
	err := waitUntilGone(ctx, path, 10*time.Millisecond, 0)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file was removed: %v", err)
	}
}

func TestWaitReportsOtherStatErrors(t *testing.T) {
	file := newFile(t)
	err := waitUntilGone(context.Background(), filepath.Join(file, "child"), 10*time.Millisecond, 0)
	if err == nil || errors.Is(err, errTimeout) || errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want a stat error", err)
	}
}

func TestWaitTimeoutWinsOverNothing(t *testing.T) {
	path := newFile(t)
	removeAfter(path, 20*time.Millisecond)
	if err := waitUntilGone(context.Background(), path, 5*time.Millisecond, time.Second); err != nil {
		t.Errorf("error = %v, want nil when the file is removed before the timeout", err)
	}
}
