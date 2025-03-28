package vegr

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"git.sr.ht/~nelsam/hel/vegr/ret"
)

type methodPrefs struct {
	variadic bool
}

type Option func(methodPrefs) methodPrefs

func Variadic() Option {
	return func(m methodPrefs) methodPrefs {
		m.variadic = true
		return m
	}
}

// Method is a type which can track method call arguments and return values.
type Method[T, U any] struct {
	rcvr     string
	name     string
	variadic bool
	block    ret.Blocker
	in       chan T
	out      chan U
}

// NewMethod creates a new method struct using buffer as the channel buffer for
// input and output channels.
func NewMethod[T, U any](receiver, name string, buffer int, opts ...Option) Method[T, U] {
	var prefs methodPrefs
	for _, opt := range opts {
		prefs = opt(prefs)
	}
	return Method[T, U]{
		rcvr:     receiver,
		name:     name,
		variadic: prefs.variadic,
		block:    BlockChan(),
		in:       make(chan T, buffer),
		out:      make(chan U, buffer),
	}
}

// Block returns the blocking channel for m.
func (m Method[T, U]) Block() ret.Blocker {
	return m.block
}

// In returns the input channel for m.
func (m Method[T, U]) In() chan T {
	return m.in
}

// InDesc describes the parameters for this method.
func (m Method[T, U]) InDesc() string {
	t := reflect.TypeFor[T]()
	return tupleDesc(t, true, m.Variadic())
}

// Out returns the output channel for m.
func (m Method[T, U]) Out() chan U {
	return m.out
}

// OutDesc describes the parameters for this method.
func (m Method[T, U]) OutDesc() string {
	t := reflect.TypeFor[U]()
	return tupleDesc(t, false, false, ret.PanicFieldIdx(t))
}

// String describes the method's type signature.
func (m Method[T, U]) String() string {
	meth := fmt.Sprintf("%s.%s%s", m.rcvr, m.name, m.InDesc())
	out := m.OutDesc()
	if out != "" {
		meth = fmt.Sprintf("%s %s", meth, out)
	}
	return meth
}

// Variadic reports whether or not this method's final argument is a variadic
// argument.
func (m Method[T, U]) Variadic() bool {
	return m.variadic
}

func tupleDesc(t reflect.Type, forceParens bool, isVariadic bool, ignored ...int) string {
	var elems []string
	for i := 0; i < t.NumField(); i++ {
		if slices.Contains(ignored, i) {
			continue
		}
		typStr := t.Field(i).Type.String()
		typStr = strings.ReplaceAll(typStr, "interface {}", "any")
		if i == t.NumField()-1 && isVariadic {
			typStr = strings.Replace(typStr, "[]", "...", 1)
		}
		elems = append(elems, typStr)
	}
	if !forceParens {
		switch len(elems) {
		case 0:
			return ""
		case 1:
			return elems[0]
		}
	}
	return fmt.Sprintf("(%s)", strings.Join(elems, ", "))
}
