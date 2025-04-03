package v1_test

import (
	"errors"

	metricsHelpers "code.cloudfoundry.org/go-metric-registry/testhelpers"
	egress "code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress/v1"
	"github.com/cloudfoundry/sonde-go/events"
	"google.golang.org/protobuf/proto"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("EventMarshaller", func() {
	var (
		marshaller      *egress.EventMarshaller
		mockChainWriter *MockBatchChainByteWriter
		metricClient    *metricsHelpers.SpyMetricsRegistry
		ctrl            *gomock.Controller
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockChainWriter = NewMockBatchChainByteWriter(ctrl)
		metricClient = metricsHelpers.NewMetricsRegistry()
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	JustBeforeEach(func() {
		marshaller = egress.NewMarshaller(metricClient)
		marshaller.SetWriter(mockChainWriter)
	})

	Describe("Write", func() {
		var envelope *events.Envelope

		Context("with a nil writer", func() {
			BeforeEach(func() {
				envelope = &events.Envelope{
					Origin:    proto.String("The Negative Zone"),
					EventType: events.Envelope_LogMessage.Enum(),
				}
			})

			JustBeforeEach(func() {
				marshaller.SetWriter(nil)
			})

			It("does not panic", func() {
				Expect(func() {
					marshaller.Write(envelope)
				}).ToNot(Panic())
			})
		})

		Context("with a valid writer", func() {
			BeforeEach(func() {
				envelope = &events.Envelope{
					Origin:    proto.String("The Negative Zone"),
					EventType: events.Envelope_LogMessage.Enum(),
				}
			})

			It("writes the marshalled envelope to the writer", func() {
				mockChainWriter.EXPECT().Write(gomock.Any()).Return(nil)
				marshaller.Write(envelope)
			})

			It("increments the egress counter", func() {
				mockChainWriter.EXPECT().Write(gomock.Any()).Return(nil)
				marshaller.Write(envelope)
				Expect(metricClient.GetMetric("egress", map[string]string{"metric_version": "1.0"}).Value()).To(Equal(float64(1)))
			})

			Context("when the writer returns an error", func() {
				It("does not increment the counter", func() {
					mockChainWriter.EXPECT().Write(gomock.Any()).Return(errors.New("write error"))
					marshaller.Write(envelope)
					Expect(metricClient.GetMetric("egress", map[string]string{"metric_version": "1.0"}).Value()).To(Equal(float64(0)))
				})
			})
		})
	})

	Describe("SetWriter", func() {
		It("writes to the new writer", func() {
			newWriter := NewMockBatchChainByteWriter(ctrl)
			newWriter.EXPECT().Write(gomock.Any()).Return(nil)
			marshaller.SetWriter(newWriter)
			envelope := &events.Envelope{
				Origin:    proto.String("The Negative Zone"),
				EventType: events.Envelope_LogMessage.Enum(),
			}
			marshaller.Write(envelope)
		})

		It("uses the new writer for subsequent writes", func() {
			oldWriter := mockChainWriter
			newWriter := NewMockBatchChainByteWriter(ctrl)
			marshaller.SetWriter(newWriter)
			envelope := &events.Envelope{
				Origin:    proto.String("The Negative Zone"),
				EventType: events.Envelope_LogMessage.Enum(),
			}
			newWriter.EXPECT().Write(gomock.Any()).Return(nil)
			marshaller.Write(envelope)
			oldWriter.EXPECT().Write(gomock.Any()).Times(0)
		})
	})
})
