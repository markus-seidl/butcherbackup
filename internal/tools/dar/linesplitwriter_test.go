package dar

import (
	"io"
	"slices"
	"strings"
	"sync"
	"testing"
)

func writeChunks(w io.Writer, text string, size int) {
	for len(text) > 0 {
		n := min(size, len(text))
		io.WriteString(w, text[:n])
		text = text[n:]
	}
}

func TestWriterAssemblesLinesFromChunks(t *testing.T) {
	whole := parseFixtures(t, "restore_corrupt.stdout", "restore_corrupt.stderr")
	for _, size := range []int{1, 2, 3, 7, 64, 1000, 1 << 20} {
		p := NewVerboseParser()
		for _, name := range []string{"restore_corrupt.stdout", "restore_corrupt.stderr"} {
			w := p.NewWriter()
			writeChunks(w, fixture(t, name), size)
			w.Close()
		}
		if p.CurrentFile() != whole.CurrentFile() {
			t.Errorf("chunk size %d: CurrentFile() = %q, want %q", size, p.CurrentFile(), whole.CurrentFile())
		}
		if !slices.Equal(p.Errors(), whole.Errors()) {
			t.Errorf("chunk size %d: Errors() = %+v, want %+v", size, p.Errors(), whole.Errors())
		}
	}
}

func TestWriterReturnsLengthAndHandlesNoTrailingNewline(t *testing.T) {
	p := NewVerboseParser()
	w := p.NewWriter()
	n, err := w.Write([]byte("Adding file to archive: /data/last.txt"))
	if n != len("Adding file to archive: /data/last.txt") || err != nil {
		t.Errorf("Write() = %d, %v", n, err)
	}
	if got := p.CurrentFile(); got != "" {
		t.Errorf("CurrentFile() = %q before newline or Close, want empty", got)
	}
	w.Close()
	if got := p.CurrentFile(); got != "/data/last.txt" {
		t.Errorf("CurrentFile() = %q after Close, want /data/last.txt", got)
	}
}

func TestWritersKeepSeparatePartialLines(t *testing.T) {
	p := NewVerboseParser()
	out, errOut := p.NewWriter(), p.NewWriter()
	io.WriteString(out, "Adding file to archive: /da")
	io.WriteString(errOut, "FATAL error, aborting operation: bo")
	io.WriteString(out, "ta/a.txt\n")
	io.WriteString(errOut, "om\n")
	if p.CurrentFile() != "/data/a.txt" {
		t.Errorf("CurrentFile() = %q", p.CurrentFile())
	}
	if errs := p.Errors(); len(errs) != 1 || errs[0].Message != "boom" {
		t.Errorf("Errors() = %+v", errs)
	}
}

func TestConcurrentWriters(t *testing.T) {
	p := NewVerboseParser()
	var wg sync.WaitGroup
	feed := func(text string, size int) {
		defer wg.Done()
		w := p.NewWriter()
		writeChunks(w, text, size)
		w.Close()
	}
	var files, errs strings.Builder
	for range 200 {
		files.WriteString("Adding file to archive: /data/file.txt\n")
		errs.WriteString("Error while saving /data/file.txt: Permission denied\n")
	}
	wg.Add(3)
	go feed(files.String(), 5)
	go feed(errs.String(), 7)
	go func() {
		defer wg.Done()
		for range 200 {
			p.CurrentFile()
			p.Errors()
		}
	}()
	wg.Wait()
	if p.CurrentFile() != "/data/file.txt" {
		t.Errorf("CurrentFile() = %q", p.CurrentFile())
	}
	if n := len(p.Errors()); n != 200 {
		t.Errorf("%d errors, want 200", n)
	}
}

func TestParseText(t *testing.T) {
	p := NewVerboseParser()
	p.ParseText("Adding file to archive: /a\r\nAdding file to archive: /b")
	if p.CurrentFile() != "/b" {
		t.Errorf("CurrentFile() = %q, want /b", p.CurrentFile())
	}
}
