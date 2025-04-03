# Prometheus Scraper

This document provides a detailed analysis of the Prometheus Scraper component, its architecture, implementation details, and usage.

## Overview

The Prometheus Scraper is a component of the loggregator-agent-release that scrapes Prometheus metrics endpoints from Cloud Foundry components and applications, converts them into the Loggregator envelope format, and forwards them to the Loggregator system. This allows Cloud Foundry operators to leverage existing Prometheus instrumentation while using the Loggregator system for metrics collection and processing.

## Architecture

The Prometheus Scraper follows a polling-based architecture:

```
┌──────────────────────────────────────────────┐
│                Prometheus Scraper             │
│                                              │
│  ┌─────────┐    ┌─────────┐    ┌──────────┐  │
│  │ Config  │───▶│ Scraper │───▶│ Metrics  │  │
│  │ Parser  │    │ Manager │    │ Emitter  │  │
│  └─────────┘    └─────────┘    └──────────┘  │
│        │             │               │       │
└────────┼─────────────┼───────────────┼───────┘
         ▼             ▼               ▼
┌──────────────┐  ┌──────────┐   ┌────────────┐
│ Configuration│  │ Prometheus│   │ Loggregator│
│ Files/Env    │  │ Endpoints │   │ System     │
└──────────────┘  └──────────┘   └────────────┘
```

### Key Components

1. **Config Parser**:
   - Reads configuration from files and environment variables
   - Discovers target endpoints to scrape
   - Parses scrape intervals and configurations

2. **Scraper Manager**:
   - Manages scraping workers for each target
   - Handles scheduling of scrape operations
   - Implements parallel scraping with limits

3. **Metrics Emitter**:
   - Converts Prometheus metrics to Loggregator envelope format
   - Handles different metric types (counter, gauge, histogram)
   - Routes metrics to the Loggregator system

## Implementation Details

### Main Application

The Prometheus Scraper's main function initializes the application and its components:

```go
// Simplified from cmd/prom-scraper/main.go
func main() {
    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("failed to load config: %s", err)
    }
    
    // Set up metrics client
    metricClient := metrics.NewRegistry(...)
    
    // Set up the scraper manager
    scrapeManager := scraper.NewManager(
        cfg.DefaultScrapeInterval,
        metricClient,
    )
    
    // Set up the metrics converter
    metricsConverter := promql.NewConverter()
    
    // Set up the metrics emitter
    metricsEmitter := loggregator.NewClient(...)
    
    // Configure and start the scraper
    app := application.New(
        cfg,
        scrapeManager,
        metricsConverter,
        metricsEmitter,
        metricClient,
    )
    app.Start()
}
```

### Configuration

The Prometheus Scraper is configured via environment variables and configuration files:

```go
// Configuration structure
type Config struct {
    DefaultScrapeInterval    time.Duration `env:"SCRAPE_INTERVAL,              required"`
    MetricsExpirationInterval time.Duration `env:"METRICS_EXPIRATION_INTERVAL"`
    PathsToScrapeConfigs     []string      `env:"PATHS_TO_SCRAPE_CONFIGS"`
    MetricsServersPortsSlice []int         `env:"METRICS_SERVERS_PORTS,        required"`
    AdditionalScrapeConfigs  string        `env:"ADDITIONAL_SCRAPE_CONFIGS"`
    DefaultSourceID          string        `env:"DEFAULT_SOURCE_ID,            required"`
    LoggregatorIngressAddr   string        `env:"LOGGREGATOR_AGENT_ADDR,       required"`
    LoggregatorCAPath        string        `env:"LOGGREGATOR_AGENT_CA_FILE_PATH"`
    LoggregatorCertPath      string        `env:"LOGGREGATOR_AGENT_CERT_FILE_PATH"`
    LoggregatorKeyPath       string        `env:"LOGGREGATOR_AGENT_KEY_FILE_PATH"`
    MetricsPort              uint16        `env:"METRICS_PORT"`
    MetricsCAPath            string        `env:"METRICS_CA_FILE_PATH"`
    MetricsCertPath          string        `env:"METRICS_CERT_FILE_PATH"`
    MetricsKeyPath           string        `env:"METRICS_KEY_FILE_PATH"`
    DebugPort                uint16        `env:"DEBUG_PORT"`
    APIServer                bool          `env:"API_SERVER"`
    APIServerPort            uint16        `env:"API_SERVER_PORT"`
    APIServerCAPath          string        `env:"API_SERVER_CA_FILE_PATH"`
    APIServerCertPath        string        `env:"API_SERVER_CERT_FILE_PATH"`
    APIServerKeyPath         string        `env:"API_SERVER_KEY_FILE_PATH"`
    UseOtelCollector         bool          `env:"USE_OTEL_COLLECTOR"`
}
```

Key configuration parameters include:
- **SCRAPE_INTERVAL**: Default interval for scraping Prometheus endpoints
- **PATHS_TO_SCRAPE_CONFIGS**: Paths to files containing scrape configurations
- **DEFAULT_SOURCE_ID**: Default source ID for metrics
- **LOGGREGATOR_AGENT_ADDR**: Address of the Loggregator Agent

