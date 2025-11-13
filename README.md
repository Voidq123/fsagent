# FSAgent - FreeSWITCH Metrics Collection Agent

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

FSAgent is a high-performance Golang application that connects to multiple FreeSWITCH instances via ESL (Event Socket Library), collects real-time RTCP and QoS metrics, and exports them to OpenTelemetry Collector. It provides comprehensive call quality monitoring with support for multi-dimensional metrics including per-leg, per-call, and per-domain analysis.

## Features

- **Multi-Instance Support**: Connect to multiple FreeSWITCH servers simultaneously
- **Real-Time RTCP Metrics**: Monitor jitter, packet loss, and fraction lost during active calls
- **QoS Summary Metrics**: Collect comprehensive call quality metrics (MOS, jitter, packet loss) at call termination
- **Flexible Storage**: Choose between in-memory (fast) or Redis (persistent) state storage
- **OpenTelemetry Export**: Native OTLP/gRPC export to OpenTelemetry Collector
- **Multi-Dimensional Labels**: Track metrics by channel_id (per-leg), correlation_id (per-call), and domain_name (per-tenant)
- **Auto-Reconnection**: Exponential backoff reconnection with keepalive mechanism
- **Health Endpoints**: Built-in health checks and Prometheus-format internal metrics
- **Structured Logging**: JSON or text format with configurable log levels

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         FSAgent                              │
│                                                              │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐           │
│  │ FS Conn 1  │  │ FS Conn 2  │  │ FS Conn N  │           │
│  │  Manager   │  │  Manager   │  │  Manager   │           │
│  └─────┬──────┘  └─────┬──────┘  └─────┬──────┘           │
│        │                │                │                   │
│        └────────────────┴────────────────┘                   │
│                         │                                    │
│                  ┌──────▼──────┐                            │
│                  │    Event     │                            │
│                  │  Processor   │                            │
│                  └──────┬───────┘                            │
│                         │                                    │
│         ┌───────────────┼───────────────┐                   │
│         │               │               │                   │
│    ┌────▼────┐   ┌─────▼─────┐   ┌────▼────┐              │
│    │  RTCP   │   │    QoS    │   │  State  │              │
│    │Calculator│   │Calculator │   │  Store  │              │
│    └────┬────┘   └─────┬─────┘   └────┬────┘              │
│         │               │               │                   │
│         └───────────────┴───────────────┘                   │
│                         │                                    │
│                  ┌──────▼──────┐                            │
│                  │  Metrics     │                            │
│                  │  Exporter    │                            │
│                  └──────┬───────┘                            │
└─────────────────────────┼────────────────────────────────────┘
                          │
                          ▼
                  ┌──────────────┐
                  │ OpenTelemetry│
                  │  Collector   │
                  └──────────────┘
```

For detailed architecture diagrams including sequence diagrams, data flow, and deployment architectures, see [ARCHITECTURE.md](ARCHITECTURE.md).

## Quick Start

### Prerequisites

- Go 1.21 or higher
- FreeSWITCH with ESL enabled
- OpenTelemetry Collector (optional, for metrics export)
- Redis (optional, for persistent state storage)

### Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/fsagent.git
cd fsagent

# Install dependencies
go mod download

# Build the application
go build -o fsagent ./cmd/fsagent

# Or install directly
go install ./cmd/fsagent
```

### Configuration

1. Copy the example configuration:
```bash
cp config.example.yaml config.yaml
```

2. Edit `config.yaml` with your FreeSWITCH connection details:
```yaml
freeswitch_instances:
  - name: fs1
    host: 192.168.1.10
    port: 8021
    password: ClueCon

storage:
  type: memory

opentelemetry:
  endpoint: localhost:4317
  insecure: true

http:
  port: 8080

logging:
  level: info
  format: json
```

3. Run FSAgent:
```bash
./fsagent
```

### Using Environment Variables

Instead of a config file, you can use environment variables:

```bash
export FSAGENT_FREESWITCH_INSTANCES='[{"name":"fs1","host":"192.168.1.10","port":8021,"password":"ClueCon"}]'
export FSAGENT_STORAGE_TYPE=memory
export FSAGENT_OTEL_ENDPOINT=localhost:4317
export FSAGENT_HTTP_PORT=8080
export FSAGENT_LOG_LEVEL=info

./fsagent
```

## Configuration Options

### FreeSWITCH Instances

Configure one or more FreeSWITCH instances to monitor:

