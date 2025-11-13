package calculator

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/luongdev/fsagent/pkg/connection"
	"github.com/luongdev/fsagent/pkg/logger"
	"github.com/luongdev/fsagent/pkg/store"
)

// QoSCalculator processes CHANNEL_DESTROY events and calculates QoS metrics
type QoSCalculator interface {
	// CalculateMetrics processes CHANNEL_DESTROY event and returns QoS metrics
	CalculateMetrics(ctx context.Context, event *connection.FSEvent, instanceName string) (*QoSMetrics, error)
}

// QoSMetrics represents calculated QoS metrics
type QoSMetrics struct {
	Timestamp     time.Time
	InstanceName  string
	ChannelID     string // Unique-ID for per-leg monitoring
	CorrelationID string // SIP Call-ID for per-call aggregation
	DomainName    string // SIP domain for filtering by tenant/domain

	// Quality Metrics
	MOSScore  float64
	AvgJitter float64 // (min + max) / 2
	MinJitter float64
	MaxJitter float64
	Delta     float64 // mean interval

	// Traffic Metrics
	TotalPackets int64 // in + out
	PacketLoss   int64 // in + out skip packets
	TotalBytes   int64 // in + out

	// Codec Information
	CodecName string
	CodecPT   int
	PTime     int
	ClockRate int

	// Endpoints
	SrcIP   string
	SrcPort uint16
	DstIP   string
	DstPort uint16

	// Timing
	ReportTimestamp int64
}

// qosCalculator implements QoSCalculator interface
type qosCalculator struct {
	store store.StateStore
}

// NewQoSCalculator creates a new QoS calculator
func NewQoSCalculator(store store.StateStore) QoSCalculator {
	return &qosCalculator{
		store: store,
	}
}

// CalculateMetrics processes CHANNEL_DESTROY event and returns QoS metrics
func (qc *qosCalculator) CalculateMetrics(ctx context.Context, event *connection.FSEvent, instanceName string) (*QoSMetrics, error) {
	// Check for variable_rtp_use_codec_rate presence - only process if exists
	if event.GetHeader("variable_rtp_use_codec_rate") == "" {
		return nil, fmt.Errorf("variable_rtp_use_codec_rate not present, skipping QoS calculation")
	}

	channelID := event.GetHeader("Unique-ID")
	if channelID == "" {
		return nil, fmt.Errorf("CHANNEL_DESTROY event missing Unique-ID")
	}

	// Initialize metrics
	metrics := &QoSMetrics{
		Timestamp:    time.Now(),
		InstanceName: instanceName,
		ChannelID:    channelID,
	}

	// Extract MOS score
	if err := qc.extractQualityMetrics(event, metrics); err != nil {
		logger.ErrorWithFields(map[string]interface{}{
			"channel_id":  channelID,
			"fs_instance": instanceName,
			"error":       err.Error(),
		}, "Failed to extract quality metrics")
		return nil, fmt.Errorf("failed to extract quality metrics: %w", err)
	}

	// Extract traffic metrics
	if err := qc.extractTrafficMetrics(event, metrics); err != nil {
		logger.ErrorWithFields(map[string]interface{}{
			"channel_id":  channelID,
			"fs_instance": instanceName,
			"error":       err.Error(),
		}, "Failed to extract traffic metrics")
		return nil, fmt.Errorf("failed to extract traffic metrics: %w", err)
	}

	// Extract codec information
	if err := qc.extractCodecInfo(event, metrics); err != nil {
		logger.ErrorWithFields(map[string]interface{}{
			"channel_id":  channelID,
			"fs_instance": instanceName,
			"error":       err.Error(),
		}, "Failed to extract codec info")
		return nil, fmt.Errorf("failed to extract codec info: %w", err)
	}

	// Retrieve correlation_id and domain_name from state, or extract from event
	if err := qc.extractStateAndDomain(ctx, event, metrics); err != nil {
		logger.ErrorWithFields(map[string]interface{}{
			"channel_id":  channelID,
			"fs_instance": instanceName,
			"error":       err.Error(),
		}, "Failed to extract state and domain")
		return nil, fmt.Errorf("failed to extract state and domain: %w", err)
	}

	logger.DebugWithFields(map[string]interface{}{
		"channel_id":     channelID,
		"correlation_id": metrics.CorrelationID,
		"fs_instance":    instanceName,
		"mos_score":      metrics.MOSScore,
		"avg_jitter_ms":  metrics.AvgJitter,
		"min_jitter_ms":  metrics.MinJitter,
		"max_jitter_ms":  metrics.MaxJitter,
		"delta_ms":       metrics.Delta,
		"packet_loss":    metrics.PacketLoss,
		"total_packets":  metrics.TotalPackets,
		"codec_name":     metrics.CodecName,
	}, "QoS metrics calculated successfully")

	return metrics, nil
}

