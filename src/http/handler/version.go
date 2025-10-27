package handler

import (
	"encoding/json"
	"net/http"
)

// These variables are set at build time via ldflags
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

func (api *API) RegisterVersionApi() {
	api.mux.HandleFunc("/api/version", api.handleVersion)
}

func (api *API) handleVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	versionInfo := VersionInfo{
		Version:   Version,
		Commit:    Commit,
		BuildDate: Date,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	_ = enc.Encode(versionInfo)
}
