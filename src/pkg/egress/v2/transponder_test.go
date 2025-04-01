package v2_test

import (
	"errors"
	"sync"
	"time"

	"code.cloudfoundry.org/go-loggregator/v10/rpc/loggregator_v2"
	metricsHelpers "code.cloudfoundry.org/go-metric-registry/testhelpers"
	egress "code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// SimpleNexter is a simple implementation of the Nexter interface for testing
type SimpleNexter struct {
	mu        sync.Mutex
	envelopes chan *loggregator_v2.Envelope
	ok        chan bool
}

func NewSimpleNexter() *SimpleNexter {
	return &SimpleNexter{
		envelopes: make(chan *loggregator_v2.Envelope, 100),
		ok:        make(chan bool, 100),
	}
}

func (n *SimpleNexter) TryNext() (*loggregator_v2.Envelope, bool) {
	n.mu.Lock()
	defer n.mu.Unlock()

	select {
	case env := <-n.envelopes:
		return env, <-n.ok
	default:
		return nil, false
	}
}

func (n *SimpleNexter) AddEnvelope(env *loggregator_v2.Envelope, nextOk bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.envelopes <- env
	n.ok <- nextOk
}

// SimpleBatchWriter is a simple implementation of the BatchWriter interface for testing
type SimpleBatchWriter struct {
	mu      sync.Mutex
	batches chan []*loggregator_v2.Envelope
	errors  chan error
}

func NewSimpleBatchWriter() *SimpleBatchWriter {
	return &SimpleBatchWriter{
		batches: make(chan []*loggregator_v2.Envelope, 100),
		errors:  make(chan error, 100),
	}
}

func (w *SimpleBatchWriter) Write(msgs []*loggregator_v2.Envelope) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.batches <- msgs

	select {
	case err := <-w.errors:
		return err
	default:
		return nil
	}
}

func (w *SimpleBatchWriter) AddError(err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.errors <- err
}

var _ = Describe("Transponder", func() {
	It("reads from the buffer to the writer", func() {
		envelope := &loggregator_v2.Envelope{SourceId: "uuid"}
		nexter := NewSimpleNexter()
		nexter.AddEnvelope(envelope, true)
		writer := NewSimpleBatchWriter()

		spy := metricsHelpers.NewMetricsRegistry()

		tx := egress.NewTransponder(nexter, writer, 1, time.Nanosecond, spy)
		go tx.Start()

		var batch []*loggregator_v2.Envelope
		Eventually(writer.batches).Should(Receive(&batch))
		Expect(batch).To(Equal([]*loggregator_v2.Envelope{envelope}))
	})

	Describe("batching", func() {
		It("emits once the batch count has been reached", func() {
			envelope := &loggregator_v2.Envelope{SourceId: "uuid"}
			nexter := NewSimpleNexter()
			writer := NewSimpleBatchWriter()

			for i := 0; i < 6; i++ {
				nexter.AddEnvelope(envelope, true)
			}

			spy := metricsHelpers.NewMetricsRegistry()

			tx := egress.NewTransponder(nexter, writer, 5, time.Minute, spy)
			go tx.Start()

			var batch []*loggregator_v2.Envelope
			Eventually(writer.batches).Should(Receive(&batch))
			Expect(batch).To(HaveLen(5))
		})

		It("emits once the batch interval has been reached", func() {
			envelope := &loggregator_v2.Envelope{SourceId: "uuid"}
			nexter := NewSimpleNexter()
			writer := NewSimpleBatchWriter()

			nexter.AddEnvelope(envelope, true)

			spy := metricsHelpers.NewMetricsRegistry()

			tx := egress.NewTransponder(nexter, writer, 5, time.Millisecond, spy)
			go tx.Start()

			var batch []*loggregator_v2.Envelope
			Eventually(writer.batches).Should(Receive(&batch))
			Expect(batch).To(HaveLen(1))
		})

		It("clears batch upon egress failure", func() {
			envelope := &loggregator_v2.Envelope{SourceId: "uuid"}
			nexter := NewSimpleNexter()
			writer := NewSimpleBatchWriter()
			writer.AddError(errors.New("some-error"))

			for i := 0; i < 6; i++ {
				nexter.AddEnvelope(envelope, true)
			}

			spy := metricsHelpers.NewMetricsRegistry()

			tx := egress.NewTransponder(nexter, writer, 5, time.Minute, spy)
			go tx.Start()

			Eventually(writer.batches).Should(HaveLen(1))
			Consistently(writer.batches).Should(HaveLen(1))
		})

		It("emits egress and dropped metric", func() {
			envelope := &loggregator_v2.Envelope{SourceId: "uuid"}
			nexter := NewSimpleNexter()
			writer := NewSimpleBatchWriter()

			for i := 0; i < 6; i++ {
				nexter.AddEnvelope(envelope, true)
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
