package api

import (
	"encoding/json"
	"net/http"
	"time"

	"dhrishti/types"

	"github.com/gorilla/websocket"
)

/*
WSGraphHandler streams the latest graph state over WebSocket.

Why WebSockets here?

- Polling creates a "visibility tax": changes can only be seen on the next tick.
- Observability workflows want *reaction time* (close, fail, recover) to be near-real-time.

*/
type WSGraphHandler struct {
	Graph *types.Graph

	// How often we emit snapshots to connected clients.
	// 200ms feels snappy while staying lightweight.
	Interval time.Duration
}

var wsUpgrader = websocket.Upgrader{
	// Observability UIs are frequently served from a separate dev server.
	// We allow all origins for local development simplicity.
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (h *WSGraphHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	interval := h.Interval
	if interval <= 0 {
		interval = 200 * time.Millisecond
	}

	// Ensure dead clients don't keep goroutines alive forever.
	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Drain incoming control frames (ping/pong/close).
	go func() {
		for {
			if _, _, readErr := conn.ReadMessage(); readErr != nil {
				return
			}
		}
	}()

	push := func() bool {
		msg := BuildGraphResponse(h.Graph)
		return conn.WriteMessage(websocket.TextMessage, mustJSON(msg)) != nil
	}

	if push() {
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		if push() {
			return
		}
	}
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

