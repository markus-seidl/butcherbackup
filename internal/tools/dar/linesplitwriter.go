package dar

import (
	"bytes"
	"io"
)

// LineSplitWriter turns written chunks into complete lines for a VerboseParser. Each
// writer keeps its own partial line, so give stdout and stderr one writer each.
type LineSplitWriter struct {
	parser *VerboseParser
	buf    []byte
}

var _ io.WriteCloser = (*LineSplitWriter)(nil)

// NewWriter returns a writer that feeds the parser line by line.
func (p *VerboseParser) NewWriter() *LineSplitWriter { return &LineSplitWriter{parser: p} }

func (w *LineSplitWriter) Write(data []byte) (int, error) {
	w.buf = append(w.buf, data...)
	for {
		i := bytes.IndexByte(w.buf, '\n')
		if i < 0 {
			break
		}
		w.parser.ParseLine(string(w.buf[:i]))
		w.buf = w.buf[i+1:]
	}
	return len(data), nil
}

// Close parses a trailing line that has no newline.
func (w *LineSplitWriter) Close() error {
	if len(w.buf) > 0 {
		w.parser.ParseLine(string(w.buf))
		w.buf = nil
	}
	return nil
}
