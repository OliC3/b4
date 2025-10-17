package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/daniellavrushin/b4/config"
)

// MetricsCollector collects and stores metrics for the dashboard
type MetricsCollector struct {
	mu sync.RWMutex

	// Counters
	TotalConnections    uint64 `json:"total_connections"`
	ActiveFlows         uint64 `json:"active_flows"`
	PacketsProcessed    uint64 `json:"packets_processed"`
	BytesProcessed      uint64 `json:"bytes_processed"`
	TCPConnections      uint64 `json:"tcp_connections"`
	UDPConnections      uint64 `json:"udp_connections"`
	TargetedConnections uint64 `json:"targeted_connections"`

	// Time series data (last 60 data points, 1 per second)
	ConnectionRate []TimeSeriesPoint `json:"connection_rate"`
	PacketRate     []TimeSeriesPoint `json:"packet_rate"`

	// Top domains
	TopDomains map[string]uint64 `json:"top_domains"`

	// Protocol distribution
	ProtocolDist map[string]uint64 `json:"protocol_dist"`

	// Geographic distribution (if geoip enabled)
	GeoDist map[string]uint64 `json:"geo_dist"`

	// System info
	StartTime      time.Time      `json:"start_time"`
	Uptime         string         `json:"uptime"`
	CPUUsage       float64        `json:"cpu_usage"`
	MemoryUsage    MemoryStats    `json:"memory_usage"`
	WorkerStatus   []WorkerHealth `json:"worker_status"`
	NFQueueStatus  string         `json:"nfqueue_status"`
	IPTablesStatus string         `json:"iptables_status"`

	// Recent activity
	RecentConnections []ConnectionLog `json:"recent_connections"`
	RecentEvents      []SystemEvent   `json:"recent_events"`

	// Real-time rates
	CurrentCPS float64 `json:"current_cps"` // Connections per second
	CurrentPPS float64 `json:"current_pps"` // Packets per second

	// Internal tracking
	lastUpdate      time.Time
	lastConnCount   uint64
	lastPacketCount uint64
}

type TimeSeriesPoint struct {
	Timestamp int64   `json:"timestamp"`
	Value     float64 `json:"value"`
}

type MemoryStats struct {
	Allocated      uint64  `json:"allocated"`
	TotalAllocated uint64  `json:"total_allocated"`
	System         uint64  `json:"system"`
	NumGC          uint32  `json:"num_gc"`
	HeapAlloc      uint64  `json:"heap_alloc"`
	HeapInuse      uint64  `json:"heap_inuse"`
	Percent        float64 `json:"percent"`
}

type WorkerHealth struct {
	ID        int    `json:"id"`
	Status    string `json:"status"`
	Processed uint64 `json:"processed"`
}

type ConnectionLog struct {
	Timestamp   time.Time `json:"timestamp"`
	Protocol    string    `json:"protocol"`
	Domain      string    `json:"domain"`
	Source      string    `json:"source"`
	Destination string    `json:"destination"`
	IsTarget    bool      `json:"is_target"`
}

type SystemEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
}

var (
	metricsCollector *MetricsCollector
	metricsOnce      sync.Once
)

// GetMetricsCollector returns the singleton metrics collector
func GetMetricsCollector() *MetricsCollector {
	metricsOnce.Do(func() {
		metricsCollector = &MetricsCollector{
			StartTime:         time.Now(),
			TopDomains:        make(map[string]uint64),
			ProtocolDist:      make(map[string]uint64),
			GeoDist:           make(map[string]uint64),
			ConnectionRate:    make([]TimeSeriesPoint, 0, 60),
			PacketRate:        make([]TimeSeriesPoint, 0, 60),
			RecentConnections: make([]ConnectionLog, 0, 10),
			RecentEvents:      make([]SystemEvent, 0, 20),
			WorkerStatus:      make([]WorkerHealth, 0),
			NFQueueStatus:     "active",
			IPTablesStatus:    "active",
			lastUpdate:        time.Now(),
		}

		// Start the metrics update goroutine
		go metricsCollector.updateLoop()
	})
	return metricsCollector
}

