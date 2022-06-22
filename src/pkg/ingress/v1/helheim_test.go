// This file was generated by git.sr.ht/~nelsam/hel.  Do not
// edit this code by hand unless you *really* know what you're
// doing.  Expect any changes made manually to be overwritten
// the next time hel regenerates this file.

package v1_test

import (
	"github.com/cloudfoundry/sonde-go/events"
)

type mockEnvelopeWriter struct {
	WriteCalled chan bool
	WriteInput  struct {
		Event chan *events.Envelope
	}
}

func newMockEnvelopeWriter() *mockEnvelopeWriter {
	m := &mockEnvelopeWriter{}
	m.WriteCalled = make(chan bool, 100)
	m.WriteInput.Event = make(chan *events.Envelope, 100)
	return m
}
func (m *mockEnvelopeWriter) Write(event *events.Envelope) {
	m.WriteCalled <- true
	m.WriteInput.Event <- event
}
