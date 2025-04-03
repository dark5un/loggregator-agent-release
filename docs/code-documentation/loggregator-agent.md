# Loggregator Agent

This document provides a detailed analysis of the Loggregator Agent component, its architecture, implementation details, and usage.

## Overview

The Loggregator Agent is the primary component in the loggregator-agent-release that receives logs and metrics from local sources on a VM and forwards them to Dopplers for further processing and distribution. It serves as a critical link in the data pipeline of Cloud Foundry's logging and metrics system.

## Architecture

The Loggregator Agent follows a modular architecture with several key components:

```
┌───────────────────────────────────────────────────────┐
│                   Loggregator Agent                   │
│                                                       │
│  ┌─────────────┐     ┌──────────┐     ┌────────────┐  │
│  │   Ingress   │     │  Diodes  │     │   Egress   │  │
│  │  Receivers  │────▶│ (Buffer) │────▶│  Writers   │  │
│  └─────────────┘     └──────────┘     └────────────┘  │
│         ▲                                    │        │
└─────────│────────────────────────────────────│────────┘
          │                                    │
          │                                    ▼
┌─────────│────────────┐            ┌─────────────────┐
│ Local Applications   │            │    Dopplers     │
│ and System Components│            │                 │
└──────────────────────┘            └─────────────────┘
```

### Key Components

1. **Ingress Receivers**: 
   - Receive logs and metrics via v1 (UDP) and v2 (gRPC) protocols
   - Parse and validate incoming data
   - Convert to internal envelope format

2. **Diodes**:
   - Provide non-blocking buffers between ingress and egress
   - Implement backpressure handling
   - Allow for graceful degradation under load

3. **Egress Writers**:
   - Batch messages for efficient transmission
   - Maintain connections to Dopplers
   - Implement retry logic and circuit breaking
   - Handle TLS for secure communication

## Implementation Details

### V1 and V2 Apps

The Loggregator Agent implements both V1 (legacy) and V2 protocols through separate applications initialized in `agent.go`:

```go
// From agent.go
appV1 := NewV1App(a.config, clientCreds, metricClient)
go appV1.Start()

appV2 := NewV2App(a.config, clientCreds, serverCreds, metricClient)
appV2.Start()
```

#### V1 App (Legacy UDP Protocol)

The V1 App handles the legacy UDP-based protocol:

```go
// Simplified V1App implementation
type V1App struct {
    config       *Config
    creds        credentials.TransportCredentials
    metricClient MetricClient
}

func (a *V1App) Start() {
    // Set up UDP listener
    udpListener, err := net.ListenPacket("udp4", fmt.Sprintf(":%d", a.config.IncomingUDPPort))
    
    // Create NetworkReader for UDP messages
    networkReader, err := ingress.NewNetworkReader(udpListener)
    
    // Set up unmarshaller and message converter
    unmarshaller := ingress.NewUnMarshaller()
    
    // Connect to egress writer
    sender := egress.NewGRPCSender(a.config.RouterAddr, a.creds)
    
    // Start processing loop
    for {
        data, addr := networkReader.Read()
        message := unmarshaller.UnMarshal(data)
        // Process and forward message
        sender.Send(message)
    }
}
```

#### V2 App (gRPC Protocol)

The V2 App implements the newer gRPC-based protocol:

```go
// Simplified V2App implementation
type V2App struct {
    config       *Config
    clientCreds  credentials.TransportCredentials
    serverCreds  credentials.TransportCredentials
    metricClient MetricClient
}

func (a *V2App) Start() {
    // Set up gRPC server
    grpcServer := grpc.NewServer(grpc.Creds(a.serverCreds))
    
    // Create ingress receiver
    receiver := ingress.NewReceiver(...)
    
    // Register server
    loggregator_v2.RegisterIngressServer(grpcServer, receiver)
    
    // Set up egress client
    egressClient := egress.NewClient(a.config.RouterAddr, a.clientCreds)
    
    // Start server
    lis, err := net.Listen("tcp", fmt.Sprintf(":%d", a.config.GRPC.Port))
    grpcServer.Serve(lis)
}
```

### Configuration

The Loggregator Agent is configured via environment variables, with sensible defaults provided:

```go
// Default configuration values
cfg := Config{
    MetricBatchIntervalMilliseconds: 60000,
    MetricSourceID:                  "metron",
    IncomingUDPPort:                 3457,
    MetricsServer: config.MetricsServer{
        Port: 14824,
    },
    GRPC: GRPC{
        Port: 3458,
    },
}
```

Key configuration parameters include:
- **AGENT_PORT**: The gRPC port for receiving V2 protocol messages (default: 3458)
- **AGENT_INCOMING_UDP_PORT**: The UDP port for receiving V1 protocol messages (default: 3457)
- **ROUTER_ADDR**: The address of the Doppler service
- **AGENT_CERT_FILE**, **AGENT_KEY_FILE**, **AGENT_CA_FILE**: TLS credentials
- **AGENT_METRIC_BATCH_INTERVAL_MILLISECONDS**: Controls batching behavior (default: 60000)

