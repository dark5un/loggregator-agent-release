# Transponder Tests

## Overview

The Transponder is responsible for reading envelopes from a diode, batching them, and sending them to a writer. The tests in `pkg/egress/v2/transponder_test.go` verify its functionality, focusing on batching behavior, error handling, and metric emission.

## Current Test Coverage

### Core Functionality

1. **Basic Operation**: Tests that the Transponder reads from a Nexter and writes to a BatchWriter.
   ```go
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
   ```

2. **No Envelopes**: Tests that the writer is not called when there are no envelopes.
   ```go
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
   ```

### Batching Behavior

1. **Batch Size**: Tests that the Transponder emits once it reaches the batch size.
   ```go
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
   ```

2. **Batch Interval**: Tests that the Transponder emits once it reaches the batch interval.
   ```go
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
   ```

### Error Handling

1. **Writer Errors**: Tests that the Transponder continues processing when the writer returns an error.
   ```go
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
   ```

2. **Batch Clearing**: Tests that the batch is cleared upon a write error.
   ```go
   It("clears batch upon egress failure", func() {
       // [Test implementation]
   })
   ```

### Metrics

1. **Metric Emission**: Tests that the Transponder emits egress and dropped metrics.
   ```go
   It("emits egress and dropped metric", func() {
       // [Test implementation]
   })
   ```

## Suggested Additional Test Cases

The following test cases would improve coverage of the Transponder component:

### 1. Test Concurrent Envelope Processing

Test that the Transponder handles multiple envelopes being processed concurrently:

```go
It("handles concurrent envelope processing correctly", func() {
    nexter := mocks.NewNexter(mockT)
    writer := mocks.NewBatchWriter(mockT)
    
    // Set up a large number of envelope returns
    for i := 0; i < 100; i++ {
        envelope := &loggregator_v2.Envelope{SourceId: fmt.Sprintf("uuid-%d", i)}
        nexter.On("TryNext").Return(envelope, true).Once()
    }
    nexter.On("TryNext").Return(nil, false)
    
    // Accept any batch of envelopes
    writer.On("Write", mock.Anything).Return(nil)
    
    // Use a small batch size and interval
    tx := v2.NewTransponder(nexter, writer, 10, 50*time.Millisecond, metricsHelpers.NewMetricsRegistry())
    
    // Start multiple concurrent transponders (simulating high load)
    var wg sync.WaitGroup
    for i := 0; i < 5; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            tx.Start()
        }()
    }
    
    // Wait for all envelopes to be processed
    Eventually(func() int {
        return writer.GetCalls()
    }).Should(BeNumerically(">=", 10))
})
```

### 2. Test Partial Batch Flushing

Test that partial batches are flushed when the batch interval is reached:

```go
It("flushes partial batches when interval is reached", func() {
    nexter := mocks.NewNexter(mockT)
    writer := mocks.NewBatchWriter(mockT)
    envelope := &loggregator_v2.Envelope{SourceId: "uuid"}
    
    // Return 3 envelopes (less than batch size)
    nexter.On("TryNext").Return(envelope, true).Times(3)
    nexter.On("TryNext").Return(nil, false).After(100 * time.Millisecond)
    
    // Should still write the 3 envelopes as a partial batch
    writer.On("Write", mock.MatchedBy(func(batch []*loggregator_v2.Envelope) bool {
        return len(batch) == 3
    })).Return(nil)
    
    tx := v2.NewTransponder(nexter, writer, 5, 50*time.Millisecond, metricsHelpers.NewMetricsRegistry())
    go tx.Start()
    
    Eventually(func() int {
        return writer.GetCalls()
    }, 500*time.Millisecond, 10*time.Millisecond).Should(Equal(1))
})
```

### 3. Test Envelope Ordering

Test that envelopes are written in the order they are received:

