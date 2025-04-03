package v1_test

import (
	"time"

	egress "code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress/v1"
	"github.com/cloudfoundry/sonde-go/events"
	"google.golang.org/protobuf/proto"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MessageAggregator", func() {
	var (
		mockWriter        *MockEnvelopeWriter
		messageAggregator *egress.MessageAggregator
		originalTTL       time.Duration
		ctrl              *gomock.Controller
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockWriter = NewMockEnvelopeWriter(ctrl)
		messageAggregator = egress.NewAggregator(
			mockWriter,
		)
		originalTTL = egress.MaxTTL
	})

	AfterEach(func() {
		ctrl.Finish()
		egress.MaxTTL = originalTTL
	})

	It("passes value messages through", func() {
		inputMessage := createValueMessage()
		mockWriter.EXPECT().Write(inputMessage)
		messageAggregator.Write(inputMessage)
	})

	It("handles concurrent writes without data race", func() {
		inputMessage := createValueMessage()
		done := make(chan struct{})

		// Set up expectations before starting the goroutine
		for i := 0; i < 40; i++ {
			mockWriter.EXPECT().Write(inputMessage)
		}

		go func() {
			defer GinkgoRecover()
			defer close(done)
			for i := 0; i < 40; i++ {
				messageAggregator.Write(inputMessage)
			}
		}()

		<-done
	})

	Describe("counter processing", func() {
		It("sets the Total field on a CounterEvent ", func() {
			// Set up expectation before writing
			mockWriter.EXPECT().Write(gomock.Any()).DoAndReturn(func(e *events.Envelope) {
				Expect(e.GetEventType()).To(Equal(events.Envelope_CounterEvent))
				expectCorrectCounterNameDeltaAndTotal(e, "total", 4, 4)
			})

			// Write the event
			messageAggregator.Write(createCounterMessage("total", "fake-origin-4", nil))
		})

		It("accumulates Deltas for CounterEvents with the same name, origin, and tags", func() {
			// Set up expectations for three counter events
			mockWriter.EXPECT().Write(gomock.Any()).DoAndReturn(func(e *events.Envelope) {
				expectCorrectCounterNameDeltaAndTotal(e, "total", 4, 4)
			})
			mockWriter.EXPECT().Write(gomock.Any()).DoAndReturn(func(e *events.Envelope) {
				expectCorrectCounterNameDeltaAndTotal(e, "total", 4, 8)
			})
			mockWriter.EXPECT().Write(gomock.Any()).DoAndReturn(func(e *events.Envelope) {
				expectCorrectCounterNameDeltaAndTotal(e, "total", 4, 12)
			})

			// Write the events
			messageAggregator.Write(createCounterMessage(
				"total",
				"fake-origin-4",
				map[string]string{
					"protocol": "tcp",
				},
			))
			messageAggregator.Write(createCounterMessage(
				"total",
				"fake-origin-4",
				map[string]string{
					"protocol": "tcp",
				},
			))
			messageAggregator.Write(createCounterMessage(
				"total",
				"fake-origin-4",
				map[string]string{
					"protocol": "tcp",
				},
			))
		})

		It("overwrites aggregated total when total is set", func() {
			// Set up expectations for three counter events
			mockWriter.EXPECT().Write(gomock.Any()).DoAndReturn(func(e *events.Envelope) {
				expectCorrectCounterNameDeltaAndTotal(e, "total", 4, 4)
			})
			mockWriter.EXPECT().Write(gomock.Any()).DoAndReturn(func(e *events.Envelope) {
				expectCorrectCounterNameDeltaAndTotal(e, "total", 0, 101)
			})
			mockWriter.EXPECT().Write(gomock.Any()).DoAndReturn(func(e *events.Envelope) {
				expectCorrectCounterNameDeltaAndTotal(e, "total", 4, 105)
			})

			// Write the events
			messageAggregator.Write(createCounterMessage(
				"total",
				"fake-origin-4",
				map[string]string{
					"protocol": "tcp",
				},
			))
			messageAggregator.Write(createCounterMessageWithTotal(
				"total",
				"fake-origin-4",
				map[string]string{
					"protocol": "tcp",
				},
			))
			messageAggregator.Write(createCounterMessage(
				"total",
				"fake-origin-4",
				map[string]string{
					"protocol": "tcp",
				},
			))
		})

		It("accumulates differently-named counters separately", func() {
			// Set up expectations for two counter events
			mockWriter.EXPECT().Write(gomock.Any()).DoAndReturn(func(e *events.Envelope) {
				expectCorrectCounterNameDeltaAndTotal(e, "total1", 4, 4)
			})
			mockWriter.EXPECT().Write(gomock.Any()).DoAndReturn(func(e *events.Envelope) {
				expectCorrectCounterNameDeltaAndTotal(e, "total2", 4, 4)
			})

			// Write the events
			messageAggregator.Write(createCounterMessage("total1", "fake-origin-4", nil))
			messageAggregator.Write(createCounterMessage("total2", "fake-origin-4", nil))
		})

		It("accumulates differently-tagged counters separately", func() {
			// Set up expectations for four counter events
			mockWriter.EXPECT().Write(gomock.Any()).DoAndReturn(func(e *events.Envelope) {
				expectCorrectCounterNameDeltaAndTotal(e, "total", 4, 4)
			})
			mockWriter.EXPECT().Write(gomock.Any()).DoAndReturn(func(e *events.Envelope) {
				expectCorrectCounterNameDeltaAndTotal(e, "total", 4, 4)
			})
			mockWriter.EXPECT().Write(gomock.Any()).DoAndReturn(func(e *events.Envelope) {
				expectCorrectCounterNameDeltaAndTotal(e, "total", 4, 8)
			})
			mockWriter.EXPECT().Write(gomock.Any()).DoAndReturn(func(e *events.Envelope) {
				expectCorrectCounterNameDeltaAndTotal(e, "total", 4, 4)
			})

			// Write the events
			By("writing protocol tagged counters")
			messageAggregator.Write(createCounterMessage(
				"total",
				"fake-origin-4",
				map[string]string{
					"protocol": "grpc",
				},
			))
			messageAggregator.Write(createCounterMessage(
				"total",
				"fake-origin-4",
				map[string]string{
					"protocol": "tcp",
				},
			))
			messageAggregator.Write(createCounterMessage(
				"total",
				"fake-origin-4",
				map[string]string{
					"protocol": "grpc",
				},
			))

			By("writing counters tagged with key/value strings split differently")
			messageAggregator.Write(createCounterMessage(
				"total",
				"fake-origin-4",
				map[string]string{
					"proto": "other",
				},
			))
		})

		It("does not accumulate for counters when receiving a non-counter event", func() {
			// Set up expectations for two events
			mockWriter.EXPECT().Write(gomock.Any()).DoAndReturn(func(e *events.Envelope) {
				Expect(e.GetEventType()).To(Equal(events.Envelope_ValueMetric))
			})
			mockWriter.EXPECT().Write(gomock.Any()).DoAndReturn(func(e *events.Envelope) {
				Expect(e.GetEventType()).To(Equal(events.Envelope_CounterEvent))
				expectCorrectCounterNameDeltaAndTotal(e, "counter1", 4, 4)
			})

			// Write the events
			messageAggregator.Write(createValueMessage())
			messageAggregator.Write(createCounterMessage("counter1", "fake-origin-4", nil))
		})

		It("accumulates independently for different origins", func() {
			// Set up expectations for three counter events
			mockWriter.EXPECT().Write(gomock.Any()).DoAndReturn(func(e *events.Envelope) {
				Expect(e.GetOrigin()).To(Equal("fake-origin-4"))
				expectCorrectCounterNameDeltaAndTotal(e, "counter1", 4, 4)
			})
			mockWriter.EXPECT().Write(gomock.Any()).DoAndReturn(func(e *events.Envelope) {
				Expect(e.GetOrigin()).To(Equal("fake-origin-5"))
				expectCorrectCounterNameDeltaAndTotal(e, "counter1", 4, 4)
			})
			mockWriter.EXPECT().Write(gomock.Any()).DoAndReturn(func(e *events.Envelope) {
				Expect(e.GetOrigin()).To(Equal("fake-origin-4"))
				expectCorrectCounterNameDeltaAndTotal(e, "counter1", 4, 8)
			})

			// Write the events
			messageAggregator.Write(createCounterMessage("counter1", "fake-origin-4", nil))
			messageAggregator.Write(createCounterMessage("counter1", "fake-origin-5", nil))
			messageAggregator.Write(createCounterMessage("counter1", "fake-origin-4", nil))
		})
	})
})

