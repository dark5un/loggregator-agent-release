package vegr

import (
	"fmt"
	"reflect"
	"slices"
	"time"

	"git.sr.ht/~nelsam/hel/vegr/ret"
)

// TestingT represents the properties of *testing.T that vegr uses.
type TestingT interface {
	Helper()
	Failed() bool
	Logf(string, ...any)
	Fatalf(string, ...any)
	Cleanup(func())
}

// ReturnMethod represents the methods that PopulateReturns uses from a Method.
type ReturnMethod[T any] interface {
	Block() ret.Blocker
	Out() chan T
}

// PopulateReturns handles populating return value addresses with values from channels in a mock.
func PopulateReturns[T any](t TestingT, timeout time.Duration, mock ReturnMethod[T], addrs ...any) {
	t.Helper()

	doneCh := make(chan struct{})
	t.Cleanup(func() { <-doneCh })
	defer func() {
		defer close(doneCh)
		if r := recover(); r != nil {
			if t.Failed() {
				return
			}
			panic(r)
		}
	}()

	var setters []reflect.Value
	for _, a := range addrs {
		v := reflect.ValueOf(a)
		if v.Kind() != reflect.Ptr {
			panic(fmt.Errorf("hel: PopulateReturns was called with non-pointer type [%T]", a))
		}
		if v.IsNil() {
			panic(fmt.Errorf("hel: PopulateReturns was called with nil pointer of type [%T]", a))
		}
		setters = append(setters, v.Elem())
	}

	returns := len(addrs)
	outTyp := reflect.TypeFor[T]()
	fields := outTyp.NumField()
	panicIdx := ret.PanicFieldIdx(outTyp)
	if panicIdx >= 0 {
		fields--
		setters = slices.Insert(setters, panicIdx, reflect.Value{})
	}
	if returns != fields {
		panic(fmt.Errorf("hel: PopulateReturns got %d addresses but needs %d", returns, fields))
	}

	deadline := time.NewTimer(timeout)
	defer deadline.Stop()

	reportTimeout := func(msg string, args ...any) {
		defer func() {
			if r := recover(); r != nil {
				panic(fmt.Errorf("hel: panic calling t.Fatalf from mock method [%v]: %v", mock, r))
			}
		}()
		t.Fatalf(msg, args...)
	}

	select {
	case <-mock.Block():
		defer func() {
			mock.Block() <- struct{}{}
		}()
		if t.Failed() {
			return
		}
	case <-deadline.C:
		if t.Failed() {
			return
		}
		reportTimeout("hel: mock method [%v] timed out after %v waiting to be unblocked", mock, timeout)
	}
	select {
	case ret, ok := <-mock.Out():
		if t.Failed() {
			return
		}
		if !ok {
			panic(fmt.Errorf("hel: PopulateReturns called on closed mock [%v]", mock))
		}
		recv := reflect.ValueOf(ret)
		for i := 0; i < recv.NumField(); i++ {
			v := recv.Field(i)
			if i == panicIdx {
				val := v.Interface()
				if val != nil {
					defer panic(val)
				}
				continue
			}
			setters[i].Set(v)
		}
	case <-deadline.C:
		if t.Failed() {
			return
		}
		reportTimeout("hel: mock method [%v] timed out after %v waiting for return", mock, timeout)
	}
}
