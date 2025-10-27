// src/http/handler/metrics.go
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/daniellavrushin/b4/metrics"
)

// Re-export for backward compatibility
type MetricsCollector = metrics.MetricsCollector
type WorkerHealth = metrics.WorkerHealth

func GetMetricsCollector() *metrics.MetricsCollector {
	return metrics.GetMetricsCollector()
}

// RegisterMetricsApi registers the metrics API endpoints
func (api *API) RegisterMetricsApi() {
	api.mux.HandleFunc("/api/metrics", api.getMetrics)
	api.mux.HandleFunc("/api/metrics/summary", api.getMetricsSummary)
}

func (a *API) getMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	metricsData := metrics.GetMetricsCollector().GetSnapshot()

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	_ = enc.Encode(metricsData)
}

func (a *API) getMetricsSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	m := metrics.GetMetricsCollector().GetSnapshot()

	summary := map[string]interface{}{
		"total_connections": m.TotalConnections,
		"active_flows":      m.ActiveFlows,
		"current_cps":       m.CurrentCPS,
		"current_pps":       m.CurrentPPS,
		"uptime":            m.Uptime,
		"memory_percent":    m.MemoryUsage.Percent,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	_ = enc.Encode(summary)
}
