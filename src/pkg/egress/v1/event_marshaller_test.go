package v1_test

import (
	metricsHelpers "code.cloudfoundry.org/go-metric-registry/testhelpers"
	egress "code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress/v1"
	"code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress/v1/mocks"
	"code.cloudfoundry.org/loggregator-agent-release/src/pkg/testhelpers"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/proto"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("EventMarshaller", func() {
	var (
		marshaller      *egress.EventMarshaller
		mockChainWriter *mocks.BatchChainByteWriter
		metricClient    *metricsHelpers.SpyMetricsRegistry
		mockT           *testhelpers.MockTesting
	)

	BeforeEach(func() {
		mockT = testhelpers.NewMockTesting()
		mockChainWriter = mocks.NewBatchChainByteWriter(mockT)
		mockChainWriter.On("Write", mock.AnythingOfType("[]uint8")).Return(nil)
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
			})

			It("doesn't write the bytes", func() {
				marshaller.Write(envelope)
				Expect(mockChainWriter.WriteCalledCount).To(Equal(0))
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

				Expect(mockChainWriter.WriteCalledCount).To(Equal(1))
				Expect(mockChainWriter.Messages[0]).To(Equal(expected))

				metric := metricClient.GetMetric("egress", map[string]string{"metric_version": "1.0"})
				Expect(metric.Value()).To(Equal(float64(1)))
			})
		})
	})

	Describe("SetWriter", func() {
		It("writes to the new writer", func() {
			newWriter := mocks.NewBatchChainByteWriter(mockT)
			newWriter.On("Write", mock.AnythingOfType("[]uint8")).Return(nil)
			marshaller.SetWriter(newWriter)

			envelope := &events.Envelope{
				Origin:    proto.String("The Negative Zone"),
				EventType: events.Envelope_LogMessage.Enum(),
			}
			marshaller.Write(envelope)

			expected, err := proto.Marshal(envelope)
			Expect(err).ToNot(HaveOccurred())

			Expect(mockChainWriter.WriteCalledCount).To(Equal(0))
			Expect(newWriter.WriteCalledCount).To(Equal(1))
			Expect(newWriter.Messages[0]).To(Equal(expected))
		})
	})
})
