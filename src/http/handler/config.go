// src/http/handler/config.go
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/log"
	"github.com/daniellavrushin/b4/metrics"
)

func (api *API) RegisterConfigApi() {

	api.mux.HandleFunc("/api/config", api.handleConfig)
	api.mux.HandleFunc("/api/config/reset", api.resetConfig)
}

func (a *API) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.getConfig(w)
	case http.MethodPut:
		a.updateConfig(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (a *API) getConfig(w http.ResponseWriter) {
	setJsonHeader(w)

	// Calculate statistics for each set
	setsWithStats := make([]SetWithStats, len(a.cfg.Sets))
	totalDomains := 0
	totalIPs := 0

	for i, set := range a.cfg.Sets {
		// Count manual domains and IPs
		manualDomains := len(set.Targets.SNIDomains)
		manualIPs := len(set.Targets.IPs)

		// Get geosite category counts
		geositeCounts := make(map[string]int)
		geositeTotalDomains := 0
		if len(set.Targets.GeoSiteCategories) > 0 && a.geodataManager.IsGeositeConfigured() {
			counts, err := a.geodataManager.GetGeositeCategoryCounts(set.Targets.GeoSiteCategories)
			if err == nil {
				geositeCounts = counts
				for _, count := range counts {
					geositeTotalDomains += count
				}
			}
		}

		// Get geoip category counts
		geoipCounts := make(map[string]int)
		geoipTotalIPs := 0
		if len(set.Targets.GeoIpCategories) > 0 && a.geodataManager.IsGeoipConfigured() {
			counts, err := a.geodataManager.GetGeoipCategoryCounts(set.Targets.GeoIpCategories)
			if err == nil {
				geoipCounts = counts
				for _, count := range counts {
					geoipTotalIPs += count
				}
			}
		}

		setTotalDomains := manualDomains + geositeTotalDomains
		setTotalIPs := manualIPs + geoipTotalIPs

		totalDomains += setTotalDomains
		totalIPs += setTotalIPs

		setsWithStats[i] = SetWithStats{
			SetConfig: set,
			Stats: SetStatistics{
				ManualDomains:            manualDomains,
				ManualIPs:                manualIPs,
				GeositeDomains:           geositeTotalDomains,
				GeoipIPs:                 geoipTotalIPs,
				TotalDomains:             setTotalDomains,
				TotalIPs:                 setTotalIPs,
				GeositeCategoryBreakdown: geositeCounts,
				GeoipCategoryBreakdown:   geoipCounts,
			},
		}
	}

	response := ConfigResponse{
		Config:  a.cfg,
		Sets:    setsWithStats,
		Success: true,
		Message: "Configuration retrieved successfully",
	}
	enc := json.NewEncoder(w)
	_ = enc.Encode(response)
}

func (a *API) updateConfig(w http.ResponseWriter, r *http.Request) {
	var newConfig config.Config

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&newConfig); err != nil {
		log.Errorf("Failed to decode config update: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	newConfig.ConfigPath = a.cfg.ConfigPath

	// update logging level if changed
	if newConfig.System.Logging.Level != log.Level(log.CurLevel.Load()) {
		log.SetLevel(log.Level(newConfig.System.Logging.Level))
		log.Infof("Log level changed to %s", newConfig.System.Logging.Level)
	}

	a.geodataManager.UpdatePaths(newConfig.System.Geo.GeoSitePath, newConfig.System.Geo.GeoIpPath)

	// Calculate statistics for response
	setsWithStats := make([]SetWithStats, len(newConfig.Sets))
	allDomainsCount := 0
	allIpsCount := 0

	for i, set := range newConfig.Sets {
		a.loadTargetsForSetCached(set)

		manualDomains := len(set.Targets.SNIDomains)
		manualIPs := len(set.Targets.IPs)

		// Get geosite counts
		geositeCounts := make(map[string]int)
		geositeTotalDomains := 0
		if len(set.Targets.GeoSiteCategories) > 0 {
			counts, err := a.geodataManager.GetGeositeCategoryCounts(set.Targets.GeoSiteCategories)
			if err == nil {
				geositeCounts = counts
				for _, count := range counts {
					geositeTotalDomains += count
				}
			}
		}

		// Get geoip counts
		geoipCounts := make(map[string]int)
		geoipTotalIPs := 0
		if len(set.Targets.GeoIpCategories) > 0 {
			counts, err := a.geodataManager.GetGeoipCategoryCounts(set.Targets.GeoIpCategories)
			if err == nil {
				geoipCounts = counts
				for _, count := range counts {
					geoipTotalIPs += count
				}
			}
		}

		setTotalDomains := manualDomains + geositeTotalDomains
		setTotalIPs := manualIPs + geoipTotalIPs

		allDomainsCount += setTotalDomains
		allIpsCount += setTotalIPs

		setsWithStats[i] = SetWithStats{
			SetConfig: set,
			Stats: SetStatistics{
				ManualDomains:            manualDomains,
				ManualIPs:                manualIPs,
				GeositeDomains:           geositeTotalDomains,
				TotalDomains:             setTotalDomains,
				TotalIPs:                 setTotalIPs,
				GeositeCategoryBreakdown: geositeCounts,
				GeoipCategoryBreakdown:   geoipCounts,
			},
		}
	}

	if err := newConfig.Validate(); err != nil {
		log.Errorf("Invalid configuration: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := a.saveAndPushConfig(&newConfig); err != nil {
		log.Errorf("Failed to update config: %v", err)
		http.Error(w, "Failed to update config", http.StatusInternalServerError)
		return
	}

	m := metrics.GetMetricsCollector()
	m.RecordEvent("info", fmt.Sprintf("Loaded %d domains and %d IPs across %d sets", allDomainsCount, allIpsCount, len(newConfig.Sets)))
	log.Infof("Loaded %d domains and %d IPs across %d sets", allDomainsCount, allIpsCount, len(newConfig.Sets))

	response := ConfigResponse{
		Success: true,
		Message: "Configuration updated successfully",
		Config:  &newConfig,
		Sets:    setsWithStats,
	}

	setJsonHeader(w)
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	_ = enc.Encode(response)
}

func (a *API) resetConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	log.Infof("Config reset requested")

	defaultCfg := config.NewConfig()
	defaultCfg.System.Checker = a.cfg.System.Checker
	defaultCfg.ConfigPath = a.cfg.ConfigPath
	defaultCfg.System.WebServer.IsEnabled = a.cfg.System.WebServer.IsEnabled

	for _, set := range a.cfg.Sets {
		set.ResetToDefaults()
		a.loadTargetsForSetCached(set)
		defaultCfg.Sets = append(defaultCfg.Sets, set)
	}

	err := defaultCfg.Validate()

	if err != nil {
		log.Errorf("Failed to validate reset config: %v", err)
		http.Error(w, "Failed to reset config", http.StatusInternalServerError)
		return
	}

	if err := a.saveAndPushConfig(&defaultCfg); err != nil {
		log.Errorf("Failed to reset config: %v", err)
		http.Error(w, "Failed to reset config", http.StatusInternalServerError)
		return
	}

	setJsonHeader(w)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Configuration reset to defaults (domains and checker preserved)",
	})
}

func (a *API) saveAndPushConfig(cfg *config.Config) error {
	if globalPool != nil {
		err := globalPool.UpdateConfig(cfg)
		if err != nil {
			return fmt.Errorf("failed to update global pool config: %v", err)
		}
	}

	err := cfg.SaveToFile(cfg.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to save config to file: %v", err)
	}

	*a.cfg = *cfg

	return nil
}
