package dar

import (
	"slices"
	"testing"
)

func mustBuild(t *testing.T, b *Builder) *Command {
	t.Helper()
	cmd, err := b.Build()
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}
	return cmd
}

func TestArgs(t *testing.T) {
	tests := []struct {
		name string
		b    *Builder
		want []string
	}{
		{"create", NewBuilder().Create("/backup/test"), []string{"-c", "/backup/test"}},
		{"extract", NewBuilder().Extract("/backup/test"), []string{"-x", "/backup/test"}},
		{"list", NewBuilder().List("/backup/test"), []string{"-l", "/backup/test"}},
		{"isolate", NewBuilder().Isolate("/cat").Reference("/backup/test"), []string{"-C", "/cat", "-A", "/backup/test"}},
		{"root", NewBuilder().Create("a").Root("/workspace"), []string{"-c", "a", "-R", "/workspace"}},
		{"reference on extract", NewBuilder().Extract("a").Reference("cat"), []string{"-x", "a", "-A", "cat"}},
		{"on-the-fly isolate", NewBuilder().Create("a").OnFlyIsolate("cat"), []string{"-c", "a", "-@", "cat"}},
		{"verbose", NewBuilder().Create("a").Verbose(), []string{"-c", "a", "-v"}},
		{"first slice header", NewBuilder().Extract("a").FirstSliceHeader(), []string{"-x", "a", "-affs"}},
		{"slice sizes", NewBuilder().Create("a").SliceSize("100M").FirstSliceSize("200"), []string{"-c", "a", "-s", "100M", "-S", "200"}},
		{"binary size units", NewBuilder().Create("a").SliceSize("10Mi"), []string{"-c", "a", "-s", "10Mi"}},
		{"compression default", NewBuilder().Create("a").Compress(AlgorithmDefault, 0), []string{"-c", "a", "-z"}},
		{"compression algorithm", NewBuilder().Create("a").Compress(AlgorithmXz, 0), []string{"-c", "a", "-zxz"}},
		{"compression algorithm and level", NewBuilder().Create("a").Compress(AlgorithmZstd, 3), []string{"-c", "a", "-zzstd:3"}},
		{"compression level only", NewBuilder().Create("a").Compress(AlgorithmDefault, 5), []string{"-c", "a", "-z5"}},
		{"format normal", NewBuilder().List("a").Format(FormatNormal), []string{"-l", "a", "-Tnormal"}},
		{"format tree", NewBuilder().List("a").Format(FormatTree), []string{"-l", "a", "-Ttree"}},
		{"format xml", NewBuilder().List("a").Format(FormatXML), []string{"-l", "a", "-Txml"}},
		{"format slicing", NewBuilder().List("a").Format(FormatSlicing), []string{"-l", "a", "-Tslicing"}},
		{
			"several options",
			NewBuilder().Create("/backup/test").Root("/workspace").SliceSize("100M").FirstSliceSize("200").Compress(AlgorithmDefault, 0),
			[]string{"-c", "/backup/test", "-R", "/workspace", "-z", "-s", "100M", "-S", "200"},
		},
		{
			"all options in fixed order",
			NewBuilder().Verbose().SliceSize("1G").Root("/r").OnFlyIsolate("cat").Create("a").Reference("ref").Compress(AlgorithmGzip, 9).FirstSliceSize("1k").FirstSliceHeader(),
			[]string{"-c", "a", "-A", "ref", "-@", "cat", "-R", "/r", "-zgzip:9", "-s", "1G", "-S", "1k", "-affs", "-v"},
		},
		{"include mask", NewBuilder().Create("a").IncludeMask("*.txt"), []string{"-c", "a", "-I", "*.txt"}},
		{"exclude mask", NewBuilder().Create("a").ExcludeMask("tmp*"), []string{"-c", "a", "-X", "tmp*"}},
		{"include path", NewBuilder().Extract("a").IncludePath("tools/dev-vm/dev_vm.smolfile"), []string{"-x", "a", "-g", "tools/dev-vm/dev_vm.smolfile"}},
		{"exclude path", NewBuilder().Create("a").ExcludePath("docs/old"), []string{"-c", "a", "-P", "docs/old"}},
		{"filters on list", NewBuilder().List("a").IncludePath("docs").ExcludeMask("*.log"), []string{"-l", "a", "-X", "*.log", "-g", "docs"}},
		{
			"repeated filters keep order",
			NewBuilder().Create("a").IncludePath("a", "b").IncludePath("c").IncludeMask("*.go", "*.md").ExcludeMask("x"),
			[]string{"-c", "a", "-I", "*.go", "-I", "*.md", "-X", "x", "-g", "a", "-g", "b", "-g", "c"},
		},
		{
			"all four filters",
			NewBuilder().Create("a").ExcludePath("docs/old").IncludePath("docs").ExcludeMask("tmp*").IncludeMask("*.txt"),
			[]string{"-c", "a", "-I", "*.txt", "-X", "tmp*", "-g", "docs", "-P", "docs/old"},
		},
		{
			"filters after other options",
			NewBuilder().List("a").IncludePath("p").Format(FormatTree).Verbose(),
			[]string{"-l", "a", "-v", "-Ttree", "-g", "p"},
		},
		{
			"list with first slice header",
			NewBuilder().List("a").FirstSliceHeader().Verbose().Format(FormatSlicing),
			[]string{"-l", "a", "-affs", "-v", "-Tslicing"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mustBuild(t, tt.b).Args()
			if !slices.Equal(got, tt.want) {
				t.Errorf("Args() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestArgsKeepPathsAsSingleArguments(t *testing.T) {
	archive := "/my backups/test; rm -rf *"
	root := "/data/with space/$HOME"
	got := mustBuild(t, NewBuilder().Create(archive).Root(root)).Args()
	want := []string{"-c", archive, "-R", root}
	if !slices.Equal(got, want) {
		t.Errorf("Args() = %q, want %q", got, want)
	}
}

func TestBuildIsRepeatable(t *testing.T) {
	b := NewBuilder().Create("a").Root("/r").Compress(AlgorithmZstd, 3).SliceSize("10M").Verbose()
	first := mustBuild(t, b).Args()
	second := mustBuild(t, b).Args()
	if !slices.Equal(first, second) {
		t.Errorf("builds differ: %q vs %q", first, second)
	}
}

func TestArgsReturnsCopy(t *testing.T) {
	cmd := mustBuild(t, NewBuilder().Create("a"))
	cmd.Args()[0] = "changed"
	if got := cmd.Args()[0]; got != "-c" {
		t.Errorf("Args() exposed internal slice, first arg now %q", got)
	}
}

func TestProgram(t *testing.T) {
	if got := mustBuild(t, NewBuilder().Create("a")).Program(); got != "dar" {
		t.Errorf("default Program() = %q, want dar", got)
	}
	if got := mustBuild(t, NewBuilder().Create("a").Binary("/opt/dar")).Program(); got != "/opt/dar" {
		t.Errorf("Program() = %q, want /opt/dar", got)
	}
}

func TestFiltersStaySingleArguments(t *testing.T) {
	got := mustBuild(t, NewBuilder().Create("a").IncludeMask("my file*.txt").ExcludeMask("$HOME;*").IncludePath("dir with space").ExcludePath("x y/z")).Args()
	want := []string{"-c", "a", "-I", "my file*.txt", "-X", "$HOME;*", "-g", "dir with space", "-P", "x y/z"}
	if !slices.Equal(got, want) {
		t.Errorf("Args() = %q, want %q", got, want)
	}
}
