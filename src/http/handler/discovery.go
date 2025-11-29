package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/discovery"
	"github.com/daniellavrushin/b4/log"
	"github.com/daniellavrushin/b4/utils"
	"github.com/google/uuid"
)

func (api *API) RegisterDiscoveryApi() {
	api.mux.HandleFunc("/api/discovery", api.handleStartDiscovery)
	api.mux.HandleFunc("/api/discovery/start", api.handleStartCheck)
	api.mux.HandleFunc("/api/discovery/status", api.handleCheckStatus)
	api.mux.HandleFunc("/api/discovery/cancel", api.handleCancelCheck)
	api.mux.HandleFunc("/api/discovery/add", api.handleAddPresetAsSet)
}

func (api *API) handleStartCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	chckCfg := &api.cfg.System.Checker

	var req StartCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Errorf("Failed to decode check request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	domains := req.Domains
	if len(domains) == 0 {
		if len(api.cfg.Sets) > 0 {
			for _, set := range api.cfg.Sets {
				if len(set.Targets.SNIDomains) > 0 {
					domains = append(domains, set.Targets.SNIDomains...)
				}
			}
		}
		domains = append(domains, chckCfg.Domains...)
		domains = utils.FilterUniqueStrings(domains)
	}

	if len(domains) == 0 {
		http.Error(w, "No domains provided. Please specify domains to test.", http.StatusBadRequest)
		return
	}
	config := discovery.CheckConfig{
		CheckURL:               req.CheckURL,
		Timeout:                time.Duration(api.cfg.System.Checker.DiscoveryTimeoutSec) * time.Second,
		ConfigPropagateTimeout: time.Duration(api.cfg.System.Checker.ConfigPropagateMs),
	}

	suite := discovery.NewCheckSuite(config)

	go suite.Run(domains)

	response := StartCheckResponse{
		Id:          suite.Id,
		TotalChecks: len(domains),
		Message:     "Check suite started",
	}

	setJsonHeader(w)
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)
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

	chckCfg := &api.cfg.System.Checker

	var req StartCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Errorf("Failed to decode discovery request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	domains := req.Domains
	if len(domains) == 0 {
		if len(api.cfg.Sets) > 0 {
			for _, set := range api.cfg.Sets {
				if len(set.Targets.SNIDomains) > 0 {
					domains = append(domains, set.Targets.SNIDomains...)
				}
			}
		}
		domains = append(domains, chckCfg.Domains...)
		domains = utils.FilterUniqueStrings(domains)
	}

	if len(domains) == 0 {
		http.Error(w, "No domains provided. Please specify domains to test.", http.StatusBadRequest)
		return
	}

	config := discovery.CheckConfig{
		CheckURL:               req.CheckURL,
		Timeout:                time.Duration(api.cfg.System.Checker.DiscoveryTimeoutSec) * time.Second,
		ConfigPropagateTimeout: time.Duration(api.cfg.System.Checker.ConfigPropagateMs),
	}

	// Pass geodata manager for geosite-based clustering
	suite := discovery.NewDiscoverySuite(config, globalPool)

	clusters := discovery.ClusterByKnownCDN(domains)

	phase1Count := len(discovery.GetPhase1Presets())
	estimatedTests := len(clusters) * (phase1Count + 10)

	go func() {
		suite.RunDiscovery(domains)

		log.Infof("Discovery complete for %d domains (%d clusters)", len(domains), len(clusters))
		log.Infof("\n%s", suite.GetDiscoveryReport())
	}()

	response := DiscoveryStartResponse{
		Id:             suite.Id,
		TotalDomains:   len(domains),
		TotalClusters:  len(clusters),
		EstimatedTests: estimatedTests,
		Message: fmt.Sprintf("Hierarchical discovery started: %d domains in %d clusters (~%d tests)",
			len(domains), len(clusters), estimatedTests),
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

	var set config.SetConfig

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
	set.Name = set.Targets.SNIDomains[0]
	set.Targets.DomainsToMatch = []string{set.Targets.SNIDomains[0]}

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

// DiscoveryStartResponse includes cluster information
type DiscoveryStartResponse struct {
	Id             string `json:"id"`
	TotalDomains   int    `json:"total_domains"`
	TotalClusters  int    `json:"total_clusters"`
	EstimatedTests int    `json:"estimated_tests"`
	Message        string `json:"message"`
}
