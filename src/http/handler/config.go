package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/geodat"
	"github.com/daniellavrushin/b4/log"
	"github.com/daniellavrushin/b4/metrics"
)

func RegisterConfigApi(mux *http.ServeMux, cfg *config.Config) {
	api := &API{
		cfg:            cfg,
		manualDomains:  []string{}, // Initialize empty
		geositeDomains: make(map[string][]string),
	}

	// Load initial manual domains if any
	if len(cfg.Domains.SNIDomains) > 0 {
		api.manualDomains = make([]string, len(cfg.Domains.SNIDomains))
		copy(api.manualDomains, cfg.Domains.SNIDomains)
	}

	mux.HandleFunc("/api/config", api.handleConfig)
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
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	// Get all geosite domains count
	totalGeositeDomains := 0
	for _, domains := range a.geositeDomains {
		totalGeositeDomains += len(domains)
	}

	// Create response with stats
	response := ConfigResponse{
		Config: a.cfg,
		DomainStats: DomainStatistics{
			ManualDomains:     len(a.manualDomains),
			GeositeDomains:    totalGeositeDomains,
			TotalDomains:      len(a.manualDomains) + totalGeositeDomains,
			GeositeAvailable:  a.cfg.Domains.GeoSitePath != "",
			CategoryBreakdown: a.getGeoCategoryBreakdown(),
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

	// Load geosite domains if needed - but DON'T merge them into SNIDomains
	categoryStats := make(map[string]int)
	var allGeositeDomains []string

	if newConfig.Domains.GeoSitePath != "" && len(newConfig.Domains.GeoSiteCategories) > 0 {
		log.Infof("Loading domains from geodata for categories: %v", newConfig.Domains.GeoSiteCategories)

		// Clear previous geosite domains
		a.geositeDomains = make(map[string][]string)

		// Load each category separately for tracking
		for _, category := range newConfig.Domains.GeoSiteCategories {
			domains, err := geodat.LoadDomainsFromSites(newConfig.Domains.GeoSitePath, []string{category})
			if err != nil {
				log.Errorf("Failed to load category %s: %v", category, err)
				continue
			}
			a.geositeDomains[category] = domains
			categoryStats[category] = len(domains)
			log.Infof("Loaded %d domains for category: %s", len(domains), category)
			allGeositeDomains = append(allGeositeDomains, domains...)
		}

		m := metrics.GetMetricsCollector()
		m.RecordEvent("info", fmt.Sprintf("Loaded %d domains from geodata across %d categories",
			len(allGeositeDomains), len(newConfig.Domains.GeoSiteCategories)))
	} else if len(newConfig.Domains.GeoSiteCategories) == 0 {
		// Clear geosite domains if no categories selected
		a.geositeDomains = make(map[string][]string)
		log.Infof("Cleared all geosite domains")
	}

	newConfig.Domains.SNIDomains = a.manualDomains

	a.updateMainConfig(&newConfig)

	allDomainsForMatcher := make([]string, 0, len(a.manualDomains)+len(allGeositeDomains))
	allDomainsForMatcher = append(allDomainsForMatcher, a.manualDomains...)
	allDomainsForMatcher = append(allDomainsForMatcher, allGeositeDomains...)

	if globalPool != nil {
		globalPool.UpdateConfig(&newConfig, allDomainsForMatcher)
		log.Infof("Config pushed to all workers (manual: %d, geosite: %d, total: %d domains)",
			len(a.manualDomains), len(allGeositeDomains), len(allDomainsForMatcher))
	}

	// Save config to file if path is set
	if newConfig.ConfigPath != "" {
		if err := newConfig.SaveToFile(newConfig.ConfigPath); err != nil {
			log.Errorf("Failed to save config: %v", err)
		} else {
			log.Infof("Config saved to %s", newConfig.ConfigPath)
		}
	}

	// Prepare response with statistics
	totalDomains := len(a.manualDomains) + len(allGeositeDomains)
	response := map[string]interface{}{
		"success": true,
		"message": "Configuration updated successfully",
		"domain_stats": DomainStatistics{
			ManualDomains:     len(a.manualDomains),
			GeositeDomains:    len(allGeositeDomains),
			TotalDomains:      totalDomains,
			CategoryBreakdown: categoryStats,
		},
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	_ = enc.Encode(response)
}

func (a *API) updateMainConfig(newCfg *config.Config) {
	newCfg.ConfigPath = a.cfg.ConfigPath
	newCfg.WebServer.IsEnabled = a.cfg.WebServer.IsEnabled
	a.cfg = newCfg
}
