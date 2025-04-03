# Diodes: Managing Backpressure in Loggregator Agent

The `diodes` package, located in `src/pkg/diodes`, is a critical component of the Loggregator Agent that provides non-blocking data structures for managing backpressure in high-throughput messaging systems.

## Overview

Diodes are specialized ring buffers that implement a "lose newest" or "lose oldest" policy when the buffer is full. They provide thread-safe operations for high-performance, lock-free communication between goroutines.

The primary goal of diodes is to:
- Prevent slow consumers from blocking producers
- Gracefully degrade under heavy load
- Provide consistent performance
- Alert when message dropping occurs

## Types of Diodes

The package provides several types of diodes for different use cases:

### OneToOne

A `OneToOne` diode is designed for communication between a single writer and a single reader. It's the simplest diode implementation that doesn't require atomic operations for managing indices.

```go
// Creating a OneToOne diode
alerter := diodes.AlertFunc(func(missed int) {
    log.Printf("Dropped %d messages", missed)
})
d := diodes.NewOneToOne(1024, alerter)

// Writing to the diode
d.Set(diodes.GenericDataType(&myData))

// Reading from the diode
data := d.TryNext()
if data != nil {
    myData := (*MyDataType)(data)
    // Process myData
}
```

### ManyToOne

A `ManyToOne` diode allows multiple writers but a single reader. It uses atomic operations to ensure thread safety among the writers.

```go
// Creating a ManyToOne diode
d := diodes.NewManyToOne(1024, alerter)

// Writing from multiple goroutines
go func() {
    d.Set(diodes.GenericDataType(&data1))
}()
go func() {
    d.Set(diodes.GenericDataType(&data2))
}()

// Reading from a single goroutine
for {
    data := d.TryNext()
    if data != nil {
        // Process data
    }
}
```

### OneToMany

A `OneToMany` diode allows a single writer but multiple readers. Each reader maintains its own read index.

```go
// Creating a OneToMany diode
d := diodes.NewOneToMany(1024, alerter)

// Creating poller for reader 1
poller1 := d.NewPoller()

// Creating poller for reader 2
poller2 := d.NewPoller()

// Writing data
d.Set(diodes.GenericDataType(&myData))

// Reading from different goroutines
go func() {
    for {
        data := poller1.TryNext()
        if data != nil {
            // Reader 1 processing
        }
    }
}()

go func() {
    for {
        data := poller2.TryNext()
        if data != nil {
            // Reader 2 processing
        }
    }
}()
```

## Core Concepts

### Alerting

Diodes take an alerter that gets notified when messages are dropped due to buffer overflow. This allows the system to monitor and respond to backpressure situations.

```go
type Alerter interface {
    Alert(missed int)
}

// Example implementation
type AlertFunc func(missed int)

func (f AlertFunc) Alert(missed int) {
    f(missed)
}
```

### Data Types

Diodes operate on `unsafe.Pointer` for maximum flexibility, but provide the `GenericDataType` function to simplify usage:

```go
func GenericDataType(data interface{}) unsafe.Pointer {
    return unsafe.Pointer(&data)
}
```

### Policy Selection

Diodes can be configured with different dropping policies:

- **Drop Newest**: When the buffer is full, new incoming messages are dropped
- **Drop Oldest**: When the buffer is full, the oldest messages in the buffer are dropped to make room for new ones

## Implementation Details

### Internal Structure

The basic structure of a diode includes:
- A buffer of fixed size to store pointers to data
- Write and read indices for tracking positions
- An alerter for notification when dropping occurs

```go
type Diode struct {
    buffer      []unsafe.Pointer
    writeIndex  uint64
    readIndex   uint64
    alerter     Alerter
    bufferSize  uint64
    dropPolicy  DropPolicy
}
```

### Thread Safety

Different diode implementations handle thread safety differently:

- `OneToOne`: No atomic operations needed since there's a single writer and reader
- `ManyToOne`: Atomic operations for writing to ensure thread safety
- `OneToMany`: Atomic operations for reading with multiple pollers

### Performance Characteristics

Diodes are optimized for performance with the following characteristics:
- Lock-free design minimizes contention
- Cache-friendly memory layout
- Constant-time operations for both reading and writing
- Efficient handling of high-throughput scenarios

## Usage in Loggregator Agent

In the Loggregator Agent, diodes are used extensively in the ingress-to-egress pipeline:

1. **Ingress Receivers** write incoming envelopes to diodes
2. **Egress Writers** read from these diodes and forward messages to destinations
3. This decoupling ensures that slow Doppler connections don't affect the receipt of new logs

Example from the codebase:

```go
// Simplified from actual implementation
func NewAgent(config *Config) *Agent {
    // Create diode for envelope batching
    ingressDiode := diodes.NewManyToOne(10000, alerter)
    
    // Set up ingress servers to write to the diode
    receiver := ingress.NewReceiver(
        func(e *loggregator_v2.Envelope) {
            ingressDiode.Set(diodes.GenericDataType(e))
        },
        // other params...
    )
    
    // Set up egress client to read from the diode
    egressClient := egress.NewClient(
        func() *loggregator_v2.Envelope {
            data := ingressDiode.TryNext()
            if data == nil {
                return nil
            }
            return (*loggregator_v2.Envelope)(data)
        },
        // other params...
    )
    
    // Start processing
    go egressClient.Start()
    
    return &Agent{
        receiver: receiver,
        egress: egressClient,
        // other fields...
    }
}
```

## Best Practices

When working with diodes:

1. **Size appropriately**: Choose buffer sizes based on expected throughput and burstiness
2. **Monitor drops**: Always implement alerters to track message drops
3. **Policy selection**: Choose drop policies based on your application's requirements
4. **Balancing resources**: Larger buffers use more memory but reduce the chance of drops

## Testing Diodes

The diodes package includes comprehensive tests that demonstrate proper usage:

```go
func TestManyToOneDropsOldest(t *testing.T) {
    d := NewManyToOne(3, AlertFunc(func(missed int) {
        // Expect 1 message to be dropped
        if missed != 1 {
            t.Errorf("Expected 1 dropped message, got %d", missed)
        }
    }))
    
    // Fill the buffer
    d.Set(GenericDataType("a"))
    d.Set(GenericDataType("b"))
    d.Set(GenericDataType("c"))
    
    // This should cause a drop of the oldest message ("a")
    d.Set(GenericDataType("d"))
    
    // Verify proper values are read
    Expect(d.Next()).To(Equal(GenericDataType("b")))
    Expect(d.Next()).To(Equal(GenericDataType("c")))
    Expect(d.Next()).To(Equal(GenericDataType("d")))
}
```

By understanding and properly utilizing diodes, you can build high-performance, resilient systems that handle backpressure gracefully. 