package metrics

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// InternalMetrics tracks internal application metrics
type InternalMetrics struct {
	eventsReceived        map[string]map[string]*int64 // instance -> event_type -> count
	eventsProcessed       map[string]map[string]*int64 // instance -> event_type -> count
	rtcpMessagesProcessed map[string]map[string]*int64 // instance -> direction -> count
	qosMessagesGenerated  map[string]*int64            // instance -> count
	storageOperations     map[string]map[string]*int64 // operation -> status -> count
	fsConnections         map[string]*int64            // instance -> status (1=connected, 0=disconnected)
	mu                    sync.RWMutex
}

var (
	globalMetrics *InternalMetrics
	once          sync.Once
)

// GetMetrics returns the global metrics instance
func GetMetrics() *InternalMetrics {
	once.Do(func() {
		globalMetrics = &InternalMetrics{
			eventsReceived:        make(map[string]map[string]*int64),
			eventsProcessed:       make(map[string]map[string]*int64),
			rtcpMessagesProcessed: make(map[string]map[string]*int64),
			qosMessagesGenerated:  make(map[string]*int64),
			storageOperations:     make(map[string]map[string]*int64),
			fsConnections:         make(map[string]*int64),
		}
	})
	return globalMetrics
}

// IncrementEventsReceived increments the events received counter
func (m *InternalMetrics) IncrementEventsReceived(instance, eventType string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.eventsReceived[instance]; !ok {
		m.eventsReceived[instance] = make(map[string]*int64)
	}
	if _, ok := m.eventsReceived[instance][eventType]; !ok {
		var counter int64
		m.eventsReceived[instance][eventType] = &counter
	}
	atomic.AddInt64(m.eventsReceived[instance][eventType], 1)
}

// IncrementEventsProcessed increments the events processed counter
func (m *InternalMetrics) IncrementEventsProcessed(instance, eventType string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.eventsProcessed[instance]; !ok {
		m.eventsProcessed[instance] = make(map[string]*int64)
	}
	if _, ok := m.eventsProcessed[instance][eventType]; !ok {
		var counter int64
		m.eventsProcessed[instance][eventType] = &counter
	}
	atomic.AddInt64(m.eventsProcessed[instance][eventType], 1)
}

// IncrementRTCPMessagesProcessed increments the RTCP messages processed counter
func (m *InternalMetrics) IncrementRTCPMessagesProcessed(instance, direction string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.rtcpMessagesProcessed[instance]; !ok {
		m.rtcpMessagesProcessed[instance] = make(map[string]*int64)
	}
	if _, ok := m.rtcpMessagesProcessed[instance][direction]; !ok {
		var counter int64
		m.rtcpMessagesProcessed[instance][direction] = &counter
	}
	atomic.AddInt64(m.rtcpMessagesProcessed[instance][direction], 1)
}

// IncrementQoSMessagesGenerated increments the QoS messages generated counter
func (m *InternalMetrics) IncrementQoSMessagesGenerated(instance string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.qosMessagesGenerated[instance]; !ok {
		var counter int64
		m.qosMessagesGenerated[instance] = &counter
	}
	atomic.AddInt64(m.qosMessagesGenerated[instance], 1)
}

// IncrementStorageOperations increments the storage operations counter
func (m *InternalMetrics) IncrementStorageOperations(operation, status string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.storageOperations[operation]; !ok {
		m.storageOperations[operation] = make(map[string]*int64)
	}
	if _, ok := m.storageOperations[operation][status]; !ok {
		var counter int64
		m.storageOperations[operation][status] = &counter
	}
	atomic.AddInt64(m.storageOperations[operation][status], 1)
}

// SetFSConnectionStatus sets the connection status for a FreeSWITCH instance
func (m *InternalMetrics) SetFSConnectionStatus(instance string, connected bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.fsConnections[instance]; !ok {
		var status int64
		m.fsConnections[instance] = &status
	}
	if connected {
		atomic.StoreInt64(m.fsConnections[instance], 1)
	} else {
		atomic.StoreInt64(m.fsConnections[instance], 0)
	}
}

// GetPrometheusMetrics returns metrics in Prometheus format
func (m *InternalMetrics) GetPrometheusMetrics() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var output string

	// Events received
	output += "# HELP fsagent_events_received_total Total number of events received from FreeSWITCH\n"
	output += "# TYPE fsagent_events_received_total counter\n"
	for instance, eventTypes := range m.eventsReceived {
		for eventType, counter := range eventTypes {
			count := atomic.LoadInt64(counter)
			output += fmt.Sprintf("fsagent_events_received_total{instance=\"%s\",event_type=\"%s\"} %d\n", instance, eventType, count)
		}
	}

	// Events processed
	output += "# HELP fsagent_events_processed_total Total number of events processed successfully\n"
	output += "# TYPE fsagent_events_processed_total counter\n"
	for instance, eventTypes := range m.eventsProcessed {
		for eventType, counter := range eventTypes {
			count := atomic.LoadInt64(counter)
			output += fmt.Sprintf("fsagent_events_processed_total{instance=\"%s\",event_type=\"%s\"} %d\n", instance, eventType, count)
		}
	}

	// RTCP messages processed
	output += "# HELP fsagent_rtcp_messages_processed_total Total number of RTCP messages processed\n"
	output += "# TYPE fsagent_rtcp_messages_processed_total counter\n"
	for instance, directions := range m.rtcpMessagesProcessed {
		for direction, counter := range directions {
			count := atomic.LoadInt64(counter)
			output += fmt.Sprintf("fsagent_rtcp_messages_processed_total{instance=\"%s\",direction=\"%s\"} %d\n", instance, direction, count)
		}
	}

	// QoS messages generated
	output += "# HELP fsagent_qos_messages_generated_total Total number of QoS messages generated\n"
	output += "# TYPE fsagent_qos_messages_generated_total counter\n"
	for instance, counter := range m.qosMessagesGenerated {
		count := atomic.LoadInt64(counter)
		output += fmt.Sprintf("fsagent_qos_messages_generated_total{instance=\"%s\"} %d\n", instance, count)
	}

	// Storage operations
	output += "# HELP fsagent_storage_operations_total Total number of storage operations\n"
	output += "# TYPE fsagent_storage_operations_total counter\n"
	for operation, statuses := range m.storageOperations {
		for status, counter := range statuses {
			count := atomic.LoadInt64(counter)
			output += fmt.Sprintf("fsagent_storage_operations_total{operation=\"%s\",status=\"%s\"} %d\n", operation, status, count)
		}
	}

	// FS connections
	output += "# HELP fsagent_fs_connections FreeSWITCH connection status (1=connected, 0=disconnected)\n"
	output += "# TYPE fsagent_fs_connections gauge\n"
	for instance, status := range m.fsConnections {
		statusValue := atomic.LoadInt64(status)
		statusLabel := "disconnected"
		if statusValue == 1 {
			statusLabel = "connected"
		}
		output += fmt.Sprintf("fsagent_fs_connections{instance=\"%s\",status=\"%s\"} %d\n", instance, statusLabel, statusValue)
	}

	return output
}
