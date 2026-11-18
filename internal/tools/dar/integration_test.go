package dar

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestIntegration(t *testing.T) {
	if _, err := exec.LookPath("dar"); err != nil {
		t.Skip("dar not found in PATH")
	}
	ctx := context.Background()
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	restore := filepath.Join(dir, "restore")
	for _, d := range []string{src, restore} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	small := []byte("hello butcherbackup\n")
	big := make([]byte, 300*1024)
	if _, err := rand.Read(big); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "small.txt"), small, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "big.bin"), big, 0o644); err != nil {
		t.Fatal(err)
	}

	archive := filepath.Join(dir, "backup")
	catalog := filepath.Join(dir, "catalog")

	run := func(b *Builder) Result {
		t.Helper()
		cmd, err := b.Build()
		if err != nil {
			t.Fatalf("Build() error: %v", err)
		}
		res, err := cmd.Run(ctx, nil)
		if err != nil {
			t.Fatalf("dar %q failed: %v", cmd.Args(), err)
		}
		return res
	}

	run(NewBuilder().Create(archive).Root(src).Compress(AlgorithmGzip, 0).SliceSize("100k").FirstSliceSize("200").Verbose())
	slices, _ := filepath.Glob(archive + ".*.dar")
	if len(slices) < 3 {
		t.Fatalf("expected several slices, got %v", slices)
	}

	list := run(NewBuilder().List(archive).Format(FormatSlicing))
	if !strings.Contains(list.Stdout, "big.bin") || !strings.Contains(list.Stdout, "Slice(s)") {
		t.Errorf("slicing listing lacks expected content:\n%s", list.Stdout)
	}
	for _, f := range []ListFormat{FormatNormal, FormatTree, FormatXML} {
		if out := run(NewBuilder().List(archive).Format(f)); !strings.Contains(out.Stdout, "small.txt") {
			t.Errorf("listing with format %q lacks small.txt:\n%s", f, out.Stdout)
		}
	}

	run(NewBuilder().Isolate(catalog).Reference(archive))
	if _, err := os.Stat(catalog + ".1.dar"); err != nil {
		t.Fatalf("isolated catalogue not created: %v", err)
	}

	run(NewBuilder().Extract(archive).Reference(catalog).FirstSliceHeader().Root(restore))
	for name, want := range map[string][]byte{"small.txt": small, "big.bin": big} {
		got, err := os.ReadFile(filepath.Join(restore, name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s differs after restore", name)
		}
	}
}

func TestIntegrationFilters(t *testing.T) {
	if _, err := exec.LookPath("dar"); err != nil {
		t.Skip("dar not found in PATH")
	}
	ctx := context.Background()
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	files := []string{"a/one.txt", "a/two.log", "b/three.txt", "b/sub/four.txt", "b/sub/five.log", "top.txt"}
	for _, f := range files {
		p := filepath.Join(src, f)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(f+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	run := func(b *Builder) Result {
		t.Helper()
		cmd, err := b.Build()
		if err != nil {
			t.Fatalf("Build() error: %v", err)
		}
		res, err := cmd.Run(ctx, nil)
		if err != nil {
			t.Fatalf("dar %q failed: %v", cmd.Args(), err)
		}
		return res
	}
	// stored returns the archived file names (not directories) from a normal listing.
	stored := func(archive string) map[string]bool {
		t.Helper()
		got := map[string]bool{}
		for _, line := range strings.Split(run(NewBuilder().List(archive)).Stdout, "\n") {
			fields := strings.Fields(line)
			if len(fields) == 0 {
				continue
			}
			name := fields[len(fields)-1]
			for _, f := range files {
				if name == f {
					got[f] = true
				}
			}
		}
		return got
	}
	check := func(label string, got map[string]bool, want ...string) {
		t.Helper()
		if len(got) != len(want) {
			t.Errorf("%s: got %v, want %v", label, got, want)
			return
		}
		for _, w := range want {
			if !got[w] {
				t.Errorf("%s: got %v, want %v", label, got, want)
				return
			}
		}
	}

	masks := filepath.Join(dir, "masks")
	run(NewBuilder().Create(masks).Root(src).IncludeMask("*.txt").ExcludeMask("three*"))
	check("-I/-X", stored(masks), "a/one.txt", "b/sub/four.txt", "top.txt")

	paths := filepath.Join(dir, "paths")
	run(NewBuilder().Create(paths).Root(src).IncludePath("b").ExcludePath("b/sub"))
	check("-g/-P", stored(paths), "b/three.txt")

	full := filepath.Join(dir, "full")
	run(NewBuilder().Create(full).Root(src))
	check("unfiltered", stored(full), files...)

	check("list -g", func() map[string]bool {
		got := map[string]bool{}
		for _, f := range files {
			if strings.Contains(run(NewBuilder().List(full).IncludePath("a")).Stdout, f) {
				got[f] = true
			}
		}
		return got
	}(), "a/one.txt", "a/two.log")

	restore := filepath.Join(dir, "restore")
	if err := os.MkdirAll(restore, 0o755); err != nil {
		t.Fatal(err)
	}
	run(NewBuilder().Extract(full).Root(restore).IncludePath("b/sub/four.txt"))
	var restored []string
	filepath.WalkDir(restore, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			rel, _ := filepath.Rel(restore, p)
			restored = append(restored, rel)
		}
		return nil
	})
	if len(restored) != 1 || restored[0] != "b/sub/four.txt" {
		t.Errorf("extract -g restored %v, want only b/sub/four.txt", restored)
	}
}

