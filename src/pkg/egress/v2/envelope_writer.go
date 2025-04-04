package v2

import "code.cloudfoundry.org/go-loggregator/v10/rpc/loggregator_v2"

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o v2fakes/fake_writer.go . Writer
type Writer interface {
	Write(*loggregator_v2.Envelope) error
}

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o v2fakes/fake_envelope_processor.go . EnvelopeProcessor
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
