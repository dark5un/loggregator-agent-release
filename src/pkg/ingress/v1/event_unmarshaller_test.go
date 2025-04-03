package v1_test

import (
	v1 "code.cloudfoundry.org/loggregator-agent-release/src/pkg/ingress/v1"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/protobuf/proto"
)

var _ = Describe("EventUnmarshaller", func() {
	var (
		mockCtrl     *gomock.Controller
		mockWriter   *MockEnvelopeWriter
		unmarshaller *v1.EventUnmarshaller
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockWriter = NewMockEnvelopeWriter(mockCtrl)
		unmarshaller = v1.New(mockWriter)
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Context("Write", func() {
		It("unmarshalls byte arrays and writes to an EnvelopeWriter", func() {
			envelope := &events.Envelope{
				Origin:    proto.String("fake-origin"),
				EventType: events.Envelope_LogMessage.Enum(),
				LogMessage: &events.LogMessage{
					Message:     []byte("foo"),
					MessageType: events.LogMessage_OUT.Enum(),
					Timestamp:   proto.Int64(1234),
				},
			}
			message, err := proto.Marshal(envelope)
			Expect(err).ToNot(HaveOccurred())

			mockWriter.EXPECT().Write(gomock.Any())

			unmarshaller.Write(message)
		})

		It("handles bad input gracefully", func() {
			unmarshaller.Write(make([]byte, 4))
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