func TestIntegrationHooks(t *testing.T) {
	if _, err := exec.LookPath("dar"); err != nil {
		t.Skip("dar not found in PATH")
	}
	ctx := context.Background()
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	data := make([]byte, 250*1024)
	if _, err := rand.Read(data); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "big.bin"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	run := func(b *Builder) {
		t.Helper()
		cmd, err := b.Build()
		if err != nil {
			t.Fatalf("Build() error: %v", err)
		}
		if _, err := cmd.Run(ctx, nil); err != nil {
			t.Fatalf("dar %q failed: %v", cmd.Args(), err)
		}
	}
	lines := func(path string) []string {
		t.Helper()
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("hook did not run: %v", err)
		}
		return strings.Split(strings.TrimSpace(string(b)), "\n")
	}

	archive := filepath.Join(dir, "backup")
	catalog := filepath.Join(dir, "catalog")
	createLog := filepath.Join(dir, "create.log")
	run(NewBuilder().Create(archive).Root(src).SliceSize("100k").
		ExecuteBetweenSlices(`echo "%n %c" >> ` + createLog))
	got := lines(createLog)
	if len(got) != 3 || got[0] != "1 operation" || got[2] != "3 last_slice" {
		t.Errorf("create hook lines = %q, want slices 1..3 ending with last_slice", got)
	}

	run(NewBuilder().Isolate(catalog).Reference(archive))

	restoreLog := filepath.Join(dir, "restore.log")
	restore := filepath.Join(dir, "restore")
	if err := os.MkdirAll(restore, 0o755); err != nil {
		t.Fatal(err)
	}
	run(NewBuilder().Extract(archive).Reference(catalog).Root(restore).FirstSliceHeader().
		ExecuteBetweenReferenceSlices(`echo "ref %n %c" >> ` + restoreLog).
		ExecuteBetweenSlices(`echo "data %n %c" >> ` + restoreLog))
	got = lines(restoreLog)
	var sawRef, sawData bool
	for _, l := range got {
		sawRef = sawRef || strings.HasPrefix(l, "ref ")
		sawData = sawData || strings.HasPrefix(l, "data ")
	}
	if !sawRef || !sawData {
		t.Errorf("restore hook lines = %q, want both -E and -F hooks to have run", got)
	}
}

