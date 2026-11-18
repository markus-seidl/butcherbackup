package dar

import (
	"strings"
	"testing"
)

func TestValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		b    *Builder
		want string
	}{
		{"no operation", NewBuilder(), "an operation is required"},
		{"two operations", NewBuilder().Create("a").Extract("a"), "only one operation is allowed"},
		{"empty archive", NewBuilder().Create(""), "create: archive name is empty"},
		{"unknown algorithm", NewBuilder().Create("a").Compress("lz4", 0), `unsupported compression algorithm "lz4"`},
		{"negative level", NewBuilder().Create("a").Compress(AlgorithmGzip, -1), "invalid compression level -1"},
		{"size with words", NewBuilder().Create("a").SliceSize("10 megabytes"), `invalid size "10 megabytes"`},
		{"negative size", NewBuilder().Create("a").SliceSize("-5"), `invalid size "-5"`},
		{"unknown unit", NewBuilder().Create("a").SliceSize("10Q"), `invalid size "10Q"`},
		{"invalid first size", NewBuilder().Create("a").SliceSize("1M").FirstSliceSize("x"), `first slice size: invalid size "x"`},
		{"first size without size", NewBuilder().Create("a").FirstSliceSize("200"), "a slice size is required with a first slice size"},
		{"slicing on extract", NewBuilder().Extract("a").SliceSize("1M"), "slicing applies only to create and isolate"},
		{"slicing on list", NewBuilder().List("a").SliceSize("1M"), "slicing applies only to create and isolate"},
		{"unknown format", NewBuilder().List("a").Format("json"), `unsupported list format "json"`},
		{"format on create", NewBuilder().Create("a").Format(FormatXML), "the list format applies only to list"},
		{"isolate without reference", NewBuilder().Isolate("cat"), "isolate requires a reference archive"},
		{"on-the-fly on extract", NewBuilder().Extract("a").OnFlyIsolate("cat"), "applies only to create"},
		{"on-the-fly on isolate", NewBuilder().Isolate("c").Reference("a").OnFlyIsolate("cat"), "applies only to create"},
		{"empty include mask", NewBuilder().Create("a").IncludeMask(""), "include mask (-I): value is empty"},
		{"empty exclude mask", NewBuilder().Create("a").ExcludeMask("*.txt", ""), "exclude mask (-X): value is empty"},
		{"empty include path", NewBuilder().Extract("a").IncludePath(""), "include path (-g): value is empty"},
		{"empty exclude path", NewBuilder().Create("a").ExcludePath(""), "exclude path (-P): value is empty"},
		{"absolute include path", NewBuilder().Create("a").IncludePath("/etc"), `include path (-g): path "/etc" must be relative to the root directory`},
		{"absolute exclude path", NewBuilder().Create("a").ExcludePath("docs", "/var/log"), `exclude path (-P): path "/var/log" must be relative to the root directory`},
		{"include mask on isolate", NewBuilder().Isolate("c").Reference("a").IncludeMask("*.txt"), "filters apply only to create, extract and list"},
		{"exclude mask on isolate", NewBuilder().Isolate("c").Reference("a").ExcludeMask("*.txt"), "filters apply only to create, extract and list"},
		{"include path on isolate", NewBuilder().Isolate("c").Reference("a").IncludePath("a"), "filters apply only to create, extract and list"},
		{"exclude path on isolate", NewBuilder().Isolate("c").Reference("a").ExcludePath("a"), "filters apply only to create, extract and list"},
		{"empty slice hook", NewBuilder().Create("a").ExecuteBetweenSlices(""), "slice hook (-E): command is empty"},
		{"empty reference hook", NewBuilder().Create("a").Reference("r").ExecuteBetweenReferenceSlices(""), "reference slice hook (-F): command is empty"},
		{"reference hook without reference", NewBuilder().Create("a").ExecuteBetweenReferenceSlices("x"), "the reference slice hook (-F) requires a reference archive"},
		{"root on list", NewBuilder().List("a").Root("/r"), "the root directory applies only to create and extract"},
		{"root on isolate", NewBuilder().Isolate("c").Reference("a").Root("/r"), "the root directory applies only to create and extract"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := tt.b.Build()
			if err == nil {
				t.Fatalf("Build() = %q, want error containing %q", cmd.Args(), tt.want)
			}
			if cmd != nil {
				t.Errorf("Build() returned a command together with an error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %q, want it to contain %q", err, tt.want)
			}
		})
	}
}

