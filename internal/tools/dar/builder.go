package dar

import (
	"errors"
	"slices"
)

const defaultBinary = "dar"

type operationSpec struct {
	op      Operation
	archive string
}

// Builder collects dar settings through chained calls; Build validates them.
type Builder struct {
	binary         string
	ops            []operationSpec
	reference      string
	onFlyIsolate   string
	root           string
	compression    *Compression
	sliceSize      string
	firstSliceSize string
	verbose        bool
	firstSliceHdr  bool
	listFormat     ListFormat
	includeMasks   []string
	excludeMasks   []string
	includePaths   []string
	excludePaths   []string
	sliceHook      *string
	refSliceHook   *string
}

func NewBuilder() *Builder { return &Builder{} }

func (b *Builder) operation(op Operation, archive string) *Builder {
	b.ops = append(b.ops, operationSpec{op, archive})
	return b
}

func (b *Builder) Create(archive string) *Builder  { return b.operation(OpCreate, archive) }
func (b *Builder) Extract(archive string) *Builder { return b.operation(OpExtract, archive) }
func (b *Builder) List(archive string) *Builder    { return b.operation(OpList, archive) }
func (b *Builder) Isolate(archive string) *Builder { return b.operation(OpIsolate, archive) }

func (b *Builder) Binary(path string) *Builder { b.binary = path; return b }

// Reference sets -A, the reference archive or isolated catalogue.
func (b *Builder) Reference(archive string) *Builder { b.reference = archive; return b }

// OnFlyIsolate sets -@, the isolated catalogue created together with the archive.
func (b *Builder) OnFlyIsolate(archive string) *Builder { b.onFlyIsolate = archive; return b }

func (b *Builder) Root(path string) *Builder { b.root = path; return b }

func (b *Builder) Compress(algorithm Algorithm, level int) *Builder {
	b.compression = &Compression{Algorithm: algorithm, Level: level}
	return b
}

func (b *Builder) SliceSize(size string) *Builder      { b.sliceSize = size; return b }
func (b *Builder) FirstSliceSize(size string) *Builder { b.firstSliceSize = size; return b }
func (b *Builder) Verbose() *Builder                   { b.verbose = true; return b }

// FirstSliceHeader sets -affs: read the archive header from the first slice.
func (b *Builder) FirstSliceHeader() *Builder { b.firstSliceHdr = true; return b }

func (b *Builder) Format(f ListFormat) *Builder { b.listFormat = f; return b }

// ExecuteBetweenSlices sets -E: dar runs command (through a shell) whenever it
// needs a slice. The string is passed unmodified, dar macros such as %n included.
func (b *Builder) ExecuteBetweenSlices(command string) *Builder {
	b.sliceHook = &command
	return b
}

// ExecuteBetweenReferenceSlices sets -F, like -E but for the reference archive.
func (b *Builder) ExecuteBetweenReferenceSlices(command string) *Builder {
	b.refSliceHook = &command
	return b
}

// IncludeMask adds -I masks: only files whose name matches are kept.
func (b *Builder) IncludeMask(masks ...string) *Builder {
	b.includeMasks = append(b.includeMasks, masks...)
	return b
}

// ExcludeMask adds -X masks: files whose name matches are skipped.
func (b *Builder) ExcludeMask(masks ...string) *Builder {
	b.excludeMasks = append(b.excludeMasks, masks...)
	return b
}

// IncludePath adds -g paths, relative to the root directory.
func (b *Builder) IncludePath(paths ...string) *Builder {
	b.includePaths = append(b.includePaths, paths...)
	return b
}

// ExcludePath adds -P paths, relative to the root directory.
func (b *Builder) ExcludePath(paths ...string) *Builder {
	b.excludePaths = append(b.excludePaths, paths...)
	return b
}

// Build validates the settings and returns the command, or all problems found.
func (b *Builder) Build() (*Command, error) {
	if err := errors.Join(b.validate()...); err != nil {
		return nil, err
	}
	program := b.binary
	if program == "" {
		program = defaultBinary
	}
	return &Command{program: program, args: b.args()}, nil
}

func (b *Builder) args() []string {
	var args []string
	args = append(args, b.ops[0].op.flag(), b.ops[0].archive)
	if b.reference != "" {
		args = append(args, "-A", b.reference)
	}
	if b.onFlyIsolate != "" {
		args = append(args, "-@", b.onFlyIsolate)
	}
	if b.root != "" {
		args = append(args, "-R", b.root)
	}
	if b.compression != nil {
		args = append(args, b.compression.arg())
	}
	if b.sliceSize != "" {
		args = append(args, "-s", b.sliceSize)
	}
	if b.firstSliceSize != "" {
		args = append(args, "-S", b.firstSliceSize)
	}
	if b.firstSliceHdr {
		args = append(args, "-affs")
	}
	if b.verbose {
		args = append(args, "-v")
	}
	if b.listFormat != FormatNone {
		args = append(args, "-T"+string(b.listFormat))
	}
	for _, f := range b.filters() {
		for _, v := range f.values {
			args = append(args, f.flag, v)
		}
	}
	if b.sliceHook != nil {
		args = append(args, "-E", *b.sliceHook)
	}
	if b.refSliceHook != nil {
		args = append(args, "-F", *b.refSliceHook)
	}
	return args
}

// Command is a validated dar invocation.
type Command struct {
	program string
	args    []string
}

func (c *Command) Program() string { return c.program }

// Args returns a copy of the arguments, without the program name.
func (c *Command) Args() []string { return slices.Clone(c.args) }

type filter struct {
	flag   string
	name   string
	values []string
	isPath bool
}

func (b *Builder) filters() []filter {
	return []filter{
		{"-I", "include mask", b.includeMasks, false},
		{"-X", "exclude mask", b.excludeMasks, false},
		{"-g", "include path", b.includePaths, true},
		{"-P", "exclude path", b.excludePaths, true},
	}
}
