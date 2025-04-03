# Syslog Agent

This document provides a detailed analysis of the Syslog Agent component, its architecture, implementation details, and usage.

## Overview

The Syslog Agent is a component of the loggregator-agent-release that receives logs from the Loggregator system and forwards them to external syslog drains. It provides a way for applications to send their logs to third-party log management services or custom logging endpoints that support the syslog protocol.

## Architecture

The Syslog Agent follows a dedicated architecture focused on reliable log delivery:

```
┌──────────────────┐     ┌─────────────────────────────────────────────┐     ┌────────────────┐
│                  │     │               Syslog Agent                  │     │                │
│                  │     │                                             │     │                │
│   Loggregator    │     │  ┌─────────┐    ┌──────────┐    ┌────────┐ │     │  External      │
│   System         │────▶│  │ Ingress │───▶│ Binding  │───▶│ Egress │ │────▶│  Syslog Drains │
│                  │     │  │ Listener│    │ Cache    │    │ Writer │ │     │                │
│                  │     │  └─────────┘    └──────────┘    └────────┘ │     │                │
└──────────────────┘     └─────────────────────────────────────────────┘     └────────────────┘
```

### Key Components

1. **Ingress Listener**:
   - Receives logs from the Loggregator system
   - Processes and validates log envelopes
   - Filters logs based on configuration

2. **Binding Cache**:
   - Maintains a cache of syslog drain bindings
   - Periodically refreshes bindings from the Binding Cache service
   - Maps application IDs to syslog drain URLs

3. **Egress Writer**:
   - Formats logs according to syslog protocols (RFC5424, RFC3164)
   - Manages connections to syslog endpoints
   - Implements TLS for secure connections
   - Handles retry logic and backoff strategies

## Implementation Details

### Main Application

The Syslog Agent's main function initializes the application and its components:

```go
// Simplified from cmd/syslog-agent/main.go
func main() {
    // Load configuration from environment variables
    cfg := app.LoadConfig()
    
    // Set up metrics client
    metricClient := metrics.NewRegistry(...)
    
    // Create binding cache client
    bindingCache := binding.NewBindingCache(
        cfg.CacheCAFile,
        cfg.CacheClientCertFile,
        cfg.CacheClientKeyFile,
        cfg.CacheURL,
        bindingCacheMetrics,
        logger,
        time.Duration(cfg.CachePollingInterval) * time.Second,
    )
    
    // Create syslog client
    syslogClient := egress.NewSyslogClient(
        cfg.DrainSkipCertVerify,
        cfg.DrainTrustedCAFile,
        syslogTLSConfig,
        metricClient,
    )
    
    // Create binding fetcher
    fetcher := binding.NewBindingFetcher(
        bindingCache,
        time.Duration(cfg.CachePollingInterval) * time.Second,
    )
    
    // Create and start the agent application
    agent := app.NewAgent(
        cfg,
        fetcher,
        syslogClient,
        metricClient,
        logger,
    )
    agent.Start()
}
```

### Binding Cache

The Binding Cache component is responsible for retrieving and managing the syslog drain bindings:

```go
// Simplified from pkg/binding/binding_cache.go
type BindingCache struct {
    client       binding.HTTPClient
    pollingInterval time.Duration
    metrics      BindingCacheMetrics
    logger       *log.Logger
}

func (c *BindingCache) Get() (bindings []binding.Binding, err error) {
    req, err := http.NewRequest(http.MethodGet, c.cacheURL, nil)
    if err != nil {
        return nil, err
    }
    
    resp, err := c.client.Do(req)
    if err != nil {
        c.metrics.Request(0)
        return nil, err
    }
    defer resp.Body.Close()
    
    c.metrics.Request(resp.StatusCode)
    
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("unexpected response code %d", resp.StatusCode)
    }
    
    var bindingResp BindingResponse
    err = json.NewDecoder(resp.Body).Decode(&bindingResp)
    if err != nil {
        return nil, err
    }
    
    return bindingResp.Results, nil
}
```

### Egress Writer

The Egress Writer handles the actual delivery of logs to syslog endpoints:

