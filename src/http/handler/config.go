// src/http/handler/config.go
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/log"
	"github.com/daniellavrushin/b4/metrics"
	"github.com/daniellavrushin/b4/utils"
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

	totalDomains := 0
	categories := []string{}
	for _, set := range a.cfg.Sets {
		totalDomains += len(set.Domains.DomainsToMatch)

		categories = append(categories, set.Domains.GeoSiteCategories...)

	}
	categoryBreakdown, _ := a.geodataManager.GetCategoryCounts(utils.FilterUniqueStrings(categories))

	response := ConfigResponse{
		Config: a.cfg,
		DomainStats: DomainStatistics{
			TotalDomains:      totalDomains,
			GeositeAvailable:  a.geodataManager.IsConfigured(),
			CategoryBreakdown: categoryBreakdown,
		},
	}

	configCopy := *a.cfg
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

	a.geodataManager.UpdatePaths(newConfig.System.Geo.GeoSitePath, newConfig.System.Geo.GeoIpPath)

	allDomainsCount := 0
	categories := []string{}

	if newConfig.System.Geo.GeoSitePath != "" {

		for _, set := range a.cfg.Sets {
			_, err := a.applyDomainChanges(set)
			if err != nil {
				log.Errorf("Failed to apply domain changes for set '%s': %v", set.Name, err)
			}

			allDomainsCount += len(set.Domains.DomainsToMatch)
			categories = append(categories, set.Domains.GeoSiteCategories...)
			log.Infof("Loaded %d domains from geodata for set '%s'", allDomainsCount, set.Name)
		}

		m := metrics.GetMetricsCollector()
		m.RecordEvent("info", fmt.Sprintf("Loaded %d domains from geodata across %d sets",
			allDomainsCount, len(a.cfg.Sets)))
	} else if allDomainsCount == 0 {
		a.geodataManager.ClearCache()
		log.Infof("Cleared all geosite domains")
	}

	categoryBreakdown, _ := a.geodataManager.GetCategoryCounts(utils.FilterUniqueStrings(categories))

	response := map[string]interface{}{
		"success": true,
		"message": "Configuration updated successfully",
		"domain_stats": DomainStatistics{
			TotalDomains:      allDomainsCount,
			CategoryBreakdown: categoryBreakdown,
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

	defaultCfg := config.DefaultConfig
	defaultCfg.System.Checker = a.cfg.System.Checker
	defaultCfg.ConfigPath = a.cfg.ConfigPath
	defaultCfg.System.WebServer.IsEnabled = a.cfg.System.WebServer.IsEnabled

	for _, set := range a.cfg.Sets {
		defaultCfg.Sets = append(defaultCfg.Sets, set)
		set.ResetToDefaults()
		_, err := a.applyDomainChanges(set)
		if err != nil {
			log.Errorf("Failed to apply domain changes for set '%s': %v", set.Name, err)
		}
	}

	defaultCfg.MainSet.Domains = a.cfg.MainSet.Domains

	setJsonHeader(w)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Configuration reset to defaults (domains and checker preserved)",
	})
}

type domainStats struct {
	ManualDomains  int
	GeositeDomains int
	TotalDomains   int
}

func (a *API) applyDomainChanges(cfg *config.SetConfig) (domainStats, error) {
	var err error
	domains, err := a.cfg.GetDomainsForSet(cfg)
	if err != nil {
		return domainStats{}, log.Errorf("Failed to load set domains: %v", err)
	}

	geositeDomainsCount := len(cfg.Domains.DomainsToMatch) - len(cfg.Domains.SNIDomains)
	if globalPool != nil {
		globalPool.UpdateConfig(a.cfg)
		log.Infof("Config pushed to all workers (manual: %d, geosite: %d, total unique: %d domains)",
			len(cfg.Domains.SNIDomains), geositeDomainsCount, len(domains))
	}

	if a.cfg.ConfigPath != "" {
		if err := a.cfg.SaveToFile(a.cfg.ConfigPath); err != nil {
			log.Errorf("Failed to save config: %v", err)
		} else {
			log.Infof("Config saved to %s", a.cfg.ConfigPath)
		}
	}

	return domainStats{
		ManualDomains:  len(cfg.Domains.SNIDomains),
		GeositeDomains: geositeDomainsCount,
		TotalDomains:   len(cfg.Domains.DomainsToMatch),
	}, nil
}