func createValueMessage() *events.Envelope {
	return &events.Envelope{
		Origin:    proto.String("fake-origin-2"),
		EventType: events.Envelope_ValueMetric.Enum(),
		ValueMetric: &events.ValueMetric{
			Name:  proto.String("fake-metric-name"),
			Value: proto.Float64(42),
			Unit:  proto.String("fake-unit"),
		},
	}
}

func createCounterMessage(name, origin string, tags map[string]string) *events.Envelope {
	return &events.Envelope{
		Origin:    proto.String(origin),
		EventType: events.Envelope_CounterEvent.Enum(),
		CounterEvent: &events.CounterEvent{
			Name:  proto.String(name),
			Delta: proto.Uint64(4),
		},
		Tags: tags,
	}
}

func createCounterMessageWithTotal(name, origin string, tags map[string]string) *events.Envelope {
	return &events.Envelope{
		Origin:    proto.String(origin),
		EventType: events.Envelope_CounterEvent.Enum(),
		CounterEvent: &events.CounterEvent{
			Name:  proto.String(name),
			Total: proto.Uint64(101),
		},
		Tags: tags,
	}
}

func expectCorrectCounterNameDeltaAndTotal(outputMessage *events.Envelope, name string, delta uint64, total uint64) {
	Expect(outputMessage.GetCounterEvent().GetName()).To(Equal(name))
	Expect(outputMessage.GetCounterEvent().GetDelta()).To(Equal(delta))
	Expect(outputMessage.GetCounterEvent().GetTotal()).To(Equal(total))
}
