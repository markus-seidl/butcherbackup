package callback

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func intPtr(n int) *int { return &n }

func readNotification(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("invalid JSON %q: %v", data, err)
	}
	return got
}

func TestCreateNotificationAllValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notify.json")
	n := Notification{Basename: "backup", Number: intPtr(3), NumberPadded: "0003", Extension: "dar", Context: "operation"}
	if err := createNotification(path, n); err != nil {
		t.Fatal(err)
	}
	got := readNotification(t, path)
	want := map[string]any{"basename": "backup", "number": float64(3), "number_padded": "0003", "extension": "dar", "context": "operation"}
	if len(got) != len(want) {
		t.Errorf("got %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %v, want %v", k, got[k], v)
		}
	}
}

func TestCreateNotificationOmitsMissingValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notify.json")
	if err := createNotification(path, Notification{Number: intPtr(2)}); err != nil {
		t.Fatal(err)
	}
	got := readNotification(t, path)
	if len(got) != 1 || got["number"] != float64(2) {
		t.Errorf("got %v, want only number 2", got)
	}
}

func TestCreateNotificationNumberZeroIsKept(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notify.json")
	if err := createNotification(path, Notification{Number: intPtr(0)}); err != nil {
		t.Fatal(err)
	}
	if got := readNotification(t, path); got["number"] != float64(0) {
		t.Errorf("got %v, want number 0", got)
	}
}

func TestCreateNotificationSpecialCharacters(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notify.json")
	values := []string{`my "backup" 'x'`, "tab\tnew\nline", "ünïcödé 日本語 🚀", `back\slash`, "</script>&"}
	for _, v := range values {
		os.Remove(path)
		if err := createNotification(path, Notification{Basename: v, Context: v}); err != nil {
			t.Fatal(err)
		}
		got := readNotification(t, path)
		if got["basename"] != v || got["context"] != v {
			t.Errorf("round trip of %q gave %v", v, got)
		}
	}
}

func TestCreateNotificationRefusesExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notify.json")
	if err := os.WriteFile(path, []byte("previous"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := createNotification(path, Notification{Basename: "x"})
	if err == nil || !strings.Contains(err.Error(), path) || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("error = %v, want one naming %s", err, path)
	}
	if data, _ := os.ReadFile(path); string(data) != "previous" {
		t.Errorf("existing file changed to %q", data)
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 1 {
		t.Errorf("directory has %d entries, want only the existing file", len(entries))
	}
}

func TestCreateNotificationRefusesExistingDirectoryOrDanglingLink(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "d"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "nowhere"), filepath.Join(dir, "link")); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"d", "link"} {
		if err := createNotification(filepath.Join(dir, name), Notification{}); err == nil {
			t.Errorf("%s: createNotification succeeded, want error", name)
		}
	}
}

func TestCreateNotificationMissingDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "missing")
	err := createNotification(filepath.Join(dir, "notify.json"), Notification{})
	if err == nil || !strings.Contains(err.Error(), dir) {
		t.Fatalf("error = %v, want one naming %s", err, dir)
	}
	if _, statErr := os.Stat(dir); statErr == nil {
		t.Error("directory was created")
	}
}

func TestCreateNotificationRelativePath(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := createNotification("notify.json", Notification{Basename: "b"}); err != nil {
		t.Fatal(err)
	}
	if got := readNotification(t, "notify.json"); got["basename"] != "b" {
		t.Errorf("got %v", got)
	}
}

func TestCreateNotificationLeavesNoTemporaryFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notify.json")
	if err := createNotification(path, Notification{Basename: "x"}); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 || entries[0].Name() != "notify.json" {
		t.Errorf("directory entries = %v, want only notify.json", entries)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0o644 {
		t.Errorf("mode = %v, want 0644", info.Mode().Perm())
	}
}

func TestCreateNotificationNeverPartial(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notify.json")
	n := Notification{Basename: strings.Repeat("x", 50000), Number: intPtr(1), Context: "operation"}

	stop := make(chan struct{})
	var wg sync.WaitGroup
	var bad string
	var mu sync.Mutex
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			var v map[string]any
			if len(data) == 0 || json.Unmarshal(data, &v) != nil {
				mu.Lock()
				bad = "partial document of " + string(rune('0'+len(data)%10)) + " bytes"
				mu.Unlock()
				return
			}
		}
	}()
	for i := 0; i < 300; i++ {
		if err := createNotification(path, n); err != nil {
			t.Fatal(err)
		}
		os.Remove(path)
	}
	close(stop)
	wg.Wait()
	if bad != "" {
		t.Error(bad)
	}
}
