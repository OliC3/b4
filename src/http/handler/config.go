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

	// Load initial manual domains if any
	if len(api.cfg.Domains.SNIDomains) > 0 {
		api.manualDomains = make([]string, len(api.cfg.Domains.SNIDomains))
		copy(api.manualDomains, api.cfg.Domains.SNIDomains)
	}

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

	categoryBreakdown := make(map[string]int)
	totalGeositeDomains := 0

	if len(a.cfg.Domains.GeoSiteCategories) > 0 {

		counts, _ := a.geodataManager.GetCategoryCounts(a.cfg.Domains.GeoSiteCategories)
		categoryBreakdown = counts
		for _, count := range categoryBreakdown {
			totalGeositeDomains += count
		}
	}

	// Create response with stats
	response := ConfigResponse{
		Config: a.cfg,
		DomainStats: DomainStatistics{
			ManualDomains:     len(a.manualDomains),
			GeositeDomains:    totalGeositeDomains,
			TotalDomains:      len(a.manualDomains) + totalGeositeDomains,
			GeositeAvailable:  a.geodataManager.IsConfigured(),
			CategoryBreakdown: categoryBreakdown,
		},
	}

	// IMPORTANT: Return only manual domains in sni_domains field
	configCopy := *a.cfg
	configCopy.Domains.SNIDomains = a.manualDomains
	response.Config = &configCopy

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

	if err := newConfig.Validate(); err != nil {
		log.Errorf("Invalid configuration: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Store manual domains separately - these are what the user explicitly added
	a.manualDomains = make([]string, len(newConfig.Domains.SNIDomains))
	copy(a.manualDomains, newConfig.Domains.SNIDomains)
	log.Infof("Updated manual domains: %d", len(a.manualDomains))

	// Update geodata manager paths if they changed
	a.geodataManager.UpdatePaths(newConfig.Domains.GeoSitePath, newConfig.Domains.GeoIpPath)

	// Load geosite domains using the manager
	var allGeositeDomains []string
	var categoryStats map[string]int

	if newConfig.Domains.GeoSitePath != "" && len(newConfig.Domains.GeoSiteCategories) > 0 {
		log.Infof("Loading domains from geodata for categories: %v", newConfig.Domains.GeoSiteCategories)

		var err error
		allGeositeDomains, categoryStats, err = a.geodataManager.LoadCategories(newConfig.Domains.GeoSiteCategories)
		if err != nil {
			log.Errorf("Failed to load geosite categories: %v", err)
			http.Error(w, fmt.Sprintf("Failed to load geodata: %v", err), http.StatusInternalServerError)
			return
		}

		log.Infof("Loaded %d total geosite domains from %d categories", len(allGeositeDomains), len(categoryStats))
		for category, count := range categoryStats {
			log.Infof("  - %s: %d domains", category, count)
		}

		m := metrics.GetMetricsCollector()
		m.RecordEvent("info", fmt.Sprintf("Loaded %d domains from geodata across %d categories",
			len(allGeositeDomains), len(newConfig.Domains.GeoSiteCategories)))
	} else if len(newConfig.Domains.GeoSiteCategories) == 0 {
		// Clear geosite cache if no categories selected
		a.geodataManager.ClearCache()
		categoryStats = make(map[string]int)
		allGeositeDomains = []string{}
		log.Infof("Cleared all geosite domains")
	}

	// Config should only contain manual domains
	newConfig.Domains.SNIDomains = a.manualDomains

	a.updateMainConfig(&newConfig)

	// Combine all domains for the matcher
	allDomainsForMatcher := make([]string, 0, len(a.manualDomains)+len(allGeositeDomains))
	allDomainsForMatcher = append(allDomainsForMatcher, a.manualDomains...)
	allDomainsForMatcher = append(allDomainsForMatcher, allGeositeDomains...)

	// Save config to file if path is set
	if newConfig.ConfigPath != "" {
		if err := newConfig.SaveToFile(newConfig.ConfigPath); err != nil {
			log.Errorf("Failed to save config: %v", err)
		} else {
			log.Infof("Config saved to %s", newConfig.ConfigPath)
		}
	}

	// Prepare response with statistics
	totalDomains := make(map[string]struct{})
	for _, d := range a.manualDomains {
		totalDomains[d] = struct{}{}
	}
	for _, d := range allGeositeDomains {
		totalDomains[d] = struct{}{}
	}

	if globalPool != nil {
		globalPool.UpdateConfig(&newConfig, allDomainsForMatcher)
		log.Infof("Config pushed to all workers (manual: %d, geosite: %d, total: %d domains)",
			len(a.manualDomains), len(allGeositeDomains), len(totalDomains))
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Configuration updated successfully",
		"domain_stats": DomainStatistics{
			ManualDomains:     len(a.manualDomains),
			GeositeDomains:    len(allGeositeDomains),
			TotalDomains:      len(totalDomains),
			CategoryBreakdown: categoryStats,
		},
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

	// Get default config
	defaultCfg := config.DefaultConfig

	// Preserve domains and checker from current config
	defaultCfg.Domains = a.cfg.Domains
	defaultCfg.System.Checker = a.cfg.System.Checker
	defaultCfg.ConfigPath = a.cfg.ConfigPath
	defaultCfg.System.WebServer.IsEnabled = a.cfg.System.WebServer.IsEnabled

	// Update main config
	a.updateMainConfig(&defaultCfg)

	// Load geosite domains if configured
	var allGeositeDomains []string
	if a.cfg.Domains.GeoSitePath != "" && len(a.cfg.Domains.GeoSiteCategories) > 0 {
		var err error
		allGeositeDomains, _, err = a.geodataManager.LoadCategories(a.cfg.Domains.GeoSiteCategories)
		if err != nil {
			log.Errorf("Failed to load geosite domains after reset: %v", err)
		}
	}

	// Combine all domains for the matcher
	allDomainsForMatcher := make([]string, 0, len(a.manualDomains)+len(allGeositeDomains))
	allDomainsForMatcher = append(allDomainsForMatcher, a.manualDomains...)
	allDomainsForMatcher = append(allDomainsForMatcher, allGeositeDomains...)

	// Update NFQ pool
	if globalPool != nil {
		globalPool.UpdateConfig(a.cfg, allDomainsForMatcher)
		log.Infof("Config reset and pushed to all workers")
	}

	// Save to file
	if a.cfg.ConfigPath != "" {
		if err := a.cfg.SaveToFile(a.cfg.ConfigPath); err != nil {
			log.Errorf("Failed to save reset config: %v", err)
		} else {
			log.Infof("Reset config saved to %s", a.cfg.ConfigPath)
		}
	}

	setJsonHeader(w)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Configuration reset to defaults (domains and checker preserved)",
	})
}

func (a *API) updateMainConfig(newCfg *config.Config) {
	newCfg.ConfigPath = a.cfg.ConfigPath
	newCfg.System.WebServer.IsEnabled = a.cfg.System.WebServer.IsEnabled
	a.cfg = newCfg
}
