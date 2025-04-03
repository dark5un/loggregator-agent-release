package v1_test

import (
	egress "code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress/v1"
	"code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress/v1/mocks"
	"code.cloudfoundry.org/loggregator-agent-release/src/pkg/testhelpers"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/proto"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("EventWriter", func() {
	var (
		mockWriter  *mocks.EnvelopeWriter
		mockT       *testhelpers.MockTesting
		eventWriter *egress.EventWriter
	)

	BeforeEach(func() {
		mockT = testhelpers.NewMockTesting()
		mockWriter = mocks.NewEnvelopeWriter(mockT)
		mockWriter.On("Write", mock.AnythingOfType("*events.Envelope")).Return()
		eventWriter = egress.New("origin")
		eventWriter.SetWriter(mockWriter)
	})

	It("emits events with the given origin", func() {
		valueMetric := &events.ValueMetric{
			Name:  proto.String("value-name"),
			Value: proto.Float64(1.23),
			Unit:  proto.String("units"),
		}
		err := eventWriter.Emit(valueMetric)
		Expect(err).NotTo(HaveOccurred())

		Expect(mockWriter.WriteCalledCount).To(Equal(1))
		envelope := mockWriter.Envelopes[0]
		Expect(envelope.GetOrigin()).To(Equal("origin"))
		Expect(envelope.GetEventType()).To(Equal(events.Envelope_ValueMetric))

		vm := envelope.GetValueMetric()
		Expect(vm.GetName()).To(Equal("value-name"))
		Expect(vm.GetValue()).To(Equal(1.23))
		Expect(vm.GetUnit()).To(Equal("units"))
	})

	It("emits counter events with the given origin", func() {
		counterEvent := &events.CounterEvent{
			Name:  proto.String("foo"),
			Delta: proto.Uint64(1),
		}
		err := eventWriter.Emit(counterEvent)
		Expect(err).NotTo(HaveOccurred())

		Expect(mockWriter.WriteCalledCount).To(Equal(1))
		envelope := mockWriter.Envelopes[0]
		Expect(envelope.GetOrigin()).To(Equal("origin"))
		Expect(envelope.GetEventType()).To(Equal(events.Envelope_CounterEvent))

		ce := envelope.GetCounterEvent()
		Expect(ce.GetName()).To(Equal("foo"))
		Expect(ce.GetDelta()).To(Equal(uint64(1)))
	})
})
