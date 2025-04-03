# Loggregator Agent Code Documentation

This documentation provides an analysis of the Loggregator Agent codebase, including its structure, principles, and examples of usage.

## Overview

Loggregator Agent is a Cloud Foundry component that forwards logs and metrics into the Loggregator subsystem. It takes traffic from various emitter sources (diego, router, etc.) and routes that traffic to one or more dopplers. An instance of Loggregator Agent runs on each VM in an environment and is co-located on the emitter sources.

## Codebase Structure

The codebase is organized into the following main directories:

- `cmd/`: Contains the main executable packages
  - `loggregator-agent/`: The main agent implementation
  - `syslog-agent/`: Agent for handling syslog forwarding
  - `prom-scraper/`: Scraper for Prometheus metrics
  - `forwarder-agent/`: Agent for forwarding logs
  - `syslog-binding-cache/`: Cache for syslog bindings
  - `udp-forwarder/`: Agent for forwarding UDP traffic

- `pkg/`: Contains reusable packages
  - `binding/`: Binding-related functionality
  - `cache/`: Caching implementations
  - `clientpool/`: Pool of clients
  - `config/`: Configuration structures
  - `diodes/`: Data structures for message passing
  - `egress/`: Outbound traffic handling
  - `ingress/`: Inbound traffic handling
  - `plumbing/`: Utility functions and structures
  - `scraper/`: Metric scraping functionality
  - `simplecache/`: Simple caching implementations
  - `timeoutwaitgroup/`: WaitGroup with timeout functionality
  - `otelcolclient/`: OpenTelemetry collector client

- `internal/`: Internal implementation details 

- `integration_tests/`: Integration test suite

## Core Components

### Agent

The main Loggregator Agent component (in `cmd/loggregator-agent`) is responsible for:

1. Receiving logs and metrics from local sources
2. Processing and validating that data
3. Forwarding it securely to Dopplers

The agent supports both UDP and gRPC protocols, with the ability to disable UDP if needed.

#### Agent Configuration

The Agent is configured through environment variables, which include:

- Basic identity information: deployment, zone, job, index, IP
- Tags for metadata
- Protocol configurations (UDP/gRPC ports)
- TLS certificate information
- Metrics settings

### Ingress and Egress

Two fundamental concepts in the Loggregator Agent are:

- **Ingress**: Processing of incoming log and metric data
  - Implemented in `pkg/ingress`
  - Supports v1 (legacy) and v2 protocols
  - Handles validation and normalization of data

- **Egress**: Forwarding of processed data to Dopplers
  - Implemented in `pkg/egress`
  - Supports various writer implementations
  - Handles batching and retry logic

### Data Flow

The overall data flow through the Loggregator Agent follows these steps:

1. Data is received via gRPC or UDP (ingress)
2. Processed and normalized into envelopes
3. Batched for efficiency
4. Forwarded to Dopplers (egress)

This pipeline leverages non-blocking data structures (diodes) to handle backpressure appropriately.

## Design Principles

The Loggregator Agent follows several key design principles:

### 1. Security First

The codebase prioritizes security through:
- Mandatory TLS for gRPC connections
- Certificate validation
- Local-only listening for ingress traffic
- Data signing to prevent tampering

### 2. Performance and Reliability

The agent is designed for high-performance environments:
- Non-blocking data structures (diodes) for handling backpressure
- Batching of messages for efficient network usage
- Graceful degradation under load
- Retry mechanisms with exponential backoff

### 3. Observability

The agent provides comprehensive metrics about its own operation:
- Ingress and egress counters
- Processing latency
- Error rates
- Health indicators

### 4. Configurability

The agent can be configured extensively through environment variables to adapt to different deployment scenarios.

### 5. Protocol Support

The agent supports both legacy (v1) and modern (v2) protocols to ensure backward compatibility while enabling new features.

## Usage Examples

### Integrating with Loggregator Agent

Applications can send logs and metrics to Loggregator Agent using the gRPC API. Here's an example of how to implement a client:

