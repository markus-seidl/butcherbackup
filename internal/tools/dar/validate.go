package dar

import (
	"fmt"
	"path/filepath"
	"slices"
)

func (b *Builder) validate() []error {
	var errs []error
	add := func(format string, a ...any) { errs = append(errs, fmt.Errorf(format, a...)) }

	switch len(b.ops) {
	case 0:
		add("an operation is required (create, extract, list or isolate)")
	case 1:
	default:
		add("only one operation is allowed, got %d", len(b.ops))
	}
	for _, o := range b.ops {
		if o.archive == "" {
			add("%s: archive name is empty", o.op)
		}
	}

	if b.compression != nil {
		if !b.compression.Algorithm.valid() {
			add("unsupported compression algorithm %q", b.compression.Algorithm)
		}
		if b.compression.Level < 0 {
			add("invalid compression level %d", b.compression.Level)
		}
	}
	for _, s := range []struct{ name, value string }{{"slice size", b.sliceSize}, {"first slice size", b.firstSliceSize}} {
		if s.value == "" {
			continue
		}
		if _, err := ParseSize(s.value); err != nil {
			add("%s: %w", s.name, err)
		}
	}
	if b.firstSliceSize != "" && b.sliceSize == "" {
		add("a slice size is required with a first slice size")
	}
	if !b.listFormat.valid() {
		add("unsupported list format %q", string(b.listFormat))
	}

	if b.sliceHook != nil && *b.sliceHook == "" {
		add("slice hook (-E): command is empty")
	}
	if b.refSliceHook != nil {
		if *b.refSliceHook == "" {
			add("reference slice hook (-F): command is empty")
		}
		if b.reference == "" {
			add("the reference slice hook (-F) requires a reference archive")
		}
	}

	hasFilter := false
	for _, f := range b.filters() {
		for _, v := range f.values {
			hasFilter = true
			switch {
			case v == "":
				add("%s (%s): value is empty", f.name, f.flag)
			case f.isPath && filepath.IsAbs(v):
				add("%s (%s): path %q must be relative to the root directory", f.name, f.flag, v)
			}
		}
	}

	if len(b.ops) != 1 {
		return errs
	}
	op := b.ops[0].op
	if (b.sliceSize != "" || b.firstSliceSize != "") && !slices.Contains([]Operation{OpCreate, OpIsolate}, op) {
		add("slicing applies only to create and isolate, not %s", op)
	}
	if b.listFormat != FormatNone && op != OpList {
		add("the list format applies only to list, not %s", op)
	}
	if op == OpIsolate && b.reference == "" {
		add("isolate requires a reference archive")
	}
	if b.onFlyIsolate != "" && op != OpCreate {
		add("an on-the-fly isolated catalogue applies only to create, not %s", op)
	}
	if hasFilter && !slices.Contains([]Operation{OpCreate, OpExtract, OpList}, op) {
		add("filters apply only to create, extract and list, not %s", op)
	}
	if b.root != "" && !slices.Contains([]Operation{OpCreate, OpExtract}, op) {
		add("the root directory applies only to create and extract, not %s", op)
	}
	return errs
}
