package dar

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func parseFixtures(t *testing.T, names ...string) *VerboseParser {
	t.Helper()
	p := NewVerboseParser()
	for _, n := range names {
		p.ParseText(fixture(t, n))
	}
	return p
}

func TestCurrentFileKeepsLatest(t *testing.T) {
	p := NewVerboseParser()
	if got := p.CurrentFile(); got != "" {
		t.Errorf("CurrentFile() before any line = %q, want empty", got)
	}
	p.ParseLine("Adding file to archive: /data/a.txt")
	p.ParseLine("Adding file to archive: /data/b.txt")
	if got := p.CurrentFile(); got != "/data/b.txt" {
		t.Errorf("CurrentFile() = %q, want /data/b.txt", got)
	}
}

func TestCurrentFileLineKinds(t *testing.T) {
	tests := []struct{ line, want string }{
		{"Adding file to archive: /data/a.txt", "/data/a.txt"},
		{"Adding folder to archive: /data/d", "/data/d"},
		{"Adding symlink to archive: /data/sym", "/data/sym"},
		{"Adding named pipe to archive: /data/fifo", "/data/fifo"},
		{"Recording hard link into the archive: /data/hard", "/data/hard"},
		{"Restoring file's data: /restore/a.txt", "/restore/a.txt"},
		{"Adding file to archive: /data/my dir/big file.bin", "/data/my dir/big file.bin"},
		{"Adding file to archive: /data/we ird: name.txt", "/data/we ird: name.txt"},
		{"Adding file to archive: /data/crlf.txt\r", "/data/crlf.txt"},
	}
	for _, tt := range tests {
		p := NewVerboseParser()
		p.ParseLine(tt.line)
		if got := p.CurrentFile(); got != tt.want {
			t.Errorf("%q: CurrentFile() = %q, want %q", tt.line, got, tt.want)
		}
	}
}

func TestUnrelatedLinesAreIgnored(t *testing.T) {
	p := NewVerboseParser()
	p.ParseLine("Adding file to archive: /data/a.txt")
	for _, line := range []string{
		"",
		"Building the catalog object...",
		"Adding a new layer on top: Caching layer for better performances...",
		"Adding a streamed compression layer",
		"Adding reference to files that have been destroyed since reference backup...",
		"Saving Filesystem Specific Attributes for /data/other.txt",
		"Restoring file's FSA: /restore/other.txt",
		"Finished Inspecting directory /data/d , saved 0",
		" 0 inode(s) not restored (not saved in archive)",
		" 0 inode(s) not restored (overwriting policy decision)",
		" 1 inode(s) failed to restore (filesystem error)",
		"Arguments read from /etc/darrc :",
	} {
		p.ParseLine(line)
	}
	if got := p.CurrentFile(); got != "/data/a.txt" {
		t.Errorf("CurrentFile() = %q, want /data/a.txt", got)
	}
	if errs := p.Errors(); len(errs) != 0 {
		t.Errorf("Errors() = %+v, want none", errs)
	}
}

func TestCurrentFileFromFixtures(t *testing.T) {
	tests := []struct{ fixture, want string }{
		{"create_v.stdout", "/work/src/my dir/sub/we ird: name.txt"},
		{"create_diff_v.stdout", "/work/src/d"},
		{"extract_v.stdout", "/work/out/my dir/sub/we ird: name.txt"},
	}
	for _, tt := range tests {
		p := parseFixtures(t, tt.fixture)
		if got := p.CurrentFile(); got != tt.want {
			t.Errorf("%s: CurrentFile() = %q, want %q", tt.fixture, got, tt.want)
		}
		if errs := p.Errors(); len(errs) != 0 {
			t.Errorf("%s: Errors() = %+v, want none", tt.fixture, errs)
		}
	}
}

func TestFileErrors(t *testing.T) {
	tests := []struct {
		line string
		want DarError
	}{
		{
			"Error while saving /data/secret.txt: Permission denied",
			DarError{ErrorFile, "/data/secret.txt", "Permission denied", "Error while saving /data/secret.txt: Permission denied"},
		},
		{
			"Error while restoring /restore/big.bin : Bad CRC, data corruption occurred",
			DarError{ErrorFile, "/restore/big.bin", "Bad CRC, data corruption occurred", "Error while restoring /restore/big.bin : Bad CRC, data corruption occurred"},
		},
		{
			"/restore/big.bin not restored (user choice)",
			DarError{ErrorNotRestored, "/restore/big.bin", "user choice", "/restore/big.bin not restored (user choice)"},
		},
	}
	for _, tt := range tests {
		p := NewVerboseParser()
		p.ParseLine(tt.line)
		errs := p.Errors()
		if len(errs) != 1 || errs[0] != tt.want {
			t.Errorf("%q: Errors() = %+v, want [%+v]", tt.line, errs, tt.want)
		}
	}
}