### Scrape Configuration

The scrape configuration defines which endpoints to scrape:

```yaml
# Example scrape configuration
scrape_configs:
  - job_name: node
    metrics_path: /metrics
    scheme: https
    source_id: node_exporter
    scrape_interval: 15s
    static_configs:
      - targets:
        - localhost:9100
    tls_config:
      ca_file: /path/to/ca.crt
      cert_file: /path/to/client.crt
      key_file: /path/to/client.key
      insecure_skip_verify: false
```

### Scraper Implementation

The Scraper Manager handles the scheduling and execution of scrape operations:

```go
// Simplified scraper manager implementation
type Manager struct {
    scrapeInterval   time.Duration
    scrapers         map[string]*Scraper
    workChannel      chan ScraperJob
    scrapesStarted   metrics.Counter
    scrapesCompleted metrics.Counter
    scrapesError     metrics.Counter
}

func (m *Manager) Start() {
    for i := 0; i < maxConcurrentScrapes; i++ {
        go m.worker()
    }
    
    for _, scraper := range m.scrapers {
        go m.scheduleScraper(scraper)
    }
}

func (m *Manager) scheduleScraper(s *Scraper) {
    ticker := time.NewTicker(s.interval)
    for range ticker.C {
        m.workChannel <- ScraperJob{Scraper: s}
    }
}

func (m *Manager) worker() {
    for job := range m.workChannel {
        m.scrapesStarted.Inc()
        err := job.Scraper.Scrape()
        if err != nil {
            m.scrapesError.Inc()
            continue
        }
        m.scrapesCompleted.Inc()
    }
}
```

### Metrics Conversion

The Prometheus Scraper converts Prometheus metrics to Loggregator envelopes:

```go
// Simplified metrics conversion
func (c *Converter) ToEnvelopes(metrics []PromMetric, sourceID string) []*loggregator_v2.Envelope {
    var envelopes []*loggregator_v2.Envelope
    
    for _, metric := range metrics {
        switch metric.Type {
        case COUNTER:
            envelopes = append(envelopes, c.counterToEnvelope(metric, sourceID))
        case GAUGE:
            envelopes = append(envelopes, c.gaugeToEnvelope(metric, sourceID))
        case HISTOGRAM:
            envelopes = append(envelopes, c.histogramToEnvelope(metric, sourceID))
        case SUMMARY:
            envelopes = append(envelopes, c.summaryToEnvelope(metric, sourceID))
        }
    }
    
    return envelopes
}

func (c *Converter) counterToEnvelope(metric PromMetric, sourceID string) *loggregator_v2.Envelope {
    return &loggregator_v2.Envelope{
        Timestamp:  time.Now().UnixNano(),
        SourceId:   sourceID,
        Message: &loggregator_v2.Envelope_Counter{
            Counter: &loggregator_v2.Counter{
                Name:  metric.Name,
                Total: uint64(metric.Value),
            },
        },
        Tags:       metric.Labels,
    }
}
```

## Operational Considerations

### Scaling

The Prometheus Scraper is designed to scrape multiple endpoints concurrently. Key scaling considerations include:

- **Scrape Interval**: Shorter intervals provide more frequent updates but increase load
- **Concurrency Limits**: The scraper limits concurrent scrapes to avoid overwhelming targets
- **Resource Allocation**: Higher concurrency requires more CPU and network resources

### High Availability

The Prometheus Scraper ensures high availability through:

- **Multiple Instances**: Multiple scrapers can be deployed for redundancy
- **Stateless Operation**: Each scraper is independent and stateless
- **Failure Isolation**: Issues with one target don't affect others

### Security

Security considerations for the Prometheus Scraper include:

- **TLS Support**: Secure communication with Prometheus endpoints
- **Mutual TLS**: Authentication for both scraper and endpoints
- **Certificate Verification**: Option to verify endpoint certificates
- **Access Control**: Restricted access to scrape targets

## Usage Examples

### Deploying with BOSH

The Prometheus Scraper is typically deployed using BOSH:

```yaml
# Simplified BOSH manifest excerpt
instance_groups:
- name: prom_scraper
  instances: 1
  jobs:
  - name: prom_scraper
    release: loggregator-agent
    properties:
      scrape_interval: 15s
      metrics_expiration_interval: 1m
      default_source_id: prometheus
      loggregator:
        tls:
          ca_cert: "((loggregator_ca.certificate))"
          cert: "((loggregator_tls_client.certificate))"
          key: "((loggregator_tls_client.private_key))"
      scrape_configs:
        - job_name: node
          metrics_path: /metrics
          scheme: https
          static_configs:
            - targets:
              - localhost:9100
```

### Configuring Custom Scrape Targets

Custom scrape targets can be configured through configuration files:

```yaml
# /var/vcap/jobs/prom_scraper/config/config.yml
scrape_configs:
  - job_name: custom_app
    metrics_path: /custom/metrics
    source_id: custom_app
    scrape_interval: 30s
    static_configs:
      - targets:
        - app.example.com:8080
    tls_config:
      insecure_skip_verify: true
```

