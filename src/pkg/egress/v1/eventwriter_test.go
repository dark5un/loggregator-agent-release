package v1_test

import (
	"time"

	egress "code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress/v1"
	"github.com/cloudfoundry/sonde-go/events"
	"google.golang.org/protobuf/proto"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("EventWriter", func() {
	var (
		mockWriter  *mockEnvelopeWriter
		eventWriter *egress.EventWriter
	)

	BeforeEach(func() {
		t := GinkgoT()
		mockWriter = newMockEnvelopeWriter(t, time.Minute)
		eventWriter = egress.New("Africa")
	})

	Describe("Emit", func() {
		It("writes emitted events", func() {
			eventWriter.SetWriter(mockWriter)

			event := &events.ValueMetric{
				Name:  proto.String("ValueName"),
				Value: proto.Float64(13),
				Unit:  proto.String("giraffes"),
			}
			err := eventWriter.Emit(event)
			Expect(err).To(BeNil())

			var input mockEnvelopeWriter_Write_In
			Eventually(mockWriter.method.Write.Method.In()).Should(Receive(&input))
			Expect(input.Event.GetOrigin()).To(Equal("Africa"))
			Expect(input.Event.GetEventType()).To(Equal(events.Envelope_ValueMetric))
			Expect(input.Event.GetValueMetric()).To(Equal(event))
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
			eventWriter.SetWriter(mockWriter)

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

			var input mockEnvelopeWriter_Write_In
			Eventually(mockWriter.method.Write.Method.In()).Should(Receive(&input))
			Expect(input.Event).To(Equal(event))
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
