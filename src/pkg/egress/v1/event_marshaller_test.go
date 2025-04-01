package v1_test

import (
	"sync"

	metricsHelpers "code.cloudfoundry.org/go-metric-registry/testhelpers"
	egress "code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress/v1"
	"github.com/cloudfoundry/sonde-go/events"
	"google.golang.org/protobuf/proto"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// SimpleBatchChainByteWriter is a simple implementation of the BatchChainByteWriter interface for testing
type SimpleBatchChainByteWriter struct {
	messages chan []byte
	mu       sync.Mutex
}

func NewSimpleBatchChainByteWriter() *SimpleBatchChainByteWriter {
	return &SimpleBatchChainByteWriter{
		messages: make(chan []byte, 100),
	}
}

func (w *SimpleBatchChainByteWriter) Write(message []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.messages <- message
	return nil
}

var _ = Describe("EventMarshaller", func() {
	var (
		marshaller   *egress.EventMarshaller
		writer       *SimpleBatchChainByteWriter
		metricClient *metricsHelpers.SpyMetricsRegistry
	)

	BeforeEach(func() {
		writer = NewSimpleBatchChainByteWriter()
		metricClient = metricsHelpers.NewMetricsRegistry()
	})

	JustBeforeEach(func() {
		marshaller = egress.NewMarshaller(metricClient)
		marshaller.SetWriter(writer)
	})

	Describe("Write", func() {
		var envelope *events.Envelope

		Context("with a nil writer", func() {
			BeforeEach(func() {
				envelope = &events.Envelope{
					Origin:    proto.String("The Negative Zone"),
					EventType: events.Envelope_LogMessage.Enum(),
				}
			})

			JustBeforeEach(func() {
				marshaller.SetWriter(nil)
			})

			It("does not panic", func() {
				Expect(func() {
					marshaller.Write(envelope)
				}).ToNot(Panic())
			})
		})

		Context("with an invalid envelope", func() {
			BeforeEach(func() {
				envelope = &events.Envelope{}
			})

			It("doesn't write the bytes", func() {
				marshaller.Write(envelope)
				Consistently(writer.messages).ShouldNot(Receive())
			})
		})

		Context("with writer", func() {
			BeforeEach(func() {
				envelope = &events.Envelope{
					Origin:    proto.String("The Negative Zone"),
					EventType: events.Envelope_LogMessage.Enum(),
				}
			})

			It("writes messages to the writer", func() {
				marshaller.Write(envelope)
				expected, err := proto.Marshal(envelope)
				Expect(err).ToNot(HaveOccurred())
				var received []byte
				Eventually(writer.messages).Should(Receive(&received))
				Expect(received).To(Equal(expected))

				metric := metricClient.GetMetric("egress", map[string]string{"metric_version": "1.0"})
				Expect(metric.Value()).To(Equal(float64(1)))
			})
		})
	})

	Describe("SetWriter", func() {
		It("writes to the new writer", func() {
			newWriter := NewSimpleBatchChainByteWriter()
			marshaller.SetWriter(newWriter)

			envelope := &events.Envelope{
				Origin:    proto.String("The Negative Zone"),
				EventType: events.Envelope_LogMessage.Enum(),
			}
			marshaller.Write(envelope)

			expected, err := proto.Marshal(envelope)
			Expect(err).ToNot(HaveOccurred())
			Consistently(writer.messages).ShouldNot(Receive())
			var received []byte
			Eventually(newWriter.messages).Should(Receive(&received))
			Expect(received).To(Equal(expected))
		})
	})
})
