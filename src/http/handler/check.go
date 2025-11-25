package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/daniellavrushin/b4/checker"
	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/log"
	"github.com/daniellavrushin/b4/utils"
	"github.com/google/uuid"
)

func (api *API) RegisterCheckApi() {
	api.mux.HandleFunc("/api/check/start", api.handleStartCheck)
	api.mux.HandleFunc("/api/check/discovery", api.handleStartDiscovery)
	api.mux.HandleFunc("/api/check/status", api.handleCheckStatus)
	api.mux.HandleFunc("/api/check/cancel", api.handleCancelCheck)
	api.mux.HandleFunc("/api/check/add", api.handleAddPresetAsSet)
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

	if req.Timeout <= 0 {
		req.Timeout = time.Duration(chckCfg.TimeoutSeconds) * time.Second
	}
	if req.MaxConcurrent <= 0 {
		req.MaxConcurrent = chckCfg.MaxConcurrent
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
	config := checker.CheckConfig{
		CheckURL:      req.CheckURL,
		Timeout:       req.Timeout,
		MaxConcurrent: req.MaxConcurrent,
	}

	suite := checker.NewCheckSuite(config)

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

	suite, ok := checker.GetCheckSuite(testID)
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

	if err := checker.CancelCheckSuite(testID); err != nil {
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

	if req.Timeout <= 0 {
		req.Timeout = time.Duration(chckCfg.TimeoutSeconds) * time.Second
	}
	if req.MaxConcurrent <= 0 {
		req.MaxConcurrent = chckCfg.MaxConcurrent
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
	config := checker.CheckConfig{
		CheckURL:      req.CheckURL,
		Timeout:       req.Timeout,
		MaxConcurrent: req.MaxConcurrent,
	}

	presets := checker.GetTestPresets()

	suite := checker.NewDiscoverySuite(config, globalPool, presets)

	go func() {
		suite.RunDiscovery(domains)

		log.Infof("Discovery complete for %d domains", len(domains))
		log.Infof("\n%s", suite.GetDiscoveryReport())
	}()

	response := StartCheckResponse{
		Id:          suite.Id,
		TotalChecks: len(domains) * len(presets),
		Message:     fmt.Sprintf("Discovery started: %d domains × %d presets", len(domains), len(presets)),
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
