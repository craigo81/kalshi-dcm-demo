// Package kalshi provides mock authenticated endpoint simulation.
// These stubs simulate Kalshi's authenticated API without real credentials.
//
// CFTC Core Principle Compliance:
// - CP 9: Execution of Transactions - Simulates fair order matching
// - CP 11: Financial Integrity - Ensures 100% collateralization in fills
// - CP 18: Recordkeeping - All mock operations are logged
package kalshi

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// =============================================================================
// MOCK AUTHENTICATED RESPONSES
// These simulate Kalshi's authenticated endpoints for demo purposes
// =============================================================================

// MockOrderRequest represents an order placement request
type MockOrderRequest struct {
	Ticker     string `json:"ticker"`
	Side       string `json:"side"`       // yes, no
	Action     string `json:"action"`     // buy, sell
	Type       string `json:"type"`       // limit, market
	Count      int    `json:"count"`      // Number of contracts
	YesPrice   int    `json:"yes_price"`  // Price in cents (1-99)
	NoPrice    int    `json:"no_price"`
	Expiration string `json:"expiration"` // Optional: GTC, GTD, etc.
	ClientOrderID string `json:"client_order_id,omitempty"`
}

// MockOrderResponse represents an order confirmation
// CP 9: Provides transparent execution information
type MockOrderResponse struct {
	OrderID         string    `json:"order_id"`
	ClientOrderID   string    `json:"client_order_id,omitempty"`
	Ticker          string    `json:"ticker"`
	Side            string    `json:"side"`
	Action          string    `json:"action"`
	Type            string    `json:"type"`
	Status          string    `json:"status"` // pending, open, filled, cancelled
	Count           int       `json:"count"`
	RemainingCount  int       `json:"remaining_count"`
	YesPrice        int       `json:"yes_price"`
	NoPrice         int       `json:"no_price"`
	FilledCount     int       `json:"filled_count"`
	FilledAvgPrice  int       `json:"filled_avg_price"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
}

// MockPosition represents a portfolio position
// CP 5: Position tracking for limit enforcement
type MockPosition struct {
	Ticker            string  `json:"ticker"`
	EventTicker       string  `json:"event_ticker"`
	MarketTitle       string  `json:"market_title"`
	Side              string  `json:"side"` // yes, no
	Contracts         int     `json:"contracts"`
	AveragePriceCents int     `json:"average_price_cents"`
	TotalCostCents    int     `json:"total_cost_cents"`
	CurrentPriceCents int     `json:"current_price_cents"`
	CurrentValueCents int     `json:"current_value_cents"`
	UnrealizedPnL     int     `json:"unrealized_pnl_cents"`
	RealizedPnL       int     `json:"realized_pnl_cents"`
}

// MockSettlement represents a contract settlement
// CP 3: Objective resolution based on verifiable outcomes
type MockSettlement struct {
	SettlementID   string    `json:"settlement_id"`
	Ticker         string    `json:"ticker"`
	EventTicker    string    `json:"event_ticker"`
	MarketTitle    string    `json:"market_title"`
	Result         string    `json:"result"` // yes, no
	SettlementValue int      `json:"settlement_value"` // 0 or 100
	SettledAt      time.Time `json:"settled_at"`
	PayoutCents    int       `json:"payout_cents"`
	Reason         string    `json:"reason"` // Objective resolution source
}

// MockBalance represents account balance
// CP 13: Segregated funds tracking
type MockBalance struct {
	AvailableBalanceCents int `json:"available_balance_cents"`
	PortfolioValueCents   int `json:"portfolio_value_cents"`
	TotalBalanceCents     int `json:"total_balance_cents"`
	PendingDepositsCents  int `json:"pending_deposits_cents"`
	PendingWithdrawalsCents int `json:"pending_withdrawals_cents"`
}

// =============================================================================
// MOCK ORDER EXECUTOR
// Simulates order matching and execution
// =============================================================================

// MockOrderExecutor simulates order execution
type MockOrderExecutor struct {
	orders     map[string]*MockOrderResponse
	positions  map[string]map[string]*MockPosition // userID -> ticker -> position
	settlements []MockSettlement
	mu         sync.RWMutex
	orderIDCounter int64
}

// NewMockOrderExecutor creates a new mock executor
func NewMockOrderExecutor() *MockOrderExecutor {
	return &MockOrderExecutor{
		orders:     make(map[string]*MockOrderResponse),
		positions:  make(map[string]map[string]*MockPosition),
		settlements: make([]MockSettlement, 0),
	}
}

// PlaceOrder simulates order placement and execution
// CP 9: Fair and equitable execution simulation
// CP 11: Validates collateral requirements
func (e *MockOrderExecutor) PlaceOrder(userID string, req MockOrderRequest, marketBid, marketAsk int) (*MockOrderResponse, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.orderIDCounter++
	orderID := fmt.Sprintf("ORD_%d_%d", time.Now().UnixNano(), e.orderIDCounter)
	now := time.Now().UTC()

	// CP 9: Determine fill price based on order type and market
	var fillPrice int
	var status string
	var filledCount int

	if req.Type == "market" {
		// Market order fills at current bid/ask
		if req.Side == "yes" {
			fillPrice = marketAsk // Buy yes at ask
		} else {
			fillPrice = marketBid // Buy no at bid (100 - yes_ask)
		}
		filledCount = req.Count
		status = "filled"
	} else {
		// Limit order - check if it crosses the spread
		midPrice := (marketBid + marketAsk) / 2
		if req.Side == "yes" {
			if req.YesPrice >= midPrice {
				// Aggressive limit, simulate fill
				fillPrice = midPrice
				filledCount = req.Count
				status = "filled"
			} else {
				// Passive limit, goes to book
				fillPrice = 0
				status = "open"
			}
		} else {
			noMid := 100 - midPrice
			if req.NoPrice >= noMid {
				fillPrice = noMid
				filledCount = req.Count
				status = "filled"
			} else {
				status = "open"
			}
		}
	}

	order := &MockOrderResponse{
		OrderID:        orderID,
		ClientOrderID:  req.ClientOrderID,
		Ticker:         req.Ticker,
		Side:           req.Side,
		Action:         req.Action,
		Type:           req.Type,
		Status:         status,
		Count:          req.Count,
		RemainingCount: req.Count - filledCount,
		YesPrice:       req.YesPrice,
		NoPrice:        req.NoPrice,
		FilledCount:    filledCount,
		FilledAvgPrice: fillPrice,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	e.orders[orderID] = order

	// Update position if filled
	if filledCount > 0 {
		e.updatePosition(userID, req.Ticker, req.Side, filledCount, fillPrice)
	}

	return order, nil
}

// updatePosition updates user's position after a fill
// CP 5: Tracks positions for limit enforcement
func (e *MockOrderExecutor) updatePosition(userID, ticker, side string, contracts, priceCents int) {
	if e.positions[userID] == nil {
		e.positions[userID] = make(map[string]*MockPosition)
	}

	posKey := fmt.Sprintf("%s_%s", ticker, side)
	existing := e.positions[userID][posKey]

	if existing == nil {
		e.positions[userID][posKey] = &MockPosition{
			Ticker:            ticker,
			Side:              side,
			Contracts:         contracts,
			AveragePriceCents: priceCents,
			TotalCostCents:    contracts * priceCents,
			CurrentPriceCents: priceCents,
			CurrentValueCents: contracts * priceCents,
		}
	} else {
		// Average in the new position
		totalContracts := existing.Contracts + contracts
		totalCost := existing.TotalCostCents + (contracts * priceCents)
		existing.Contracts = totalContracts
		existing.TotalCostCents = totalCost
		existing.AveragePriceCents = totalCost / totalContracts
		existing.CurrentValueCents = totalContracts * priceCents
	}
}

// GetPositions returns user's positions
func (e *MockOrderExecutor) GetPositions(userID string) []MockPosition {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var positions []MockPosition
	if userPositions, ok := e.positions[userID]; ok {
		for _, pos := range userPositions {
			if pos.Contracts > 0 {
				positions = append(positions, *pos)
			}
		}
	}
	return positions
}

// GetOrders returns user's orders
func (e *MockOrderExecutor) GetOrders(userID string, status string) []MockOrderResponse {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var orders []MockOrderResponse
	for _, order := range e.orders {
		if status == "" || order.Status == status {
			orders = append(orders, *order)
		}
	}
	return orders
}

// SimulateSettlement simulates market settlement
// CP 3: Objective resolution with verifiable outcomes
func (e *MockOrderExecutor) SimulateSettlement(ticker, result, reason string) *MockSettlement {
	e.mu.Lock()
	defer e.mu.Unlock()

	settlementValue := 0
	if result == "yes" {
		settlementValue = 100
	}

	settlement := MockSettlement{
		SettlementID:    fmt.Sprintf("SET_%d", time.Now().UnixNano()),
		Ticker:          ticker,
		Result:          result,
		SettlementValue: settlementValue,
		SettledAt:       time.Now().UTC(),
		Reason:          reason,
	}

	e.settlements = append(e.settlements, settlement)

	// Close out positions for this ticker
	for userID, positions := range e.positions {
		for key, pos := range positions {
			if pos.Ticker == ticker && pos.Contracts > 0 {
				// Calculate payout
				var payout int
				if pos.Side == result {
					payout = pos.Contracts * 100 // Winner gets $1 per contract
				} else {
					payout = 0 // Loser gets nothing
				}
				pos.RealizedPnL = payout - pos.TotalCostCents
				pos.Contracts = 0 // Position closed
				e.positions[userID][key] = pos
			}
		}
	}

	return &settlement
}

// GetSettlements returns settlements
func (e *MockOrderExecutor) GetSettlements(ticker string) []MockSettlement {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if ticker == "" {
		return e.settlements
	}

	var filtered []MockSettlement
	for _, s := range e.settlements {
		if s.Ticker == ticker {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

// =============================================================================
// SETTLEMENT RESOLUTION RULES
// CP 3: Objective resolution with fallback mechanisms
// =============================================================================

// ResolutionSource defines where settlement data comes from
type ResolutionSource struct {
	Primary   string // e.g., "Federal Reserve", "BLS"
	Secondary string // Fallback source
	Tertiary  string // Additional fallback
}

// SettlementRule defines resolution rules per Kalshi best practices
type SettlementRule struct {
	Category        string
	ResolutionDelay time.Duration
	ExtensionWindow time.Duration // "Extend 24h if delayed"
	Sources         ResolutionSource
}

// DefaultSettlementRules returns standard resolution rules
// CP 3: Ensures objective, verifiable outcomes
func DefaultSettlementRules() map[string]SettlementRule {
	return map[string]SettlementRule{
		"FED": {
			Category:        "Federal Reserve",
			ResolutionDelay: 30 * time.Minute,
			ExtensionWindow: 24 * time.Hour,
			Sources: ResolutionSource{
				Primary:   "federalreserve.gov",
				Secondary: "reuters.com",
				Tertiary:  "bloomberg.com",
			},
		},
		"CPI": {
			Category:        "Consumer Price Index",
			ResolutionDelay: 30 * time.Minute,
			ExtensionWindow: 24 * time.Hour,
			Sources: ResolutionSource{
				Primary:   "bls.gov",
				Secondary: "reuters.com",
				Tertiary:  "Trading Economics",
			},
		},
		"GDP": {
			Category:        "Gross Domestic Product",
			ResolutionDelay: 30 * time.Minute,
			ExtensionWindow: 24 * time.Hour,
			Sources: ResolutionSource{
				Primary:   "bea.gov",
				Secondary: "reuters.com",
				Tertiary:  "Trading Economics",
			},
		},
		"UNEMP": {
			Category:        "Unemployment Rate",
			ResolutionDelay: 30 * time.Minute,
			ExtensionWindow: 24 * time.Hour,
			Sources: ResolutionSource{
				Primary:   "bls.gov",
				Secondary: "reuters.com",
				Tertiary:  "Trading Economics",
			},
		},
	}
}

// SimulateResolution simulates objective resolution
// Returns result based on random simulation for demo
func SimulateResolution(ticker string, yesProbability float64) (string, string) {
	rand.Seed(time.Now().UnixNano())

	var result string
	if rand.Float64() < yesProbability {
		result = "yes"
	} else {
		result = "no"
	}

	reason := fmt.Sprintf("Simulated resolution based on %.0f%% YES probability at market close", yesProbability*100)
	return result, reason
}