// updateLoop updates time-series metrics every second
func (m *MetricsCollector) updateLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		m.updateRates()
		m.updateSystemStats()
	}
}

// updateRates calculates current CPS and PPS
func (m *MetricsCollector) updateRates() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	duration := now.Sub(m.lastUpdate).Seconds()
	if duration <= 0 {
		return
	}

	// Calculate rates
	connDiff := m.TotalConnections - m.lastConnCount
	packetDiff := m.PacketsProcessed - m.lastPacketCount

	m.CurrentCPS = float64(connDiff) / duration
	m.CurrentPPS = float64(packetDiff) / duration

	// Update time series (keep last 60 points)
	nowMs := now.UnixMilli()

	m.ConnectionRate = append(m.ConnectionRate, TimeSeriesPoint{
		Timestamp: nowMs,
		Value:     m.CurrentCPS,
	})
	if len(m.ConnectionRate) > 60 {
		m.ConnectionRate = m.ConnectionRate[len(m.ConnectionRate)-60:]
	}

	m.PacketRate = append(m.PacketRate, TimeSeriesPoint{
		Timestamp: nowMs,
		Value:     m.CurrentPPS,
	})
	if len(m.PacketRate) > 60 {
		m.PacketRate = m.PacketRate[len(m.PacketRate)-60:]
	}

	// Update tracking
	m.lastUpdate = now
	m.lastConnCount = m.TotalConnections
	m.lastPacketCount = m.PacketsProcessed

	// Update uptime
	m.Uptime = formatDuration(now.Sub(m.StartTime))
}

// updateSystemStats updates CPU and memory statistics
func (m *MetricsCollector) updateSystemStats() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	m.mu.Lock()
	defer m.mu.Unlock()

	m.MemoryUsage = MemoryStats{
		Allocated:      memStats.Alloc,
		TotalAllocated: memStats.TotalAlloc,
		System:         memStats.Sys,
		NumGC:          memStats.NumGC,
		HeapAlloc:      memStats.HeapAlloc,
		HeapInuse:      memStats.HeapInuse,
		Percent:        float64(memStats.Alloc) / float64(memStats.Sys) * 100,
	}

	// CPU usage would require more complex calculation
	// For now, we'll use goroutine count as a proxy
	m.CPUUsage = float64(runtime.NumGoroutine())
}

// RecordConnection records a new connection
func (m *MetricsCollector) RecordConnection(protocol, domain, source, destination string, isTarget bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalConnections++
	m.ActiveFlows++

	// Update protocol counters
	if protocol == "TCP" {
		m.TCPConnections++
		m.ProtocolDist["TCP"]++
	} else if protocol == "UDP" {
		m.UDPConnections++
		m.ProtocolDist["UDP"]++
	}

	if isTarget {
		m.TargetedConnections++
	}

	// Update top domains
	if domain != "" {
		m.TopDomains[domain]++

		// Keep only top 10 domains
		if len(m.TopDomains) > 20 {
			m.pruneTopDomains()
		}
	}

	// Add to recent connections
	conn := ConnectionLog{
		Timestamp:   time.Now(),
		Protocol:    protocol,
		Domain:      domain,
		Source:      source,
		Destination: destination,
		IsTarget:    isTarget,
	}

	m.RecentConnections = append([]ConnectionLog{conn}, m.RecentConnections...)
	if len(m.RecentConnections) > 10 {
		m.RecentConnections = m.RecentConnections[:10]
	}
}

// RecordPacket records packet processing
func (m *MetricsCollector) RecordPacket(bytes uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.PacketsProcessed++
	m.BytesProcessed += bytes
}

// RecordEvent records a system event
func (m *MetricsCollector) RecordEvent(level, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	event := SystemEvent{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
	}

	m.RecentEvents = append([]SystemEvent{event}, m.RecentEvents...)
	if len(m.RecentEvents) > 20 {
		m.RecentEvents = m.RecentEvents[:20]
	}
}

// UpdateWorkerStatus updates the status of worker threads
func (m *MetricsCollector) UpdateWorkerStatus(workers []WorkerHealth) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.WorkerStatus = workers
}

