package dar

import (
	"bytes"
	"context"
	"crypto/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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
	if len(slices) < 3 || len(slices) == 0 {
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
