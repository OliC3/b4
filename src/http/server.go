// src/http/server.go
package http

import (
	"embed"
	"fmt"
	"io"
	stdhttp "net/http"
	"time"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/http/handler"
	"github.com/daniellavrushin/b4/http/ws"
	"github.com/daniellavrushin/b4/log"
	"github.com/daniellavrushin/b4/nfq"
)

//go:embed ui/dist/*
var uiDist embed.FS

func StartServer(cfg *config.Config, pool *nfq.Pool) (*stdhttp.Server, error) {
	if cfg.WebServer.Port == 0 {
		log.Infof("Web server disabled (port 0)")
		return nil, nil
	}

	mux := stdhttp.NewServeMux()

	// Register WebSocket endpoints
	handler.SetNFQPool(pool)
	registerWebSocketEndpoints(mux)

	// Register REST API endpoints
	registerAPIEndpoints(mux, cfg)

	// Register SPA (Single Page Application)
	handler.RegisterSpa(mux, uiDist)

	// Apply CORS middleware
	var httpHandler stdhttp.Handler = mux
	httpHandler = cors(httpHandler)

	addr := fmt.Sprintf(":%d", cfg.WebServer.Port)
	log.Infof("Starting web server on %s", addr)

	// Record startup event
	metrics := handler.GetMetricsCollector()
	metrics.RecordEvent("info", fmt.Sprintf("Web server started on port %d", cfg.WebServer.Port))

	srv := &stdhttp.Server{
		Addr:              addr,
		Handler:           httpHandler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != stdhttp.ErrServerClosed {
			log.Errorf("Web server error: %v", err)
			metrics := handler.GetMetricsCollector()
			metrics.RecordEvent("error", fmt.Sprintf("Web server error: %v", err))
		}
	}()

	return srv, nil
}

// registerWebSocketEndpoints registers all WebSocket handlers
func registerWebSocketEndpoints(mux *stdhttp.ServeMux) {
	// WebSocket endpoint for log streaming
	mux.HandleFunc("/api/ws/logs", ws.HandleLogsWebSocket)

	// WebSocket endpoint for real-time metrics
	mux.HandleFunc("/api/ws/metrics", ws.HandleMetricsWebSocket)

	log.Tracef("WebSocket endpoints registered: /api/ws/logs, /api/ws/metrics")
}

// registerAPIEndpoints registers all REST API handlers
func registerAPIEndpoints(mux *stdhttp.ServeMux, cfg *config.Config) {

	api := handler.NewAPIHandler(cfg)
	api.RegisterEndpoints(mux, cfg)

	log.Tracef("REST API endpoints registered")
}

func LogWriter() io.Writer {
	return ws.LogWriter()
}

func Shutdown() {
	// Shutdown the log hub
	ws.Shutdown()
}