// CloseConnection decrements active flows
func (m *MetricsCollector) CloseConnection() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ActiveFlows > 0 {
		m.ActiveFlows--
	}
}

// GetSnapshot returns a snapshot of current metrics
func (m *MetricsCollector) GetSnapshot() *MetricsCollector {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Create a deep copy
	snapshot := &MetricsCollector{
		TotalConnections:    m.TotalConnections,
		ActiveFlows:         m.ActiveFlows,
		PacketsProcessed:    m.PacketsProcessed,
		BytesProcessed:      m.BytesProcessed,
		TCPConnections:      m.TCPConnections,
		UDPConnections:      m.UDPConnections,
		TargetedConnections: m.TargetedConnections,
		StartTime:           m.StartTime,
		Uptime:              m.Uptime,
		CPUUsage:            m.CPUUsage,
		MemoryUsage:         m.MemoryUsage,
		NFQueueStatus:       m.NFQueueStatus,
		IPTablesStatus:      m.IPTablesStatus,
		CurrentCPS:          m.CurrentCPS,
		CurrentPPS:          m.CurrentPPS,
	}

	// Deep copy slices and maps
	snapshot.ConnectionRate = make([]TimeSeriesPoint, len(m.ConnectionRate))
	copy(snapshot.ConnectionRate, m.ConnectionRate)

	snapshot.PacketRate = make([]TimeSeriesPoint, len(m.PacketRate))
	copy(snapshot.PacketRate, m.PacketRate)

	snapshot.TopDomains = make(map[string]uint64)
	for k, v := range m.TopDomains {
		snapshot.TopDomains[k] = v
	}

	snapshot.ProtocolDist = make(map[string]uint64)
	for k, v := range m.ProtocolDist {
		snapshot.ProtocolDist[k] = v
	}

	snapshot.GeoDist = make(map[string]uint64)
	for k, v := range m.GeoDist {
		snapshot.GeoDist[k] = v
	}

	snapshot.WorkerStatus = make([]WorkerHealth, len(m.WorkerStatus))
	copy(snapshot.WorkerStatus, m.WorkerStatus)

	snapshot.RecentConnections = make([]ConnectionLog, len(m.RecentConnections))
	copy(snapshot.RecentConnections, m.RecentConnections)

	snapshot.RecentEvents = make([]SystemEvent, len(m.RecentEvents))
	copy(snapshot.RecentEvents, m.RecentEvents)

	return snapshot
}

// pruneTopDomains keeps only the top 10 domains by count
func (m *MetricsCollector) pruneTopDomains() {
	// Find minimum count in top 10
	var minCount uint64 = ^uint64(0)
	var minDomain string

	if len(m.TopDomains) <= 10 {
		return
	}

	for domain, count := range m.TopDomains {
		if count < minCount {
			minCount = count
			minDomain = domain
		}
	}

	delete(m.TopDomains, minDomain)
}

// formatDuration formats a duration into a human-readable string
func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

// RegisterMetricsApi registers the metrics API endpoints
func RegisterMetricsApi(mux *http.ServeMux, cfg *config.Config) {
	api := &API{cfg: cfg}
	mux.HandleFunc("/api/metrics", api.getMetrics)
	mux.HandleFunc("/api/metrics/summary", api.getMetricsSummary)
}

// getMetrics returns full metrics data
func (a *API) getMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	metrics := GetMetricsCollector().GetSnapshot()

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	_ = enc.Encode(metrics)
}

// getMetricsSummary returns a summary of key metrics
func (a *API) getMetricsSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	metrics := GetMetricsCollector().GetSnapshot()

	summary := map[string]interface{}{
		"total_connections": metrics.TotalConnections,
		"active_flows":      metrics.ActiveFlows,
		"current_cps":       metrics.CurrentCPS,
		"current_pps":       metrics.CurrentPPS,
		"uptime":            metrics.Uptime,
		"memory_percent":    metrics.MemoryUsage.Percent,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	_ = enc.Encode(summary)
}