```yaml
freeswitch_instances:
  - name: fs1              # Unique identifier (used in metrics labels)
    host: 192.168.1.10     # FreeSWITCH ESL host
    port: 8021             # ESL port (default: 8021)
    password: ClueCon      # ESL password
```

### Storage Backend

Choose between in-memory or Redis storage:

**In-Memory (Default)**
```yaml
storage:
  type: memory
```
- Fast, no external dependencies
- Data lost on restart
- Suitable for single-instance deployments

**Redis (Recommended for Production)**
```yaml
storage:
  type: redis
  redis:
    host: localhost
    port: 6379
    password: ""
    db: 0
```
- Persistent across restarts
- Supports multiple FSAgent instances
- 24-hour TTL for automatic cleanup

### OpenTelemetry Export

Configure OTLP/gRPC endpoint:

```yaml
opentelemetry:
  endpoint: localhost:4317
  insecure: true
  headers:                    # Optional authentication
    x-api-key: "your-key"
```

### HTTP Server

Expose health and metrics endpoints:

```yaml
http:
  port: 8080
```

Available endpoints:
- `GET /health` - Connection status for all FreeSWITCH instances
- `GET /ready` - Readiness probe (200 if at least one FS connected)
- `GET /metrics` - Prometheus-format internal metrics

### Logging

Configure structured logging:

```yaml
logging:
  level: info    # debug, info, warn, error
  format: json   # json, text
```

### Event Processing

Control which events to process:

```yaml
events:
  rtcp: true     # Real-time RTCP metrics during calls
  qos: true      # Summary metrics at call end
```

Performance tuning:
- `rtcp: false, qos: true` - Reduce event load by ~80%, only end-of-call summaries
- `rtcp: true, qos: false` - Real-time monitoring only
- Both enabled (default) - Full monitoring

## Metrics and Labels

### RTCP Metrics (Real-Time)

Exported during active calls (every ~5 seconds):

| Metric | Type | Unit | Description |
|--------|------|------|-------------|
| `freeswitch.rtcp.jitter` | Gauge | ms | Inter-arrival jitter |
| `freeswitch.rtcp.packets_lost` | Gauge | count | Incremental packet loss |
| `freeswitch.rtcp.fraction_lost` | Gauge | fraction | Fraction of packets lost |

**Labels**: `fs_instance`, `channel_id`, `correlation_id`, `domain_name`, `direction` (inbound/outbound)

### QoS Metrics (End-of-Call)

Exported when calls terminate:

| Metric | Type | Unit | Description |
|--------|------|------|-------------|
| `freeswitch.qos.mos_score` | Gauge | score | Mean Opinion Score (1-5) |
| `freeswitch.qos.jitter_avg` | Gauge | ms | Average jitter |
| `freeswitch.qos.jitter_min` | Gauge | ms | Minimum jitter |
| `freeswitch.qos.jitter_max` | Gauge | ms | Maximum jitter |
| `freeswitch.qos.total_packets` | Gauge | count | Total packets (in + out) |
| `freeswitch.qos.packet_loss` | Gauge | count | Total packet loss |
| `freeswitch.qos.total_bytes` | Gauge | bytes | Total bytes transferred |

**Labels**: `fs_instance`, `channel_id`, `correlation_id`, `domain_name`, `codec_name`, `src_ip`, `dst_ip`

### Internal Metrics (Prometheus Format)

Available at `/metrics` endpoint:

- `fsagent_events_received_total` - Total events received from FreeSWITCH
- `fsagent_events_processed_total` - Total events processed successfully
- `fsagent_rtcp_messages_processed_total` - RTCP messages processed
- `fsagent_qos_messages_generated_total` - QoS metrics generated
- `fsagent_storage_operations_total` - State store operations
- `fsagent_fs_connections` - Current FreeSWITCH connection status

### Label Dimensions

FSAgent provides multi-dimensional metrics for flexible analysis:

- **channel_id** (Unique-ID): Track individual call legs (A-leg, B-leg)
- **correlation_id** (SIP Call-ID): Aggregate metrics for entire calls
- **domain_name**: Filter and group by SIP domain/tenant
- **fs_instance**: Identify source FreeSWITCH instance
- **direction**: Distinguish inbound vs outbound metrics

Example queries:
```promql
# Average MOS score per domain
avg(freeswitch_qos_mos_score) by (domain_name)

# Packet loss rate per FreeSWITCH instance
rate(freeswitch_rtcp_packets_lost[5m]) by (fs_instance)

# Jitter for specific call
freeswitch_rtcp_jitter{correlation_id="abc123@example.com"}
```

