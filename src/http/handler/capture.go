// src/http/handler/capture.go
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/daniellavrushin/b4/capture"
	"github.com/daniellavrushin/b4/log"
)

type StartCaptureRequest struct {
	Domain     string `json:"domain"`   // Domain to capture (or "*" for all)
	Protocol   string `json:"protocol"` // "tls", "quic", or "both"
	MaxPackets int    `json:"max_packets"`
}

type CaptureSessionResponse struct {
	ID         string            `json:"id"`
	Active     bool              `json:"active"`
	Count      int               `json:"count"`
	MaxPackets int               `json:"max_packets"`
	Captures   []capture.Capture `json:"captures"`
}

func (api *API) RegisterCaptureApi() {
	api.mux.HandleFunc("/api/capture/start", api.handleStartCapture)
	api.mux.HandleFunc("/api/capture/stop", api.handleStopCapture)
	api.mux.HandleFunc("/api/capture/status", api.handleCaptureStatus)
	api.mux.HandleFunc("/api/capture/list", api.handleListCaptures)
	api.mux.HandleFunc("/api/capture/download", api.handleDownloadCapture)
}

func (api *API) handleStartCapture(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req StartCaptureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Set defaults
	if req.Protocol == "" {
		req.Protocol = "both"
	}
	if req.MaxPackets <= 0 {
		req.MaxPackets = 5
	}
	if req.Domain == "" {
		req.Domain = "*"
	}

	manager := capture.GetManager()
	session, err := manager.StartSession(req.Domain, req.Protocol, req.MaxPackets)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Infof("Started capture session %s for domain '%s', protocol '%s', max %d packets",
		session.ID, req.Domain, req.Protocol, req.MaxPackets)

	setJsonHeader(w)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"session_id": session.ID,
		"message":    "Capture session started",
	})
}

func (api *API) handleStopCapture(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.URL.Query().Get("id")
	if sessionID == "" {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}

	manager := capture.GetManager()
	if err := manager.StopSession(sessionID); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	setJsonHeader(w)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Capture session stopped",
	})
}

func (api *API) handleCaptureStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.URL.Query().Get("id")
	if sessionID == "" {
		// Return all sessions
		manager := capture.GetManager()
		sessions := manager.ListSessions()

		setJsonHeader(w)
		json.NewEncoder(w).Encode(sessions)
		return
	}

	// Return specific session
	manager := capture.GetManager()
	session, ok := manager.GetSession(sessionID)
	if !ok {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	setJsonHeader(w)
	json.NewEncoder(w).Encode(session)
}

func (api *API) handleListCaptures(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	manager := capture.GetManager()
	sessions := manager.ListSessions()

	allCaptures := []capture.Capture{}
	for _, session := range sessions {
		allCaptures = append(allCaptures, session.Captures...)
	}

	setJsonHeader(w)
	json.NewEncoder(w).Encode(allCaptures)
}
func (api *API) handleDownloadCapture(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Query().Get("file")
	if filePath == "" {
		http.Error(w, "File path required", http.StatusBadRequest)
		return
	}

	// Security: ensure file is in captures directory
	capturesDir := "./captures"
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		http.Error(w, "Invalid file path", http.StatusBadRequest)
		return
	}

	absCapturesDir, _ := filepath.Abs(capturesDir)
	if !strings.HasPrefix(absPath, absCapturesDir) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Check if file exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Extract filename from path
	filename := filepath.Base(absPath)

	// Set headers for download
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.Header().Set("Content-Type", "application/octet-stream")

	http.ServeFile(w, r, absPath)
}