func TestFileErrorWithColonInPath(t *testing.T) {
	p := NewVerboseParser()
	p.ParseLine("Adding file to archive: /data/we ird: name.txt")
	p.ParseLine("Error while saving /data/we ird: name.txt: Permission denied")
	errs := p.Errors()
	if len(errs) != 1 || errs[0].Path != "/data/we ird: name.txt" || errs[0].Message != "Permission denied" {
		t.Errorf("Errors() = %+v", errs)
	}
}

func TestFatalErrorsAreDeduplicated(t *testing.T) {
	p := NewVerboseParser()
	p.ParseLine("Aborting due to exception: Cannot read filesystem information at /x: No such file or directory")
	p.ParseLine("FATAL error, aborting operation: Cannot read filesystem information at /x: No such file or directory")
	errs := p.Errors()
	if len(errs) != 1 {
		t.Fatalf("Errors() = %+v, want one fatal error", errs)
	}
	if errs[0].Kind != ErrorFatal || errs[0].Message != "Cannot read filesystem information at /x: No such file or directory" || errs[0].Path != "" {
		t.Errorf("error = %+v", errs[0])
	}
}

func TestAbortedByPrompt(t *testing.T) {
	p := NewVerboseParser()
	p.ParseLine("Aborting program. User refused to continue while asking: slice missing")
	errs := p.Errors()
	if len(errs) != 1 || errs[0].Kind != ErrorFatal || errs[0].Message != "slice missing" {
		t.Errorf("Errors() = %+v", errs)
	}
	p.ParseLine("Aborting due to exception: slice missing")
	p.ParseLine("Aborting program. User refused to continue while asking: slice missing")
	if n := len(p.Errors()); n != 1 {
		t.Errorf("%d errors after repeats, want 1", n)
	}
	p.ParseLine("Aborting program. Something else entirely")
	if errs := p.Errors(); len(errs) != 2 || errs[1].Message != "Something else entirely" {
		t.Errorf("Errors() = %+v", errs)
	}
}

func TestLatestFileSurvivesErrors(t *testing.T) {
	p := NewVerboseParser()
	p.ParseLine("Adding file to archive: /data/a.txt")
	p.ParseLine("Error while saving /data/a.txt: Permission denied")
	p.ParseLine("Aborting due to exception: stop")
	if got := p.CurrentFile(); got != "/data/a.txt" {
		t.Errorf("CurrentFile() = %q", got)
	}
}

func TestErrorsFromFixtures(t *testing.T) {
	tests := []struct {
		fixtures []string
		want     []DarError
	}{
		{[]string{"save_denied.stdout", "save_denied.stderr"}, []DarError{
			{Kind: ErrorFile, Path: "/work/u/secret.txt", Message: "Permission denied"},
		}},
		{[]string{"restore_corrupt.stdout", "restore_corrupt.stderr"}, []DarError{
			{Kind: ErrorFile, Path: "/work/out2/my dir/big file.bin", Message: "Bad CRC, data corruption occurred"},
			{Kind: ErrorFatal, Message: "All files asked could not be restored"},
		}},
		{[]string{"missing_slice.stdout", "missing_slice.stderr"}, []DarError{
			{Kind: ErrorNotRestored, Path: "/work/out3/my dir/big file.bin", Message: "user choice"},
		}},
		{[]string{"nonexistent_root.stdout", "nonexistent_root.stderr"}, []DarError{
			{Kind: ErrorFatal, Message: "Cannot read filesystem information at /nonexistent: No such file or directory"},
		}},
		{[]string{"missing_archive.stdout", "missing_archive.stderr"}, []DarError{
			{Kind: ErrorFatal, Message: "No backup file is present in file:///work for archive nope, please provide the last file of the set."},
		}},
		{[]string{"hook_failed.stdout", "hook_failed.stderr"}, []DarError{
			{Kind: ErrorFatal, Message: "Fatal error on user command line: Ignore previous error on user command line and continue ?"},
		}},
	}
	for _, tt := range tests {
		p := parseFixtures(t, tt.fixtures...)
		got := p.Errors()
		for i := range got {
			got[i].Line = ""
		}
		if len(got) != len(tt.want) {
			t.Errorf("%v: Errors() = %+v, want %+v", tt.fixtures, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("%v: error %d = %+v, want %+v", tt.fixtures, i, got[i], tt.want[i])
			}
		}
	}
}

func TestErrorsReturnsCopy(t *testing.T) {
	p := NewVerboseParser()
	p.ParseLine("Aborting due to exception: x")
	p.Errors()[0].Message = "changed"
	if got := p.Errors()[0].Message; got != "x" {
		t.Errorf("Errors() exposed internal state, message now %q", got)
	}
}

func TestDarErrorString(t *testing.T) {
	if got := (DarError{Path: "/a", Message: "boom"}).Error(); got != "/a: boom" {
		t.Errorf("Error() = %q", got)
	}
	if got := (DarError{Message: "boom"}).Error(); got != "boom" {
		t.Errorf("Error() = %q", got)
	}
	if !strings.Contains(fixture(t, "create_v.stdout"), "Adding file to archive") {
		t.Error("fixture create_v.stdout looks wrong")
	}
}