## Usage Examples

### Single FreeSWITCH Instance

```yaml
freeswitch_instances:
  - name: fs1
    host: localhost
    port: 8021
    password: ClueCon

storage:
  type: memory

opentelemetry:
  endpoint: localhost:4317
  insecure: true
```

### Multiple FreeSWITCH Instances with Redis

```yaml
freeswitch_instances:
  - name: fs-prod-01
    host: 10.0.1.10
    port: 8021
    password: SecurePass1
  - name: fs-prod-02
    host: 10.0.1.11
    port: 8021
    password: SecurePass2

storage:
  type: redis
  redis:
    host: redis.internal
    port: 6379
    password: RedisPass
    db: 0

opentelemetry:
  endpoint: otel-collector.internal:4317
  insecure: false
  headers:
    x-api-key: "prod-api-key"
```

### High-Volume Environment (Optimized)

```yaml
freeswitch_instances:
  - name: fs-hv-01
    host: 10.0.2.10
    port: 8021
    password: HVPass

storage:
  type: redis
  redis:
    host: redis-cache.internal
    port: 6379

opentelemetry:
  endpoint: otel-collector.internal:4317
  insecure: false

logging:
  level: warn
  format: json

events:
  rtcp: false  # Disable real-time RTCP to reduce load by 80%
  qos: true    # Keep end-of-call summaries
```

## Docker Deployment

### Build Docker Image

```bash
docker build -t fsagent:latest .
```

### Run with Docker

```bash
docker run -d \
  --name fsagent \
  -p 8080:8080 \
  -v $(pwd)/config.yaml:/app/config.yaml \
  fsagent:latest
```

### Docker Compose

```yaml
version: '3.8'

services:
  fsagent:
    image: fsagent:latest
    ports:
      - "8080:8080"
    environment:
      - FSAGENT_FREESWITCH_INSTANCES=[{"name":"fs1","host":"freeswitch","port":8021,"password":"ClueCon"}]
      - FSAGENT_STORAGE_TYPE=redis
      - FSAGENT_REDIS_HOST=redis
      - FSAGENT_OTEL_ENDPOINT=otel-collector:4317
    depends_on:
      - redis
      - otel-collector
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

  otel-collector:
    image: otel/opentelemetry-collector:latest
    ports:
      - "4317:4317"
    volumes:
      - ./otel-config.yaml:/etc/otel-config.yaml
    command: ["--config=/etc/otel-config.yaml"]
```

## Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: fsagent
spec:
  replicas: 2
  selector:
    matchLabels:
      app: fsagent
  template:
    metadata:
      labels:
        app: fsagent
    spec:
      containers:
      - name: fsagent
        image: fsagent:latest
        ports:
        - containerPort: 8080
        env:
        - name: FSAGENT_STORAGE_TYPE
          value: "redis"
        - name: FSAGENT_REDIS_HOST
          value: "redis-service"
        - name: FSAGENT_OTEL_ENDPOINT
          value: "otel-collector:4317"
        envFrom:
        - configMapRef:
            name: fsagent-config
        - secretRef:
            name: fsagent-secrets
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: fsagent
spec:
  selector:
    app: fsagent
  ports:
  - port: 8080
    targetPort: 8080