func TestIntegrationVerboseParser(t *testing.T) {
	if _, err := exec.LookPath("dar"); err != nil {
		t.Skip("dar not found in PATH")
	}
	ctx := context.Background()
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	if err := os.MkdirAll(filepath.Join(src, "my dir"), 0o755); err != nil {
		t.Fatal(err)
	}
	big := make([]byte, 250*1024)
	if _, err := rand.Read(big); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "my dir", "big file.bin"), big, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "small.txt"), []byte("small\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// runParsed runs the command with the parser attached to both streams.
	runParsed := func(b *Builder) (*VerboseParser, Result, error) {
		t.Helper()
		cmd, err := b.Verbose().Build()
		if err != nil {
			t.Fatalf("Build() error: %v", err)
		}
		p := NewVerboseParser()
		out, errOut := p.NewWriter(), p.NewWriter()
		res, err := cmd.Run(ctx, ExecRunner{Stdout: out, Stderr: errOut})
		out.Close()
		errOut.Close()
		return p, res, err
	}

	archive := filepath.Join(dir, "backup")
	p, res, err := runParsed(NewBuilder().Create(archive).Root(src).SliceSize("100k").FirstSliceSize("5k"))
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	var lastAdded string
	for _, line := range strings.Split(res.Stdout, "\n") {
		if _, path, ok := strings.Cut(line, " to archive: "); ok && strings.HasPrefix(line, "Adding ") {
			lastAdded = path
		}
	}
	if lastAdded == "" || p.CurrentFile() != lastAdded {
		t.Errorf("create: CurrentFile() = %q, want last added path %q", p.CurrentFile(), lastAdded)
	}
	if !strings.HasPrefix(p.CurrentFile(), src) {
		t.Errorf("create: CurrentFile() = %q, want a path under %q", p.CurrentFile(), src)
	}
	if errs := p.Errors(); len(errs) != 0 {
		t.Errorf("create: Errors() = %+v, want none", errs)
	}

	// Corrupt data inside slice 2, which holds the big file.
	slice2 := archive + ".2.dar"
	data, err := os.ReadFile(slice2)
	if err != nil {
		t.Fatal(err)
	}
	for i := 20000; i < 20100; i++ {
		data[i] ^= 0xff
	}
	if err := os.WriteFile(slice2, data, 0o644); err != nil {
		t.Fatal(err)
	}
	restore := filepath.Join(dir, "restore")
	if err := os.MkdirAll(restore, 0o755); err != nil {
		t.Fatal(err)
	}
	p, _, err = runParsed(NewBuilder().Extract(archive).Root(restore))
	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 5 {
		t.Fatalf("extract error = %v, want exit status 5", err)
	}
	var found bool
	for _, e := range p.Errors() {
		if e.Kind == ErrorFile && strings.HasSuffix(e.Path, "big file.bin") && strings.Contains(e.Message, "Bad CRC") {
			found = true
		}
	}
	if !found {
		t.Errorf("extract: Errors() = %+v, want a Bad CRC error for big file.bin", p.Errors())
	}
	if !strings.HasPrefix(p.CurrentFile(), restore) {
		t.Errorf("extract: CurrentFile() = %q, want a path under %q", p.CurrentFile(), restore)
	}

	p, _, err = runParsed(NewBuilder().Create(filepath.Join(dir, "never")).Root("/nonexistent"))
	if !errors.As(err, &exitErr) || exitErr.Code != 2 {
		t.Fatalf("create with missing root error = %v, want exit status 2", err)
	}
	errs := p.Errors()
	if len(errs) != 1 || errs[0].Kind != ErrorFatal || !strings.Contains(errs[0].Message, "Cannot read filesystem information at /nonexistent") {
		t.Errorf("missing root: Errors() = %+v, want one fatal error", errs)
	}
}

