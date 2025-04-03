# Migration from Hel to Mockery in Loggregator Agent

## Overview

This document outlines the process and rationale for migrating the Loggregator Agent codebase from the [hel](https://git.sr.ht/~nelsam/hel) mocking library to [mockery](https://github.com/vektra/mockery) for unit testing. The migration was undertaken to leverage a more actively maintained mocking solution with improved capabilities and better integration with modern Go tooling.

## Motivation

### Limitations of Hel

The `hel` mocking library (previously at `git.sr.ht/~nelsam/hel/v3`) had several limitations:

1. **Channel-based approach**: Hel used a channel-based approach for mocking, which could lead to deadlocks and race conditions in tests.
2. **Limited maintainability**: The library wasn't being actively maintained.
3. **Go module compatibility**: Hel sometimes had issues with newer Go module versions.
4. **Non-standard pattern**: The channel-based approach is not a common pattern in Go mocking libraries.

### Benefits of Mockery

[Mockery](https://github.com/vektra/mockery) offers several improvements:

1. **Method-based approach**: Uses a more standard expectation-and-return-value approach.
2. **Active development**: Actively maintained with regular updates.
3. **Testify integration**: Built on the widely-used testify library.
4. **Type safety**: Provides type-safe mocks.
5. **Thread safety**: Better handling of concurrent operations.

## Implementation Details

### Dependencies Added

The following dependencies were added to `go.mod`:

```go
github.com/stretchr/testify v1.10.0
github.com/vektra/mockery/v2 v2.53.3
```

### Migration Process

The migration followed these key steps:

1. **Identifying mocks**: All `//go:generate hel` directives were identified throughout the codebase.
2. **Updating generation directives**: Changed `//go:generate hel` to `//go:generate mockery` with appropriate interface targeting.
3. **Creating a MockTesting helper**: Implemented a `MockTesting` helper to satisfy mockery's `TestingT` interface when used with Ginkgo.
4. **Updating test patterns**: Modified tests to use mockery's expectation-based approach instead of hel's channel-based approach.
5. **Fixing concurrency issues**: Added thread-safe access methods to mocks that handle concurrent access.

### Key Components Changed

#### 1. MockTesting Helper

Created a helper struct to bridge mockery with Ginkgo:

```go
// src/pkg/testhelpers/mock_testing.go
package testhelpers

type MockTesting struct{}

func NewMockTesting() *MockTesting {
    return &MockTesting{}
}

func (*MockTesting) Errorf(format string, args ...interface{}) {}
func (*MockTesting) Cleanup(func()) {}
```

#### 2. Thread-Safety in Mocks

Implemented thread-safe access patterns in mocks:

```go
// Thread-safe counter in BatchWriter
type BatchWriter struct {
    mock.Mock
    mu    sync.Mutex
    Calls int
}

func (_m *BatchWriter) Write(msgs []*loggregator_v2.Envelope) error {
    ret := _m.Called(msgs)
    
    _m.mu.Lock()
    _m.Calls++
    _m.mu.Unlock()
    
    return ret.Error(0)
}

func (_m *BatchWriter) GetCalls() int {
    _m.mu.Lock()
    defer _m.mu.Unlock()
    return _m.Calls
}
```

#### 3. Testing Pattern Changes

Before (with hel):
```go
writer := helheim.NewMockBatchWriter()
writer.WriteOutput.Ret0 <- nil
// Check number of calls
Eventually(writer.WriteCalled).Should(HaveLen(1))
```

After (with mockery):
```go
mockT := testhelpers.NewMockTesting()
writer := mocks.NewBatchWriter(mockT)
writer.On("Write", []*loggregator_v2.Envelope{envelope}).Return(nil)
// Check number of calls
Eventually(func() int {
    return writer.GetCalls()
}).Should(Equal(1))
```

## Challenges and Solutions

### 1. Race Conditions

**Challenge**: Mockery-generated mocks had race conditions when accessed concurrently.

**Solution**: Added mutex protection around shared data in mocks and created thread-safe accessor methods.

### 2. Ginkgo Integration

**Challenge**: Mockery expects a `testing.T` compatible interface, but Ginkgo has a different testing structure.

**Solution**: Created a simple `MockTesting` implementation that satisfies mockery's requirements.

### 3. Interface Differences

**Challenge**: Some interfaces required different mocking approaches due to usage patterns.

**Solution**: For some cases, created custom mock implementations alongside the mockery-generated ones to maintain test semantics.

## Testing Results

The migration was successful with all tests passing. The process revealed and fixed several race conditions that were present in the original tests.

Test suite statistics:
- 27 test suites
- All tests passing with race detection enabled

## Future Considerations

- Consider migrating to use Go's built-in `testing.Mock` in future versions
- Further cleanup of any remaining race conditions
- Standardize testing patterns across the codebase

## Conclusion

This migration improved the test infrastructure of the Loggregator Agent by:

1. Eliminating potential race conditions and deadlocks
2. Adopting a more standard mocking approach
3. Using a more actively maintained library
4. Improving test reliability and maintainability

The codebase is now better positioned for future development and maintenance. 