Or through environment variables:

```bash
export ADDITIONAL_SCRAPE_CONFIGS='
scrape_configs:
  - job_name: custom_app
    metrics_path: /custom/metrics
    source_id: custom_app
    static_configs:
      - targets:
        - app.example.com:8080
'
```

### API Server

The Prometheus Scraper can expose an API server for dynamic configuration:

```bash
# Enable API server
export API_SERVER=true
export API_SERVER_PORT=8081

# Configure a new scrape target
curl -X POST http://localhost:8081/v1/scrape_targets \
  -H "Content-Type: application/json" \
  -d '{
    "targets": ["app.example.com:8080"],
    "source_id": "dynamic_app",
    "instance_id": "0",
    "scrape_interval": "30s"
  }'
```

## Advanced Topics

### Metric Types

The Prometheus Scraper supports converting various Prometheus metric types:

1. **Counter**: Monotonically increasing counter
   ```go
   // Counter conversion
   &loggregator_v2.Envelope{
       Message: &loggregator_v2.Envelope_Counter{
           Counter: &loggregator_v2.Counter{
               Name:  "http_requests_total",
               Total: 12345,
           },
       },
   }
   ```

2. **Gauge**: Value that can go up and down
   ```go
   // Gauge conversion
   &loggregator_v2.Envelope{
       Message: &loggregator_v2.Envelope_Gauge{
           Gauge: &loggregator_v2.Gauge{
               Metrics: map[string]*loggregator_v2.GaugeValue{
                   "cpu_usage": {
                       Unit:  "percentage",
                       Value: 45.2,
                   },
               },
           },
       },
   }
   ```

3. **Histogram**: Observations bucketed by value ranges
   ```go
   // Histogram conversion
   // Converted to multiple gauge metrics
   &loggregator_v2.Envelope{
       Message: &loggregator_v2.Envelope_Gauge{
           Gauge: &loggregator_v2.Gauge{
               Metrics: map[string]*loggregator_v2.GaugeValue{
                   "http_request_duration_bucket_0.1": {Value: 12},
                   "http_request_duration_bucket_0.5": {Value: 34},
                   "http_request_duration_bucket_1.0": {Value: 56},
                   "http_request_duration_sum": {Value: 123.45},
                   "http_request_duration_count": {Value: 78},
               },
           },
       },
   }
   ```

### OpenTelemetry Integration

The Prometheus Scraper can also forward metrics to an OpenTelemetry Collector:

```go
// Simplified OpenTelemetry integration
func NewOtelMetricsEmitter(config *Config) (*OtelEmitter, error) {
    // Set up OpenTelemetry exporter
    exporter, err := otlpmetricgrpc.New(
        context.Background(),
        otlpmetricgrpc.WithEndpoint(config.OtelCollectorAddr),
        otlpmetricgrpc.WithTLSCredentials(...),
    )
    
    // Set up meter provider
    meterProvider := sdkmetric.NewMeterProvider(
        sdkmetric.WithReader(
            sdkmetric.NewPeriodicReader(exporter),
        ),
    )
    
    return &OtelEmitter{
        meter: meterProvider.Meter("prom-scraper"),
    }, nil
}
```

### Metrics Filtering

The Prometheus Scraper supports filtering metrics:

```yaml
# Metric relabeling configuration
scrape_configs:
  - job_name: node
    metric_relabel_configs:
      - source_labels: [__name__]
        regex: "go_.*"
        action: drop
      - source_labels: [__name__]
        regex: "process_.*"
        action: keep
```

## Troubleshooting

Common issues and their solutions:

1. **Connection Issues**:
   - Check network connectivity to Prometheus endpoints
   - Verify TLS certificates and configuration
   - Check for firewall rules blocking access

2. **Scrape Failures**:
   - Check Prometheus endpoint health
   - Verify authentication if required
   - Check for rate limiting on the target

3. **Metric Processing Issues**:
   - Check for unsupported metric types
   - Verify metric naming conventions
   - Check for duplicated metric names

## Monitoring

The Prometheus Scraper exposes its own metrics for monitoring:

```bash
# Query the metrics endpoint
curl -k https://localhost:metrics_port/metrics
```

Key metrics include:
- `prom_scraper.scrapes_started`: Number of scrape operations started
- `prom_scraper.scrapes_completed`: Number of scrape operations completed
- `prom_scraper.scrapes_error`: Number of scrape operations that failed
- `prom_scraper.scrape_duration_seconds`: Duration of scrape operations
- `prom_scraper.metrics_emitted`: Number of metrics emitted to Loggregator

## References

- [Prometheus Exposition Format](https://prometheus.io/docs/instrumenting/exposition_formats/)
- [Loggregator V2 API](https://github.com/cloudfoundry/loggregator-api/blob/master/v2/envelope.proto)
- [OpenTelemetry Metrics](https://opentelemetry.io/docs/concepts/signals/metrics/) 