package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/daniellavrushin/b4/geodat"
	"github.com/daniellavrushin/b4/log"
)

func (api *API) RegisterGeositeApi() {
	api.mux.HandleFunc("/api/geosite", api.handleGeoSite)
	api.mux.HandleFunc("/api/geosite/category", api.previewGeoCategory)
	api.mux.HandleFunc("/api/geosite/domain", api.addGeositeDomain)
}

func (a *API) handleGeoSite(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.getGeositeTags(w)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (a *API) addGeositeDomain(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req AddDomainRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		log.Errorf("Failed to decode add domain request: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate domain
	if req.Domain == "" {
		http.Error(w, "Domain cannot be empty", http.StatusBadRequest)
		return
	}

	// Check if domain already exists in manual domains
	for _, existingDomain := range a.manualDomains {
		if existingDomain == req.Domain {
			response := AddDomainResponse{
				Success:      false,
				Message:      fmt.Sprintf("Domain '%s' already exists in manual domains", req.Domain),
				Domain:       req.Domain,
				TotalDomains: len(a.manualDomains),
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	// Add domain to manual domains
	a.manualDomains = append(a.manualDomains, req.Domain)
	log.Infof("Added domain '%s' to manual domains list", req.Domain)

	// Update the config with new manual domains
	a.cfg.Domains.SNIDomains = make([]string, len(a.manualDomains))
	copy(a.cfg.Domains.SNIDomains, a.manualDomains)

	// Prepare all domains for the matcher (manual + geosite)
	allGeositeDomains := []string{}
	for _, domains := range a.geositeDomains {
		allGeositeDomains = append(allGeositeDomains, domains...)
	}

	allDomainsForMatcher := make([]string, 0, len(a.manualDomains)+len(allGeositeDomains))
	allDomainsForMatcher = append(allDomainsForMatcher, a.manualDomains...)
	allDomainsForMatcher = append(allDomainsForMatcher, allGeositeDomains...)

	// Update the NFQ pool with new configuration
	if globalPool != nil {
		globalPool.UpdateConfig(a.cfg, allDomainsForMatcher)
		log.Infof("Updated NFQ pool with new domain (manual: %d, geosite: %d, total: %d)",
			len(a.manualDomains), len(allGeositeDomains), len(allDomainsForMatcher))
	}

	// Save config to file if path is set
	if a.cfg.ConfigPath != "" {
		if err := a.cfg.SaveToFile(a.cfg.ConfigPath); err != nil {
			log.Errorf("Failed to save config after adding domain: %v", err)
			// Don't return error to client, domain was added successfully
		} else {
			log.Infof("Config saved to %s after adding domain", a.cfg.ConfigPath)
		}
	}

	// Send success response
	response := AddDomainResponse{
		Success:       true,
		Message:       fmt.Sprintf("Successfully added domain '%s'", req.Domain),
		Domain:        req.Domain,
		TotalDomains:  len(a.manualDomains),
		ManualDomains: a.manualDomains,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (a *API) getGeositeTags(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)

	if a.cfg.Domains.GeoSitePath == "" {
		log.Tracef("Geosite path is not configured")
		_ = enc.Encode(GeositeResponse{Tags: []string{}})
		return
	}

	tags, err := geodat.ListGeoSiteTags(a.cfg.Domains.GeoSitePath)
	if err != nil {
		http.Error(w, "Failed to load geosite tags: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := GeositeResponse{
		Tags: tags,
	}

	_ = enc.Encode(response)
}

func (a *API) previewGeoCategory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	category := r.URL.Query().Get("tag")
	if category == "" {
		http.Error(w, "Tag category parameter required", http.StatusBadRequest)
		return
	}

	if a.cfg.Domains.GeoSitePath == "" {
		http.Error(w, "Geosite path not configured", http.StatusBadRequest)
		return
	}

	domains, err := geodat.LoadDomainsFromSites(a.cfg.Domains.GeoSitePath, []string{category})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load category: %v", err), http.StatusInternalServerError)
		return
	}

	// Limit preview to first 100 domains
	previewLimit := 100
	preview := domains
	if len(domains) > previewLimit {
		preview = domains[:previewLimit]
	}

	response := map[string]interface{}{
		"category":      category,
		"total_domains": len(domains),
		"preview_count": len(preview),
		"preview":       preview,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	_ = enc.Encode(response)
}

func (a *API) getGeoCategoryBreakdown() map[string]int {
	breakdown := make(map[string]int)
	for category, domains := range a.geositeDomains {
		breakdown[category] = len(domains)
	}
	return breakdown
}
