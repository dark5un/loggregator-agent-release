# Message Aggregator Tests

## Overview

The MessageAggregator component in `pkg/egress/v1` is responsible for aggregating counter events before forwarding them to the next component in the pipeline. The tests in `pkg/egress/v1/message_aggregator_test.go` verify its correct behavior for counter accumulation, value message passing, and thread safety.

## Current Test Coverage

### Basic Functionality

1. **Value Message Passing**: Tests that non-counter messages pass through unchanged.
   ```go
   It("passes value messages through", func() {
       inputMessage := createValueMessage()
       messageAggregator.Write(inputMessage)

       Expect(mockWriter.GetWriteCalledCount()).To(Equal(1))
       Expect(mockWriter.GetEnvelopeAt(0)).To(Equal(inputMessage))
   })
   ```

2. **Thread Safety**: Tests that concurrent writes don't cause data races.
   ```go
   It("handles concurrent writes without data race", func() {
       inputMessage := createValueMessage()
       done := make(chan struct{})
       go func() {
           defer close(done)
           for i := 0; i < 40; i++ {
               messageAggregator.Write(inputMessage)
           }
       }()
       for i := 0; i < 40; i++ {
           messageAggregator.Write(inputMessage)
       }
       <-done
   })
   ```

### Counter Processing

1. **Setting Total Field**: Tests that the Total field is set on CounterEvents.
   ```go
   It("sets the Total field on a CounterEvent ", func() {
       messageAggregator.Write(createCounterMessage("total", "fake-origin-4", nil))

       Expect(mockWriter.GetWriteCalledCount()).To(Equal(1))
       outputMessage := mockWriter.GetEnvelopeAt(0)
       Expect(outputMessage.GetEventType()).To(Equal(events.Envelope_CounterEvent))
       expectCorrectCounterNameDeltaAndTotal(outputMessage, "total", 4, 4)
   })
   ```

2. **Accumulating Deltas**: Tests that Deltas are accumulated for CounterEvents with the same name, origin, and tags.
   ```go
   It("accumulates Deltas for CounterEvents with the same name, origin, and tags", func() {
       // [Test implementation]
   })
   ```

3. **Overwriting Total**: Tests that the aggregated total is overwritten when a Total is explicitly set.
   ```go
   It("overwrites aggregated total when total is set", func() {
       // [Test implementation]
   })
   ```

4. **Different Counters**: Tests that differently-named counters are accumulated separately.
   ```go
   It("accumulates differently-named counters separately", func() {
       // [Test implementation]
   })
   ```

5. **Different Tags**: Tests that differently-tagged counters are accumulated separately.
   ```go
   It("accumulates differently-tagged counters separately", func() {
       // [Test implementation]
   })
   ```

6. **Non-Counter Events**: Tests that non-counter events don't affect counter accumulation.
   ```go
   It("does not accumulate for counters when receiving a non-counter event", func() {
       // [Test implementation]
   })
   ```

7. **Different Origins**: Tests that counters with different origins are accumulated independently.
   ```go
   It("accumulates independently for different origins", func() {
       // [Test implementation]
   })
   ```

## Suggested Additional Test Cases

The following test cases would improve coverage of the MessageAggregator component:

### 1. Test TTL Behavior

The MessageAggregator has a TTL (Time To Live) mechanism that should expire counters that haven't been updated in a while. This isn't tested in the current suite.

```go
It("expires counters after TTL", func() {
    // Set a short TTL for testing
    originalTTL := egress.MaxTTL
    egress.MaxTTL = 10 * time.Millisecond
    defer func() { egress.MaxTTL = originalTTL }()
    
    // First counter event
    messageAggregator.Write(createCounterMessage("expired-counter", "fake-origin", nil))
    
    // Wait for TTL to expire
    time.Sleep(20 * time.Millisecond)
    
    // Write same counter again - should reset to initial value rather than accumulate
    messageAggregator.Write(createCounterMessage("expired-counter", "fake-origin", nil))
    
    Expect(mockWriter.GetWriteCalledCount()).To(Equal(2))
    expectCorrectCounterNameDeltaAndTotal(mockWriter.GetEnvelopeAt(0), "expired-counter", 4, 4)
    expectCorrectCounterNameDeltaAndTotal(mockWriter.GetEnvelopeAt(1), "expired-counter", 4, 4) // Reset to 4, not 8
})
```

### 2. Test Large Counter Values

Test that the MessageAggregator correctly handles large counter values to ensure there are no overflow issues:

