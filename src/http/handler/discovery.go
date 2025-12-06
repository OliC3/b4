package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/discovery"
	"github.com/daniellavrushin/b4/log"
	"github.com/google/uuid"
)

func (api *API) RegisterDiscoveryApi() {
	api.mux.HandleFunc("/api/discovery", api.handleStartDiscovery)
	api.mux.HandleFunc("/api/discovery/status", api.handleCheckStatus)
	api.mux.HandleFunc("/api/discovery/cancel", api.handleCancelCheck)
	api.mux.HandleFunc("/api/discovery/add", api.handleAddPresetAsSet)
	api.mux.HandleFunc("/api/discovery/similar", api.handleFindSimilarSets)
	api.mux.HandleFunc("/api/config/sets/", api.handleSetDomains)
	api.mux.HandleFunc("/api/discovery/fingerprint", api.handleFingerprint)

}

func (api *API) handleCheckStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	testID := r.URL.Query().Get("id")
	if testID == "" {
		http.Error(w, "Check ID required", http.StatusBadRequest)
		return
	}

	suite, ok := discovery.GetCheckSuite(testID)
	if !ok {
		http.Error(w, "Check suite not found", http.StatusNotFound)
		return
	}

	snapshot := suite.GetSnapshot()

	setJsonHeader(w)
	json.NewEncoder(w).Encode(snapshot)
}

func (api *API) handleCancelCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	testID := r.URL.Query().Get("id")
	if testID == "" {
		http.Error(w, "Check ID required", http.StatusBadRequest)
		return
	}

	if err := discovery.CancelCheckSuite(testID); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	log.Infof("Canceled test suite %s", testID)

	setJsonHeader(w)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Check suite canceled",
	})
}

func (api *API) handleStartDiscovery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req DiscoveryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Errorf("Failed to decode discovery request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Domain == "" {
		http.Error(w, "Domain is required", http.StatusBadRequest)
		return
	}

	suite := discovery.NewDiscoverySuite(req.CheckURL, globalPool, req.Domain)

	phase1Count := len(discovery.GetPhase1Presets())

	go func() {
		suite.RunDiscovery()
		log.Infof("Discovery complete for %s", req.Domain)
	}()

	response := DiscoveryResponse{
		Id:             suite.Id,
		Domain:         req.Domain,
		EstimatedTests: phase1Count + 15, // rough estimate
		Message:        fmt.Sprintf("Discovery started for %s", req.Domain),
	}

	setJsonHeader(w)
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)
}

func (api *API) handleAddPresetAsSet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var set = config.NewSetConfig()

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&set); err != nil {
		log.Errorf("Failed to decode config update: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	set.Id = uuid.New().String()

	if len(set.Targets.SNIDomains) == 0 {
		log.Errorf("At least one SNI domain is required")
		http.Error(w, "At least one SNI domain is required", http.StatusBadRequest)
		return
	}
	if set.Name == "" {
		set.Name = set.Targets.SNIDomains[0]
	}

	api.loadTargetsForSetCached(&set)

	// Ensure all target arrays are initialized (not null)
	if set.Targets.IPs == nil {
		set.Targets.IPs = []string{}
	}
	if set.Targets.IpsToMatch == nil {
		set.Targets.IpsToMatch = []string{}
	}
	if set.Targets.GeoSiteCategories == nil {
		set.Targets.GeoSiteCategories = []string{}
	}
	if set.Targets.GeoIpCategories == nil {
		set.Targets.GeoIpCategories = []string{}
	}

	// Ensure TCP WinValues is initialized
	if set.TCP.WinValues == nil {
		set.TCP.WinValues = []int{0, 1460, 8192, 65535}
	}
	if set.TCP.WinMode == "" {
		set.TCP.WinMode = "off"
	}
	if set.TCP.DesyncMode == "" {
		set.TCP.DesyncMode = "off"
	}

	// Ensure Faking SNIMutation is initialized
	if set.Faking.SNIMutation.Mode == "" {
		set.Faking.SNIMutation.Mode = "off"
	}
	if set.Faking.SNIMutation.FakeSNIs == nil {
		set.Faking.SNIMutation.FakeSNIs = []string{}
	}

	api.cfg.Sets = append([]*config.SetConfig{&set}, api.cfg.Sets...)

	if api.cfg.MainSet == nil {
		api.cfg.MainSet = &set
	}

	// Save configuration
	if err := api.saveAndPushConfig(api.cfg); err != nil {
		log.Errorf("Failed to save config: %v", err)
		http.Error(w, "Failed to save configuration", http.StatusInternalServerError)
		return
	}

	setJsonHeader(w)
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Added '%s' configuration", set.Name),
	})
}

func (api *API) handleFindSimilarSets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var incoming config.SetConfig
	if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	type SimilarSet struct {
		Id      string   `json:"id"`
		Name    string   `json:"name"`
		Domains []string `json:"domains"`
	}

	var similar []SimilarSet

	for _, set := range api.cfg.Sets {
		if !set.Enabled {
			continue
		}
		if setsHaveSimilarConfig(set, &incoming) {
			similar = append(similar, SimilarSet{
				Id:      set.Id,
				Name:    set.Name,
				Domains: set.Targets.SNIDomains,
			})
		}
	}

	setJsonHeader(w)
	json.NewEncoder(w).Encode(similar)
}

func (api *API) handleSetDomains(w http.ResponseWriter, r *http.Request) {

	path := r.URL.Path
	parts := strings.Split(strings.TrimPrefix(path, "/api/config/sets/"), "/")
	if len(parts) < 2 || parts[1] != "domains" {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	setId := parts[0]

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Domain string `json:"domain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Find set and add domain
	for _, set := range api.cfg.Sets {
		if set.Id == setId {
			set.Targets.SNIDomains = append(set.Targets.SNIDomains, req.Domain)
			set.Targets.DomainsToMatch = append(set.Targets.DomainsToMatch, req.Domain)

			if err := api.saveAndPushConfig(api.cfg); err != nil {
				http.Error(w, "Failed to save", http.StatusInternalServerError)
				return
			}

			setJsonHeader(w)
			json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
			return
		}
	}

	http.Error(w, "Set not found", http.StatusNotFound)
}

func setsHaveSimilarConfig(a, b *config.SetConfig) bool {
	return a.Fragmentation.Strategy == b.Fragmentation.Strategy &&
		a.Fragmentation.ReverseOrder == b.Fragmentation.ReverseOrder &&
		a.Fragmentation.MiddleSNI == b.Fragmentation.MiddleSNI &&
		a.Faking.Strategy == b.Faking.Strategy &&
		a.Faking.TTL == b.Faking.TTL &&
		a.Faking.SNI == b.Faking.SNI &&
		a.TCP.DropSACK == b.TCP.DropSACK
}

func (api *API) handleFingerprint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Domain string `json:"domain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Domain == "" {
		http.Error(w, "Domain required", http.StatusBadRequest)
		return
	}

	timeout := time.Duration(api.cfg.System.Checker.DiscoveryTimeoutSec) * time.Second
	prober := discovery.NewDPIProber(req.Domain, timeout)

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	fingerprint := prober.Fingerprint(ctx)

	setJsonHeader(w)
	json.NewEncoder(w).Encode(fingerprint.ToJSON())
}
