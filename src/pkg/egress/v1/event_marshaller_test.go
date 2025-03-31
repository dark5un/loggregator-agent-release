package v1_test

import (
	"time"

	metricsHelpers "code.cloudfoundry.org/go-metric-registry/testhelpers"
	egress "code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress/v1"
	"github.com/cloudfoundry/sonde-go/events"
	"google.golang.org/protobuf/proto"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("EventMarshaller", func() {
	var (
		marshaller      *egress.EventMarshaller
		mockChainWriter *mockBatchChainByteWriter
		metricClient    *metricsHelpers.SpyMetricsRegistry
	)

	BeforeEach(func() {
		t := GinkgoT()
		mockChainWriter = newMockBatchChainByteWriter(t, time.Minute)
		metricClient = metricsHelpers.NewMetricsRegistry()
	})

	JustBeforeEach(func() {
		marshaller = egress.NewMarshaller(metricClient)
		marshaller.SetWriter(mockChainWriter)
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
				close(mockChainWriter.method.Write.Method.Out())
			})

			It("doesn't write the bytes", func() {
				marshaller.Write(envelope)
				Consistently(mockChainWriter.method.Write.Method.In()).ShouldNot(Receive())
			})
		})

		Context("with writer", func() {
			BeforeEach(func() {
				close(mockChainWriter.method.Write.Method.Out())
				envelope = &events.Envelope{
					Origin:    proto.String("The Negative Zone"),
					EventType: events.Envelope_LogMessage.Enum(),
				}
			})

			It("writes messages to the writer", func() {
				marshaller.Write(envelope)
				expected, err := proto.Marshal(envelope)
				Expect(err).ToNot(HaveOccurred())
				Expect(mockChainWriter.method.Write.Method.In()).Should(Receive(Equal(expected)))

				metric := metricClient.GetMetric("egress", map[string]string{"metric_version": "1.0"})
				Expect(metric.Value()).To(Equal(float64(1)))
			})
		})
	})

	Describe("SetWriter", func() {
		It("writes to the new writer", func() {
			t := GinkgoT()
			newWriter := newMockBatchChainByteWriter(t, time.Minute)
			close(newWriter.method.Write.Method.Out())
			marshaller.SetWriter(newWriter)

			envelope := &events.Envelope{
				Origin:    proto.String("The Negative Zone"),
				EventType: events.Envelope_LogMessage.Enum(),
			}
			marshaller.Write(envelope)

			expected, err := proto.Marshal(envelope)
			Expect(err).ToNot(HaveOccurred())
			Consistently(mockChainWriter.method.Write.Method.In()).ShouldNot(Receive())
			Eventually(newWriter.method.Write.Method.In()).Should(Receive(Equal(expected)))
		})
	})
})
