package v2_test

import (
	"errors"
	"sync"

	"code.cloudfoundry.org/go-loggregator/v10/rpc/loggregator_v2"
	v2 "code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// SimpleSingleWriter is a simple implementation of the Writer interface for testing
type SimpleSingleWriter struct {
	mu       sync.Mutex
	messages chan *loggregator_v2.Envelope
	errors   chan error
}

func NewSimpleSingleWriter() *SimpleSingleWriter {
	return &SimpleSingleWriter{
		messages: make(chan *loggregator_v2.Envelope, 100),
		errors:   make(chan error, 100),
	}
}

func (w *SimpleSingleWriter) Write(msg *loggregator_v2.Envelope) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.messages <- msg

	select {
	case err := <-w.errors:
		return err
	default:
		return nil
	}
}

func (w *SimpleSingleWriter) AddError(err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.errors <- err
}

// Simple implementation of the EnvelopeProcessor interface for testing
type SimpleProcessor struct {
	processErr error
}

func (p *SimpleProcessor) Process(*loggregator_v2.Envelope) error {
	return p.processErr
}

var _ = Describe("EnvelopeWriter", func() {
	It("processes envelopes before writing", func() {
		writer := NewSimpleSingleWriter()
		tagger := v2.NewTagger(nil)
		ew := v2.NewEnvelopeWriter(writer, v2.NewCounterAggregator(tagger.TagEnvelope))
		Expect(ew.Write(buildCounterEnvelope(10, "name-1", "origin-1"))).To(Succeed())

		var receivedEnvelope *loggregator_v2.Envelope
		Eventually(writer.messages).Should(Receive(&receivedEnvelope))
		Expect(receivedEnvelope.GetCounter().GetDelta()).To(Equal(uint64(10)))
	})

	It("returns an error if the processor fails", func() {
		writer := NewSimpleSingleWriter()
		processor := &SimpleProcessor{processErr: errors.New("expected error")}
		ew := v2.NewEnvelopeWriter(writer, processor)
		Expect(ew.Write(buildCounterEnvelope(10, "name-1", "origin-1"))).ToNot(Succeed())
	})
})