func TestIntegrationFindSlices(t *testing.T) {
	if _, err := exec.LookPath("dar"); err != nil {
		t.Skip("dar not found in PATH")
	}
	ctx := context.Background()
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	contents := map[string][]byte{
		"big file.bin": make([]byte, 350*1024),
		"small.txt":    []byte("small\n"),
		"other.txt":    []byte("other\n"),
		"tail.bin":     make([]byte, 150*1024),
	}
	for name, data := range contents {
		if _, err := rand.Read(data[:min(len(data), 300*1024)]); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(src, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	run := func(b *Builder) Result {
		t.Helper()
		cmd, err := b.Build()
		if err != nil {
			t.Fatalf("Build() error: %v", err)
		}
		res, err := cmd.Run(ctx, nil)
		if err != nil {
			t.Fatalf("dar %q failed: %v", cmd.Args(), err)
		}
		return res
	}

	archive := filepath.Join(dir, "backup")
	catalog := filepath.Join(dir, "catalog")
	run(NewBuilder().Create(archive).Root(src).SliceSize("100k").FirstSliceSize("5k"))
	run(NewBuilder().Isolate(catalog).Reference(archive))
	listing := run(NewBuilder().List(archive).Format(FormatSlicing)).Stdout
	all, err := filepath.Glob(archive + ".*.dar")
	if err != nil || len(all) < 5 {
		t.Fatalf("expected at least 5 slices, got %v (%v)", all, err)
	}

	for name, want := range contents {
		needed, matched, err := FindSlices(strings.NewReader(listing), "^"+regexp.QuoteMeta(name)+"$")
		if err != nil {
			t.Fatal(err)
		}
		if len(matched) != 1 || len(needed) == 0 {
			t.Fatalf("%s: needed = %v, matched = %+v", name, needed, matched)
		}
		if name == "big file.bin" && len(needed) < 4 {
			t.Errorf("%s spans %v, want at least 4 slices", name, needed)
		}

		// Keep only the slices FindSlices reports plus slice 1, whose header
		// dar reads with -affs, and restore the file from that partial set.
		keep := map[int]bool{1: true}
		for _, n := range needed {
			keep[n] = true
		}
		partial := filepath.Join(dir, "partial-"+strings.ReplaceAll(name, " ", "_"))
		if err := os.MkdirAll(partial, 0o755); err != nil {
			t.Fatal(err)
		}
		for n := range keep {
			data, err := os.ReadFile(fmt.Sprintf("%s.%d.dar", archive, n))
			if err != nil {
				t.Fatalf("%s: slice %d does not exist: %v", name, n, err)
			}
			if err := os.WriteFile(filepath.Join(partial, fmt.Sprintf("backup.%d.dar", n)), data, 0o644); err != nil {
				t.Fatal(err)
			}
		}
		restore := filepath.Join(dir, "restore-"+strings.ReplaceAll(name, " ", "_"))
		if err := os.MkdirAll(restore, 0o755); err != nil {
			t.Fatal(err)
		}
		run(NewBuilder().Extract(filepath.Join(partial, "backup")).Reference(catalog).
			FirstSliceHeader().Root(restore).IncludePath(name))
		got, err := os.ReadFile(filepath.Join(restore, name))
		if err != nil || !bytes.Equal(got, want) {
			t.Errorf("%s: not restored from slices %v (err %v)", name, needed, err)
		}
	}
}

type callbackNotification struct {
	Basename     string `json:"basename"`
	Number       int    `json:"number"`
	NumberPadded string `json:"number_padded"`
	Extension    string `json:"extension"`
	Context      string `json:"context"`
}

func buildButcherbackup(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "butcherbackup")
	cmd := exec.Command("go", "build", "-o", bin, "butcherbackup")
	cmd.Dir = filepath.Join("..", "..", "..")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}
	return bin
}

func callbackHook(bin, notify string) string {
	return bin + " callback --basename %b --number %n --number-padded %N --extension %e --context %c " + notify
}

// playApp acts as the backup app: for every notification it reads the JSON,
// calls check, records it and deletes the file so dar continues.
func playApp(t *testing.T, notify string, check func(callbackNotification)) (stop func() []callbackNotification) {
	t.Helper()
	var (
		mu       sync.Mutex
		got      []callbackNotification
		finished = make(chan struct{})
		exited   = make(chan struct{})
	)
	go func() {
		defer close(exited)
		for {
			select {
			case <-finished:
				return
			default:
			}
			data, err := os.ReadFile(notify)
			if err != nil {
				time.Sleep(2 * time.Millisecond)
				continue
			}
			var n callbackNotification
			if err := json.Unmarshal(data, &n); err != nil {
				t.Errorf("invalid notification %q: %v", data, err)
				os.Remove(notify)
				continue
			}
			check(n)
			mu.Lock()
			got = append(got, n)
			mu.Unlock()
			os.Remove(notify)
		}
	}()
	return func() []callbackNotification {
		close(finished)
		<-exited
		mu.Lock()
		defer mu.Unlock()
		return got
	}
}

