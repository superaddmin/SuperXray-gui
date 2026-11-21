// Package websocket provides WebSocket hub for real-time updates and notifications.
package websocket

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/superaddmin/SuperXray-gui/v2/logger"
)

// MessageType represents the type of WebSocket message
type MessageType string

const (
	MessageTypeStatus       MessageType = "status"       // Server status update
	MessageTypeTraffic      MessageType = "traffic"      // Traffic statistics update
	MessageTypeInbounds     MessageType = "inbounds"     // Inbounds list update
	MessageTypeNotification MessageType = "notification" // System notification
	MessageTypeXrayState    MessageType = "xray_state"   // Xray state change
	MessageTypeOutbounds    MessageType = "outbounds"    // Outbounds list update
	MessageTypeInvalidate   MessageType = "invalidate"   // Lightweight signal telling frontend to re-fetch data via REST
)

// Message represents a WebSocket message
type Message struct {
	Type    MessageType `json:"type"`
	Payload any         `json:"payload"`
	Time    int64       `json:"time"`
}

// Client represents a WebSocket client connection
type Client struct {
	ID     string
	Send   chan []byte
	Hub    *Hub
	Topics map[MessageType]bool // Subscribed topics
	closed bool                 // Protected by the owning hub's mutex
}

// Hub maintains the set of active clients and broadcasts messages to them
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Inbound messages from clients
	broadcast chan []byte

	// Mutex for thread-safe operations
	mu sync.RWMutex

	// Context for graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	ctx, cancel := context.WithCancel(context.Background())

	return &Hub{
		clients:   make(map[*Client]bool),
		broadcast: make(chan []byte, 2048), // Increased from 256 to 2048 for high load
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("WebSocket hub panic recovered:", r)
			// Restart the hub loop
			go h.Run()
		}
	}()

	for {
		select {
		case <-h.ctx.Done():
			// Graceful shutdown: close all clients
			h.mu.Lock()
			for client := range h.clients {
				h.removeClientLocked(client)
			}
			h.mu.Unlock()
			logger.Info("WebSocket hub stopped gracefully")
			return

		case message := <-h.broadcast:
			if message == nil {
				continue
			}
			h.deliver(message, "")
		}
	}
}

// Enqueuing is nonblocking. Membership, sends, and channel closure share one lock.
func (h *Hub) deliver(message []byte, topic MessageType) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for client := range h.clients {
		if topic != "" && len(client.Topics) != 0 && !client.Topics[topic] {
			continue
		}
		select {
		case client.Send <- message:
		default:
			h.removeClientLocked(client)
		}
	}
}

func (h *Hub) removeClientLocked(client *Client) {
	delete(h.clients, client)
	if !client.closed {
		client.closed = true
		close(client.Send)
	}
}

// Broadcast sends a message to all connected clients
func (h *Hub) Broadcast(messageType MessageType, payload any) {
	if h == nil {
		return
	}
	if payload == nil {
		logger.Warning("Attempted to broadcast nil payload")
		return
	}

	// Skip all work if no clients are connected
	if h.GetClientCount() == 0 {
		return
	}

	msg := Message{
		Type:    messageType,
		Payload: payload,
		Time:    getCurrentTimestamp(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		logger.Error("Failed to marshal WebSocket message:", err)
		return
	}

	// If message exceeds size limit, send a lightweight invalidate notification
	// instead of dropping it entirely — the frontend will re-fetch via REST API
	const maxMessageSize = 10 * 1024 * 1024 // 10MB
	if len(data) > maxMessageSize {
		logger.Debugf("WebSocket message too large (%d bytes) for type %s, sending invalidate signal", len(data), messageType)
		h.broadcastInvalidate(messageType)
		return
	}

	// Non-blocking send with timeout to prevent delays
	select {
	case h.broadcast <- data:
	case <-time.After(100 * time.Millisecond):
		logger.Warning("WebSocket broadcast channel is full, dropping message")
	case <-h.ctx.Done():
		// Hub is shutting down
	}
}

// BroadcastToTopic sends a message only to clients subscribed to the specific topic
func (h *Hub) BroadcastToTopic(messageType MessageType, payload any) {
	if h == nil {
		return
	}
	if payload == nil {
		logger.Warning("Attempted to broadcast nil payload to topic")
		return
	}

	// Skip all work if no clients are connected
	if h.GetClientCount() == 0 {
		return
	}

	msg := Message{
		Type:    messageType,
		Payload: payload,
		Time:    getCurrentTimestamp(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		logger.Error("Failed to marshal WebSocket message:", err)
		return
	}

	// If message exceeds size limit, send a lightweight invalidate notification
	const maxMessageSize = 10 * 1024 * 1024 // 10MB
	if len(data) > maxMessageSize {
		logger.Debugf("WebSocket message too large (%d bytes) for type %s, sending invalidate signal", len(data), messageType)
		h.broadcastInvalidate(messageType)
		return
	}

	h.deliver(data, messageType)
}

// GetClientCount returns the number of connected clients
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// Register registers a new client with the hub
func (h *Hub) Register(client *Client) {
	if h == nil || client == nil || client.Send == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if client.closed {
		return
	}
	if h.ctx.Err() != nil {
		h.removeClientLocked(client)
		return
	}
	h.clients[client] = true
}

// Unregister unregisters a client from the hub
func (h *Hub) Unregister(client *Client) {
	if h == nil || client == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[client] {
		h.removeClientLocked(client)
	}
}

// Stop gracefully stops the hub and closes all connections
func (h *Hub) Stop() {
	if h == nil {
		return
	}
	if h.cancel != nil {
		h.cancel()
	}
}

// broadcastInvalidate sends a lightweight invalidate message to all clients,
// telling them to re-fetch the specified data type via REST API.
// This is used when the full payload exceeds the WebSocket message size limit.
func (h *Hub) broadcastInvalidate(originalType MessageType) {
	msg := Message{
		Type:    MessageTypeInvalidate,
		Payload: map[string]string{"type": string(originalType)},
		Time:    getCurrentTimestamp(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		logger.Error("Failed to marshal invalidate message:", err)
		return
	}

	// Non-blocking send with timeout
	select {
	case h.broadcast <- data:
	case <-time.After(100 * time.Millisecond):
		logger.Warning("WebSocket broadcast channel is full, dropping invalidate message")
	case <-h.ctx.Done():
	}
}

// getCurrentTimestamp returns current Unix timestamp in milliseconds
func getCurrentTimestamp() int64 {
	return time.Now().UnixMilli()
}