```go
package main

import (
    "context"
    "log"
    "time"

    "code.cloudfoundry.org/go-loggregator/v10/rpc/loggregator_v2"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials"
)

func main() {
    // Set up TLS credentials
    creds, err := credentials.NewClientTLSFromFile("path/to/ca.crt", "doppler")
    if err != nil {
        log.Fatalf("Failed to create TLS credentials: %v", err)
    }

    // Connect to the agent
    conn, err := grpc.Dial("localhost:3458", grpc.WithTransportCredentials(creds))
    if err != nil {
        log.Fatalf("Failed to dial agent: %v", err)
    }
    defer conn.Close()

    // Create a client
    client := loggregator_v2.NewIngressClient(conn)

    // Create a sender
    sender, err := client.Sender(context.Background())
    if err != nil {
        log.Fatalf("Failed to create sender: %v", err)
    }

    // Send a log message
    err = sender.Send(&loggregator_v2.Envelope{
        Timestamp: time.Now().UnixNano(),
        SourceId:  "my-app",
        Message: &loggregator_v2.Envelope_Log{
            Log: &loggregator_v2.Log{
                Payload: []byte("Hello, Loggregator!"),
                Type:    loggregator_v2.Log_OUT,
            },
        },
        Tags: map[string]string{
            "source_type": "APP",
        },
    })
    if err != nil {
        log.Fatalf("Failed to send log: %v", err)
    }

    log.Println("Log sent successfully")
}
```

### Configuring the Agent

To configure the Loggregator Agent, you would set environment variables like:

```bash
# Identity
export AGENT_DEPLOYMENT=cf
export AGENT_ZONE=z1
export AGENT_JOB=loggregator_agent
export AGENT_INDEX=0
export AGENT_IP=10.0.0.1

# TLS Configuration
export AGENT_CA_FILE=/path/to/ca.crt
export AGENT_CERT_FILE=/path/to/agent.crt
export AGENT_KEY_FILE=/path/to/agent.key

# Network configuration
export AGENT_PORT=3458
export AGENT_INCOMING_UDP_PORT=3457
export ROUTER_ADDR=doppler.service.cf.internal:8082

# Optional settings
export AGENT_DISABLE_UDP=false
export AGENT_METRIC_BATCH_INTERVAL_MILLISECONDS=60000
export USE_RFC3339=true
```

### Processing Metrics

The Loggregator Agent can also receive and forward metrics. Here's an example of sending a counter metric:

```go
// Send a counter metric
err = sender.Send(&loggregator_v2.Envelope{
    Timestamp: time.Now().UnixNano(),
    SourceId:  "my-app",
    Message: &loggregator_v2.Envelope_Counter{
        Counter: &loggregator_v2.Counter{
            Name:  "requests",
            Delta: 1,
            Total: 100,
        },
    },
    Tags: map[string]string{
        "source_type": "APP",
    },
})
```

## Component Documentation

For more detailed information about specific components, refer to the following documentation:

- [Loggregator Agent](loggregator-agent.md) - Detailed documentation of the main agent component
- [Syslog Agent](syslog-agent.md) - Documentation of the syslog forwarding component
- [Prometheus Scraper](prom-scraper.md) - Documentation of the Prometheus metrics integration
- [Ingress and Egress](ingress-egress.md) - In-depth explanation of the data flow components
- [Diodes](diodes.md) - Details on the backpressure handling mechanism
- [Integration Example](integration-example.md) - Complete example of integrating with the agent

## Key Concepts

### Envelopes

The core data structure in the Loggregator system is the Envelope, which wraps all data with metadata:

```protobuf
message Envelope {
    int64 timestamp = 1;
    string source_id = 2;
    string instance_id = 3;
    
    oneof message {
        Log log = 4;
        Counter counter = 5;
        Gauge gauge = 6;
        Timer timer = 7;
        Event event = 8;
    }
    
    map<string, string> tags = 9;
}
```

### Metrics Types

The Loggregator Agent supports several metric types:

1. **Counter**: Monotonically increasing counters
2. **Gauge**: Values that can go up and down
3. **Timer**: Measurements of timing information
4. **Event**: Discrete events with a title and body

## Contributing

To contribute to the Loggregator Agent codebase:

1. Set up a development environment with Go 1.x
2. Clone the repository
3. Run tests using `go test ./...`
4. Make changes and submit a pull request 