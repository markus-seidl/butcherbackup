package dar

import (
	"regexp"
	"slices"
	"strings"
	"testing"
)

func parseListing(t *testing.T, name string) []SliceEntry {
	t.Helper()
	entries, err := ParseSlicing(strings.NewReader(fixture(t, name)))
	if err != nil {
		t.Fatal(err)
	}
	return entries
}

func TestParseSlicingFixture(t *testing.T) {
	want := []SliceEntry{
		{"top", EntryFile, []SliceRange{{1, 1}}},
		{"d", EntryDirectory, []SliceRange{{1, 1}}},
		{"d/f1", EntryFile, []SliceRange{{1, 1}}},
		{"d/sym", EntrySymlink, nil},
		{"d/hard", EntryFile, []SliceRange{{1, 1}}},
		{"d/fifo", EntryOther, nil},
		{"my dir", EntryDirectory, []SliceRange{{1, 1}}},
		{"my dir/big file.bin", EntryFile, []SliceRange{{1, 4}}},
		{"my dir/sub", EntryDirectory, []SliceRange{{4, 4}}},
		{"my dir/sub/we ird: name.txt", EntryFile, []SliceRange{{4, 4}}},
	}
	got := parseListing(t, "slicing_listing.stdout")
	if len(got) != len(want) {
		t.Fatalf("got %d entries %+v, want %d", len(got), got, len(want))
	}
	for i := range want {
		if got[i].Name != want[i].Name || got[i].Type != want[i].Type || !slices.Equal(got[i].Slices, want[i].Slices) {
			t.Errorf("entry %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestParseSlicingLines(t *testing.T) {
	tests := []struct {
		name string
		line string
		want *SliceEntry
	}{
		{"single slice", "4\t [Saved][ ]       [-L-][     ][ ] -rw-r--r-- my dir/sub/we ird: name.txt", &SliceEntry{"my dir/sub/we ird: name.txt", EntryFile, []SliceRange{{4, 4}}}},
		{"range", "1-4\t [Saved][ ]       [-L-][     ][ ] -rw-r--r-- big.bin", &SliceEntry{"big.bin", EntryFile, []SliceRange{{1, 4}}}},
		{"no slice", "\t [Saved][-]       [---][-----][ ] lrwxrwxrwx link", &SliceEntry{"link", EntrySymlink, nil}},
		{"list of slices", "1,3-4\t [Saved][ ]       [-L-][     ][ ] -rw-r--r-- odd", &SliceEntry{"odd", EntryFile, []SliceRange{{1, 1}, {3, 4}}}},
		{"directory", "2\t [Saved][-]       [-L-][   0%][ ] drwxr-xr-x dir", &SliceEntry{"dir", EntryDirectory, []SliceRange{{2, 2}}}},
		{"leading and trailing spaces in name", "1\t [Saved][ ]       [-L-][     ][ ] -rw-r--r--  x y ", &SliceEntry{" x y ", EntryFile, []SliceRange{{1, 1}}}},
		{"carriage return", "1\t [Saved][ ]       [-L-][     ][ ] -rw-r--r-- crlf\r", &SliceEntry{"crlf", EntryFile, []SliceRange{{1, 1}}}},
		{"header", "Slice(s)|[Data ][D][ EA  ][FSA][Compr][S]|Permission| Filemane", nil},
		{"separator", "--------+--------------------------------+----------+-----------------------------", nil},
		{"footer", "All displayed files have their data in slice range [1-4]", nil},
		{"dashes", "-----", nil},
		{"removed entry", "\t [--- REMOVED ENTRY ----]d/sym", nil},
		{"empty", "", nil},
		{"verbose noise", "Adding file to archive: /data/a.txt", nil},
		{"bad range", "4-2\t [Saved][ ]       [-L-][     ][ ] -rw-r--r-- bad", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSlicing(strings.NewReader(tt.line + "\n"))
			if err != nil {
				t.Fatal(err)
			}
			if tt.want == nil {
				if len(got) != 0 {
					t.Errorf("got %+v, want no entry", got)
				}
				return
			}
			if len(got) != 1 || got[0].Name != tt.want.Name || got[0].Type != tt.want.Type || !slices.Equal(got[0].Slices, tt.want.Slices) {
				t.Errorf("got %+v, want %+v", got, *tt.want)
			}
		})
	}
}

func TestParseSlicingDifferentialFixture(t *testing.T) {
	entries := parseListing(t, "slicing_diff_listing.stdout")
	byName := map[string]SliceEntry{}
	for _, e := range entries {
		byName[e.Name] = e
	}
	if e := byName["top"]; !slices.Equal(e.Slices, []SliceRange{{1, 1}}) {
		t.Errorf("top = %+v, want slice 1", e)
	}
	for _, name := range []string{"d/f1", "d/hard", "my dir/big file.bin", "my dir/sub/we ird: name.txt"} {
		e, ok := byName[name]
		if !ok || len(e.Slices) != 0 {
			t.Errorf("%s = %+v (present %v), want an entry without slices", name, e, ok)
		}
	}
	if _, ok := byName["d/sym"]; ok {
		t.Error("removed entry d/sym was reported as an entry")
	}
}

func TestNeededSlices(t *testing.T) {
	entries := parseListing(t, "slicing_listing.stdout")
	tests := []struct {
		pattern     string
		wantSlices  []int
		wantMatches []string
	}{
		{`big file`, []int{1, 2, 3, 4}, []string{"my dir/big file.bin"}},
		{`^top$`, []int{1}, []string{"top"}},
		{`\.txt$`, []int{4}, []string{"my dir/sub/we ird: name.txt"}},
		{`^d/`, []int{1}, []string{"d/f1", "d/sym", "d/hard", "d/fifo"}},
		{`we ird: name`, []int{4}, []string{"my dir/sub/we ird: name.txt"}},
		{`sym`, []int{}, []string{"d/sym"}},
		{`^nothing-matches$`, []int{}, nil},
		{`.`, []int{1, 2, 3, 4}, nil},
		{`^(top|my dir/sub/.*)$`, []int{1, 4}, []string{"top", "my dir/sub/we ird: name.txt"}},
	}
	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			gotSlices, matched := NeededSlices(entries, regexp.MustCompile(tt.pattern))
			if !slices.Equal(gotSlices, tt.wantSlices) {
				t.Errorf("slices = %v, want %v", gotSlices, tt.wantSlices)
			}
			if tt.wantMatches == nil {
				return
			}
			var names []string
			for _, e := range matched {
				names = append(names, e.Name)
			}
			if !slices.Equal(names, tt.wantMatches) {
				t.Errorf("matched = %q, want %q", names, tt.wantMatches)
			}
		})
	}
}

