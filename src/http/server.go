package http

import (
	"embed"
	"fmt"
	stdhttp "net/http"
	"time"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/http/handler"
	"github.com/daniellavrushin/b4/log"
)

//go:embed ui/dist/*
var uiDist embed.FS

func StartServer(cfg *config.Config) (*stdhttp.Server, error) {
	if cfg.WebServer.Port == 0 {
		log.Infof("Web server disabled (port 0)")
		return nil, nil
	}
	mux := stdhttp.NewServeMux()

	mux.HandleFunc("/api/ws/logs", wsHandler)

	// Register internal http handlers
	handler.RegisterConfigApi(mux, cfg)
	handler.RegisterSpa(mux, uiDist)

	// Apply CORS middleware
	var handler stdhttp.Handler = mux
	handler = cors(handler)

	addr := fmt.Sprintf(":%d", cfg.WebServer.Port)
	log.Infof("Starting web server on %s", addr)
	srv := &stdhttp.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != stdhttp.ErrServerClosed {
			log.Errorf("Web server error: %v", err)
		}
	}()
	return srv, nil
}
