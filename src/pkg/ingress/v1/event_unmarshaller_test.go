package v1_test

import (
	ingress "code.cloudfoundry.org/loggregator-agent-release/src/pkg/ingress/v1"
	"code.cloudfoundry.org/loggregator-agent-release/src/pkg/ingress/v1/mocks"
	"code.cloudfoundry.org/loggregator-agent-release/src/pkg/testhelpers"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/proto"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("EventUnmarshaller", func() {
	var (
		mockWriter   *mocks.EnvelopeWriter
		unmarshaller *ingress.EventUnmarshaller
		mockT        *testhelpers.MockTesting
	)

	BeforeEach(func() {
		mockT = testhelpers.NewMockTesting()
		mockWriter = mocks.NewEnvelopeWriter(mockT)
		mockWriter.On("Write", mock.AnythingOfType("*events.Envelope")).Return()
		unmarshaller = ingress.NewUnMarshaller(mockWriter)
	})

	Context("when the message contains a valid v1 envelope", func() {
		It("unmarshalls the bytes and handles them", func() {
			envelope := &events.Envelope{
				Origin:    proto.String("fake-origin-3"),
				EventType: events.Envelope_LogMessage.Enum(),
				LogMessage: &events.LogMessage{
					Message:     []byte("test message"),
					MessageType: events.LogMessage_OUT.Enum(),
					Timestamp:   proto.Int64(123456789),
				},
				Tags: map[string]string{
					"foo": "bar",
				},
			}
			message, _ := proto.Marshal(envelope)

			unmarshaller.Write(message)
			mockWriter.AssertNumberOfCalls(mockT, "Write", 1)
		})
	})

	Context("when unmarshalling fails", func() {
		It("does not write to the next writer", func() {
			message := []byte("this is not a valid v1 envelope")
			unmarshaller.Write(message)
			mockWriter.AssertNumberOfCalls(mockT, "Write", 0)
		})
	})

	Context("when the v1 envelope is missing required fields", func() {
		It("does not handle the event", func() {
			envelope := &events.Envelope{
				Origin:    proto.String("fake-origin-3"),
				EventType: events.Envelope_LogMessage.Enum(),
				// LogMessage: &events.LogMessage{},
			}
			bytes, err := proto.Marshal(envelope)
			Expect(err).ToNot(HaveOccurred())

			unmarshaller.Write(bytes)
			mockWriter.AssertNumberOfCalls(mockT, "Write", 0)
		})
	})

	Context("when the message has useful Tags", func() {
		It("unmarshalls the bytes and handles them", func() {
			envelope := &events.Envelope{
				Origin:    proto.String("fake-origin-3"),
				EventType: events.Envelope_LogMessage.Enum(),
				LogMessage: &events.LogMessage{
					Message:     []byte("test message"),
					MessageType: events.LogMessage_OUT.Enum(),
					Timestamp:   proto.Int64(123456789),
				},
				Tags: map[string]string{
					"source_id": "some-app-id",
				},
			}
			bytes, err := proto.Marshal(envelope)
			Expect(err).ToNot(HaveOccurred())

			unmarshaller.Write(bytes)
			mockWriter.AssertNumberOfCalls(mockT, "Write", 1)
		})
	})

	Context("when the message doesn't have Tags", func() {
		It("unmarshalls the bytes and handles them", func() {
			envelope := &events.Envelope{
				Origin:    proto.String("fake-origin-3"),
				EventType: events.Envelope_LogMessage.Enum(),
				LogMessage: &events.LogMessage{
					Message:     []byte("test message"),
					MessageType: events.LogMessage_OUT.Enum(),
					Timestamp:   proto.Int64(123456789),
				},
			}
			bytes, err := proto.Marshal(envelope)
			Expect(err).ToNot(HaveOccurred())

			unmarshaller.Write(bytes)
			mockWriter.AssertNumberOfCalls(mockT, "Write", 1)
		})
	})
})

func NewValueMetric(name string, value float64, unit string) *events.ValueMetric {
	return &events.ValueMetric{
		Name:  proto.String(name),
		Value: proto.Float64(value),
		Unit:  proto.String(unit),
	}
}
