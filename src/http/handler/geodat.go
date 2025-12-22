package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/daniellavrushin/b4/log"
)

type GeodatDownloadRequest struct {
	GeositeURL      string `json:"geosite_url"`
	GeoipURL        string `json:"geoip_url"`
	DestinationPath string `json:"destination_path"`
}

type GeodatDownloadResponse struct {
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	GeositePath string `json:"geosite_path"`
	GeoipPath   string `json:"geoip_path"`
	GeositeSize int64  `json:"geosite_size"`
	GeoipSize   int64  `json:"geoip_size"`
}

type GeodatSource struct {
	Name       string `json:"name"`
	GeositeURL string `json:"geosite_url"`
	GeoipURL   string `json:"geoip_url"`
}

func (api *API) RegisterGeodatApi() {
	api.mux.HandleFunc("/api/geodat/download", api.handleGeodatDownload)
	api.mux.HandleFunc("/api/geodat/sources", api.handleGeodatSources)
	api.mux.HandleFunc("/api/geodat/info", api.handleFileInfo)
}

var geodatSources = []GeodatSource{
	{
		Name:       "Loyalsoldier",
		GeositeURL: "https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geosite.dat",
		GeoipURL:   "https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geoip.dat",
	},
	{
		Name:       "RUNET Freedom",
		GeositeURL: "https://raw.githubusercontent.com/runetfreedom/russia-v2ray-rules-dat/release/geosite.dat",
		GeoipURL:   "https://raw.githubusercontent.com/runetfreedom/russia-v2ray-rules-dat/release/geoip.dat",
	},
	{
		Name:       "Nidelon",
		GeositeURL: "https://github.com/Nidelon/ru-block-v2ray-rules/releases/latest/download/geosite.dat",
		GeoipURL:   "https://github.com/Nidelon/ru-block-v2ray-rules/releases/latest/download/geoip.dat",
	},
	{
		Name:       "DustinWin",
		GeositeURL: "https://github.com/DustinWin/ruleset_geodata/releases/download/mihomo/geosite.dat",
		GeoipURL:   "https://github.com/DustinWin/ruleset_geodata/releases/download/mihomo/geoip.dat",
	},
	{
		Name:       "Chocolate4U",
		GeositeURL: "https://raw.githubusercontent.com/Chocolate4U/Iran-v2ray-rules/release/geosite.dat",
		GeoipURL:   "https://raw.githubusercontent.com/Chocolate4U/Iran-v2ray-rules/release/geoip.dat",
	},
}

func (api *API) handleGeodatSources(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	setJsonHeader(w)
	json.NewEncoder(w).Encode(geodatSources)
}

func (api *API) handleGeodatDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req GeodatDownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.GeositeURL == "" || req.GeoipURL == "" || req.DestinationPath == "" {
		http.Error(w, "Geosite URL, GeoIP URL, and destination path required", http.StatusBadRequest)
		return
	}

	// Create destination directory
	if err := os.MkdirAll(req.DestinationPath, 0755); err != nil {
		log.Errorf("Failed to create directory: %v", err)
		http.Error(w, fmt.Sprintf("Failed to create directory: %v", err), http.StatusInternalServerError)
		return
	}

	geositePath := filepath.Join(req.DestinationPath, "geosite.dat")
	geoipPath := filepath.Join(req.DestinationPath, "geoip.dat")

	// Download geosite.dat
	geositeSize, err := downloadFile(req.GeositeURL, geositePath)
	if err != nil {
		log.Errorf("Failed to download geosite.dat: %v", err)
		http.Error(w, fmt.Sprintf("Failed to download geosite.dat: %v", err), http.StatusInternalServerError)
		return
	}

	// Download geoip.dat
	geoipSize, err := downloadFile(req.GeoipURL, geoipPath)
	if err != nil {
		log.Errorf("Failed to download geoip.dat: %v", err)
		http.Error(w, fmt.Sprintf("Failed to download geoip.dat: %v", err), http.StatusInternalServerError)
		return
	}

	// Update config
	api.cfg.System.Geo.GeoSitePath = geositePath
	api.cfg.System.Geo.GeoIpPath = geoipPath
	api.cfg.System.Geo.GeoSiteURL = req.GeositeURL
	api.cfg.System.Geo.GeoIpURL = req.GeoipURL

	if err := api.saveAndPushConfig(api.cfg); err != nil {
		log.Errorf("Failed to save config: %v", err)
		http.Error(w, "Failed to save configuration", http.StatusInternalServerError)
		return
	}

	api.geodataManager.UpdatePaths(geositePath, geoipPath)
	api.geodataManager.ClearCache()

	for _, set := range api.cfg.Sets {
		log.Infof("Reloading geo targets for set: %s", set.Name)
		api.loadTargetsForSetCached(set)
	}

	log.Infof("Downloaded geodat files: geosite.dat (%d bytes), geoip.dat (%d bytes)", geositeSize, geoipSize)

	response := GeodatDownloadResponse{
		Success:     true,
		Message:     "Geodat files downloaded successfully",
		GeositePath: geositePath,
		GeoipPath:   geoipPath,
		GeositeSize: geositeSize,
		GeoipSize:   geoipSize,
	}

	setJsonHeader(w)
	json.NewEncoder(w).Encode(response)
}

func downloadFile(url, filepath string) (int64, error) {
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return 0, err
	}
	defer out.Close()

	size, err := io.Copy(out, resp.Body)
	return size, err
}

func (api *API) handleFileInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "Path parameter required", http.StatusBadRequest)
		return
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			setJsonHeader(w)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"exists": false,
			})
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	setJsonHeader(w)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"exists":        true,
		"size":          info.Size(),
		"last_modified": info.ModTime().Format(time.RFC3339),
	})
}
