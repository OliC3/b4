package handler

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/daniellavrushin/b4/log"
)

func (api *API) RegisterIntegrationApi() {
	api.mux.HandleFunc("/api/integration/ipinfo", api.getIpInfo)
}

func (a *API) getIpInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	ip := r.URL.Query().Get("ip")
	if ip == "" {
		http.Error(w, "IP parameter required", http.StatusBadRequest)
		return
	}

	token := a.cfg.System.API.IPInfoToken
	if token == "" {
		http.Error(w, "IPInfo token not configured", http.StatusBadRequest)
		return
	}

	cleanIP := ip
	if idx := strings.Index(cleanIP, ":"); idx != -1 {
		cleanIP = cleanIP[:idx]
	}
	cleanIP = strings.Trim(cleanIP, "[]")

	url := fmt.Sprintf("https://ipinfo.io/%s?token=%s", cleanIP, token)
	resp, err := http.Get(url)
	if err != nil {
		log.Errorf("Failed to fetch IP info: %v", err)
		http.Error(w, "Failed to fetch IP info", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "IPInfo API error", resp.StatusCode)
		return
	}

	setJsonHeader(w)
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, resp.Body)
}
