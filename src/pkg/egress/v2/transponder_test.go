package v2_test

import (
	"errors"
	"sync"
	"time"

	"code.cloudfoundry.org/go-loggregator/v10/rpc/loggregator_v2"
	metricsHelpers "code.cloudfoundry.org/go-metric-registry/testhelpers"
	v2 "code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress/v2"
	"code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress/v2/mocks"
	"code.cloudfoundry.org/loggregator-agent-release/src/pkg/testhelpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// MockNexter and MockBatchWriter implement custom mocks for parts of the test that
// need the hel-style channel-based approach
type MockNexter struct {
	TryNextCalled chan bool
	TryNextOutput struct {
		Ret0 chan *loggregator_v2.Envelope
		Ret1 chan bool
	}
}

func NewMockNexter() *MockNexter {
	m := &MockNexter{}
	m.TryNextCalled = make(chan bool, 100)
	m.TryNextOutput.Ret0 = make(chan *loggregator_v2.Envelope, 100)
	m.TryNextOutput.Ret1 = make(chan bool, 100)
	return m
}

func (m *MockNexter) TryNext() (*loggregator_v2.Envelope, bool) {
	m.TryNextCalled <- true
	return <-m.TryNextOutput.Ret0, <-m.TryNextOutput.Ret1
}

type MockBatchWriter struct {
	mu          sync.Mutex
	WriteCalled chan bool
	WriteInput  struct {
		Msgs chan []*loggregator_v2.Envelope
	}
	WriteOutput struct {
		Ret0 chan error
	}
	Calls int
}

func NewMockBatchWriter() *MockBatchWriter {
	m := &MockBatchWriter{}
	m.WriteCalled = make(chan bool, 100)
	m.WriteInput.Msgs = make(chan []*loggregator_v2.Envelope, 100)
	m.WriteOutput.Ret0 = make(chan error, 100)
	return m
}

func (m *MockBatchWriter) Write(msgs []*loggregator_v2.Envelope) error {
	m.WriteCalled <- true
	m.WriteInput.Msgs <- msgs

	m.mu.Lock()
	m.Calls++
	m.mu.Unlock()

	return <-m.WriteOutput.Ret0
}

func (m *MockBatchWriter) GetCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Calls
}