func TestIntegrationCallback(t *testing.T) {
	if _, err := exec.LookPath("dar"); err != nil {
		t.Skip("dar not found in PATH")
	}
	bin := buildButcherbackup(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	data := make([]byte, 250*1024)
	if _, err := rand.Read(data); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "big.bin"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(dir, "backup")
	notify := filepath.Join(dir, "notify.json")

	// Create: every notification must arrive before dar starts the next slice.
	stop := playApp(t, notify, func(n callbackNotification) {
		if _, err := os.Stat(fmt.Sprintf("%s.%d.dar", archive, n.Number+1)); err == nil {
			t.Errorf("slice %d already exists while slice %d is being handled: dar did not wait", n.Number+1, n.Number)
		}
	})
	cmd, err := NewBuilder().Create(archive).Root(src).SliceSize("100k").
		ExecuteBetweenSlices(callbackHook(bin, notify)).Build()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cmd.Run(ctx, nil); err != nil {
		t.Fatalf("dar create failed: %v", err)
	}
	notes := stop()

	slices, _ := filepath.Glob(archive + ".*.dar")
	if len(slices) < 3 || len(notes) != len(slices) {
		t.Fatalf("%d notifications for %d slices: %+v", len(notes), len(slices), notes)
	}
	for i, n := range notes {
		wantContext := "operation"
		if i == len(notes)-1 {
			wantContext = "last_slice"
		}
		if n.Number != i+1 || n.Context != wantContext || n.Basename != "backup" || n.Extension != "dar" || n.NumberPadded != fmt.Sprint(i+1) {
			t.Errorf("notification %d = %+v, want slice %d with context %s", i, n, i+1, wantContext)
		}
		if _, err := os.Stat(fmt.Sprintf("%s.%d.dar", archive, n.Number)); err != nil {
			t.Errorf("notification for slice %d but the slice file does not exist: %v", n.Number, err)
		}
	}
	if _, err := os.Stat(notify); err == nil {
		t.Error("notification file still exists after dar finished")
	}

	// Extract: the first call is the init context, then operation.
	restore := filepath.Join(dir, "restore")
	if err := os.MkdirAll(restore, 0o755); err != nil {
		t.Fatal(err)
	}
	stop = playApp(t, notify, func(callbackNotification) {})
	cmd, err = NewBuilder().Extract(archive).Root(restore).
		ExecuteBetweenSlices(callbackHook(bin, notify)).Build()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cmd.Run(ctx, nil); err != nil {
		t.Fatalf("dar extract failed: %v", err)
	}
	notes = stop()
	if len(notes) < 2 || notes[0].Context != "init" || notes[1].Context != "operation" || notes[1].Number != 1 {
		t.Errorf("extract notifications = %+v, want init first, then operation from slice 1", notes)
	}
	if got, err := os.ReadFile(filepath.Join(restore, "big.bin")); err != nil || !bytes.Equal(got, data) {
		t.Errorf("restored file differs (err %v)", err)
	}
}

func TestIntegrationCallbackFailureStopsDar(t *testing.T) {
	if _, err := exec.LookPath("dar"); err != nil {
		t.Skip("dar not found in PATH")
	}
	bin := buildButcherbackup(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	data := make([]byte, 250*1024)
	if _, err := rand.Read(data); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "big.bin"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(dir, "backup")
	notify := filepath.Join(dir, "notify.json")
	if err := os.WriteFile(notify, []byte("left over from an earlier run"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd, err := NewBuilder().Create(archive).Root(src).SliceSize("100k").
		ExecuteBetweenSlices(callbackHook(bin, notify)).Verbose().Build()
	if err != nil {
		t.Fatal(err)
	}
	p := NewVerboseParser()
	out, errOut := p.NewWriter(), p.NewWriter()
	_, err = cmd.Run(ctx, ExecRunner{Stdout: out, Stderr: errOut})
	out.Close()
	errOut.Close()

	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 6 {
		t.Fatalf("dar error = %v, want exit status 6 (failed user command)", err)
	}
	// On abort dar closes the archive with a tiny terminator slice, so a stopped
	// run has no data in slice 2 (a full slice would be 100 KiB).
	if info, err := os.Stat(archive + ".2.dar"); err == nil && info.Size() > 1024 {
		t.Errorf("slice 2 holds %d bytes: dar continued although the hook failed", info.Size())
	}
	if info, err := os.Stat(archive + ".1.dar"); err != nil || info.Size() != 100*1024 {
		t.Errorf("slice 1 = %v (err %v), want a full 100 KiB slice", info, err)
	}
	var fatal bool
	for _, e := range p.Errors() {
		fatal = fatal || (e.Kind == ErrorFatal && strings.Contains(e.Message, "user command"))
	}
	if !fatal {
		t.Errorf("Errors() = %+v, want a fatal user command error", p.Errors())
	}
	if got, _ := os.ReadFile(notify); string(got) != "left over from an earlier run" {
		t.Errorf("existing notification file was changed to %q", got)
	}
}
