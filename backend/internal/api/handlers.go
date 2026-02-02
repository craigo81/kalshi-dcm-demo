// Package api provides HTTP handlers for the DCM demo.
// All handlers include CFTC Core Principle compliance annotations.
package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"

	"github.com/kalshi-dcm-demo/backend/internal/auth"
	"github.com/kalshi-dcm-demo/backend/internal/compliance"
	"github.com/kalshi-dcm-demo/backend/internal/kalshi"
	"github.com/kalshi-dcm-demo/backend/internal/mock"
	"github.com/kalshi-dcm-demo/backend/internal/models"
)

// =============================================================================
// HANDLER DEPENDENCIES
// =============================================================================

type Handler struct {
	store       *mock.Store
	kalshi      *kalshi.Client
	surveillance *compliance.SurveillanceEngine
}

func NewHandler(store *mock.Store, kalshiClient *kalshi.Client, surveillance *compliance.SurveillanceEngine) *Handler {
	return &Handler{
		store:       store,
		kalshi:      kalshiClient,
		surveillance: surveillance,
	}
}

// =============================================================================
// RESPONSE HELPERS
// =============================================================================

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Code    string      `json:"code,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func respondError(w http.ResponseWriter, status int, message, code string) {
	respondJSON(w, status, APIResponse{
		Success: false,
		Error:   message,
		Code:    code,
	})
}

func respondSuccess(w http.ResponseWriter, data interface{}, meta interface{}) {
	respondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

// =============================================================================
// HEALTH CHECK
// =============================================================================

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	respondSuccess(w, map[string]interface{}{
		"status":     "healthy",
		"service":    "kalshi-dcm-demo",
		"version":    "1.0.0",
		"timestamp":  time.Now().UTC(),
		"compliance": "CFTC Core Principles compliant",
	}, nil)
}

// =============================================================================
// AUTHENTICATION HANDLERS
// Core Principle 17: Fitness Standards - User eligibility
// =============================================================================

type SignupRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	StateCode   string `json:"state_code"`
	DateOfBirth string `json:"date_of_birth"` // YYYY-MM-DD
	IsUSResident bool  `json:"is_us_resident"`
}

// Signup registers a new user account.
// Core Principle 17: Initial eligibility check for US residency.
func (h *Handler) Signup(w http.ResponseWriter, r *http.Request) {
	var req SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST")
		return
	}

	// Validate required fields
	if req.Email == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "Email and password required", "MISSING_FIELDS")
		return
	}

	// Core Principle 17: Check US residency requirement
	if !req.IsUSResident {
		respondError(w, http.StatusForbidden,
			"Trading is only available to US residents", "US_RESIDENCY_REQUIRED")
		return
	}

	// Validate state (some states may have restrictions)
	restrictedStates := map[string]bool{
		// Example: Some prediction markets have state restrictions
	}
	if restrictedStates[req.StateCode] {
		respondError(w, http.StatusForbidden,
			"Trading is not available in your state", "STATE_RESTRICTED")
		return
	}

	// Parse date of birth
	dob, err := time.Parse("2006-01-02", req.DateOfBirth)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid date of birth format", "INVALID_DOB")
		return
	}

	// Check age (must be 18+)
	age := time.Now().Year() - dob.Year()
	if age < 18 {
		respondError(w, http.StatusForbidden, "Must be 18 or older to trade", "AGE_RESTRICTED")
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Registration failed", "INTERNAL_ERROR")
		return
	}

	ip := auth.GetClientIP(r)

	// Create user
	user, err := h.store.CreateUser(
		req.Email,
		string(hashedPassword),
		req.FirstName,
		req.LastName,
		req.StateCode,
		dob,
		req.IsUSResident,
		ip,
	)
	if err != nil {
		if err == mock.ErrUserExists {
			respondError(w, http.StatusConflict, "Email already registered", "USER_EXISTS")
			return
		}
		respondError(w, http.StatusInternalServerError, "Registration failed", "INTERNAL_ERROR")
		return
	}

	// Create wallet (Core Principle 13: Segregated funds)
	h.store.CreateWallet(user.ID, ip)

	// Generate JWT
	token, err := auth.GenerateToken(user.ID, user.Email, string(user.Status), false)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Token generation failed", "INTERNAL_ERROR")
		return
	}

	respondSuccess(w, map[string]interface{}{
		"user":  user,
		"token": token,
		"next_step": "kyc_required",
		"message": "Account created. Please complete KYC verification to start trading.",
	}, nil)
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login authenticates a user and returns a JWT.
// Core Principle 18: Logs authentication events for audit trail.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST")
		return
	}

	user, err := h.store.GetUserByEmail(req.Email)
	if err != nil {
		// Don't reveal if email exists or not
		respondError(w, http.StatusUnauthorized, "Invalid credentials", "INVALID_CREDENTIALS")
		return
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		respondError(w, http.StatusUnauthorized, "Invalid credentials", "INVALID_CREDENTIALS")
		return
	}

	// Check if suspended/banned (Core Principle 17)
	if user.Status == models.UserStatusSuspended {
		respondError(w, http.StatusForbidden, "Account suspended", "ACCOUNT_SUSPENDED")
		return
	}
	if user.Status == models.UserStatusBanned {
		respondError(w, http.StatusForbidden, "Account banned", "ACCOUNT_BANNED")
		return
	}

	ip := auth.GetClientIP(r)

	// Record login (Core Principle 18)
	h.store.RecordLogin(user.ID, ip)

	verified := user.Status == models.UserStatusVerified
	token, err := auth.GenerateToken(user.ID, user.Email, string(user.Status), verified)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Token generation failed", "INTERNAL_ERROR")
		return
	}

	respondSuccess(w, map[string]interface{}{
		"user":  user,
		"token": token,
	}, nil)
}

// GetProfile returns current user profile.
func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "Unauthorized", "UNAUTHORIZED")
		return
	}

	user, err := h.store.GetUser(claims.UserID)
	if err != nil {
		respondError(w, http.StatusNotFound, "User not found", "USER_NOT_FOUND")
		return
	}

	kyc, _ := h.store.GetKYCRecord(claims.UserID)
	wallet, _ := h.store.GetWallet(claims.UserID)

	respondSuccess(w, map[string]interface{}{
		"user":   user,
		"kyc":    kyc,
		"wallet": wallet,
	}, nil)
}

// =============================================================================
// KYC HANDLERS
// Core Principle 17: Fitness Standards
// =============================================================================

type KYCSubmitRequest struct {
	DocumentType   string `json:"document_type"` // drivers_license, passport, state_id
	DocumentNumber string `json:"document_number"`
	// In production: Would include document image upload
}

// SubmitKYC initiates KYC verification.
// Core Principle 17: Verifies participant eligibility.
func (h *Handler) SubmitKYC(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "Unauthorized", "UNAUTHORIZED")
		return
	}

	var req KYCSubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST")
		return
	}

	validDocTypes := map[string]bool{
		"drivers_license": true,
		"passport":        true,
		"state_id":        true,
	}
	if !validDocTypes[req.DocumentType] {
		respondError(w, http.StatusBadRequest, "Invalid document type", "INVALID_DOC_TYPE")
		return
	}

	ip := auth.GetClientIP(r)

	record, err := h.store.CreateKYCRecord(claims.UserID, req.DocumentType, req.DocumentNumber, ip)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "KYC submission failed", "INTERNAL_ERROR")
		return
	}

	// MOCK: Auto-approve after delay (demo only)
	// In production: Would integrate with identity verification service
	go func() {
		time.Sleep(3 * time.Second) // Simulate verification delay
		h.store.MockKYCApproval(claims.UserID, true, "")
	}()

	respondSuccess(w, map[string]interface{}{
		"kyc_record": record,
		"message":    "KYC submitted. Verification typically takes 1-3 business days.",
	}, nil)
}

// GetKYCStatus returns current KYC verification status.
func (h *Handler) GetKYCStatus(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "Unauthorized", "UNAUTHORIZED")
		return
	}

	record, err := h.store.GetKYCRecord(claims.UserID)
	if err != nil {
		respondError(w, http.StatusNotFound, "KYC record not found", "KYC_NOT_FOUND")
		return
	}

	if record == nil {
		respondSuccess(w, map[string]interface{}{
			"status": "not_started",
			"message": "Please submit KYC documents to start verification.",
		}, nil)
		return
	}

	respondSuccess(w, record, nil)
}

// =============================================================================
// WALLET HANDLERS
// Core Principle 11: Financial Integrity (100% collateralization)
// Core Principle 13: Financial Resources (Segregated funds)
// =============================================================================

type DepositRequest struct {
	AmountUSD float64 `json:"amount_usd"`
	// In production: Would include ACH details, bank info, etc.
}

// GetWallet returns user's wallet balance.
// Core Principle 13: Shows segregated funds status.
func (h *Handler) GetWallet(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "Unauthorized", "UNAUTHORIZED")
		return
	}

	wallet, err := h.store.GetWallet(claims.UserID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Wallet not found", "WALLET_NOT_FOUND")
		return
	}

	respondSuccess(w, wallet, nil)
}

// Deposit adds funds to wallet (mock ACH).
// Core Principle 13: Funds segregation tracking.
func (h *Handler) Deposit(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "Unauthorized", "UNAUTHORIZED")
		return
	}

	var req DepositRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST")
		return
	}

	if req.AmountUSD <= 0 {
		respondError(w, http.StatusBadRequest, "Amount must be positive", "INVALID_AMOUNT")
		return
	}

	// Demo limits
	if req.AmountUSD > 10000 {
		respondError(w, http.StatusBadRequest, "Maximum deposit is $10,000", "AMOUNT_EXCEEDED")
		return
	}

	ip := auth.GetClientIP(r)
	reference := "MOCK_ACH_" + time.Now().Format("20060102150405")

	tx, err := h.store.Deposit(claims.UserID, req.AmountUSD, reference, ip)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Deposit failed", "DEPOSIT_FAILED")
		return
	}

	wallet, _ := h.store.GetWallet(claims.UserID)

	respondSuccess(w, map[string]interface{}{
		"transaction": tx,
		"wallet":      wallet,
		"message":     "Deposit completed successfully",
	}, nil)
}

// GetTransactions returns transaction history.
// Core Principle 18: Recordkeeping.
func (h *Handler) GetTransactions(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "Unauthorized", "UNAUTHORIZED")
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	transactions, err := h.store.GetTransactions(claims.UserID, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch transactions", "INTERNAL_ERROR")
		return
	}

	respondSuccess(w, transactions, nil)
}

// =============================================================================
// MARKET HANDLERS (Real Kalshi API)
// Core Principle 3: Contracts not readily susceptible to manipulation
// =============================================================================

// GetMarkets fetches live markets from Kalshi.
// Core Principle 3: Focus on economic binaries (low manipulation risk).
func (h *Handler) GetMarkets(w http.ResponseWriter, r *http.Request) {
	params := kalshi.MarketParams{
		Status:       r.URL.Query().Get("status"),
		SeriesTicker: r.URL.Query().Get("series_ticker"),
		EventTicker:  r.URL.Query().Get("event_ticker"),
		Cursor:       r.URL.Query().Get("cursor"),
	}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		if parsed, err := strconv.Atoi(limit); err == nil {
			params.Limit = parsed
		}
	}
	if params.Limit == 0 {
		params.Limit = 20
	}

	response, err := h.kalshi.GetMarkets(params)
	if err != nil {
		respondError(w, http.StatusServiceUnavailable, "Failed to fetch markets", "KALSHI_ERROR")
		return
	}

	// Convert to internal models with risk classification
	var markets []models.KalshiMarket
	for _, m := range response.Markets {
		markets = append(markets, m.ToMarket())
	}

	respondSuccess(w, markets, map[string]interface{}{
		"cursor":   response.Cursor,
		"exchange": "kalshi",
	})
}

// GetMarket fetches a single market.
func (h *Handler) GetMarket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ticker := vars["ticker"]

	if ticker == "" {
		respondError(w, http.StatusBadRequest, "Market ticker required", "MISSING_TICKER")
		return
	}

	market, err := h.kalshi.GetMarket(ticker)
	if err != nil {
		respondError(w, http.StatusNotFound, "Market not found", "MARKET_NOT_FOUND")
		return
	}

	respondSuccess(w, market.ToMarket(), nil)
}

// GetOrderbook fetches market orderbook.
// Core Principle 9: Transparency in execution.
func (h *Handler) GetOrderbook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ticker := vars["ticker"]

	depth := 10
	if d := r.URL.Query().Get("depth"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil {
			depth = parsed
		}
	}

	orderbook, err := h.kalshi.GetOrderbook(ticker, depth)
	if err != nil {
		respondError(w, http.StatusServiceUnavailable, "Failed to fetch orderbook", "KALSHI_ERROR")
		return
	}

	respondSuccess(w, orderbook, nil)
}

// GetEvents fetches Kalshi events.
func (h *Handler) GetEvents(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}
	cursor := r.URL.Query().Get("cursor")

	response, err := h.kalshi.GetEvents(status, limit, cursor)
	if err != nil {
		respondError(w, http.StatusServiceUnavailable, "Failed to fetch events", "KALSHI_ERROR")
		return
	}

	respondSuccess(w, response.Events, map[string]interface{}{
		"cursor": response.Cursor,
	})
}

// GetSeries fetches Kalshi series.
func (h *Handler) GetSeries(w http.ResponseWriter, r *http.Request) {
	cursor := r.URL.Query().Get("cursor")
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	response, err := h.kalshi.GetSeries(cursor, limit)
	if err != nil {
		respondError(w, http.StatusServiceUnavailable, "Failed to fetch series", "KALSHI_ERROR")
		return
	}

	respondSuccess(w, response.Series, map[string]interface{}{
		"cursor": response.Cursor,
	})
}

// =============================================================================
// TRADING HANDLERS (Mock)
// Core Principle 9: Execution of Transactions
// Core Principle 11: Financial Integrity
// =============================================================================

type PlaceOrderRequest struct {
	MarketTicker string `json:"market_ticker"`
	Side         string `json:"side"`       // yes, no
	Type         string `json:"type"`       // limit, market
	Quantity     int    `json:"quantity"`   // Number of contracts
	PriceCents   int    `json:"price_cents"` // 1-99
}

// PreTradeCheck validates an order before placement.
// Core Principle 11: Ensures 100% collateralization.
// Core Principle 5: Checks position limits.
func (h *Handler) PreTradeCheck(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "Unauthorized", "UNAUTHORIZED")
		return
	}

	var req PlaceOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST")
		return
	}

	side := models.OrderSide(req.Side)
	check := h.surveillance.ValidateOrder(claims.UserID, req.MarketTicker, side, req.Quantity, req.PriceCents)

	respondSuccess(w, check, nil)
}

// PlaceOrder submits a trading order (mock).
// Core Principle 9: Fair and equitable execution.
// Core Principle 11: Pre-trade margin check.
func (h *Handler) PlaceOrder(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "Unauthorized", "UNAUTHORIZED")
		return
	}

	var req PlaceOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST")
		return
	}

	// Validate inputs
	if req.MarketTicker == "" {
		respondError(w, http.StatusBadRequest, "Market ticker required", "MISSING_TICKER")
		return
	}
	if req.Side != "yes" && req.Side != "no" {
		respondError(w, http.StatusBadRequest, "Side must be 'yes' or 'no'", "INVALID_SIDE")
		return
	}
	if req.Quantity <= 0 || req.Quantity > 1000 {
		respondError(w, http.StatusBadRequest, "Quantity must be 1-1000", "INVALID_QUANTITY")
		return
	}
	if req.PriceCents < 1 || req.PriceCents > 99 {
		respondError(w, http.StatusBadRequest, "Price must be 1-99 cents", "INVALID_PRICE")
		return
	}

	side := models.OrderSide(req.Side)
	orderType := models.OrderTypeLimit
	if req.Type == "market" {
		orderType = models.OrderTypeMarket
	}

	// Verify market exists and is open
	market, err := h.kalshi.GetMarket(req.MarketTicker)
	if err != nil {
		respondError(w, http.StatusNotFound, "Market not found", "MARKET_NOT_FOUND")
		return
	}
	// Check for open/active status (Kalshi may use different values)
	// Also handle case variations
	marketStatus := strings.ToLower(market.Status)
	isOpen := marketStatus == "open" || marketStatus == "active" || marketStatus == "trading"
	if !isOpen {
		respondError(w, http.StatusBadRequest, "Market is not open for trading (status: "+market.Status+")", "MARKET_CLOSED")
		return
	}

	ip := auth.GetClientIP(r)

	// Create order (includes compliance checks)
	order, err := h.store.CreateOrder(
		claims.UserID,
		req.MarketTicker,
		market.EventTicker,
		side,
		orderType,
		req.Quantity,
		req.PriceCents,
		ip,
	)

	if err != nil {
		switch err {
		case mock.ErrInsufficientFunds:
			respondError(w, http.StatusBadRequest, "Insufficient funds", "INSUFFICIENT_FUNDS")
		case mock.ErrPositionLimitExceeded:
			respondError(w, http.StatusBadRequest, "Position limit exceeded", "POSITION_LIMIT")
		case mock.ErrKYCRequired:
			respondError(w, http.StatusForbidden, "KYC verification required", "KYC_REQUIRED")
		case mock.ErrTradingHalted:
			respondError(w, http.StatusServiceUnavailable, "Trading is halted", "TRADING_HALTED")
		case mock.ErrUserSuspended:
			respondError(w, http.StatusForbidden, "Account suspended", "ACCOUNT_SUSPENDED")
		default:
			respondError(w, http.StatusInternalServerError, "Order failed", "ORDER_FAILED")
		}
		return
	}

	// MOCK: Simulate fill for demo
	// In production: Would route to Kalshi's authenticated API
	go func() {
		time.Sleep(500 * time.Millisecond) // Simulate matching delay
		h.store.MockFillOrder(order.ID, req.PriceCents)
	}()

	wallet, _ := h.store.GetWallet(claims.UserID)

	respondSuccess(w, map[string]interface{}{
		"order":   order,
		"wallet":  wallet,
		"message": "Order submitted successfully",
	}, nil)
}

// GetOrders returns user's order history.
// Core Principle 18: Order recordkeeping.
func (h *Handler) GetOrders(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "Unauthorized", "UNAUTHORIZED")
		return
	}

	var status *models.OrderStatus
	if s := r.URL.Query().Get("status"); s != "" {
		os := models.OrderStatus(s)
		status = &os
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	orders, err := h.store.GetOrders(claims.UserID, status, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch orders", "INTERNAL_ERROR")
		return
	}

	respondSuccess(w, orders, nil)
}

// =============================================================================
// PORTFOLIO HANDLERS
// Core Principle 5: Position monitoring
// =============================================================================

// GetPositions returns open positions.
// Core Principle 5: Position limits visibility.
func (h *Handler) GetPositions(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "Unauthorized", "UNAUTHORIZED")
		return
	}

	positions, err := h.store.GetPositions(claims.UserID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch positions", "INTERNAL_ERROR")
		return
	}

	// Enrich with current market prices
	for i := range positions {
		market, err := h.kalshi.GetMarket(positions[i].MarketTicker)
		if err == nil {
			var currentPrice int
			if positions[i].Side == models.OrderSideYes {
				currentPrice = market.YesBid
			} else {
				currentPrice = market.NoBid
			}
			positions[i].CurrentValue = float64(positions[i].Quantity*currentPrice) / 100.0
			positions[i].UnrealizedPnL = positions[i].CurrentValue - positions[i].CostBasisUSD
		}
	}

	// Calculate totals
	var totalValue, totalPnL float64
	for _, pos := range positions {
		totalValue += pos.CurrentValue
		totalPnL += pos.UnrealizedPnL
	}

	respondSuccess(w, map[string]interface{}{
		"positions":      positions,
		"total_value":    totalValue,
		"total_pnl":      totalPnL,
		"position_count": len(positions),
	}, nil)
}

// GetPortfolioSummary returns portfolio overview.
func (h *Handler) GetPortfolioSummary(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "Unauthorized", "UNAUTHORIZED")
		return
	}

	wallet, _ := h.store.GetWallet(claims.UserID)
	positions, _ := h.store.GetPositions(claims.UserID)
	user, _ := h.store.GetUser(claims.UserID)

	var positionValue, unrealizedPnL float64
	for _, pos := range positions {
		positionValue += pos.CurrentValue
		unrealizedPnL += pos.UnrealizedPnL
	}

	exposure := h.store.GetUserExposure(claims.UserID)

	respondSuccess(w, map[string]interface{}{
		"wallet": map[string]interface{}{
			"available":    wallet.AvailableUSD,
			"locked":       wallet.LockedUSD,
			"total":        wallet.AvailableUSD + wallet.LockedUSD,
		},
		"positions": map[string]interface{}{
			"count":          len(positions),
			"total_value":    positionValue,
			"unrealized_pnl": unrealizedPnL,
		},
		"limits": map[string]interface{}{
			"position_limit":   user.PositionLimitUSD,
			"current_exposure": exposure,
			"utilization":      (exposure / user.PositionLimitUSD) * 100,
		},
	}, nil)
}

// =============================================================================
// COMPLIANCE HANDLERS
// Core Principle 4: Market surveillance
// Core Principle 18: Audit trail
// =============================================================================

// GetAuditLog returns user's audit trail.
// Core Principle 18: Recordkeeping access.
func (h *Handler) GetAuditLog(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "Unauthorized", "UNAUTHORIZED")
		return
	}

	since := time.Now().AddDate(0, -1, 0) // Last 30 days
	if s := r.URL.Query().Get("since"); s != "" {
		if parsed, err := time.Parse(time.RFC3339, s); err == nil {
			since = parsed
		}
	}

	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	entries := h.store.GetAuditLog(claims.UserID, since, limit)

	respondSuccess(w, entries, nil)
}
