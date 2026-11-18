package dar

import (
	"io"
	"regexp"
	"strings"
	"sync"
)

type ErrorKind string

const (
	// ErrorFile is an error about one file, such as "Error while saving <path>: <reason>".
	ErrorFile ErrorKind = "file"
	// ErrorNotRestored is a file dar did not restore, such as "<path> not restored (user choice)".
	ErrorNotRestored ErrorKind = "not-restored"
	// ErrorFatal is an error that stops dar.
	ErrorFatal ErrorKind = "fatal"
)

// DarError is an error reported in dar's verbose output.
type DarError struct {
	Kind    ErrorKind
	Path    string // empty for fatal errors
	Message string
	Line    string // the original output line
}

func (e DarError) Error() string {
	if e.Path != "" {
		return e.Path + ": " + e.Message
	}
	return e.Message
}

var (
	currentFilePatterns = []*regexp.Regexp{
		regexp.MustCompile(`^Adding \S.*? to archive: (.+)$`),
		regexp.MustCompile(`^Recording hard link into the archive: (.+)$`),
		regexp.MustCompile(`^Restoring file's data: (.+)$`),
	}
	fileErrorPattern   = regexp.MustCompile(`^Error while \w+ (.+?) ?: (.+)$`)
	fileErrorPrefix    = regexp.MustCompile(`^Error while \w+ `)
	notRestoredPattern = regexp.MustCompile(`^(\S.*?) not restored \((.+)\)$`)
	fatalPrefixes      = []string{"Aborting due to exception: ", "FATAL error, aborting operation: "}
	abortPrefix        = "Aborting program. "
	abortMessagePrefix = []string{"User refused to continue while asking: ", "An error occurred concerning user command execution: "}
)

// VerboseParser follows the output of dar run with -v. It keeps the latest file
// dar reports working on and collects the errors it reports. It is safe for
// concurrent use, so stdout and stderr can be fed from different goroutines.
// dar's verbose output does not report slice numbers, so the parser does not either.
type VerboseParser struct {
	mu      sync.Mutex
	current string
	errs    []DarError
}

func NewVerboseParser() *VerboseParser { return &VerboseParser{} }

// CurrentFile returns the latest file dar reported, or "" if none yet.
func (p *VerboseParser) CurrentFile() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.current
}

// Errors returns the errors reported so far, in order.
func (p *VerboseParser) Errors() []DarError {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]DarError(nil), p.errs...)
}

// ParseLine parses one output line; lines that are not recognized are ignored.
func (p *VerboseParser) ParseLine(line string) {
	line = strings.TrimRight(line, "\r\n")
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, re := range currentFilePatterns {
		if m := re.FindStringSubmatch(line); m != nil {
			p.current = m[1]
			return
		}
	}
	if fileErrorPrefix.MatchString(line) {
		if e, ok := p.fileError(line); ok {
			p.errs = append(p.errs, e)
		}
		return
	}
	if m := notRestoredPattern.FindStringSubmatch(line); m != nil {
		p.errs = append(p.errs, DarError{Kind: ErrorNotRestored, Path: m[1], Message: m[2], Line: line})
		return
	}
	for _, prefix := range fatalPrefixes {
		if msg, ok := strings.CutPrefix(line, prefix); ok {
			p.addFatal(msg, line)
			return
		}
	}
	if rest, ok := strings.CutPrefix(line, abortPrefix); ok {
		for _, prefix := range abortMessagePrefix {
			if msg, ok := strings.CutPrefix(rest, prefix); ok {
				rest = msg
				break
			}
		}
		p.addFatal(rest, line)
	}
}

// fileError splits "Error while <verb> <path>[ ]: <reason>". Paths may contain
// ": ", so when the line names the current file, that path is preferred.
func (p *VerboseParser) fileError(line string) (DarError, bool) {
	if p.current != "" {
		rest := fileErrorPrefix.ReplaceAllString(line, "")
		if after, ok := strings.CutPrefix(rest, p.current); ok {
			for _, sep := range []string{" : ", ": "} {
				if msg, ok := strings.CutPrefix(after, sep); ok {
					return DarError{Kind: ErrorFile, Path: p.current, Message: msg, Line: line}, true
				}
			}
		}
	}
	m := fileErrorPattern.FindStringSubmatch(line)
	if m == nil {
		return DarError{}, false
	}
	return DarError{Kind: ErrorFile, Path: m[1], Message: m[2], Line: line}, true
}

// addFatal records a fatal error unless one with the same message exists, since
// dar prints the same message on stdout and on stderr.
func (p *VerboseParser) addFatal(msg, line string) {
	for _, e := range p.errs {
		if e.Kind == ErrorFatal && e.Message == msg {
			return
		}
	}
	p.errs = append(p.errs, DarError{Kind: ErrorFatal, Message: msg, Line: line})
}

// ParseText parses a complete output text, line by line.
func (p *VerboseParser) ParseText(text string) {
	w := p.NewWriter()
	io.WriteString(w, text)
	w.Close()
}
