package v2_test

import (
	"errors"
	"sync"
	"time"

	"code.cloudfoundry.org/go-loggregator/v10/rpc/loggregator_v2"
	metricsHelpers "code.cloudfoundry.org/go-metric-registry/testhelpers"
	egress "code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress/v2"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Transponder", func() {
	var (
		ctrl       *gomock.Controller
		mockNexter *MockNexter
		mockWriter *MockBatchWriter
		spy        *metricsHelpers.SpyMetricsRegistry
		tx         *egress.Transponder
		wg         sync.WaitGroup
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockNexter = NewMockNexter(ctrl)
		mockWriter = NewMockBatchWriter(ctrl)
		spy = metricsHelpers.NewMetricsRegistry()
		wg = sync.WaitGroup{}
	})

	AfterEach(func() {
		if tx != nil {
			tx.Stop()
			// Wait for the Transponder to stop
			Eventually(func() bool {
				select {
				case <-tx.Done:
					return true
				default:
					return false
				}
			}).Should(BeTrue())
			wg.Wait()
		}
	})

	It("reads from the buffer to the writer", func() {
		envelope := &loggregator_v2.Envelope{SourceId: "uuid"}
		mockNexter.EXPECT().TryNext().Return(envelope, true).Times(1)
		mockWriter.EXPECT().Write([]*loggregator_v2.Envelope{envelope}).Return(nil)
		// After the first call, return false to stop the loop
		mockNexter.EXPECT().TryNext().Return(nil, false).AnyTimes()

		tx = egress.NewTransponder(mockNexter, mockWriter, 1, time.Nanosecond, spy)
		wg.Add(1)
		go func() {
			defer GinkgoRecover()
			defer wg.Done()
			tx.Start()
		}()

		Eventually(func() bool {
			return hasMetric(spy, "egress", map[string]string{"metric_version": "2.0"})()
		}).Should(BeTrue())

		// Give the Transponder time to process all envelopes
		time.Sleep(100 * time.Millisecond)
	})

	Describe("batching", func() {
		It("emits once the batch count has been reached", func() {
			envelope := &loggregator_v2.Envelope{SourceId: "uuid"}
			for i := 0; i < 6; i++ {
				mockNexter.EXPECT().TryNext().Return(envelope, true)
			}
			mockWriter.EXPECT().Write(gomock.Any()).Return(nil)
			// After the batch is complete, return false to stop the loop
			mockNexter.EXPECT().TryNext().Return(nil, false).AnyTimes()

			tx = egress.NewTransponder(mockNexter, mockWriter, 5, time.Minute, spy)
			wg.Add(1)
			go func() {
				defer GinkgoRecover()
				defer wg.Done()
				tx.Start()
			}()

			Eventually(func() bool {
				return hasMetric(spy, "egress", map[string]string{"metric_version": "2.0"})()
			}).Should(BeTrue())

			// Give the Transponder time to process all envelopes
			time.Sleep(100 * time.Millisecond)
		})

		It("emits once the batch duration has been reached", func() {
			envelope := &loggregator_v2.Envelope{SourceId: "uuid"}
			mockNexter.EXPECT().TryNext().Return(envelope, true).Times(1)
			mockWriter.EXPECT().Write([]*loggregator_v2.Envelope{envelope}).Return(nil)
			// After the first call, return false to stop the loop
			mockNexter.EXPECT().TryNext().Return(nil, false).AnyTimes()

			tx = egress.NewTransponder(mockNexter, mockWriter, 100, time.Nanosecond, spy)
			wg.Add(1)
			go func() {
				defer GinkgoRecover()
				defer wg.Done()
				tx.Start()
			}()

			Eventually(func() bool {
				return hasMetric(spy, "egress", map[string]string{"metric_version": "2.0"})()
			}).Should(BeTrue())

			// Give the Transponder time to process all envelopes
			time.Sleep(100 * time.Millisecond)
		})
	})

	Describe("error handling", func() {
		It("continues processing when the writer returns an error", func() {
			envelope := &loggregator_v2.Envelope{SourceId: "uuid"}
			mockNexter.EXPECT().TryNext().Return(envelope, true).Times(1)
			mockWriter.EXPECT().Write([]*loggregator_v2.Envelope{envelope}).Return(errors.New("write error"))
			// After the first call, return false to stop the loop
			mockNexter.EXPECT().TryNext().Return(nil, false).AnyTimes()

			tx = egress.NewTransponder(mockNexter, mockWriter, 1, time.Nanosecond, spy)
			wg.Add(1)
			go func() {
				defer GinkgoRecover()
				defer wg.Done()
				tx.Start()
			}()

			Eventually(func() bool {
				return hasMetric(spy, "dropped", map[string]string{"direction": "egress", "metric_version": "2.0"})()
			}).Should(BeTrue())

			// Give the Transponder time to process all envelopes
			time.Sleep(100 * time.Millisecond)
		})
	})
})

func hasMetric(mc *metricsHelpers.SpyMetricsRegistry, metricName string, tags map[string]string) func() bool {
	return func() bool {
		return mc.HasMetric(metricName, tags)
	}
}