```go
It("handles large counter values correctly", func() {
    // Create a counter message with a large delta
    largeValue := uint64(18446744073709551000) // Close to max uint64
    
    largeCounter := &events.Envelope{
        Origin:    proto.String("fake-origin"),
        EventType: events.Envelope_CounterEvent.Enum(),
        CounterEvent: &events.CounterEvent{
            Name:  proto.String("large-counter"),
            Delta: proto.Uint64(largeValue),
        },
    }
    
    // Write twice to test accumulation
    messageAggregator.Write(largeCounter)
    messageAggregator.Write(largeCounter)
    
    Expect(mockWriter.GetWriteCalledCount()).To(Equal(2))
    
    // First write should have delta = large value, total = large value
    firstOutput := mockWriter.GetEnvelopeAt(0)
    Expect(firstOutput.GetCounterEvent().GetDelta()).To(Equal(largeValue))
    Expect(firstOutput.GetCounterEvent().GetTotal()).To(Equal(largeValue))
    
    // Second write should have delta = large value, total = 2*large value (or wrap around if overflow)
    secondOutput := mockWriter.GetEnvelopeAt(1)
    Expect(secondOutput.GetCounterEvent().GetDelta()).To(Equal(largeValue))
    
    // Check for overflow - this test might need adjustment based on expected behavior
    expectedTotal := largeValue * 2
    if expectedTotal < largeValue { // Overflow check
        // If overflow is expected behavior, adjust this expectation
    }
    Expect(secondOutput.GetCounterEvent().GetTotal()).To(Equal(expectedTotal))
})
```

### 3. Test Many Different Counter Types

Test that the MessageAggregator can handle a large number of different counter types simultaneously:

```go
It("handles many different counter types simultaneously", func() {
    // Create a large number of different counters
    numCounters := 1000
    for i := 0; i < numCounters; i++ {
        counter := createCounterMessage(
            fmt.Sprintf("counter-%d", i),
            fmt.Sprintf("origin-%d", i % 10),
            map[string]string{
                "tag": fmt.Sprintf("value-%d", i % 20),
            },
        )
        messageAggregator.Write(counter)
    }
    
    // Write all counters again to test accumulation
    for i := 0; i < numCounters; i++ {
        counter := createCounterMessage(
            fmt.Sprintf("counter-%d", i),
            fmt.Sprintf("origin-%d", i % 10),
            map[string]string{
                "tag": fmt.Sprintf("value-%d", i % 20),
            },
        )
        messageAggregator.Write(counter)
    }
    
    Expect(mockWriter.GetWriteCalledCount()).To(Equal(numCounters * 2))
    
    // Sample a few counters to verify accumulation
    for i := 0; i < 10; i++ {
        idx := i * 100
        firstWriteIdx := idx
        secondWriteIdx := idx + numCounters
        
        firstOutput := mockWriter.GetEnvelopeAt(firstWriteIdx)
        secondOutput := mockWriter.GetEnvelopeAt(secondWriteIdx)
        
        counterName := fmt.Sprintf("counter-%d", idx)
        Expect(firstOutput.GetCounterEvent().GetName()).To(Equal(counterName))
        Expect(secondOutput.GetCounterEvent().GetName()).To(Equal(counterName))
        
        Expect(firstOutput.GetCounterEvent().GetDelta()).To(Equal(uint64(4)))
        Expect(firstOutput.GetCounterEvent().GetTotal()).To(Equal(uint64(4)))
        
        Expect(secondOutput.GetCounterEvent().GetDelta()).To(Equal(uint64(4)))
        Expect(secondOutput.GetCounterEvent().GetTotal()).To(Equal(uint64(8)))
    }
})
```

### 4. Test Memory Usage

Test that the MessageAggregator doesn't leak memory when processing a large number of counters:

```go
It("doesn't leak memory when processing many counters", func() {
    // This test might require modifications to the MessageAggregator to expose internal state
    
    // Create a large number of unique counters to fill the cache
    numUniqueCounters := 10000
    for i := 0; i < numUniqueCounters; i++ {
        counter := createCounterMessage(
            fmt.Sprintf("unique-counter-%d", i),
            "origin",
            nil,
        )
        messageAggregator.Write(counter)
    }
    
    // Get memory stats before and after garbage collection
    // This would require exposing the internal counter map or using runtime.GC()
    
    // Write a new set of counters
    for i := numUniqueCounters; i < numUniqueCounters*2; i++ {
        counter := createCounterMessage(
            fmt.Sprintf("unique-counter-%d", i),
            "origin",
            nil,
        )
        messageAggregator.Write(counter)
    }
    
    // Verify that older counters were eventually pruned
    // This would require modifications to expose internal state
})
```

### 5. Test Nil or Invalid Input

Test that the MessageAggregator handles nil or invalid input gracefully:

```go
It("handles nil or invalid input gracefully", func() {
    // Test with nil message
    Expect(func() { messageAggregator.Write(nil) }).NotTo(Panic())
    
    // Test with nil CounterEvent
    nilCounter := &events.Envelope{
        Origin:    proto.String("fake-origin"),
        EventType: events.Envelope_CounterEvent.Enum(),
        CounterEvent: nil,
    }
    Expect(func() { messageAggregator.Write(nilCounter) }).NotTo(Panic())
    
    // Test with nil CounterEvent fields
    emptyCounter := &events.Envelope{
        Origin:    proto.String("fake-origin"),
        EventType: events.Envelope_CounterEvent.Enum(),
        CounterEvent: &events.CounterEvent{
            Name:  nil,
            Delta: nil,
        },
    }
    Expect(func() { messageAggregator.Write(emptyCounter) }).NotTo(Panic())
})
```

## Conclusion

The current MessageAggregator tests provide good coverage of the basic functionality and counter accumulation behavior. The suggested additional test cases would improve coverage in areas such as TTL behavior, large counter values, high cardinality, memory usage, and error handling.

Implementing these additional tests would help ensure the MessageAggregator component is robust under a wider variety of conditions and edge cases. 