package v1_test

import (
	egress "code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress/v1"
	"github.com/cloudfoundry/sonde-go/events"
	"google.golang.org/protobuf/proto"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// SimpleEnvelopeWriter is defined in message_aggregator_test.go
// No need to redeclare it here

var _ = Describe("Tagger", func() {
	It("tags events with the given deployment name, job, index and IP address", func() {
		writer := NewSimpleEnvelopeWriter()
		tagger := egress.NewTagger(
			"test-deployment",
			"test-job",
			"2",
			"123.123.123.123",
			writer,
		)
		envelope := &events.Envelope{
			EventType: events.Envelope_ValueMetric.Enum(),
			ValueMetric: &events.ValueMetric{
				Name:  proto.String("metricName"),
				Value: proto.Float64(2.0),
				Unit:  proto.String("seconds"),
			},
		}

		tagger.Write(envelope)

		var receivedEvent *events.Envelope
		Eventually(writer.envelopes).Should(Receive(&receivedEvent))
		expected := &events.Envelope{
			EventType: events.Envelope_ValueMetric.Enum(),
			ValueMetric: &events.ValueMetric{
				Name:  proto.String("metricName"),
				Value: proto.Float64(2.0),
				Unit:  proto.String("seconds"),
			},
			Deployment: proto.String("test-deployment"),
			Job:        proto.String("test-job"),
			Index:      proto.String("2"),
			Ip:         proto.String("123.123.123.123"),
		}
		Eventually(receivedEvent).Should(Equal(expected))
	})

	Context("doesn't overwrite", func() {
		var (
			writer   *SimpleEnvelopeWriter
			tagger   *egress.Tagger
			envelope *events.Envelope
		)

		BeforeEach(func() {
			writer = NewSimpleEnvelopeWriter()

			tagger = egress.NewTagger(
				"test-deployment",
				"test-job",
				"2",
				"123.123.123.123",
				writer,
			)

			envelope = &events.Envelope{
				EventType: events.Envelope_ValueMetric.Enum(),
				ValueMetric: &events.ValueMetric{
					Name:  proto.String("metricName"),
					Value: proto.Float64(2.0),
					Unit:  proto.String("seconds"),
				},
			}
		})

		It("when deployment is already set", func() {
			envelope.Deployment = proto.String("another-deployment")
			tagger.Write(envelope)

			var writtenEnvelope *events.Envelope
			Eventually(writer.envelopes).Should(Receive(&writtenEnvelope))
			Eventually(*writtenEnvelope.Deployment).Should(Equal("another-deployment"))
		})

		It("when job is already set", func() {
			envelope.Job = proto.String("another-job")
			tagger.Write(envelope)

			var writtenEnvelope *events.Envelope
			Eventually(writer.envelopes).Should(Receive(&writtenEnvelope))
			Eventually(*writtenEnvelope.Job).Should(Equal("another-job"))
		})

		It("when index is already set", func() {
			envelope.Index = proto.String("3")
			tagger.Write(envelope)

			var writtenEnvelope *events.Envelope
			Eventually(writer.envelopes).Should(Receive(&writtenEnvelope))
			Eventually(*writtenEnvelope.Index).Should(Equal("3"))
		})

		It("when ip is already set", func() {
			envelope.Ip = proto.String("1.1.1.1")
			tagger.Write(envelope)

			var writtenEnvelope *events.Envelope
			Eventually(writer.envelopes).Should(Receive(&writtenEnvelope))
			Eventually(*writtenEnvelope.Ip).Should(Equal("1.1.1.1"))
		})
	})
})
