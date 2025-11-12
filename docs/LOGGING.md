# Logging Configuration

FSAgent sử dụng structured logger với các log levels để dễ dàng quản lý logs trong môi trường development và production.

## Log Levels

- **DEBUG**: Thông tin chi tiết cho debugging (correlation IDs, media info, event processing details)
- **INFO**: Thông tin quan trọng về hoạt động của hệ thống (connections, RTCP metrics, channel lifecycle)
- **WARN**: Cảnh báo về các vấn đề không nghiêm trọng (connection failures, retry attempts)
- **ERROR**: Lỗi nghiêm trọng cần xử lý (calculation errors, processing failures)

## Configuration

### Via Config File (config.yaml)

```yaml
logging:
  level: info    # debug, info, warn, error
  format: text   # text or json (reserved for future use)
```

### Via Environment Variable

```bash
export FSAGENT_LOG_LEVEL=debug
./fsagent
```

## Production Recommendations

Trong môi trường production, nên set log level là `info` hoặc `warn` để:
- Giảm volume của logs
- Tránh log ra thông tin nhạy cảm (correlation IDs, channel details)
- Cải thiện performance

```yaml
# Production config
logging:
  level: info
```

## Development Recommendations

Trong môi trường development, set log level là `debug` để xem chi tiết:

```yaml
# Development config
logging:
  level: debug
```

## Log Examples

### DEBUG Level
```
[DEBUG] Processing event: CHANNEL_CREATE from instance: fs1
[DEBUG] 🔗 Correlation ID from variable_sip_call_id: abc123@192.168.1.1
[DEBUG] ✅ Created channel state: channel_id=xyz, correlation_id=abc123, domain=example.com
[DEBUG] 📡 Updated media info: channel_id=xyz, correlation_id=abc123, local=10.0.0.1:16384, remote=10.0.0.2:20000
```

### INFO Level
```
[INFO] FSAgent starting with log level: info
[INFO] State store initialized successfully
[INFO] Successfully connected to FreeSWITCH instance: fs1 at 192.168.13.137:8021
[INFO] 📊 RTCP metrics: channel_id=xyz, correlation_id=abc123, domain=example.com, direction=inbound, jitter=2.50ms, packets_lost=5
[INFO] 🔚 Channel destroyed: channel_id=xyz, correlation_id=abc123, instance=fs1
```

### WARN Level
```
[WARN] Initial connection failed for instance fs1: connection refused
[WARN] Keepalive failed for instance fs1: timeout
[WARN] Event channel full for instance fs1, dropping event
```

### ERROR Level
```
[ERROR] Error calculating RTCP metrics: channel state not found
[ERROR] Error processing event from instance fs1: invalid event format
[ERROR] No FreeSWITCH connections established
```

## Runtime Log Level Change

Hiện tại log level được set khi khởi động application. Để thay đổi log level, cần restart application với config mới.

Future enhancement: Có thể thêm HTTP endpoint để thay đổi log level runtime mà không cần restart.
