# Integration Example: Using Loggregator Agent

This document provides a complete example of how to integrate an application with the Loggregator Agent for sending logs and metrics.

## Overview

Integrating with Loggregator Agent involves:
1. Setting up a gRPC client
2. Establishing a secure connection
3. Sending log and metric data via the appropriate APIs

## Prerequisites

- Loggregator Agent running and accessible (typically on localhost:3458)
- TLS certificates for secure communication
- Go application with the following dependencies:
  ```go
  import (
      "code.cloudfoundry.org/go-loggregator/v10/rpc/loggregator_v2"
      "google.golang.org/grpc"
      "google.golang.org/grpc/credentials"
  )
  ```

## Complete Application Example

Below is a complete sample application that demonstrates how to:
- Connect to Loggregator Agent
- Send logs, counters, gauges, and events
- Handle errors and reconnection

```go
package main

import (
    "context"
    "crypto/tls"
    "crypto/x509"
    "flag"
    "fmt"
    "io/ioutil"
    "log"
    "os"
    "time"

    "code.cloudfoundry.org/go-loggregator/v10/rpc/loggregator_v2"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials"
)

var (
    agentAddr    = flag.String("agent-addr", "localhost:3458", "Loggregator Agent address")
    caFile       = flag.String("ca-file", "", "CA certificate file")
    certFile     = flag.String("cert-file", "", "Client certificate file")
    keyFile      = flag.String("key-file", "", "Client private key file")
    sourceID     = flag.String("source-id", "example-app", "Source ID for logs and metrics")
    logInterval  = flag.Duration("log-interval", 1*time.Second, "Interval to send logs")
    metricCount  = flag.Int("metric-count", 10, "Number of metrics to send per batch")
)

func main() {
    flag.Parse()

    // Set up TLS credentials
    tlsConfig, err := loadTLSConfig(*caFile, *certFile, *keyFile)
    if err != nil {
        log.Fatalf("Failed to load TLS config: %v", err)
    }
    creds := credentials.NewTLS(tlsConfig)

    // Connect to the agent
    conn, err := grpc.Dial(*agentAddr, grpc.WithTransportCredentials(creds))
    if err != nil {
        log.Fatalf("Failed to dial agent: %v", err)
    }
    defer conn.Close()

    // Create a client
    client := loggregator_v2.NewIngressClient(conn)
    
    // Create a streaming sender
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    sender, err := client.Sender(ctx)
    if err != nil {
        log.Fatalf("Failed to create sender: %v", err)
    }

    // Create a batch sender for metrics
    batchSender, err := client.BatchSender(ctx)
    if err != nil {
        log.Fatalf("Failed to create batch sender: %v", err)
    }

    // Start sending logs
    go sendLogs(sender, *sourceID, *logInterval)
    
    // Start sending metrics
    go sendMetricsBatch(batchSender, *sourceID, *metricCount)

    // Keep the application running
    fmt.Println("Sending logs and metrics to Loggregator Agent...")
    fmt.Println("Press Ctrl+C to exit")
    
    // Block forever
    select {}
}

func sendLogs(sender loggregator_v2.Ingress_SenderClient, sourceID string, interval time.Duration) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()
    
    for t := range ticker.C {
        logMessage := fmt.Sprintf("Application log message at %s", t.Format(time.RFC3339))
        
        err := sender.Send(&loggregator_v2.Envelope{
            Timestamp: time.Now().UnixNano(),
            SourceId:  sourceID,
            Message: &loggregator_v2.Envelope_Log{
                Log: &loggregator_v2.Log{
                    Payload: []byte(logMessage),
                    Type:    loggregator_v2.Log_OUT,
                },
            },
            Tags: map[string]string{
                "source_type": "APP",
                "instance_id": "0",
            },
        })
        
        if err != nil {
            log.Printf("Error sending log: %v", err)
        } else {
            log.Printf("Sent log: %s", logMessage)
        }
    }
}

func sendMetricsBatch(sender loggregator_v2.Ingress_BatchSenderClient, sourceID string, count int) {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()
    
    counterTotal := int64(0)
    
    for range ticker.C {
        batch := &loggregator_v2.EnvelopeBatch{
            Batch: make([]*loggregator_v2.Envelope, 0, count),
        }

        // Add a counter
        counterTotal += int64(count)
        batch.Batch = append(batch.Batch, &loggregator_v2.Envelope{
            Timestamp: time.Now().UnixNano(),
            SourceId:  sourceID,
            Message: &loggregator_v2.Envelope_Counter{
                Counter: &loggregator_v2.Counter{
                    Name:  "requests",
                    Delta: int64(count),
                    Total: counterTotal,
                },
            },
            Tags: map[string]string{
                "source_type": "APP",
            },
        })

        // Add a gauge
        batch.Batch = append(batch.Batch, &loggregator_v2.Envelope{
            Timestamp: time.Now().UnixNano(),
            SourceId:  sourceID,
            Message: &loggregator_v2.Envelope_Gauge{
                Gauge: &loggregator_v2.Gauge{
                    Metrics: map[string]*loggregator_v2.GaugeValue{
                        "memory": {
                            Unit:  "bytes",
                            Value: 1024 * 1024 * 100, // 100 MB
                        },
                        "cpu": {
                            Unit:  "percentage",
                            Value: 35.5,
                        },
                    },
                },
            },
        })

        // Add an event
        batch.Batch = append(batch.Batch, &loggregator_v2.Envelope{
            Timestamp: time.Now().UnixNano(),
            SourceId:  sourceID,
            Message: &loggregator_v2.Envelope_Event{
                Event: &loggregator_v2.Event{
                    Title: "Application Status",
                    Body:  "Application is healthy",
                },
            },
        })

        err := sender.Send(batch)
        if err != nil {
            log.Printf("Error sending metrics batch: %v", err)
        } else {
            log.Printf("Sent metrics batch with %d envelopes", len(batch.Batch))
        }
    }
}

func loadTLSConfig(caFile, certFile, keyFile string) (*tls.Config, error) {
    // Load CA cert
    caCert, err := ioutil.ReadFile(caFile)
    if err != nil {
        return nil, fmt.Errorf("failed to read CA cert file: %v", err)
    }

    caCertPool := x509.NewCertPool()
    if !caCertPool.AppendCertsFromPEM(caCert) {
        return nil, fmt.Errorf("failed to parse CA cert")
    }

    // Load client cert/key
    cert, err := tls.LoadX509KeyPair(certFile, keyFile)
    if err != nil {
        return nil, fmt.Errorf("failed to load client cert/key: %v", err)
    }

    return &tls.Config{
        RootCAs:      caCertPool,
        Certificates: []tls.Certificate{cert},
        ServerName:   "doppler", // Must match the CN in the server certificate
    }, nil
}
```

