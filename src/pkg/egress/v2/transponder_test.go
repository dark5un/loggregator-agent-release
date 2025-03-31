package v2_test

import (
	"errors"
	"time"

	"code.cloudfoundry.org/go-loggregator/v10/rpc/loggregator_v2"
	metricsHelpers "code.cloudfoundry.org/go-metric-registry/testhelpers"
	egress "code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Transponder", func() {
	It("reads from the buffer to the writer", func() {
		t := GinkgoT()
		envelope := &loggregator_v2.Envelope{SourceId: "uuid"}
		nexter := newMockNexter(t, time.Minute)
		nexter.method.TryNext.Method.Out() <- mockNexter_TryNext_Out{Ret0: envelope, Ret1: true}
		writer := newMockBatchWriter(t, time.Minute)
		close(writer.method.Write.Method.Out())

		spy := metricsHelpers.NewMetricsRegistry()

		tx := egress.NewTransponder(nexter, writer, 1, time.Nanosecond, spy)
		go tx.Start()

		Eventually(nexter.method.TryNext.Method.In()).Should(Receive())
		Eventually(writer.method.Write.Method.In()).Should(Receive(Equal([]*loggregator_v2.Envelope{envelope})))
	})

	Describe("batching", func() {
		It("emits once the batch count has been reached", func() {
			t := GinkgoT()
			envelope := &loggregator_v2.Envelope{SourceId: "uuid"}
			nexter := newMockNexter(t, time.Minute)
			writer := newMockBatchWriter(t, time.Minute)
			close(writer.method.Write.Method.Out())

			for i := 0; i < 6; i++ {
				nexter.method.TryNext.Method.Out() <- mockNexter_TryNext_Out{Ret0: envelope, Ret1: true}
			}

			spy := metricsHelpers.NewMetricsRegistry()

			tx := egress.NewTransponder(nexter, writer, 5, time.Minute, spy)
			go tx.Start()

			var batch []*loggregator_v2.Envelope
			Eventually(writer.method.Write.Method.In()).Should(Receive(&batch))
			Expect(batch).To(HaveLen(5))
		})

		It("emits once the batch interval has been reached", func() {
			t := GinkgoT()
			envelope := &loggregator_v2.Envelope{SourceId: "uuid"}
			nexter := newMockNexter(t, time.Minute)
			writer := newMockBatchWriter(t, time.Minute)
			close(writer.method.Write.Method.Out())

			nexter.method.TryNext.Method.Out() <- mockNexter_TryNext_Out{Ret0: envelope, Ret1: true}
			close(nexter.method.TryNext.Method.Out())

			spy := metricsHelpers.NewMetricsRegistry()

			tx := egress.NewTransponder(nexter, writer, 5, time.Millisecond, spy)
			go tx.Start()

			var batch []*loggregator_v2.Envelope
			Eventually(writer.method.Write.Method.In()).Should(Receive(&batch))
			Expect(batch).To(HaveLen(1))
		})

		It("clears batch upon egress failure", func() {
			t := GinkgoT()
			envelope := &loggregator_v2.Envelope{SourceId: "uuid"}
			nexter := newMockNexter(t, time.Minute)
			writer := newMockBatchWriter(t, time.Minute)

			go func() {
				for {
					writer.method.Write.Method.Out() <- mockBatchWriter_Write_Out{Ret0: errors.New("some-error")}
				}
			}()

			for i := 0; i < 6; i++ {
				nexter.method.TryNext.Method.Out() <- mockNexter_TryNext_Out{Ret0: envelope, Ret1: true}
			}

			spy := metricsHelpers.NewMetricsRegistry()

			tx := egress.NewTransponder(nexter, writer, 5, time.Minute, spy)
			go tx.Start()

			Eventually(writer.method.Write.Method.In()).Should(HaveLen(1))
			Consistently(writer.method.Write.Method.In()).Should(HaveLen(1))
		})

		It("emits egress and dropped metric", func() {
			t := GinkgoT()
			envelope := &loggregator_v2.Envelope{SourceId: "uuid"}
			nexter := newMockNexter(t, time.Minute)
			writer := newMockBatchWriter(t, time.Minute)
			close(writer.method.Write.Method.Out())

			for i := 0; i < 6; i++ {
				nexter.method.TryNext.Method.Out() <- mockNexter_TryNext_Out{Ret0: envelope, Ret1: true}
			}

			spy := metricsHelpers.NewMetricsRegistry()
			tx := egress.NewTransponder(nexter, writer, 5, time.Minute, spy)
			go tx.Start()

			Eventually(hasMetric(spy, "egress", map[string]string{"metric_version": "2.0"}))
			Eventually(hasMetric(spy, "dropped", map[string]string{"direction": "egress", "metric_version": "2.0"}))

		})
	})
})

func hasMetric(mc *metricsHelpers.SpyMetricsRegistry, metricName string, tags map[string]string) func() bool {
	return func() bool {
		return mc.HasMetric(metricName, tags)
	}
}
