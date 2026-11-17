package dar

import (
	"fmt"
	"regexp"
)

type Operation int

const (
	OpNone Operation = iota
	OpCreate
	OpExtract
	OpList
	OpIsolate
)

func (o Operation) flag() string {
	switch o {
	case OpCreate:
		return "-c"
	case OpExtract:
		return "-x"
	case OpList:
		return "-l"
	case OpIsolate:
		return "-C"
	}
	return ""
}

func (o Operation) String() string {
	switch o {
	case OpCreate:
		return "create"
	case OpExtract:
		return "extract"
	case OpList:
		return "list"
	case OpIsolate:
		return "isolate"
	}
	return "none"
}

type Algorithm string

const (
	AlgorithmDefault Algorithm = ""
	AlgorithmGzip    Algorithm = "gzip"
	AlgorithmBzip2   Algorithm = "bzip2"
	AlgorithmXz      Algorithm = "xz"
	AlgorithmZstd    Algorithm = "zstd"
)

func (a Algorithm) valid() bool {
	switch a {
	case AlgorithmDefault, AlgorithmGzip, AlgorithmBzip2, AlgorithmXz, AlgorithmZstd:
		return true
	}
	return false
}

// Compression describes the -z option. A zero Level means "not specified".
type Compression struct {
	Algorithm Algorithm
	Level     int
}

func (c Compression) arg() string {
	arg := "-z" + string(c.Algorithm)
	if c.Level > 0 {
		if c.Algorithm != AlgorithmDefault {
			arg += ":"
		}
		arg += fmt.Sprint(c.Level)
	}
	return arg
}

type ListFormat string

const (
	FormatNone    ListFormat = ""
	FormatNormal  ListFormat = "normal"
	FormatTree    ListFormat = "tree"
	FormatXML     ListFormat = "xml"
	FormatSlicing ListFormat = "slicing"
)

func (f ListFormat) valid() bool {
	switch f {
	case FormatNone, FormatNormal, FormatTree, FormatXML, FormatSlicing:
		return true
	}
	return false
}

// Size is a dar size such as "100M" or "10Mi", passed to dar unchanged.
type Size string

var sizePattern = regexp.MustCompile(`^[0-9]+([kKMGTPEZY]|[kMGTPEZY]i)?$`)

func ParseSize(s string) (Size, error) {
	if !sizePattern.MatchString(s) {
		return "", fmt.Errorf("invalid size %q: expected a number with an optional unit such as k, M, G or Mi", s)
	}
	return Size(s), nil
}
