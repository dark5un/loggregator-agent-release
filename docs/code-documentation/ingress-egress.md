# Ingress and Egress Components

This document details the ingress and egress components of the Loggregator Agent, explaining how data flows through the system.

## Ingress Overview

The ingress component is responsible for receiving and processing incoming log and metric data before passing it to other components. It's implemented in the `pkg/ingress` directory, with separate implementations for the v1 (legacy) and v2 protocols.

### V1 Ingress

Located in `pkg/ingress/v1`, this component handles the legacy protocol used by older CF components.

Key components:
- **Network Reader**: Reads data from UDP listeners
- **Unmarshaller**: Converts binary data into message structures
- **Event Marshaller**: Processes events into envelopes

### V2 Ingress

Located in `pkg/ingress/v2`, this component handles the newer protocol with richer message types.

The main component is the **Receiver**, which:
- Implements the gRPC server interface defined in the loggregator_v2 protobuf
- Provides multiple endpoints for receiving data (Send, Sender, BatchSender)
- Validates incoming envelopes
- Sets proper source IDs on envelopes
- Forwards data to the DataSetter interface

```go
// Simplified version of Receiver from pkg/ingress/v2/receiver.go
type Receiver struct {
    loggregator_v2.UnimplementedIngressServer

    dataSetter           DataSetter
    ingressMetric        func(uint64)
    originMappingsMetric func(uint64)
}

func (s *Receiver) Sender(sender loggregator_v2.Ingress_SenderServer) error {
    for {
        e, err := sender.Recv()
        if err != nil {
            log.Printf("Failed to receive data: %s", err)
            return err
        }
        e.SourceId = s.sourceID(e)
        s.dataSetter.Set(e)
        s.ingressMetric(1)
    }
}
```

## Egress Overview

The egress component is responsible for forwarding processed data to its final destination, typically Dopplers. It's implemented in the `pkg/egress` directory.

### Writer Implementations

Multiple writer implementations exist for different protocols and destinations:

- **gRPC Writer**: Sends data to gRPC endpoints
- **TCP Writer**: Sends data over raw TCP
- **Batching Writer**: Accumulates messages for batch sending
- **Retry Writer**: Wraps other writers and implements retry logic

### Connector

The egress connector component manages connections to destinations:

- Maintains a pool of connections to Dopplers
- Implements service discovery of Dopplers
- Handles connection failures and retries

```go
// Simplified example of connector usage
func NewGRPCConnector(opts ...ConnectorOption) *GRPCConnector {
    c := &GRPCConnector{
        pool:    make(map[string]clientPool),
        lookup:  lookupFromAddrs([]string{"doppler.service.cf.internal:8082"}),
        clients: 5,
        urls:    []string{"doppler.service.cf.internal:8082"},
    }
    
    for _, o := range opts {
        o(c)
    }
    
    return c
}
```

## Data Flow Example

Here's how data typically flows through these components:

1. **Ingress**: A log message is received via gRPC
   ```go
   // Client sends a log message
   client.Send(&loggregator_v2.Envelope{
       Message: &loggregator_v2.Envelope_Log{
           Log: &loggregator_v2.Log{
               Payload: []byte("App log message"),
           },
       },
   })
   ```

2. **Processing**: The Receiver validates and enhances the envelope
   ```go
   // In the Receiver
   e.SourceId = s.sourceID(e)
   s.dataSetter.Set(e)
   ```

3. **Diode**: The message passes through a diode for backpressure handling
   ```go
   // Simplified diode usage
   d := diodes.NewOneToOne(1024, alerter)
   d.Set(diodeEnvelope)
   ```

4. **Egress**: A writer sends the message to its destination
   ```go
   // In a gRPC writer
   sender.Send(&loggregator_v2.EnvelopeBatch{
       Batch: batch,
   })
   ```

## Handling Backpressure with Diodes

One of the key components in the system is the diode, which handles backpressure situations:

```go
// Simplified diode example
type Diode struct {
    buffer      []unsafe.Pointer
    writeIndex  uint64
    readIndex   uint64
    alerter     Alerter
}

func (d *Diode) Set(data unsafe.Pointer) {
    // If buffer is full, alert and drop older messages
    if d.writeIndex - d.readIndex > d.size {
        d.alerter.Alert(int(d.writeIndex - d.readIndex))
        d.readIndex++
    }
    
    // Write data to buffer
    d.buffer[d.writeIndex%d.size] = data
    d.writeIndex++
}
```

The diode ensures that the system degrades gracefully under high load by dropping less important messages rather than causing backups throughout the system.

## Metrics and Monitoring

Both ingress and egress components emit metrics for monitoring:

- **Ingress metrics**:
  - Number of received envelopes
  - Parsing errors
  - Source ID mappings

- **Egress metrics**:
  - Number of sent envelopes
  - Batch sizes
  - Errors/failed attempts
  - Retry counts
  - Connection status

These metrics can be used to monitor the health and performance of the Loggregator Agent. 