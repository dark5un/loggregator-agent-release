package v1_test

import (
	egress "code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress/v1"
	"github.com/cloudfoundry/sonde-go/events"
	"google.golang.org/protobuf/proto"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Tagger", func() {
	var (
		mockWriter *MockEnvelopeWriter
		ctrl       *gomock.Controller
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockWriter = NewMockEnvelopeWriter(ctrl)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("tags events with the given deployment name, job, index and IP address", func() {
		t := egress.NewTagger(
			"test-deployment",
			"test-job",
			"2",
			"123.123.123.123",
			mockWriter,
		)
		envelope := &events.Envelope{
			EventType: events.Envelope_ValueMetric.Enum(),
			ValueMetric: &events.ValueMetric{
				Name:  proto.String("metricName"),
				Value: proto.Float64(2.0),
				Unit:  proto.String("seconds"),
			},
		}

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

		mockWriter.EXPECT().Write(expected)
		t.Write(envelope)
	})

	Context("doesn't overwrite", func() {
		var (
			t        *egress.Tagger
			envelope *events.Envelope
		)

		BeforeEach(func() {
			t = egress.NewTagger(
				"test-deployment",
				"test-job",
				"2",
				"123.123.123.123",
				mockWriter,
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

			mockWriter.EXPECT().Write(gomock.Any()).DoAndReturn(func(e *events.Envelope) {
				Expect(*e.Deployment).To(Equal("another-deployment"))
			})

			t.Write(envelope)
		})

		It("when job is already set", func() {
			envelope.Job = proto.String("another-job")

			mockWriter.EXPECT().Write(gomock.Any()).DoAndReturn(func(e *events.Envelope) {
				Expect(*e.Job).To(Equal("another-job"))
			})

			t.Write(envelope)
		})

		It("when index is already set", func() {
			envelope.Index = proto.String("3")

			mockWriter.EXPECT().Write(gomock.Any()).DoAndReturn(func(e *events.Envelope) {
				Expect(*e.Index).To(Equal("3"))
			})

			t.Write(envelope)
		})

		It("when ip is already set", func() {
			envelope.Ip = proto.String("1.1.1.1")

			mockWriter.EXPECT().Write(gomock.Any()).DoAndReturn(func(e *events.Envelope) {
				Expect(*e.Ip).To(Equal("1.1.1.1"))
			})

			t.Write(envelope)
		})
	})
})
