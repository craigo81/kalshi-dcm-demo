// Surveillance & Risk Operator Dashboard
// CFTC Core Principle 4: Prevention of Market Disruption
// Provides real-time monitoring and control for compliance officers.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
)

// =============================================================================
// CONFIGURATION
// =============================================================================

type Config struct {
	Port            string
	BackendAPIURL   string // Main DCM demo API
	RefreshInterval time.Duration
}

func loadConfig() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}

	backendURL := os.Getenv("BACKEND_API_URL")
	if backendURL == "" {
		backendURL = "http://localhost:8080/api/v1"
	}

	return &Config{
		Port:            port,
		BackendAPIURL:   backendURL,
		RefreshInterval: 5 * time.Second,
	}
}

// =============================================================================
// DATA MODELS
// =============================================================================

// Alert represents a compliance alert
type Alert struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"`
	Severity     string    `json:"severity"`
	UserID       string    `json:"user_id"`
	MarketTicker string    `json:"market_ticker"`
	Description  string    `json:"description"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	ResolvedAt   *time.Time `json:"resolved_at,omitempty"`
	ResolvedBy   string    `json:"resolved_by,omitempty"`
	Notes        string    `json:"notes,omitempty"`
}

// User represents user summary for surveillance
type UserSummary struct {
	ID             string    `json:"id"`
	Email          string    `json:"email"`
	Status         string    `json:"status"`
	PositionLimit  float64   `json:"position_limit"`
	CurrentExposure float64  `json:"current_exposure"`
	OpenPositions  int       `json:"open_positions"`
	AlertCount     int       `json:"alert_count"`
	LastActivity   time.Time `json:"last_activity"`
}

// MarketStatus represents market surveillance status
type MarketStatus struct {
	Ticker        string    `json:"ticker"`
	Status        string    `json:"status"`
	IsHalted      bool      `json:"is_halted"`
	HaltReason    string    `json:"halt_reason,omitempty"`
	Volume24h     int       `json:"volume_24h"`
	AlertCount    int       `json:"alert_count"`
	LastPrice     int       `json:"last_price"`
	PriceChange   float64   `json:"price_change_24h"`
}

// DashboardStats for overview metrics
type DashboardStats struct {
	ActiveUsers       int       `json:"active_users"`
	OpenPositions     int       `json:"open_positions"`
	TotalVolume24h    float64   `json:"total_volume_24h"`
	OpenAlerts        int       `json:"open_alerts"`
	CriticalAlerts    int       `json:"critical_alerts"`
	HaltedMarkets     int       `json:"halted_markets"`
	SystemStatus      string    `json:"system_status"`
	LastUpdated       time.Time `json:"last_updated"`
}

// =============================================================================
// IN-MEMORY STORE (Demo)
// =============================================================================

type Store struct {
	alerts      []Alert
	users       []UserSummary
	markets     []MarketStatus
	stats       DashboardStats
	globalHalt  bool
	mu          sync.RWMutex
}

func NewStore() *Store {
	s := &Store{
		alerts:  make([]Alert, 0),
		users:   make([]UserSummary, 0),
		markets: make([]MarketStatus, 0),
	}
	s.seedDemoData()
	return s
}

func (s *Store) seedDemoData() {
	now := time.Now().UTC()

	// Demo alerts
	s.alerts = []Alert{
		{
			ID:           "alert_001",
			Type:         "position_limit",
			Severity:     "high",
			UserID:       "user_123",
			MarketTicker: "FED-RATE-MAR",
			Description:  "User approaching 90% of position limit",
			Status:       "open",
			CreatedAt:    now.Add(-2 * time.Hour),
		},
		{
			ID:           "alert_002",
			Type:         "wash_trade",
			Severity:     "medium",
			UserID:       "user_456",
			MarketTicker: "CPI-FEB",
			Description:  "Potential wash trade pattern detected",
			Status:       "open",
			CreatedAt:    now.Add(-30 * time.Minute),
		},
		{
			ID:           "alert_003",
			Type:         "unusual_activity",
			Severity:     "low",
			UserID:       "user_789",
			MarketTicker: "GDP-Q1",
			Description:  "Unusual trading volume spike",
			Status:       "resolved",
			CreatedAt:    now.Add(-24 * time.Hour),
			ResolvedAt:   timePtr(now.Add(-23 * time.Hour)),
			ResolvedBy:   "admin@dcm.com",
			Notes:        "Normal activity during earnings season",
		},
	}

	// Demo users
	s.users = []UserSummary{
		{
			ID:              "user_123",
			Email:           "trader1@example.com",
			Status:          "verified",
			PositionLimit:   25000,
			CurrentExposure: 22500,
			OpenPositions:   5,
			AlertCount:      1,
			LastActivity:    now.Add(-5 * time.Minute),
		},
		{
			ID:              "user_456",
			Email:           "trader2@example.com",
			Status:          "verified",
			PositionLimit:   25000,
			CurrentExposure: 8000,
			OpenPositions:   3,
			AlertCount:      1,
			LastActivity:    now.Add(-15 * time.Minute),
		},
		{
			ID:              "user_789",
			Email:           "trader3@example.com",
			Status:          "kyc_pending",
			PositionLimit:   25000,
			CurrentExposure: 0,
			OpenPositions:   0,
			AlertCount:      0,
			LastActivity:    now.Add(-2 * time.Hour),
		},
	}

	// Demo markets
	s.markets = []MarketStatus{
		{
			Ticker:      "FED-RATE-MAR",
			Status:      "open",
			IsHalted:    false,
			Volume24h:   15420,
			AlertCount:  1,
			LastPrice:   65,
			PriceChange: 2.5,
		},
		{
			Ticker:      "CPI-FEB",
			Status:      "open",
			IsHalted:    false,
			Volume24h:   8930,
			AlertCount:  1,
			LastPrice:   48,
			PriceChange: -1.2,
		},
		{
			Ticker:      "GDP-Q1",
			Status:      "open",
			IsHalted:    false,
			Volume24h:   3210,
			AlertCount:  0,
			LastPrice:   72,
			PriceChange: 5.0,
		},
	}

	// Calculate stats
	s.updateStats()
}

func (s *Store) updateStats() {
	openAlerts := 0
	criticalAlerts := 0
	for _, a := range s.alerts {
		if a.Status == "open" {
			openAlerts++
			if a.Severity == "critical" || a.Severity == "high" {
				criticalAlerts++
			}
		}
	}

	haltedMarkets := 0
	totalVolume := 0.0
	for _, m := range s.markets {
		if m.IsHalted {
			haltedMarkets++
		}
		totalVolume += float64(m.Volume24h)
	}

	openPositions := 0
	for _, u := range s.users {
		openPositions += u.OpenPositions
	}

	status := "operational"
	if s.globalHalt {
		status = "halted"
	} else if criticalAlerts > 0 {
		status = "warning"
	}

	s.stats = DashboardStats{
		ActiveUsers:    len(s.users),
		OpenPositions:  openPositions,
		TotalVolume24h: totalVolume,
		OpenAlerts:     openAlerts,
		CriticalAlerts: criticalAlerts,
		HaltedMarkets:  haltedMarkets,
		SystemStatus:   status,
		LastUpdated:    time.Now().UTC(),
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}

// =============================================================================
// WEBSOCKET HUB
// =============================================================================

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for demo
	},
}

type Hub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan interface{}
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mu         sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan interface{}, 100),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

func (h *Hub) Run() {
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
				client.Close()
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				if err := client.WriteJSON(message); err != nil {
					client.Close()
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) Broadcast(msgType string, data interface{}) {
	h.broadcast <- map[string]interface{}{
		"type":      msgType,
		"data":      data,
		"timestamp": time.Now().UTC(),
	}
}

// =============================================================================
// HANDLERS
// =============================================================================

type Handler struct {
	store  *Store
	hub    *Hub
	config *Config
}

func NewHandler(store *Store, hub *Hub, config *Config) *Handler {
	return &Handler{
		store:  store,
		hub:    hub,
		config: config,
	}
}

// Dashboard Stats
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	h.store.mu.RLock()
	defer h.store.mu.RUnlock()

	h.store.updateStats()
	respondJSON(w, http.StatusOK, h.store.stats)
}

// Alerts
func (h *Handler) GetAlerts(w http.ResponseWriter, r *http.Request) {
	h.store.mu.RLock()
	defer h.store.mu.RUnlock()

	status := r.URL.Query().Get("status")
	severity := r.URL.Query().Get("severity")

	var filtered []Alert
	for _, a := range h.store.alerts {
		if status != "" && a.Status != status {
			continue
		}
		if severity != "" && a.Severity != severity {
			continue
		}
		filtered = append(filtered, a)
	}

	respondJSON(w, http.StatusOK, filtered)
}

type ResolveAlertRequest struct {
	Notes      string `json:"notes"`
	ResolvedBy string `json:"resolved_by"`
}

func (h *Handler) ResolveAlert(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	alertID := vars["id"]

	var req ResolveAlertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	h.store.mu.Lock()
	defer h.store.mu.Unlock()

	for i := range h.store.alerts {
		if h.store.alerts[i].ID == alertID {
			now := time.Now().UTC()
			h.store.alerts[i].Status = "resolved"
			h.store.alerts[i].ResolvedAt = &now
			h.store.alerts[i].ResolvedBy = req.ResolvedBy
			h.store.alerts[i].Notes = req.Notes

			h.hub.Broadcast("alert_resolved", h.store.alerts[i])
			respondJSON(w, http.StatusOK, h.store.alerts[i])
			return
		}
	}

	respondError(w, http.StatusNotFound, "Alert not found")
}

// Users
func (h *Handler) GetUsers(w http.ResponseWriter, r *http.Request) {
	h.store.mu.RLock()
	defer h.store.mu.RUnlock()

	respondJSON(w, http.StatusOK, h.store.users)
}

func (h *Handler) SuspendUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	h.store.mu.Lock()
	defer h.store.mu.Unlock()

	for i := range h.store.users {
		if h.store.users[i].ID == userID {
			h.store.users[i].Status = "suspended"
			h.hub.Broadcast("user_suspended", h.store.users[i])
			respondJSON(w, http.StatusOK, h.store.users[i])
			return
		}
	}

	respondError(w, http.StatusNotFound, "User not found")
}

// Markets
func (h *Handler) GetMarkets(w http.ResponseWriter, r *http.Request) {
	h.store.mu.RLock()
	defer h.store.mu.RUnlock()

	respondJSON(w, http.StatusOK, h.store.markets)
}

type HaltMarketRequest struct {
	Reason     string `json:"reason"`
	InitiatedBy string `json:"initiated_by"`
}

func (h *Handler) HaltMarket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ticker := vars["ticker"]

	var req HaltMarketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	h.store.mu.Lock()
	defer h.store.mu.Unlock()

	for i := range h.store.markets {
		if h.store.markets[i].Ticker == ticker {
			h.store.markets[i].IsHalted = true
			h.store.markets[i].HaltReason = req.Reason
			h.store.markets[i].Status = "halted"

			h.hub.Broadcast("market_halted", map[string]interface{}{
				"ticker":      ticker,
				"reason":      req.Reason,
				"initiated_by": req.InitiatedBy,
				"timestamp":   time.Now().UTC(),
			})
			respondJSON(w, http.StatusOK, h.store.markets[i])
			return
		}
	}

	respondError(w, http.StatusNotFound, "Market not found")
}

func (h *Handler) ResumeMarket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ticker := vars["ticker"]

	h.store.mu.Lock()
	defer h.store.mu.Unlock()

	for i := range h.store.markets {
		if h.store.markets[i].Ticker == ticker {
			h.store.markets[i].IsHalted = false
			h.store.markets[i].HaltReason = ""
			h.store.markets[i].Status = "open"

			h.hub.Broadcast("market_resumed", map[string]interface{}{
				"ticker":    ticker,
				"timestamp": time.Now().UTC(),
			})
			respondJSON(w, http.StatusOK, h.store.markets[i])
			return
		}
	}

	respondError(w, http.StatusNotFound, "Market not found")
}

// Global Halt
func (h *Handler) GlobalHalt(w http.ResponseWriter, r *http.Request) {
	var req HaltMarketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	h.store.mu.Lock()
	defer h.store.mu.Unlock()

	h.store.globalHalt = true
	for i := range h.store.markets {
		h.store.markets[i].IsHalted = true
		h.store.markets[i].HaltReason = "GLOBAL HALT: " + req.Reason
		h.store.markets[i].Status = "halted"
	}

	h.hub.Broadcast("global_halt", map[string]interface{}{
		"reason":       req.Reason,
		"initiated_by": req.InitiatedBy,
		"timestamp":    time.Now().UTC(),
	})

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "halted",
		"reason":  req.Reason,
		"markets": len(h.store.markets),
	})
}

func (h *Handler) GlobalResume(w http.ResponseWriter, r *http.Request) {
	h.store.mu.Lock()
	defer h.store.mu.Unlock()

	h.store.globalHalt = false
	for i := range h.store.markets {
		h.store.markets[i].IsHalted = false
		h.store.markets[i].HaltReason = ""
		h.store.markets[i].Status = "open"
	}

	h.hub.Broadcast("global_resume", map[string]interface{}{
		"timestamp": time.Now().UTC(),
	})

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "operational",
		"markets": len(h.store.markets),
	})
}

// WebSocket
func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	h.hub.register <- conn

	// Send initial state
	h.store.mu.RLock()
	conn.WriteJSON(map[string]interface{}{
		"type": "initial_state",
		"data": map[string]interface{}{
			"stats":   h.store.stats,
			"alerts":  h.store.alerts,
			"markets": h.store.markets,
		},
	})
	h.store.mu.RUnlock()

	// Read messages (keep connection alive)
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			h.hub.unregister <- conn
			break
		}
	}
}

// Health
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"service":   "surveillance-dashboard",
		"version":   "1.0.0",
		"timestamp": time.Now().UTC(),
	})
}

// =============================================================================
// RESPONSE HELPERS
// =============================================================================

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// =============================================================================
// MAIN
// =============================================================================

func main() {
	config := loadConfig()
	store := NewStore()
	hub := NewHub()
	handler := NewHandler(store, hub, config)

	// Start WebSocket hub
	go hub.Run()

	// Start periodic stats broadcast
	go func() {
		ticker := time.NewTicker(config.RefreshInterval)
		for range ticker.C {
			store.mu.Lock()
			store.updateStats()
			stats := store.stats
			store.mu.Unlock()
			hub.Broadcast("stats_update", stats)
		}
	}()

	// Router
	r := mux.NewRouter()
	api := r.PathPrefix("/api").Subrouter()

	// Health
	api.HandleFunc("/health", handler.HealthCheck).Methods("GET")

	// Dashboard
	api.HandleFunc("/stats", handler.GetStats).Methods("GET")

	// Alerts
	api.HandleFunc("/alerts", handler.GetAlerts).Methods("GET")
	api.HandleFunc("/alerts/{id}/resolve", handler.ResolveAlert).Methods("POST")

	// Users
	api.HandleFunc("/users", handler.GetUsers).Methods("GET")
	api.HandleFunc("/users/{id}/suspend", handler.SuspendUser).Methods("POST")

	// Markets
	api.HandleFunc("/markets", handler.GetMarkets).Methods("GET")
	api.HandleFunc("/markets/{ticker}/halt", handler.HaltMarket).Methods("POST")
	api.HandleFunc("/markets/{ticker}/resume", handler.ResumeMarket).Methods("POST")

	// Global controls
	api.HandleFunc("/halt", handler.GlobalHalt).Methods("POST")
	api.HandleFunc("/resume", handler.GlobalResume).Methods("POST")

	// WebSocket
	r.HandleFunc("/ws", handler.HandleWebSocket)

	// Static files - check for React build first, then fall back to legacy static
	staticDir := "./frontend/dist"
	if _, err := os.Stat(staticDir); os.IsNotExist(err) {
		staticDir = "./static" // Legacy static HTML
		log.Println("📄 Serving legacy static HTML")
	} else {
		log.Println("⚛️  Serving React build from frontend/dist")
	}
	r.PathPrefix("/").Handler(http.FileServer(http.Dir(staticDir)))

	// CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	addr := fmt.Sprintf(":%s", config.Port)
	log.Printf("🔍 Surveillance Dashboard starting on http://localhost%s", addr)
	log.Printf("📊 WebSocket available at ws://localhost%s/ws", addr)
	log.Printf("🔗 Backend API: %s", config.BackendAPIURL)

	if err := http.ListenAndServe(addr, c.Handler(r)); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