// extractQualityMetrics extracts MOS score and jitter metrics
func (qc *qosCalculator) extractQualityMetrics(event *connection.FSEvent, metrics *QoSMetrics) error {
	// Extract MOS score from variable_rtp_audio_in_mos
	if mosStr := event.GetHeader("variable_rtp_audio_in_mos"); mosStr != "" {
		if mos, err := strconv.ParseFloat(mosStr, 64); err == nil {
			metrics.MOSScore = mos
		}
	}

	// Extract min jitter variance
	if minJitterStr := event.GetHeader("variable_rtp_audio_in_jitter_min_variance"); minJitterStr != "" {
		if minJitter, err := strconv.ParseFloat(minJitterStr, 64); err == nil {
			metrics.MinJitter = minJitter
		}
	}

	// Extract max jitter variance
	if maxJitterStr := event.GetHeader("variable_rtp_audio_in_jitter_max_variance"); maxJitterStr != "" {
		if maxJitter, err := strconv.ParseFloat(maxJitterStr, 64); err == nil {
			metrics.MaxJitter = maxJitter
		}
	}

	// Calculate average jitter: (min + max) / 2
	if metrics.MinJitter > 0 || metrics.MaxJitter > 0 {
		metrics.AvgJitter = (metrics.MinJitter + metrics.MaxJitter) / 2.0
	}

	// Extract delta (mean interval)
	if deltaStr := event.GetHeader("variable_rtp_audio_in_mean_interval"); deltaStr != "" {
		if delta, err := strconv.ParseFloat(deltaStr, 64); err == nil {
			metrics.Delta = delta
		}
	}

	return nil
}

// extractTrafficMetrics sums inbound and outbound traffic metrics
func (qc *qosCalculator) extractTrafficMetrics(event *connection.FSEvent, metrics *QoSMetrics) error {
	var inboundPackets, outboundPackets int64
	var inboundBytes, outboundBytes int64
	var inboundSkip, outboundSkip int64

	// Extract inbound packet count
	if inPacketsStr := event.GetHeader("variable_rtp_audio_in_packet_count"); inPacketsStr != "" {
		if packets, err := strconv.ParseInt(inPacketsStr, 10, 64); err == nil {
			inboundPackets = packets
		}
	}

	// Extract outbound packet count
	if outPacketsStr := event.GetHeader("variable_rtp_audio_out_packet_count"); outPacketsStr != "" {
		if packets, err := strconv.ParseInt(outPacketsStr, 10, 64); err == nil {
			outboundPackets = packets
		}
	}

	// Sum total packets
	metrics.TotalPackets = inboundPackets + outboundPackets

	// Extract inbound byte count (media bytes)
	if inBytesStr := event.GetHeader("variable_rtp_audio_in_media_bytes"); inBytesStr != "" {
		if bytes, err := strconv.ParseInt(inBytesStr, 10, 64); err == nil {
			inboundBytes = bytes
		}
	}

	// Extract outbound byte count (media bytes)
	if outBytesStr := event.GetHeader("variable_rtp_audio_out_media_bytes"); outBytesStr != "" {
		if bytes, err := strconv.ParseInt(outBytesStr, 10, 64); err == nil {
			outboundBytes = bytes
		}
	}

	// Sum total bytes
	metrics.TotalBytes = inboundBytes + outboundBytes

	// Extract inbound skip packet count (packet loss)
	if inSkipStr := event.GetHeader("variable_rtp_audio_in_skip_packet_count"); inSkipStr != "" {
		if skip, err := strconv.ParseInt(inSkipStr, 10, 64); err == nil {
			inboundSkip = skip
		}
	}

	// Extract outbound skip packet count (packet loss)
	if outSkipStr := event.GetHeader("variable_rtp_audio_out_skip_packet_count"); outSkipStr != "" {
		if skip, err := strconv.ParseInt(outSkipStr, 10, 64); err == nil {
			outboundSkip = skip
		}
	}

	// Sum total packet loss
	metrics.PacketLoss = inboundSkip + outboundSkip

	return nil
}

