package dar

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

type EntryType int

const (
	EntryOther EntryType = iota
	EntryFile
	EntryDirectory
	EntrySymlink
)

// SliceRange is an inclusive range of slice numbers.
type SliceRange struct{ First, Last int }

// SliceEntry is one entry of a `dar -l -Tslicing` listing.
type SliceEntry struct {
	Name   string
	Type   EntryType
	Slices []SliceRange // empty for entries without data, such as symlinks
}

// entryPattern matches "<slices>\t [flags]x5 <10-char permission> <name>". The
// name is everything after the permission, so spaces and colons are kept.
var entryPattern = regexp.MustCompile(`^([0-9,-]*)\t\s*(?:\[[^\]]*\]\s*){5}(\S{10}) (.*)$`)

// ParseSlicing reads a slicing listing. Header, footer and any other line that
// is not an entry (including "REMOVED ENTRY" lines) is ignored.
func ParseSlicing(r io.Reader) ([]SliceEntry, error) {
	var entries []SliceEntry
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		m := entryPattern.FindStringSubmatch(strings.TrimRight(sc.Text(), "\r"))
		if m == nil {
			continue
		}
		ranges, ok := parseSliceColumn(m[1])
		if !ok {
			continue
		}
		entries = append(entries, SliceEntry{Name: m[3], Type: entryType(m[2][0]), Slices: ranges})
	}
	return entries, sc.Err()
}

func entryType(c byte) EntryType {
	switch c {
	case '-':
		return EntryFile
	case 'd':
		return EntryDirectory
	case 'l':
		return EntrySymlink
	}
	return EntryOther
}

func parseSliceColumn(col string) ([]SliceRange, bool) {
	if col == "" {
		return nil, true
	}
	var ranges []SliceRange
	for _, part := range strings.Split(col, ",") {
		first, last, isRange := strings.Cut(part, "-")
		a, err := strconv.Atoi(first)
		if err != nil {
			return nil, false
		}
		b := a
		if isRange {
			if b, err = strconv.Atoi(last); err != nil || b < a {
				return nil, false
			}
		}
		ranges = append(ranges, SliceRange{a, b})
	}
	return ranges, true
}

// NeededSlices returns the sorted, unique slices holding the data of entries
// whose full name matches re, together with the matched entries in listing order.
// A matching directory contributes its own slice.
func NeededSlices(entries []SliceEntry, re *regexp.Regexp) ([]int, []SliceEntry) {
	var matched []SliceEntry
	set := map[int]bool{}
	for _, e := range entries {
		if !re.MatchString(e.Name) {
			continue
		}
		matched = append(matched, e)
		for _, r := range e.Slices {
			for n := r.First; n <= r.Last; n++ {
				set[n] = true
			}
		}
	}
	needed := make([]int, 0, len(set))
	for n := range set {
		needed = append(needed, n)
	}
	slices.Sort(needed)
	return needed, matched
}

// FindSlices parses a slicing listing and returns the slices needed for the
// entries whose name matches pattern (a regular expression, unanchored).
func FindSlices(r io.Reader, pattern string) ([]int, []SliceEntry, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid file expression %q: %w", pattern, err)
	}
	entries, err := ParseSlicing(r)
	if err != nil {
		return nil, nil, err
	}
	needed, matched := NeededSlices(entries, re)
	return needed, matched, nil
}
