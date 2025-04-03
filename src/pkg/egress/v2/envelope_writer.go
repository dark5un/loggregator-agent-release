//go:generate mockgen -package v2_test -destination mock_writer_test.go -source envelope_writer.go Writer
//go:generate mockgen -package v2_test -destination mock_envelope_processor_test.go -source envelope_writer.go EnvelopeProcessor

package v2

import "code.cloudfoundry.org/go-loggregator/v10/rpc/loggregator_v2"

type Writer interface {
	Write(*loggregator_v2.Envelope) error
}

type EnvelopeProcessor interface {
	Process(*loggregator_v2.Envelope) error
}

type EnvelopeWriter struct {
	writer    Writer
	processor EnvelopeProcessor
}

func NewEnvelopeWriter(w Writer, ps EnvelopeProcessor) EnvelopeWriter {
	return EnvelopeWriter{
		writer:    w,
		processor: ps,
	}
}

func (ew EnvelopeWriter) Write(env *loggregator_v2.Envelope) error {
	err := ew.processor.Process(env)
	if err != nil {
		return err
	}

	return ew.writer.Write(env)
}