// extractCodecInfo extracts codec information and media endpoints
func (qc *qosCalculator) extractCodecInfo(event *connection.FSEvent, metrics *QoSMetrics) error {
	// Extract codec name
	if codecName := event.GetHeader("variable_rtp_use_codec_name"); codecName != "" {
		metrics.CodecName = codecName
	}

	// Extract codec payload type
	if codecPTStr := event.GetHeader("variable_rtp_use_codec_pt"); codecPTStr != "" {
		if pt, err := strconv.Atoi(codecPTStr); err == nil {
			metrics.CodecPT = pt
		}
	}

	// Extract ptime (packetization time)
	if ptimeStr := event.GetHeader("variable_rtp_use_codec_ptime"); ptimeStr != "" {
		if ptime, err := strconv.Atoi(ptimeStr); err == nil {
			metrics.PTime = ptime
		}
	}

	// Extract clock rate
	if clockRateStr := event.GetHeader("variable_rtp_use_codec_rate"); clockRateStr != "" {
		if rate, err := strconv.Atoi(clockRateStr); err == nil {
			metrics.ClockRate = rate
		}
	}

	// Extract local media IP and port
	if localIP := event.GetHeader("variable_local_media_ip"); localIP != "" {
		metrics.SrcIP = localIP
	}

	if localPortStr := event.GetHeader("variable_local_media_port"); localPortStr != "" {
		if port, err := strconv.ParseUint(localPortStr, 10, 16); err == nil {
			metrics.SrcPort = uint16(port)
		}
	}

	// Extract remote media IP and port
	if remoteIP := event.GetHeader("variable_remote_media_ip"); remoteIP != "" {
		metrics.DstIP = remoteIP
	}

	if remotePortStr := event.GetHeader("variable_remote_media_port"); remotePortStr != "" {
		if port, err := strconv.ParseUint(remotePortStr, 10, 16); err == nil {
			metrics.DstPort = uint16(port)
		}
	}

	// Extract report timestamp
	if timestampStr := event.GetHeader("Event-Date-Timestamp"); timestampStr != "" {
		if timestamp, err := strconv.ParseInt(timestampStr, 10, 64); err == nil {
			metrics.ReportTimestamp = timestamp
		}
	}

	return nil
}

// extractStateAndDomain retrieves correlation_id and domain_name from state or event
func (qc *qosCalculator) extractStateAndDomain(ctx context.Context, event *connection.FSEvent, metrics *QoSMetrics) error {
	channelID := metrics.ChannelID

	// Try to get channel state
	state, err := qc.store.Get(ctx, channelID)
	if err == nil {
		// State found - use correlation_id and domain_name from state
		metrics.CorrelationID = state.CorrelationID
		metrics.DomainName = state.DomainName

		// Delete channel state after metrics calculation
		if delErr := qc.store.Delete(ctx, channelID); delErr != nil {
			// Log warning but don't fail - metrics are already calculated
			logger.WarnWithFields(map[string]interface{}{
				"channel_id":     channelID,
				"correlation_id": metrics.CorrelationID,
				"error":          delErr.Error(),
			}, "Failed to delete channel state after QoS calculation")
		}

		return nil
	}

	// State not found - extract from event
	// Extract correlation_id using priority: Other-Leg-Unique-ID → Unique-ID → variable_sip_call_id → variable_global_call_id
	if correlationID := event.GetHeader("Other-Leg-Unique-ID"); correlationID != "" {
		metrics.CorrelationID = correlationID
	} else if correlationID := event.GetHeader("Unique-ID"); correlationID != "" {
		metrics.CorrelationID = correlationID
	} else if correlationID := event.GetHeader("variable_sip_call_id"); correlationID != "" {
		metrics.CorrelationID = correlationID
	} else if correlationID := event.GetHeader("variable_global_call_id"); correlationID != "" {
		metrics.CorrelationID = correlationID
	}

	// Extract domain_name using priority: variable_domain_name → variable_sip_from_host → variable_sip_to_host
	if domainName := event.GetHeader("variable_domain_name"); domainName != "" {
		metrics.DomainName = domainName
	} else if domainName := event.GetHeader("variable_sip_from_host"); domainName != "" {
		metrics.DomainName = domainName
	} else if domainName := event.GetHeader("variable_sip_to_host"); domainName != "" {
		metrics.DomainName = domainName
	}
	// If no domain found, leave as empty string (default)

	return nil
}
