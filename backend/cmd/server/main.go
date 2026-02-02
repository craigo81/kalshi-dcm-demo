// Kalshi DCM Demo - CFTC Compliant Binary Contracts Trading Platform
//
// This demo implements a trading platform for binary event contracts
// that routes to Kalshi as the designated contract market (DCM).
//
// CFTC Core Principles Implemented:
// - Core Principle 2: Compliance with CEA Rules
// - Core Principle 3: Contracts Not Readily Susceptible to Manipulation
// - Core Principle 4: Prevention of Market Disruption
// - Core Principle 5: Position Limits
// - Core Principle 9: Execution of Transactions
// - Core Principle 11: Financial Integrity (100% Collateralization)
// - Core Principle 13: Financial Resources (Segregated Funds)
// - Core Principle 17: Fitness Standards
// - Core Principle 18: Recordkeeping and Reporting
//
// WARNING: This is a DEMO application. Do not use in production without:
// - Proper database integration
// - Real KYC/AML verification services
// - Kalshi authenticated API integration
// - Security audits and penetration testing
// - Legal review for regulatory compliance

package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/kalshi-dcm-demo/backend/internal/api"
	"github.com/kalshi-dcm-demo/backend/internal/compliance"
	"github.com/kalshi-dcm-demo/backend/internal/kalshi"
	"github.com/kalshi-dcm-demo/backend/internal/mock"
	"github.com/kalshi-dcm-demo/backend/internal/ws"
)

func main() {
	log.Println("===========================================")
	log.Println("  Kalshi DCM Demo - CFTC Compliant Platform")
	log.Println("===========================================")
	log.Println("")
	log.Println("Core Principles: 2, 3, 4, 5, 9, 11, 13, 17, 18")
	log.Println("")

	// Configuration
	port := getEnv("PORT", "8080")
	kalshiURL := getEnv("KALSHI_API_URL", kalshi.DefaultBaseURL)
	dataDir := getEnv("DATA_DIR", "./data")
	persistenceEnabled := getEnv("ENABLE_PERSISTENCE", "true") == "true"

	log.Printf("Starting server on port %s", port)
	log.Printf("Kalshi API: %s", kalshiURL)
	log.Printf("Persistence: %v (dir: %s)", persistenceEnabled, dataDir)

	// Initialize components
	// Persistent store for CP 18: 5-year recordkeeping
	store := mock.NewStoreWithPersistence(mock.PersistenceConfig{
		Enabled:          persistenceEnabled,
		DataDir:          dataDir,
		AutoSaveInterval: 5 * time.Minute,
		RetentionYears:   5,
	})
	log.Println("✓ Persistent data store initialized")

	// Kalshi API client for real market data (Core Principle 3)
	kalshiClient := kalshi.NewClient(kalshiURL, 30*time.Second)
	log.Println("✓ Kalshi API client initialized")

	// Surveillance engine (Core Principles 4, 5)
	surveillance := compliance.NewSurveillanceEngine(store)
	log.Println("✓ Surveillance engine initialized")

	// WebSocket hub for real-time updates (Core Principle 9)
	wsHub := ws.NewHub(kalshiClient)
	go wsHub.Run()
	log.Println("✓ WebSocket hub started")

	// API handlers
	handler := api.NewHandler(store, kalshiClient, surveillance)

	// Create router with all routes
	router := api.NewRouter(handler)

	// Create a new mux for WebSocket + API routes
	mainRouter := mux.NewRouter()
	mainRouter.HandleFunc("/ws", wsHub.ServeWS)
	mainRouter.PathPrefix("/").Handler(router)

	// Configure HTTP server
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mainRouter,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("✓ Server listening on http://localhost:%s", port)
		log.Println("")
		log.Println("API Endpoints:")
		log.Println("  POST /api/v1/auth/signup     - Register new user")
		log.Println("  POST /api/v1/auth/login      - Authenticate user")
		log.Println("  GET  /api/v1/markets         - List Kalshi markets")
		log.Println("  GET  /api/v1/markets/{ticker} - Get market details")
		log.Println("  POST /api/v1/kyc             - Submit KYC verification")
		log.Println("  POST /api/v1/wallet/deposit  - Mock deposit funds")
		log.Println("  POST /api/v1/orders          - Place trading order")
		log.Println("  GET  /api/v1/positions       - View open positions")
		log.Println("  GET  /api/v1/portfolio       - Portfolio summary")
		log.Println("  WS   /ws                     - Real-time market data")
		log.Println("")
		log.Println("Press Ctrl+C to stop")

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("")
	log.Println("Shutting down server...")

	// Save data before shutdown (CP 18: Recordkeeping)
	store.Stop()
	log.Println("✓ Data persisted")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped gracefully")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
