package v1_test

import (
	egress "code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress/v1"
	"github.com/cloudfoundry/sonde-go/events"
	"google.golang.org/protobuf/proto"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("EventWriter", func() {
	var (
		writer      *SimpleEnvelopeWriter
		eventWriter *egress.EventWriter
	)

	BeforeEach(func() {
		writer = NewSimpleEnvelopeWriter()
		eventWriter = egress.New("Africa")
	})

	Describe("Emit", func() {
		It("writes emitted events", func() {
			eventWriter.SetWriter(writer)

			event := &events.ValueMetric{
				Name:  proto.String("ValueName"),
				Value: proto.Float64(13),
				Unit:  proto.String("giraffes"),
			}
			err := eventWriter.Emit(event)
			Expect(err).To(BeNil())

			var receivedEnvelope *events.Envelope
			Eventually(writer.envelopes).Should(Receive(&receivedEnvelope))
			Expect(receivedEnvelope.GetOrigin()).To(Equal("Africa"))
			Expect(receivedEnvelope.GetEventType()).To(Equal(events.Envelope_ValueMetric))
			Expect(receivedEnvelope.GetValueMetric()).To(Equal(event))
		})

		It("returns an error with a sane message when emitting without a writer", func() {
			event := &events.ValueMetric{
				Name:  proto.String("ValueName"),
				Value: proto.Float64(13),
				Unit:  proto.String("giraffes"),
			}
			err := eventWriter.Emit(event)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("EventWriter: No envelope writer set (see SetWriter)"))
		})
	})

	Describe("EmitEnvelope", func() {
		It("writes emitted events", func() {
			eventWriter.SetWriter(writer)

			event := &events.Envelope{
				Origin:    proto.String("foo"),
				EventType: events.Envelope_ValueMetric.Enum(),
				ValueMetric: &events.ValueMetric{
					Name:  proto.String("ValueName"),
					Value: proto.Float64(13),
					Unit:  proto.String("giraffes"),
				},
			}
			err := eventWriter.EmitEnvelope(event)
			Expect(err).To(BeNil())

			var receivedEnvelope *events.Envelope
			Eventually(writer.envelopes).Should(Receive(&receivedEnvelope))
			Expect(receivedEnvelope).To(Equal(event))
		})

		It("returns an error with a sane message when emitting without a writer", func() {
			event := &events.Envelope{
				Origin:    proto.String("foo"),
				EventType: events.Envelope_ValueMetric.Enum(),
				ValueMetric: &events.ValueMetric{
					Name:  proto.String("ValueName"),
					Value: proto.Float64(13),
					Unit:  proto.String("giraffes"),
				},
			}
			err := eventWriter.EmitEnvelope(event)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("EventWriter: No envelope writer set (see SetWriter)"))
		})
	})
})