func TestNeededSlicesSortedAndUnique(t *testing.T) {
	entries := []SliceEntry{
		{"b", EntryFile, []SliceRange{{2, 3}}},
		{"a", EntryFile, []SliceRange{{1, 2}}},
		{"c", EntryFile, []SliceRange{{5, 5}, {2, 2}}},
	}
	got, matched := NeededSlices(entries, regexp.MustCompile(`.`))
	if want := []int{1, 2, 3, 5}; !slices.Equal(got, want) {
		t.Errorf("slices = %v, want %v", got, want)
	}
	if len(matched) != 3 || matched[0].Name != "b" {
		t.Errorf("matched = %+v, want listing order", matched)
	}
}

func TestFindSlices(t *testing.T) {
	got, matched, err := FindSlices(strings.NewReader(fixture(t, "slicing_listing.stdout")), `big file`)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, []int{1, 2, 3, 4}) || len(matched) != 1 {
		t.Errorf("slices = %v, matched = %+v", got, matched)
	}

	got, matched, err = FindSlices(strings.NewReader(fixture(t, "slicing_listing.stdout")), `nope`)
	if err != nil || len(got) != 0 || len(matched) != 0 {
		t.Errorf("no match: slices = %v, matched = %+v, err = %v", got, matched, err)
	}
}

func TestFindSlicesInvalidExpression(t *testing.T) {
	got, matched, err := FindSlices(strings.NewReader(fixture(t, "slicing_listing.stdout")), `(unclosed`)
	if err == nil || !strings.Contains(err.Error(), "(unclosed") {
		t.Errorf("error = %v, want it to name the expression", err)
	}
	if got != nil || matched != nil {
		t.Errorf("slices = %v, matched = %v, want none", got, matched)
	}
}