```go
// Simplified from pkg/egress/syslog_writer.go
type Writer struct {
    url         *url.URL
    drainScope  string
    client      HTTPClient
    sourceIndex string
    logger      *log.Logger
    metrics     SyslogWriterMetrics
}

func (w *Writer) Write(env *loggregator_v2.Envelope) error {
    msg, err := w.buildMsg(env)
    if err != nil {
        return err
    }
    
    // Send message to syslog endpoint
    switch w.url.Scheme {
    case "https", "syslog-tls":
        return w.sendTLS(msg)
    case "syslog":
        return w.sendTCP(msg)
    case "syslog-udp":
        return w.sendUDP(msg)
    default:
        return fmt.Errorf("unsupported scheme: %s", w.url.Scheme)
    }
}

func (w *Writer) buildMsg(env *loggregator_v2.Envelope) (string, error) {
    switch w.drainScope {
    case binding.APP, binding.ALL:
        return w.buildAppMsg(env)
    default:
        return "", fmt.Errorf("invalid drain scope: %s", w.drainScope)
    }
}
```

### Configuration

The Syslog Agent is configured via environment variables:

```go
// Default configuration values
type Config struct {
    BindingCacheAPIURL         string `env:"BINDING_CACHE_URL,                required"`
    APICAFile                  string `env:"API_CA_FILE_PATH,                 required"`
    APICertFile                string `env:"API_CERT_FILE_PATH,               required"`
    APIKeyFile                 string `env:"API_KEY_FILE_PATH,                required"`
    APICommonName              string `env:"API_COMMON_NAME,                  required"`
    CacheCAFile                string `env:"CACHE_CA_FILE_PATH,               required"`
    CacheCertFile              string `env:"CACHE_CERT_FILE_PATH,             required"`
    CacheKeyFile               string `env:"CACHE_KEY_FILE_PATH,              required"`
    CacheCommonName            string `env:"CACHE_COMMON_NAME,                required"`
    CacheURL                   string `env:"CACHE_URL,                        required"`
    DrainSkipCertVerify        bool   `env:"DRAIN_SKIP_CERT_VERIFY,           required"`
    DrainTrustedCAFile         string `env:"DRAIN_TRUSTED_CA_FILE_PATH"`
    MetricsPort                uint16 `env:"METRICS_PORT"`
    MetricsCAFile              string `env:"METRICS_CA_FILE_PATH"`
    MetricsCertFile            string `env:"METRICS_CERT_FILE_PATH"`
    MetricsKeyFile             string `env:"METRICS_KEY_FILE_PATH"`
    MetricsCommonName          string `env:"METRICS_COMMON_NAME"`
    PProfPort                  uint16 `env:"PPROF_PORT"`
    IngressPort                uint16 `env:"INGRESS_PORT,                     required"`
    IngressCAFile              string `env:"INGRESS_CA_FILE_PATH,             required"`
    IngressCertFile            string `env:"INGRESS_CERT_FILE_PATH,           required"`
    IngressKeyFile             string `env:"INGRESS_KEY_FILE_PATH,            required"`
    IngressCommonName          string `env:"INGRESS_COMMON_NAME,              required"`
    CachePollingInterval       int    `env:"CACHE_POLLING_INTERVAL,           required"`
    IdleDrainTimeout           time.Duration
    OmitAppMetadata            bool   `env:"OMIT_APP_METADATA"`
    MaxDrains                  int `env:"MAX_DRAINS"`
}
```

Key configuration parameters include:
- **BINDING_CACHE_URL**: URL of the binding cache API
- **CACHE_URL**: URL of the syslog binding cache service
- **DRAIN_SKIP_CERT_VERIFY**: Whether to skip TLS certificate verification for drains
- **INGRESS_PORT**: Port to receive logs from Loggregator
- **CACHE_POLLING_INTERVAL**: How often to refresh the binding cache
- **MAX_DRAINS**: Maximum number of drains per app

## Operational Considerations

### Scaling

The Syslog Agent is designed to handle forwarding of logs to multiple syslog drains. Key scaling considerations include:

- **Connection Pooling**: The agent maintains connection pools to syslog endpoints
- **Concurrency Control**: The agent limits concurrent connections to avoid overwhelming targets
- **Rate Limiting**: The agent implements rate limiting to protect syslog endpoints
- **Maximum Drains**: The agent can be configured to limit the number of drains per application

### High Availability

The Syslog Agent ensures high availability through:

- **Stateless Operation**: The agent maintains minimal state
- **Automatic Recovery**: The agent automatically reconnects to endpoints after failures
- **Multiple Instances**: Multiple agent instances can be deployed for redundancy

### Security

Security considerations for the Syslog Agent include:

