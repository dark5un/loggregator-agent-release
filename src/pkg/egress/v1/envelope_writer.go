//go:generate mockgen -package v1_test -destination mock_envelope_writer_test.go -source envelope_writer.go EnvelopeWriter

package v1

import "github.com/cloudfoundry/sonde-go/events"

type EnvelopeWriter interface {
	Write(event *events.Envelope)
}
