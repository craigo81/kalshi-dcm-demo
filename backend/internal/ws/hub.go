// Package ws provides WebSocket support for real-time market updates.
// Core Principle 9: Transparent, real-time market information.
package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kalshi-dcm-demo/backend/internal/kalshi"
)

// =============================================================================
// WEBSOCKET UPGRADER
// =============================================================================

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production: Validate origin
		return true
	},
}

// =============================================================================
// MESSAGE TYPES
// =============================================================================

type MessageType string

const (
	MsgTypeSubscribe   MessageType = "subscribe"
	MsgTypeUnsubscribe MessageType = "unsubscribe"
	MsgTypeMarketData  MessageType = "market_data"
	MsgTypeOrderbook   MessageType = "orderbook"
	MsgTypeError       MessageType = "error"
	MsgTypePing        MessageType = "ping"
	MsgTypePong        MessageType = "pong"
)

type WSMessage struct {
	Type    MessageType     `json:"type"`
	Channel string          `json:"channel,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// =============================================================================
// CLIENT
// =============================================================================

type Client struct {
	hub          *Hub
	conn         *websocket.Conn
	send         chan []byte
	subscriptions map[string]bool
	mu           sync.RWMutex
}

func NewClient(hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		hub:          hub,
		conn:         conn,
		send:         make(chan []byte, 256),
		subscriptions: make(map[string]bool),
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		var msg WSMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case MsgTypeSubscribe:
			c.mu.Lock()
			c.subscriptions[msg.Channel] = true
			c.mu.Unlock()
		case MsgTypeUnsubscribe:
			c.mu.Lock()
			delete(c.subscriptions, msg.Channel)
			c.mu.Unlock()
		case MsgTypePing:
			pong, _ := json.Marshal(WSMessage{Type: MsgTypePong})
			c.send <- pong
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) isSubscribed(channel string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.subscriptions[channel]
}

// =============================================================================
// HUB - Manages all WebSocket connections
// =============================================================================

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	kalshi     *kalshi.Client
	mu         sync.RWMutex
}

func NewHub(kalshiClient *kalshi.Client) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		kalshi:     kalshiClient,
	}
}

func (h *Hub) Run() {
	// Start market data polling
	go h.pollMarketData()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// pollMarketData fetches and broadcasts market updates.
// Core Principle 9: Real-time market transparency.
func (h *Hub) pollMarketData() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Fetch open markets
		response, err := h.kalshi.GetMarkets(kalshi.MarketParams{
			Status: "open",
			Limit:  50,
		})
		if err != nil {
			log.Printf("Market poll error: %v", err)
			continue
		}

		// Broadcast to subscribed clients
		for _, market := range response.Markets {
			channel := "market:" + market.Ticker
			data, _ := json.Marshal(market.ToMarket())

			msg, _ := json.Marshal(WSMessage{
				Type:    MsgTypeMarketData,
				Channel: channel,
				Data:    data,
			})

			h.mu.RLock()
			for client := range h.clients {
				if client.isSubscribed(channel) || client.isSubscribed("market:*") {
					select {
					case client.send <- msg:
					default:
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// ServeWS handles WebSocket upgrade requests.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := NewClient(h, conn)
	h.register <- client

	go client.writePump()
	go client.readPump()
}
