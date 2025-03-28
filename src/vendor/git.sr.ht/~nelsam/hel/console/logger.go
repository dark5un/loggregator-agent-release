package console

import (
	"context"
	"fmt"
	"io"
	"slices"
	"sync"
)

type ctxKey struct{}

// Logger is a type which may be used to write logs to a console.
type Logger struct {
	w       io.Writer
	pfx     string
	verbose bool

	// printed is used by the Once option to only print a given message once.
	printed sync.Map
}

type Opt func(*Logger)

func Verbose() Opt {
	return func(l *Logger) {
		l.verbose = true
	}
}

type newlinePos int

const (
	newlineEnd newlinePos = iota
	newlineStart
	newlineNone
)

type printPrefs struct {
	once      bool
	fmtArgs   []any
	verbose   bool
	newlineAt newlinePos
}

type PrintOption func(printPrefs) printPrefs

// Once ensures that this message is only printed once. This is useful for
// messages that might get repeated by multiple goroutines.
func Once() PrintOption {
	return func(p printPrefs) printPrefs {
		p.once = true
		return p
	}
}

// Fmt adds format args to the print statement.
func Fmt(args ...any) PrintOption {
	return func(p printPrefs) printPrefs {
		p.fmtArgs = args
		return p
	}
}

// AsVerbose marks the message as a verbose message, only printing if the
// console logger is in verbose mode.
func AsVerbose() PrintOption {
	return func(p printPrefs) printPrefs {
		p.verbose = true
		return p
	}
}

// NewlinePfx changes the message to _start_ with a newline, rather than end
// with one. Loggers usually end every message with a newline. This moves the
// newline to the start.
func NewlinePfx() PrintOption {
	return func(p printPrefs) printPrefs {
		p.newlineAt = newlineStart
		return p
	}
}

// NewLogger constructs a Logger to write to w.
func NewLogger(w io.Writer, opts ...Opt) *Logger {
	l := &Logger{w: w}
	for _, o := range opts {
		o(l)
	}
	return l
}

// WithPrefix returns a new Logger with pfx added. Any existing prefix in l will
// be prepended.
func (l *Logger) WithPrefix(pfx string) *Logger {
	if l.pfx != "" {
		pfx = fmt.Sprintf("%s%s", l.pfx, pfx)
	}
	return &Logger{
		w:       l.w,
		pfx:     pfx,
		verbose: l.verbose,
	}
}

// Print prints a line of output to the console.
func (l *Logger) Print(output string, opts ...PrintOption) error {
	var p printPrefs
	for _, o := range opts {
		p = o(p)
	}
	if p.verbose && !l.verbose {
		return nil
	}
	if len(p.fmtArgs) > 0 {
		output = fmt.Sprintf(output, p.fmtArgs...)
	}
	if p.once {
		if _, existed := l.printed.LoadOrStore(output, struct{}{}); existed {
			return nil
		}
	}
	msg := append([]byte(l.pfx), []byte(output)...)
	switch p.newlineAt {
	case newlineStart:
		msg = slices.Insert(msg, 0, '\n')
	case newlineEnd:
		msg = append(msg, '\n')
	}
	if _, err := l.w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("console: could not write log: %w", err)
	}
	return nil
}

// Ctx returns a child context.Context with l set as the context's logger.
func (l *Logger) Ctx(c context.Context) context.Context {
	return context.WithValue(c, ctxKey{}, l)
}

// FromCtx retrieves a Logger from c.
func FromCtx(c context.Context) *Logger {
	l := c.Value(ctxKey{})
	if l == nil {
		return NewLogger(io.Discard)
	}
	return l.(*Logger)
}
