package v1_test

import (
	"errors"

	metricsHelpers "code.cloudfoundry.org/go-metric-registry/testhelpers"
	v1 "code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress/v1"
	"code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress/v1/v1fakes"
	"github.com/cloudfoundry/sonde-go/events"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/protobuf/proto"
)

var _ = Describe("EventMarshaller", func() {
	var (
		marshaller      *v1.EventMarshaller
		metricClient    *metricsHelpers.SpyMetricsRegistry
		fakeBytesWriter *v1fakes.FakeBatchChainByteWriter
		envelope        *events.Envelope
	)

	BeforeEach(func() {
		fakeBytesWriter = new(v1fakes.FakeBatchChainByteWriter)
		metricClient = metricsHelpers.NewMetricsRegistry()
		marshaller = v1.NewMarshaller(metricClient)
		marshaller.SetWriter(fakeBytesWriter)

		envelope = &events.Envelope{
			Origin:    proto.String("origin"),
			EventType: events.Envelope_HttpStartStop.Enum(),
			ValueMetric: &events.ValueMetric{
				Name:  proto.String("name"),
				Value: proto.Float64(0),
				Unit:  proto.String("unit"),
			},
		}
	})

	Describe("Write", func() {
		Context("with writer", func() {
			var err error
			var message []byte

			BeforeEach(func() {
				message, err = proto.Marshal(envelope)
				Expect(err).ToNot(HaveOccurred())
			})

			It("writes messages to the writer", func() {
				marshaller.Write(envelope)

				Expect(fakeBytesWriter.WriteCallCount()).To(Equal(1))
				Expect(fakeBytesWriter.WriteArgsForCall(0)).To(Equal(message))

				metric := metricClient.GetMetric("egress", map[string]string{"metric_version": "1.0"})
				Expect(metric.Value()).To(Equal(float64(1)))
			})

			It("returns an error when the writer fails", func() {
				fakeBytesWriter.WriteReturns(errors.New("boom"))

				marshaller.Write(envelope)

				Expect(fakeBytesWriter.WriteCallCount()).To(Equal(1))
				Expect(fakeBytesWriter.WriteArgsForCall(0)).To(Equal(message))

				metric := metricClient.GetMetric("egress", map[string]string{"metric_version": "1.0"})
				Expect(metric.Value()).To(Equal(float64(0)))
			})
		})
	})

	Describe("SetWriter", func() {
		It("writes to the new writer", func() {
			newWriter := new(v1fakes.FakeBatchChainByteWriter)
			marshaller.SetWriter(newWriter)

			envelope := &events.Envelope{
				Origin:    proto.String("The Negative Zone"),
				EventType: events.Envelope_LogMessage.Enum(),
			}
			marshaller.Write(envelope)

			expected, err := proto.Marshal(envelope)
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeBytesWriter.WriteCallCount()).To(Equal(0))
			Expect(newWriter.WriteCallCount()).To(Equal(1))
			Expect(newWriter.WriteArgsForCall(0)).To(Equal(expected))
		})
	})
})
