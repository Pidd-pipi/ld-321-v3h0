package ws

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/agridispatch/agridispatch/internal/constants"
	"github.com/gorilla/websocket"
)

// Hub WebSocket 连接管理与农机实时位置推送。
type Hub struct {
	clients map[*websocket.Conn]bool
	mu      sync.Mutex
	logger  *slog.Logger
	upgrader websocket.Upgrader
}

// NewHub 创建连接管理器。
func NewHub(logger *slog.Logger) *Hub {
	return &Hub{
		clients: make(map[*websocket.Conn]bool),
		logger:  logger,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

// ServeWS 升级 HTTP 为 WebSocket 并加入推送。
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Warn("websocket upgrade failed", "err", err)
		return
	}
	h.mu.Lock()
	h.clients[conn] = true
	h.mu.Unlock()
	h.logger.Info("websocket client connected", "remote", conn.RemoteAddr().String())

	defer func() {
		h.mu.Lock()
		delete(h.clients, conn)
		h.mu.Unlock()
		_ = conn.Close()
	}()

	// 读取循环（丢弃客户端消息，保持连接）
	conn.SetReadDeadline(time.Now().Add(120 * time.Second))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(120 * time.Second))
		return nil
	})
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				_ = conn.Close()
				return
			}
		}
	}()
	// 定时推送模拟轨迹
	ticker := time.NewTicker(constants.WSPushIntervalSeconds * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		msg := map[string]interface{}{
			"type":   "track",
			"time":   time.Now().Format(time.RFC3339),
			"points": h.sampleTracks(),
		}
		data, _ := json.Marshal(msg)
		h.mu.Lock()
		for c := range h.clients {
			if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
				h.logger.Warn("ws write failed", "err", err)
				_ = c.Close()
				delete(h.clients, c)
			}
		}
		h.mu.Unlock()
	}
}

// sampleTracks 生成模拟实时轨迹点。
func (h *Hub) sampleTracks() []map[string]interface{} {
	base := []struct {
		Code string
		Lon  float64
		Lat  float64
	}{
		{"NJ-2026-001", 116.316, 39.985},
		{"NJ-2026-002", 116.301, 39.972},
	}
	now := time.Now().Format("2006-01-02 15:04")
	out := make([]map[string]interface{}, 0, len(base))
	for i, b := range base {
		out = append(out, map[string]interface{}{
			"machineCode":   b.Code,
			"taskType":      []string{"耕地", "播种"}[i],
			"capturedAt":    now,
			"longitude":     b.Lon + float64(i)*0.001,
			"latitude":      b.Lat + float64(i)*0.001,
			"speed":         6.0 + float64(i)*2.0,
			"fieldBoundary": []string{"北岭 1 号田", "西坡旱地"}[i],
		})
	}
	return out
}