```

## Monitoring and Observability

### Health Check

```bash
curl http://localhost:8080/health
```

Response:
```json
{
  "status": "healthy",
  "connections": {
    "fs1": {
      "connected": true,
      "last_event": "2024-01-15T10:30:45Z"
    },
    "fs2": {
      "connected": false,
      "last_error": "connection refused"
    }
  }
}
```

### Readiness Probe

```bash
curl http://localhost:8080/ready
```

Returns 200 if at least one FreeSWITCH connection is active, 503 otherwise.

### Internal Metrics

```bash
curl http://localhost:8080/metrics
```

Returns Prometheus-format metrics for FSAgent itself.

## Troubleshooting

### Connection Issues

**Problem**: FSAgent cannot connect to FreeSWITCH

**Solutions**:
1. Verify ESL is enabled in FreeSWITCH:
   ```bash
   fs_cli -x "event_socket status"
   ```

2. Check ESL password in FreeSWITCH config:
   ```xml
   <!-- /etc/freeswitch/autoload_configs/event_socket.conf.xml -->
   <param name="password" value="ClueCon"/>
   ```

3. Verify network connectivity:
   ```bash
   telnet <freeswitch-host> 8021
   ```

4. Check FSAgent logs:
   ```bash
   ./fsagent 2>&1 | grep -i "connection"
   ```

### No Metrics Exported

**Problem**: Metrics not appearing in OpenTelemetry Collector

**Solutions**:
1. Verify OTel Collector is running:
   ```bash
   curl http://localhost:13133/  # Health check endpoint
   ```

2. Check FSAgent can reach OTel endpoint:
   ```bash
   telnet <otel-host> 4317
   ```

3. Enable debug logging:
   ```yaml
   logging:
     level: debug
     format: text
   ```

4. Check for export errors in logs:
   ```bash
   ./fsagent 2>&1 | grep -i "export"
   ```

### High Memory Usage

**Problem**: FSAgent consuming too much memory

**Solutions**:
1. Use Redis instead of in-memory storage:
   ```yaml
   storage:
     type: redis
   ```

2. Disable RTCP events if not needed:
   ```yaml
   events:
     rtcp: false
     qos: true
   ```

3. Monitor channel state count:
   ```bash
   curl http://localhost:8080/metrics | grep storage_operations
   ```

### Missing Domain Names

**Problem**: `domain_name` label is empty in metrics

**Solutions**:
1. Ensure FreeSWITCH is setting domain variables:
   ```xml
   <!-- In dialplan -->
   <action application="set" data="domain_name=${domain_name}"/>
   ```

2. Check SIP headers are present:
   ```bash
   fs_cli -x "uuid_dump <uuid>" | grep -i domain
   ```

3. FSAgent falls back to these headers in order:
   - `variable_domain_name`
   - `variable_sip_from_host`
   - `variable_sip_to_host`

### Reconnection Loops

**Problem**: FSAgent constantly reconnecting to FreeSWITCH

**Solutions**:
1. Check FreeSWITCH logs for ESL errors:
   ```bash
   tail -f /var/log/freeswitch/freeswitch.log | grep -i "event socket"
   ```

2. Verify password is correct in config

3. Check for network issues or firewalls

4. Increase keepalive interval if needed (requires code change)

## Performance Tuning

### Event Load Optimization

For high-volume environments (1000+ concurrent calls):

1. **Disable RTCP events** (reduces load by ~80%):
   ```yaml
   events:
     rtcp: false
     qos: true
   ```

2. **Use Redis for state storage** (better memory management):
   ```yaml
   storage:
     type: redis
   ```

3. **Reduce log level**:
   ```yaml
   logging:
     level: warn
   ```

### Scaling

- **Horizontal**: Deploy multiple FSAgent instances with Redis storage
- **Vertical**: Increase CPU/memory for single instance
- **Per-Instance**: One FSAgent per FreeSWITCH for isolation

## Development

### Project Structure

```
fsagent/
├── cmd/
│   └── fsagent/          # Main application entry point
│       └── main.go
├── pkg/                  # Public libraries and interfaces
│   ├── calculator/       # RTCP and QoS metric calculators
│   │   ├── rtcp.go
│   │   └── qos.go
│   ├── config/           # Configuration structures
│   │   └── config.go
│   ├── connection/       # FreeSWITCH ESL connection management
│   │   └── manager.go
│   ├── exporter/         # OpenTelemetry metrics exporter
│   │   └── metrics_exporter.go
│   ├── logger/           # Structured logging
│   │   └── logger.go
│   ├── metrics/          # Internal metrics
│   │   └── internal_metrics.go
│   ├── processor/        # Event processing and routing
│   │   └── event_processor.go
│   ├── server/           # HTTP server
│   │   └── http_server.go
│   └── store/            # State storage (in-memory and Redis)
│       └── state_store.go
├── config.example.yaml   # Example configuration
├── Dockerfile            # Docker build configuration
├── docker-compose.yml    # Docker Compose setup
├── go.mod
├── go.sum
└── README.md
```

### Building from Source

```bash
# Clone repository
git clone [repository]
cd fsagent

# Install dependencies
go mod download

# Run tests
go test ./...

# Build
go build -o fsagent ./cmd/fsagent

# Run
./fsagent
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./pkg/calculator/...

# Verbose output
go test -v ./...
```

## Contributing

Contributions are welcome! Please follow these guidelines:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- [FreeSWITCH](https://freeswitch.org/) - Open-source telephony platform
- [OpenTelemetry](https://opentelemetry.io/) - Observability framework