- **TLS for Syslog**: Support for secure syslog connections using TLS
- **Certificate Verification**: Option to verify syslog endpoint certificates
- **Mutual TLS**: Authentication for communication with binding cache and Loggregator
- **Isolated Communication**: Communication is restricted to authorized components

## Using Syslog Agent

### Configuring Applications to Use Syslog Drains

In Cloud Foundry, applications can be configured to use syslog drains through:

```bash
# Using the CF CLI to bind a syslog drain
cf bind-service my-app my-log-drain -c '{"syslog_drain_url":"syslog://syslog-drain.example.com:514"}'
```

Or in an application manifest:
```yaml
applications:
- name: my-app
  services:
  - name: my-log-drain
    parameters:
      syslog_drain_url: syslog://syslog-drain.example.com:514
```

### Supported Syslog Formats

The Syslog Agent supports multiple syslog formats:

1. **RFC5424** (default): Modern syslog format with structured data
   ```
   <14>1 2020-07-21T20:33:58.123456+00:00 my-app web.1 - [cloudfoundry@18060 app_id="12345" source_type="APP"] Application log message
   ```

2. **RFC3164**: Legacy syslog format
   ```
   <14>Jul 21 20:33:58 my-app web.1: Application log message
   ```

### Monitoring Syslog Agent

The Syslog Agent exposes metrics via its metrics endpoint:

```bash
# Query the metrics endpoint
curl -k https://localhost:metrics_port/metrics
```

Key metrics include:
- `syslog_agent.ingress`: Number of log envelopes received
- `syslog_agent.egress`: Number of log messages sent to drains
- `syslog_agent.dropped`: Number of log messages dropped
- `syslog_agent.binding_refresh_count`: Number of binding cache refreshes
- `syslog_agent.active_drains`: Number of active syslog drain connections

## Advanced Topics

### Custom TLS Configuration

The Syslog Agent allows for custom TLS configuration for syslog drains:

```yaml
# BOSH manifest excerpt
properties:
  syslog_agent:
    drain_trusted_ca_file: |
      -----BEGIN CERTIFICATE-----
      MIIEXzCCA0egAwIBAgIJAJVGTLmRnfTLMA0GCSqGSIb3DQEBCwUAMEIxCzAJBgNV
      ...
      -----END CERTIFICATE-----
```

### Log Filtering

The Syslog Agent can filter logs based on drain scope:

- **APP**: Only application logs
- **ALL**: All logs including system components

```go
// Example binding configuration
type Binding struct {
    AppID     string `json:"app_id"`
    Hostname  string `json:"hostname"`
    DrainURL  string `json:"drain_url"`
    DrainType string `json:"drain_type"` // "logs" or "metrics"
    DrainScope string `json:"drain_scope"` // "app" or "all"
}
```

### Retry Logic

The Syslog Agent implements sophisticated retry logic for failed syslog connections:

```go
// Simplified retry implementation
func (w *Writer) sendWithRetry(msg string) error {
    var err error
    
    for i := 0; i < maxRetries; i++ {
        err = w.send(msg)
        if err == nil {
            return nil
        }
        
        delay := calculateBackoff(i)
        time.Sleep(delay)
    }
    
    return fmt.Errorf("failed after %d retries: %s", maxRetries, err)
}

func calculateBackoff(retry int) time.Duration {
    // Exponential backoff with jitter
    delay := baseDelay * math.Pow(2, float64(retry))
    jitter := rand.Float64() * 0.5 * delay
    return time.Duration(delay + jitter)
}
```

## Troubleshooting

Common issues and their solutions:

1. **Binding Cache Connectivity**:
   - Check network connectivity to the binding cache
   - Verify TLS certificates
   - Ensure the binding cache service is running

2. **Syslog Drain Connectivity**:
   - Check network connectivity to syslog endpoints
   - Verify TLS configuration if using secure syslog
   - Check firewall rules

3. **Message Delivery Issues**:
   - Check syslog server logs for rejected messages
   - Verify message format compatibility
   - Check rate limiting on the syslog endpoint

## References

- [RFC5424 Syslog Protocol](https://tools.ietf.org/html/rfc5424)
- [RFC3164 BSD Syslog Protocol](https://tools.ietf.org/html/rfc3164)
- [Cloud Foundry Syslog Drain Documentation](https://docs.cloudfoundry.org/devguide/services/log-management.html) 