package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/geodat"
	"github.com/daniellavrushin/b4/log"
)

func RegisterGeositeApi(mux *http.ServeMux, cfg *config.Config) {
	api := &API{cfg: cfg}
	mux.HandleFunc("/api/geosite", api.handleGeoSite)
	mux.HandleFunc("/api/geosite/category", api.previewGeoCategory)
}

func (a *API) handleGeoSite(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.getGeositeTags(w)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
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
