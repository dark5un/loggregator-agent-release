package v1_test

import (
	egress "code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress/v1"
	"github.com/cloudfoundry/sonde-go/events"
	"google.golang.org/protobuf/proto"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("EventWriter", func() {
	var (
		mockWriter  *MockEnvelopeWriter
		eventWriter *egress.EventWriter
		ctrl        *gomock.Controller
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockWriter = NewMockEnvelopeWriter(ctrl)
		eventWriter = egress.New("Africa")
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("Emit", func() {
		It("writes emitted events", func() {
			eventWriter.SetWriter(mockWriter)

			event := &events.ValueMetric{
				Name:  proto.String("ValueName"),
				Value: proto.Float64(13),
				Unit:  proto.String("giraffes"),
			}

			mockWriter.EXPECT().Write(gomock.Any()).DoAndReturn(func(e *events.Envelope) {
				Expect(e.GetOrigin()).To(Equal("Africa"))
				Expect(e.GetEventType()).To(Equal(events.Envelope_ValueMetric))
				Expect(e.GetValueMetric()).To(Equal(event))
			})

			err := eventWriter.Emit(event)
			Expect(err).To(BeNil())
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

			mockWriter.EXPECT().Write(event)

			err := eventWriter.EmitEnvelope(event)
			Expect(err).To(BeNil())
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
