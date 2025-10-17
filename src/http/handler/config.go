package handler

import (
	"encoding/json"
	"net/http"

	"github.com/daniellavrushin/b4/config"
)

func RegisterConfigApi(mux *http.ServeMux, cfg *config.Config) {
	api := &API{cfg: cfg}
	mux.HandleFunc("/api/config", api.getConfig)
}

func (a *API) getConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	_ = enc.Encode(a.cfg)
}