var _ = Describe("Transponder", func() {
	var (
		mockT *testhelpers.MockTesting
	)

	BeforeEach(func() {
		mockT = testhelpers.NewMockTesting()
	})

	It("reads from a diode and writes to a writer", func() {
		nexter := mocks.NewNexter(mockT)
		envelope := &loggregator_v2.Envelope{SourceId: "uuid"}
		writer := mocks.NewBatchWriter(mockT)

		nexter.On("TryNext").Return(envelope, true).Once()
		nexter.On("TryNext").Return(nil, false)
		writer.On("Write", []*loggregator_v2.Envelope{envelope}).Return(nil)

		tx := v2.NewTransponder(nexter, writer, 1, time.Minute, metricsHelpers.NewMetricsRegistry())
		go tx.Start()

		Eventually(func() int {
			return writer.GetCalls()
		}).Should(Equal(1))
	})

	It("doesn't call writer when there are no envelopes", func() {
		nexter := mocks.NewNexter(mockT)
		writer := mocks.NewBatchWriter(mockT)

		nexter.On("TryNext").Return(nil, false)

		tx := v2.NewTransponder(nexter, writer, 1, time.Minute, metricsHelpers.NewMetricsRegistry())
		go tx.Start()

		Consistently(func() int {
			return writer.GetCalls()
		}).Should(Equal(0))
	})

	It("emits once it reaches the batch size", func() {
		nexter := mocks.NewNexter(mockT)
		writer := mocks.NewBatchWriter(mockT)

		envelope := &loggregator_v2.Envelope{SourceId: "uuid"}
		nexter.On("TryNext").Return(envelope, true).Times(3)
		nexter.On("TryNext").Return(nil, false)
		writer.On("Write", []*loggregator_v2.Envelope{envelope, envelope, envelope}).Return(nil)

		tx := v2.NewTransponder(nexter, writer, 3, time.Minute, metricsHelpers.NewMetricsRegistry())
		go tx.Start()

		Eventually(func() int {
			return writer.GetCalls()
		}).Should(Equal(1))
	})

	It("emits once it reaches the batch interval", func() {
		nexter := mocks.NewNexter(mockT)
		writer := mocks.NewBatchWriter(mockT)
		envelope := &loggregator_v2.Envelope{SourceId: "uuid"}

		nexter.On("TryNext").Return(envelope, true).Once()
		nexter.On("TryNext").Return(nil, false)
		writer.On("Write", []*loggregator_v2.Envelope{envelope}).Return(nil)

		tx := v2.NewTransponder(nexter, writer, 10, 10*time.Millisecond, metricsHelpers.NewMetricsRegistry())
		go tx.Start()

		Eventually(func() int {
			return writer.GetCalls()
		}, 2*time.Second, 10*time.Millisecond).Should(Equal(1))
	})

	It("ignores writer errors and continues", func() {
		nexter := mocks.NewNexter(mockT)
		writer := mocks.NewBatchWriter(mockT)
		envelope := &loggregator_v2.Envelope{SourceId: "uuid"}

		nexter.On("TryNext").Return(envelope, true).Times(4)
		nexter.On("TryNext").Return(nil, false)

		writer.On("Write", []*loggregator_v2.Envelope{envelope, envelope}).Return(errors.New("some-error")).Once()
		writer.On("Write", []*loggregator_v2.Envelope{envelope, envelope}).Return(nil).Once()

		tx := v2.NewTransponder(nexter, writer, 2, time.Minute, metricsHelpers.NewMetricsRegistry())
		go tx.Start()

		Eventually(func() int {
			return writer.GetCalls()
		}).Should(Equal(2))
	})

	Describe("batching", func() {
		It("emits once the batch count has been reached", func() {
			envelope := &loggregator_v2.Envelope{SourceId: "uuid"}
			nexter := NewMockNexter()
			writer := NewMockBatchWriter()
			close(writer.WriteOutput.Ret0)

			for i := 0; i < 6; i++ {
				nexter.TryNextOutput.Ret0 <- envelope
				nexter.TryNextOutput.Ret1 <- true
			}

			spy := metricsHelpers.NewMetricsRegistry()

			tx := v2.NewTransponder(nexter, writer, 5, time.Minute, spy)
			go tx.Start()

			var batch []*loggregator_v2.Envelope
			Eventually(writer.WriteInput.Msgs).Should(Receive(&batch))
			Expect(batch).To(HaveLen(5))
		})

		It("emits once the batch interval has been reached", func() {
			envelope := &loggregator_v2.Envelope{SourceId: "uuid"}
			nexter := NewMockNexter()
			writer := NewMockBatchWriter()
			close(writer.WriteOutput.Ret0)

			nexter.TryNextOutput.Ret0 <- envelope
			nexter.TryNextOutput.Ret1 <- true
			close(nexter.TryNextOutput.Ret0)
			close(nexter.TryNextOutput.Ret1)

			spy := metricsHelpers.NewMetricsRegistry()

			tx := v2.NewTransponder(nexter, writer, 5, time.Millisecond, spy)
			go tx.Start()

			var batch []*loggregator_v2.Envelope
			Eventually(writer.WriteInput.Msgs).Should(Receive(&batch))
			Expect(batch).To(HaveLen(1))
		})

		It("clears batch upon egress failure", func() {
			envelope := &loggregator_v2.Envelope{SourceId: "uuid"}
			nexter := NewMockNexter()
			writer := NewMockBatchWriter()

			go func() {
				for {
					writer.WriteOutput.Ret0 <- errors.New("some-error")
				}
			}()

			for i := 0; i < 6; i++ {
				nexter.TryNextOutput.Ret0 <- envelope
				nexter.TryNextOutput.Ret1 <- true
			}

			spy := metricsHelpers.NewMetricsRegistry()

			tx := v2.NewTransponder(nexter, writer, 5, time.Minute, spy)
			go tx.Start()

			Eventually(writer.WriteCalled).Should(HaveLen(1))
			Consistently(writer.WriteCalled).Should(HaveLen(1))
		})

		It("emits egress and dropped metric", func() {
			envelope := &loggregator_v2.Envelope{SourceId: "uuid"}
			nexter := NewMockNexter()
			writer := NewMockBatchWriter()
			close(writer.WriteOutput.Ret0)

			for i := 0; i < 6; i++ {
				nexter.TryNextOutput.Ret0 <- envelope
				nexter.TryNextOutput.Ret1 <- true
			}

			spy := metricsHelpers.NewMetricsRegistry()
			tx := v2.NewTransponder(nexter, writer, 5, time.Minute, spy)
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
