// Package api provides routing for the DCM demo API.
package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rs/cors"

	"github.com/kalshi-dcm-demo/backend/internal/auth"
)

// NewRouter creates and configures the API router.
func NewRouter(h *Handler) http.Handler {
	r := mux.NewRouter()

	// API versioning
	api := r.PathPrefix("/api/v1").Subrouter()

	// ==========================================================================
	// PUBLIC ROUTES (No authentication required)
	// ==========================================================================

	// Health check
	api.HandleFunc("/health", h.HealthCheck).Methods("GET", "OPTIONS")

	// Authentication
	api.HandleFunc("/auth/signup", h.Signup).Methods("POST", "OPTIONS")
	api.HandleFunc("/auth/login", h.Login).Methods("POST", "OPTIONS")

	// Public market data (from Kalshi)
	api.HandleFunc("/markets", h.GetMarkets).Methods("GET", "OPTIONS")
	api.HandleFunc("/markets/{ticker}", h.GetMarket).Methods("GET", "OPTIONS")
	api.HandleFunc("/markets/{ticker}/orderbook", h.GetOrderbook).Methods("GET", "OPTIONS")
	api.HandleFunc("/events", h.GetEvents).Methods("GET", "OPTIONS")
	api.HandleFunc("/series", h.GetSeries).Methods("GET", "OPTIONS")

	// ==========================================================================
	// AUTHENTICATED ROUTES (Requires valid JWT)
	// ==========================================================================

	authenticated := api.PathPrefix("").Subrouter()
	authenticated.Use(auth.AuthMiddleware)

	// User profile
	authenticated.HandleFunc("/profile", h.GetProfile).Methods("GET", "OPTIONS")

	// KYC
	authenticated.HandleFunc("/kyc", h.GetKYCStatus).Methods("GET", "OPTIONS")
	authenticated.HandleFunc("/kyc", h.SubmitKYC).Methods("POST", "OPTIONS")

	// Wallet
	authenticated.HandleFunc("/wallet", h.GetWallet).Methods("GET", "OPTIONS")
	authenticated.HandleFunc("/wallet/deposit", h.Deposit).Methods("POST", "OPTIONS")
	authenticated.HandleFunc("/wallet/transactions", h.GetTransactions).Methods("GET", "OPTIONS")

	// Audit trail
	authenticated.HandleFunc("/audit", h.GetAuditLog).Methods("GET", "OPTIONS")

	// ==========================================================================
	// TRADING ROUTES (Requires authentication; KYC checked in handlers)
	// Core Principle 17: Fitness Standards enforcement via store checks
	// Note: We check user.Status in handlers against the store (source of truth)
	// rather than relying on JWT claims which may be stale after KYC approval
	// ==========================================================================

	// Pre-trade check (Core Principle 11)
	authenticated.HandleFunc("/orders/check", h.PreTradeCheck).Methods("POST", "OPTIONS")

	// Trading (Core Principle 9)
	authenticated.HandleFunc("/orders", h.PlaceOrder).Methods("POST", "OPTIONS")
	authenticated.HandleFunc("/orders", h.GetOrders).Methods("GET", "OPTIONS")

	// Portfolio (Core Principle 5)
	authenticated.HandleFunc("/positions", h.GetPositions).Methods("GET", "OPTIONS")
	authenticated.HandleFunc("/portfolio", h.GetPortfolioSummary).Methods("GET", "OPTIONS")

	// ==========================================================================
	// CORS CONFIGURATION
	// ==========================================================================

	c := cors.New(cors.Options{
		AllowedOrigins: []string{
			"http://localhost:3000",
			"http://localhost:5173",
			"http://127.0.0.1:3000",
			"http://127.0.0.1:5173",
		},
		AllowedMethods: []string{
			"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS",
		},
		AllowedHeaders: []string{
			"Accept",
			"Authorization",
			"Content-Type",
			"X-Requested-With",
			"X-CSRF-Token",
		},
		ExposedHeaders: []string{
			"Link",
			"X-Total-Count",
		},
		AllowCredentials: true,
		MaxAge:           300,
	})

	return c.Handler(r)
}
