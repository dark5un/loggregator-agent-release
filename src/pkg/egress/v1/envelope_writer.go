package v1

import "github.com/cloudfoundry/sonde-go/events"

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o v1fakes/fake_envelope_writer.go . EnvelopeWriter
type EnvelopeWriter interface {
	Write(event *events.Envelope)
}