```go
It("maintains envelope order in batches", func() {
    nexter := mocks.NewNexter(mockT)
    writer := mocks.NewBatchWriter(mockT)
    
    // Create envelopes with sequential IDs
    envelopes := []*loggregator_v2.Envelope{
        {SourceId: "uuid-1"},
        {SourceId: "uuid-2"},
        {SourceId: "uuid-3"},
    }
    
    for _, env := range envelopes {
        nexter.On("TryNext").Return(env, true).Once()
    }
    nexter.On("TryNext").Return(nil, false)
    
    // Capture the actual batch for inspection
    var capturedBatch []*loggregator_v2.Envelope
    writer.On("Write", mock.Anything).Run(func(args mock.Arguments) {
        capturedBatch = args.Get(0).([]*loggregator_v2.Envelope)
    }).Return(nil)
    
    tx := v2.NewTransponder(nexter, writer, 3, time.Minute, metricsHelpers.NewMetricsRegistry())
    go tx.Start()
    
    Eventually(func() int {
        return writer.GetCalls()
    }).Should(Equal(1))
    
    // Verify order is maintained
    Expect(capturedBatch).To(HaveLen(3))
    Expect(capturedBatch[0].SourceId).To(Equal("uuid-1"))
    Expect(capturedBatch[1].SourceId).To(Equal("uuid-2"))
    Expect(capturedBatch[2].SourceId).To(Equal("uuid-3"))
})
```

### 4. Test Resource Cleanup

Test that resources are properly released when the Transponder stops:

```go
It("cleans up resources when stopped", func() {
    nexter := mocks.NewNexter(mockT)
    writer := mocks.NewBatchWriter(mockT)
    
    stopCh := make(chan struct{})
    
    // Mock a stoppable Transponder with access to internal state
    tx := v2.NewTransponder(nexter, writer, 10, time.Minute, metricsHelpers.NewMetricsRegistry())
    
    go func() {
        // Run for a short time then signal stop
        time.Sleep(100 * time.Millisecond)
        close(stopCh)
    }()
    
    // Start should exit cleanly when signaled
    tx.Start()
    
    // Additional assertions checking resource cleanup could be added here
    // This would require modifying the Transponder to be stoppable and
    // exposing internal state for test inspection
})
```

### 5. Test Metrics Accuracy

Test that metrics accurately reflect the number of processed and dropped envelopes:

```go
It("accurately counts processed and dropped envelopes", func() {
    nexter := mocks.NewNexter(mockT)
    writer := mocks.NewBatchWriter(mockT)
    envelope := &loggregator_v2.Envelope{SourceId: "uuid"}
    
    // Return 10 envelopes
    nexter.On("TryNext").Return(envelope, true).Times(10)
    nexter.On("TryNext").Return(nil, false)
    
    // First batch succeeds
    writer.On("Write", mock.MatchedBy(func(batch []*loggregator_v2.Envelope) bool {
        return len(batch) == 5
    })).Return(nil).Once()
    
    // Second batch fails
    writer.On("Write", mock.MatchedBy(func(batch []*loggregator_v2.Envelope) bool {
        return len(batch) == 5
    })).Return(errors.New("write error")).Once()
    
    spy := metricsHelpers.NewMetricsRegistry()
    tx := v2.NewTransponder(nexter, writer, 5, time.Minute, spy)
    go tx.Start()
    
    Eventually(func() int {
        return writer.GetCalls()
    }).Should(Equal(2))
    
    // Verify metrics
    Eventually(func() float64 {
        return spy.GetMetricValue("egress", map[string]string{"metric_version": "2.0"})
    }).Should(Equal(5.0)) // 5 successful writes
    
    Eventually(func() float64 {
        return spy.GetMetricValue("dropped", map[string]string{"direction": "egress", "metric_version": "2.0"})
    }).Should(Equal(5.0)) // 5 dropped writes
})
```

## Conclusion

The current Transponder tests provide good coverage of the basic functionality, batching behavior, error handling, and metric emission. The suggested additional test cases would improve coverage in areas such as concurrency, partial batch handling, ordering, resource cleanup, and metric accuracy.

Implementing these additional tests would help ensure the Transponder component is robust under a wider variety of conditions. 