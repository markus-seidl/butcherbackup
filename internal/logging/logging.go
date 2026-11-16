package logging

import (
	"io"

	"github.com/rs/zerolog"
)

// New returns a logger that writes only the message of each event to w.
func New(w io.Writer) zerolog.Logger {
	cw := zerolog.ConsoleWriter{
		Out:        w,
		NoColor:    true,
		PartsOrder: []string{zerolog.MessageFieldName},
	}
	return zerolog.New(cw)
}
