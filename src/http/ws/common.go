// src/http/ws/common.go
package ws

import (
	"net/http"

	"github.com/gorilla/websocket"
)

// Upgrader is a shared WebSocket upgrader configuration
var Upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from any origin
		// In production, you might want to restrict this
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}
