# Testing Best Practices for Loggregator Agent

## Overview

This document outlines best practices for writing tests in the Loggregator Agent codebase. Following these guidelines will help ensure tests are effective, maintainable, and consistent across the codebase.

## General Best Practices

### 1. Test Structure

- **Use Descriptive Names**: Test names should clearly describe what is being tested and what the expected behavior is.
  ```go
  It("emits once it reaches the batch size", func() {
      // Test implementation
  })
  ```

- **Follow the AAA Pattern**: Arrange, Act, Assert.
  ```go
  It("should do something", func() {
      // Arrange - set up the test environment
      writer := mocks.NewBatchWriter(mockT)
      
      // Act - perform the action being tested
      tx := v2.NewTransponder(nexter, writer, 1, time.Minute, metricsHelpers.NewMetricsRegistry())
      go tx.Start()
      
      // Assert - verify the expected outcome
      Eventually(func() int {
          return writer.GetCalls()
      }).Should(Equal(1))
  })
  ```

- **One Assertion per Test**: When possible, focus each test on a single behavior or assertion.

### 2. Mocking

- **Use Mockery for Mocks**: Use mockery to generate mock implementations of interfaces.
  ```go
  mockT := testhelpers.NewMockTesting()
  writer := mocks.NewBatchWriter(mockT)
  writer.On("Write", mock.Anything).Return(nil)
  ```

- **Make Mocks Thread-Safe**: When using mocks in concurrent tests, ensure they are thread-safe.
  ```go
  type ThreadSafeMock struct {
      mu sync.Mutex
      // other fields
  }
  
  func (m *ThreadSafeMock) Method() {
      m.mu.Lock()
      defer m.mu.Unlock()
      // implementation
  }
  ```

- **Set Clear Expectations**: Be explicit about what methods should be called and what they should return.
  ```go
  writer.On("Write", []*loggregator_v2.Envelope{envelope}).Return(nil).Once()
  ```

### 3. Asynchronous Testing

- **Use Eventually for Async Operations**: When testing asynchronous behavior, use Ginkgo's `Eventually` helper.
  ```go
  Eventually(func() int {
      return writer.GetCalls()
  }).Should(Equal(1))
  ```

- **Use Consistently to Verify Stability**: Use `Consistently` to ensure a condition remains true for a period.
  ```go
  Consistently(func() int {
      return writer.GetCalls()
  }).Should(Equal(0))
  ```

- **Set Appropriate Timeouts**: Configure timeouts that are long enough to avoid flaky tests but short enough to catch issues quickly.
  ```go
  Eventually(func() int {
      return writer.GetCalls()
  }, 2*time.Second, 10*time.Millisecond).Should(Equal(1))
  ```

### 4. Test Coverage

- **Test Happy Path**: Ensure the component works as expected under normal conditions.
- **Test Error Handling**: Verify that the component handles errors gracefully.
- **Test Edge Cases**: Cover boundary conditions, empty inputs, large inputs, etc.
- **Test Concurrency**: For components that operate concurrently, test thread safety.

## Specific Patterns for Loggregator Agent

### 1. Testing with MockTesting

Use the `MockTesting` helper to bridge mockery with Ginkgo:

```go
mockT := testhelpers.NewMockTesting()
mock := mocks.NewSomeMock(mockT)
```

### 2. Thread-Safe Mocks

Add thread-safety to mocks that will be used in concurrent tests:

```go
// Add mutex protection and accessor methods
type ThreadSafeMock struct {
    mu sync.Mutex
    Calls int
}

// Thread-safe counter
func (m *ThreadSafeMock) IncrementCalls() {
    m.mu.Lock()
    m.Calls++
    m.mu.Unlock()
}

// Thread-safe getter
func (m *ThreadSafeMock) GetCalls() int {
    m.mu.Lock()
    defer m.mu.Unlock()
    return m.Calls
}
```

### 3. Testing Batching Behavior

When testing components that batch events:

```go
It("emits once it reaches the batch size", func() {
    // Configure batch size
    batchSize := 3
    
    // Set up expectations for multiple events
    for i := 0; i < batchSize; i++ {
        nexter.On("TryNext").Return(envelope, true).Once()
    }
    nexter.On("TryNext").Return(nil, false)
    
    // Expect a single write with the full batch
    writer.On("Write", []*loggregator_v2.Envelope{envelope, envelope, envelope}).Return(nil)
    
    // Create the component with the configured batch size
    tx := v2.NewTransponder(nexter, writer, batchSize, time.Minute, metricsHelpers.NewMetricsRegistry())
    go tx.Start()
    
    // Verify the batch was written
    Eventually(func() int {
        return writer.GetCalls()
    }).Should(Equal(1))
})
```

### 4. Testing Metrics

When testing components that emit metrics:

```go
It("emits metrics when processing messages", func() {
    // Use a metrics spy to capture metrics
    spy := metricsHelpers.NewMetricsRegistry()
    
    // Set up and run the component
    component := NewComponent(spy)
    component.Process()
    
    // Verify metrics were emitted
    Eventually(func() bool {
        return spy.HasMetric("metric_name", map[string]string{"label": "value"})
    }).Should(BeTrue())
    
    Eventually(func() float64 {
        return spy.GetMetricValue("metric_name", map[string]string{"label": "value"})
    }).Should(Equal(42.0))
})
```

### 5. Testing Error Handling

When testing error handling:

```go
It("handles errors gracefully", func() {
    // Configure the mock to return an error
    mock.On("SomeMethod").Return(nil, errors.New("expected error"))
    
    // Verify the component handles the error properly
    Expect(func() {
        component.DoSomething()
    }).NotTo(Panic())
    
    // Verify error was logged or handled in some expected way
    // ...
})
```

## Test Performance

- **Minimize Test Duration**: Tests should be as fast as possible while still being reliable.
- **Avoid Sleeps**: Use Ginkgo's `Eventually` and `Consistently` instead of `time.Sleep`.
- **Clean Up Resources**: Release resources in `AfterEach` or `AfterAll` blocks.

## Testing Gotchas

### 1. Goroutine Leaks

Be careful with goroutines in tests. Ensure they're properly terminated:

```go
It("doesn't leak goroutines", func() {
    done := make(chan struct{})
    
    go func() {
        defer close(done)
        // Do work
    }()
    
    // Wait for goroutine to finish
    Eventually(done).Should(BeClosed())
})
```

### 2. Race Conditions

Run tests with the race detector to catch race conditions:

```bash
go test -race ./...
```

### 3. Flaky Tests

If a test is flaky:
- Increase timeouts or retry counts
- Check for race conditions
- Ensure mocks are thread-safe
- Verify cleanup between tests

## Conclusion

Following these best practices will help ensure that the Loggregator Agent tests are robust, maintainable, and effective at catching regressions. As the codebase evolves, these practices should be reviewed and updated to reflect new testing patterns and tools. 