### Metrics

The Loggregator Agent emits various metrics about its operation:

1. **Ingress Metrics**:
   - `ingress`: Number of envelopes received
   - `dropped`: Number of envelopes dropped during ingress
   - `origin_mappings`: Number of legacy envelopes with origin mapping

2. **Egress Metrics**:
   - `egress`: Number of envelopes transmitted
   - `dropped`: Number of envelopes dropped during egress
   - `retry_count`: Number of transmission retries

3. **System Metrics**:
   - `memory_usage`: Memory usage of the agent process
   - `cpu_usage`: CPU utilization of the agent process

## Operational Considerations

### Scaling

The Loggregator Agent is designed to be deployed alongside application instances and system components. Key scaling considerations include:

- **Buffer Size**: The size of diodes should be tuned based on expected message volume and burstiness
- **Batch Size**: Larger batch sizes improve throughput but introduce more latency
- **Resource Allocation**: The agent needs sufficient CPU and memory resources to handle peak loads

### High Availability

The Loggregator Agent inherently provides high availability through:

- **Per-VM Deployment**: Each VM runs its own agent, eliminating single points of failure
- **Graceful Degradation**: Under load, the agent prioritizes newer messages
- **Reconnection Logic**: The agent automatically reconnects to Dopplers when connections fail

### Security

Security is a primary concern for the Loggregator Agent:

- **TLS**: All gRPC connections use TLS with mutual authentication
- **Local-only Binding**: The agent only listens on local interfaces by default
- **Certificate Verification**: The agent verifies peer certificates for all connections

## Integration Examples

### Deploying with BOSH

The Loggregator Agent is typically deployed using BOSH:

```yaml
# Simplified BOSH manifest excerpt
instance_groups:
- name: loggregator_agent
  instances: 1
  jobs:
  - name: loggregator_agent
    release: loggregator-agent
    properties:
      loggregator:
        tls:
          ca: "((loggregator_ca.certificate))"
          agent:
            cert: "((loggregator_tls_agent.certificate))"
            key: "((loggregator_tls_agent.private_key))"
        router_addr: "doppler.service.cf.internal:8082"
```

### Pushing Logs and Metrics

Applications can push logs and metrics to the Loggregator Agent using:

```go
// Using the go-loggregator client
import "code.cloudfoundry.org/go-loggregator/v10"

func main() {
    tlsConfig := &tls.Config{/* ... */}
    
    client, err := loggregator.NewClient(
        loggregator.WithSenderAddr("localhost:3458"),
        loggregator.WithSenderTLSConfig(tlsConfig),
    )
    
    // Send a log message
    client.EmitLog(
        "Application log message",
        loggregator.WithAppInfo("app-id", "2"),
        loggregator.WithStdout(),
    )
    
    // Send a counter metric
    client.EmitCounter(
        "requests",
        loggregator.WithDelta(1),
        loggregator.WithTotal(100),
    )
}
```

## Advanced Topics

### Custom Metrics

The Loggregator Agent can emit custom metrics by using the `go-metric-registry` package:

```go
import metrics "code.cloudfoundry.org/go-metric-registry"

// Create a metric client
metricClient := metrics.NewRegistry(logger)

// Register and emit metrics
requestCounter := metricClient.NewCounter(
    "requests",
    "Number of requests received",
    metrics.WithMetricLabels(map[string]string{"protocol": "grpc"}),
)

// Increment counter
requestCounter.Add(1)
```

### Health Monitoring

The Loggregator Agent exposes a metrics endpoint that can be used for health monitoring:

```bash
# Query the metrics endpoint
curl -k https://localhost:14824/metrics
```

Example metrics output:
```
# HELP loggregator_agent_ingress Total number of envelopes received by the agent
# TYPE loggregator_agent_ingress counter
loggregator_agent_ingress 12345

# HELP loggregator_agent_dropped Total number of envelopes dropped by the agent
# TYPE loggregator_agent_dropped counter
loggregator_agent_dropped 42
```

## Troubleshooting

Common issues and their solutions:

1. **Connection Failures**: 
   - Check TLS certificates and their validity periods
   - Verify network connectivity to Dopplers
   - Ensure DNS resolution works for Doppler addresses

2. **Message Drops**:
   - Increase diode buffer sizes
   - Check for network congestion
   - Verify Doppler capacity

3. **High CPU/Memory Usage**:
   - Tune batch sizes and intervals
   - Monitor system resources
   - Consider scaling vertically (more CPU/memory)

## References

- [Loggregator Architecture](https://docs.cloudfoundry.org/loggregator/architecture.html)
- [Loggregator API](https://github.com/cloudfoundry/loggregator-api)
- [go-loggregator Client](https://github.com/cloudfoundry/go-loggregator) 