func TestValidSizes(t *testing.T) {
	for _, s := range []string{"1", "200", "100k", "100K", "100M", "2G", "1T", "1P", "1E", "1Z", "1Y", "10ki", "10Mi", "3Gi"} {
		if _, err := NewBuilder().Create("a").SliceSize(s).Build(); err != nil {
			t.Errorf("size %q rejected: %v", s, err)
		}
	}
}

func TestValidAlgorithms(t *testing.T) {
	for _, a := range []Algorithm{AlgorithmDefault, AlgorithmGzip, AlgorithmBzip2, AlgorithmXz, AlgorithmZstd} {
		if _, err := NewBuilder().Create("a").Compress(a, 0).Build(); err != nil {
			t.Errorf("algorithm %q rejected: %v", a, err)
		}
	}
}

func TestValidCombinations(t *testing.T) {
	builders := map[string]*Builder{
		"isolate with slicing": NewBuilder().Isolate("c").Reference("a").SliceSize("1M").FirstSliceSize("200").Compress(AlgorithmDefault, 0),
		"create differential":  NewBuilder().Create("d").Reference("cat").Root("/r").OnFlyIsolate("dcat"),
		"extract from catalog": NewBuilder().Extract("a").Reference("cat").FirstSliceHeader().Root("/r").Verbose(),
		"list slicing":         NewBuilder().List("a").Format(FormatSlicing).Reference("cat"),
		"create with filters":  NewBuilder().Create("a").Root("/r").IncludeMask("*.go").ExcludeMask("*_test.go").IncludePath("src").ExcludePath("src/gen"),
		"extract with filter":  NewBuilder().Extract("a").Root("/r").IncludePath("src/main.go"),
		"hooks with reference": NewBuilder().Extract("a").Reference("cat").ExecuteBetweenSlices("e").ExecuteBetweenReferenceSlices("f"),
		"list with filters":    NewBuilder().List("a").IncludePath("src").ExcludeMask("*.log"),
	}
	for name, b := range builders {
		if _, err := b.Build(); err != nil {
			t.Errorf("%s: unexpected error: %v", name, err)
		}
	}
}

func TestSeveralErrorsReportedTogether(t *testing.T) {
	_, err := NewBuilder().Isolate("cat").Root("/r").SliceSize("bad").Format("json").Compress("lz4", 0).IncludePath("/abs").IncludeMask("").ExecuteBetweenSlices("").ExecuteBetweenReferenceSlices("x").Build()
	if err == nil {
		t.Fatal("Build() succeeded, want errors")
	}
	for _, want := range []string{
		"isolate requires a reference archive",
		"the root directory applies only to create and extract",
		`invalid size "bad"`,
		`unsupported list format "json"`,
		`unsupported compression algorithm "lz4"`,
		`path "/abs" must be relative`,
		"include mask (-I): value is empty",
		"filters apply only to create, extract and list",
		"slice hook (-E): command is empty",
		"the reference slice hook (-F) requires a reference archive",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q is missing %q", err, want)
		}
	}
}

func TestParseSize(t *testing.T) {
	if got, err := ParseSize("100M"); err != nil || got != "100M" {
		t.Errorf("ParseSize(100M) = %q, %v", got, err)
	}
	if _, err := ParseSize(""); err == nil {
		t.Error("ParseSize(\"\") succeeded, want error")
	}
}