## Building and Running the Example

1. Save the code above to a file named `loggregator_example.go`
2. Build the application:
   ```bash
   go build -o loggregator_example loggregator_example.go
   ```
3. Run the application with the appropriate certificates:
   ```bash
   ./loggregator_example \
       --agent-addr=localhost:3458 \
       --ca-file=/path/to/ca.crt \
       --cert-file=/path/to/client.crt \
       --key-file=/path/to/client.key \
       --source-id=my-application
   ```

## Common Patterns and Best Practices

### Connection Management

- **Reconnection Logic**: When the connection is lost, implement backoff retry logic
- **Connection Pooling**: For high-volume applications, consider maintaining a pool of connections

```go
// Example of reconnection with backoff
func connectWithRetry(addr string, creds credentials.TransportCredentials) (*grpc.ClientConn, error) {
    backoff := 1 * time.Second
    maxBackoff := 1 * time.Minute
    
    for {
        conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(creds))
        if err == nil {
            return conn, nil
        }
        
        log.Printf("Failed to connect, retrying in %v: %v", backoff, err)
        time.Sleep(backoff)
        
        // Exponential backoff with cap
        backoff *= 2
        if backoff > maxBackoff {
            backoff = maxBackoff
        }
    }
}
```

### Batch Processing

For high-throughput scenarios, use batch processing to optimize network usage:

```go
// Collect logs in a batch
const batchSize = 100
batch := make([]*loggregator_v2.Envelope, 0, batchSize)

// Add to batch
batch = append(batch, logEnvelope)

// When batch is full or a timeout is reached, send it
if len(batch) >= batchSize {
    batchSender.Send(&loggregator_v2.EnvelopeBatch{Batch: batch})
    batch = make([]*loggregator_v2.Envelope, 0, batchSize)
}
```

### Envelope Tagging

Use consistent tagging to enable filtering and aggregation:

```go
// Common tags for all envelopes
commonTags := map[string]string{
    "source_type": "APP",
    "instance_id": os.Getenv("CF_INSTANCE_INDEX"),
    "app_id":      os.Getenv("CF_APP_ID"),
    "env":         "production",
}

// Add to envelope
envelope.Tags = commonTags
```

### Type-Specific Best Practices

#### Logs

- Include timestamp and severity
- Use structured logging when possible
- Set appropriate log type (OUT/ERR)

#### Counters

- Use consistent naming
- Choose between cumulative and delta counters based on aggregation needs
- Always set the total field

#### Gauges

- Include appropriate units
- Report related metrics in a single gauge
- Use consistent metric names

#### Events

- Keep titles short and descriptive
- Include actionable information in the body
- Use for significant application state changes

## Troubleshooting

If you encounter issues:

1. **Connection Refused**: Verify the Loggregator Agent is running and the address is correct
2. **TLS Errors**: Ensure certificates are valid and the server name matches
3. **Message Drops**: The Loggregator Agent might be under high load, consider batching or reducing volume
4. **Slow Processing**: Check network connectivity and consider using dedicated network interfaces

## Further Reading

For more information, refer to:
- [Loggregator Architecture](../loggregator-agent.md)
- [go-loggregator Client Library](https://github.com/cloudfoundry/go-loggregator)
- [Envelope Specification](https://github.com/cloudfoundry/loggregator-api/blob/master/v2/envelope.proto) 