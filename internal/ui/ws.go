package ui

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

// Event 通过 /ws/events 发送的线上海报。仪表盘按 Type 切换渲染组件（toast / badge 等）。
type Event struct {
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload,omitempty"`
	At      time.Time      `json:"at"`
}

// Hub 进程内 pub/sub，把 Event 广播给所有连接的 WebSocket 客户端。
// 使用 gorilla/websocket，使仪表盘无需轮询即可实时渲染场景+抓包更新。
type Hub struct {
	log      *slog.Logger
	mu       sync.RWMutex
	clients  map[*client]struct{}
	register chan *client
	leave    chan *client
	outbox   chan Event
}

type client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

func newHub(l *slog.Logger) *Hub {
	if l == nil {
		l = slog.Default()
	}
	return &Hub{
		log:      l,
		clients:  map[*client]struct{}{},
		register: make(chan *client, 16),
		leave:    make(chan *client, 16),
		outbox:   make(chan Event, 64),
	}
}

func (h *Hub) run() {
	for {
		select {
		case c := <-h.register:
			h.mu.Lock()
			h.clients[c] = struct{}{}
			h.mu.Unlock()
		case c := <-h.leave:
			h.mu.Lock()
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}
			h.mu.Unlock()
		case ev := <-h.outbox:
			payload, err := json.Marshal(ev)
			if err != nil {
				continue
			}
			h.mu.RLock()
			for c := range h.clients {
				select {
				case c.send <- payload:
				default:
					// Slow consumer — drop the event rather than blocking.
				}
			}
			h.mu.RUnlock()
		}
	}
}

// broadcast 把事件入队并广播给所有已连接客户端。非阻塞（队列满时直接丢弃）。
func (h *Hub) broadcast(ev Event) {
	if ev.At.IsZero() {
		ev.At = time.Now().UTC()
	}
	select {
	case h.outbox <- ev:
	default:
		// Drop when the queue is full; the dashboard is best-effort.
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// serve 把 HTTP 连接升级为 WebSocket 并注册到 hub。升级失败返回错误。
func (h *Hub) serve(c echo.Context) error {
	conn, err := upgrader.Upgrade(c.Response().Writer, c.Request(), nil)
	if err != nil {
		return err
	}
	cl := &client{hub: h, conn: conn, send: make(chan []byte, 32)}
	h.register <- cl

	go cl.writePump()
	go cl.readPump()
	return nil
}

func (c *client) readPump() {
	defer func() {
		c.hub.leave <- c
		_ = c.conn.Close()
	}()
	c.conn.SetReadLimit(1024)
	_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	for {
		// We don't expect inbound messages; reading keeps the connection alive.
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (c *client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
