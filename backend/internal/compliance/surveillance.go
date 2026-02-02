// Package compliance provides CFTC Core Principle enforcement.
// Implements surveillance, position limits, and market integrity checks.
package compliance

import (
	"fmt"
	"sync"
	"time"

	"github.com/kalshi-dcm-demo/backend/internal/mock"
	"github.com/kalshi-dcm-demo/backend/internal/models"
)

// =============================================================================
// SURVEILLANCE ENGINE
// Core Principle 4: Prevention of Market Disruption
// Core Principle 5: Position Limits
// =============================================================================

// SurveillanceEngine monitors trading activity for manipulation patterns.
type SurveillanceEngine struct {
	store *mock.Store

	// Thresholds (configurable per Core Principle 5)
	maxPositionUSD        float64
	maxOrdersPerMinute    int
	suspiciousVolumeRatio float64

	// Tracking
	orderCounts map[string][]time.Time // userID -> order timestamps
	mu          sync.RWMutex
}

// NewSurveillanceEngine creates a new surveillance engine.
func NewSurveillanceEngine(store *mock.Store) *SurveillanceEngine {
	return &SurveillanceEngine{
		store:                 store,
		maxPositionUSD:        25000.00, // Default per-user limit
		maxOrdersPerMinute:    60,       // Rate limiting
		suspiciousVolumeRatio: 0.10,     // 10% of market volume
		orderCounts:           make(map[string][]time.Time),
	}
}

// =============================================================================
// PRE-TRADE CHECKS
// Core Principle 11: Financial Integrity - 100% collateralization
// =============================================================================

// PreTradeCheck validates an order before submission.
type PreTradeCheck struct {
	Passed          bool     `json:"passed"`
	Errors          []string `json:"errors,omitempty"`
	Warnings        []string `json:"warnings,omitempty"`
	RequiredMargin  float64  `json:"required_margin_usd"`
	AvailableMargin float64  `json:"available_margin_usd"`
}

// ValidateOrder performs comprehensive pre-trade compliance checks.
// Core Principle 11: Ensures 100% collateralization.
// Core Principle 5: Enforces position limits.
func (s *SurveillanceEngine) ValidateOrder(userID, marketTicker string, side models.OrderSide, quantity, priceCents int) *PreTradeCheck {
	check := &PreTradeCheck{
		Passed:   true,
		Errors:   make([]string, 0),
		Warnings: make([]string, 0),
	}

	// Calculate required margin (100% collateralization)
	// Core Principle 11: Binary contracts require full collateral
	var marginCents int
	if side == models.OrderSideYes {
		marginCents = quantity * priceCents
	} else {
		marginCents = quantity * (100 - priceCents)
	}
	check.RequiredMargin = float64(marginCents) / 100.0

	// Get user wallet
	wallet, err := s.store.GetWallet(userID)
	if err != nil {
		check.Passed = false
		check.Errors = append(check.Errors, "Wallet not found")
		return check
	}
	check.AvailableMargin = wallet.AvailableUSD

	// Check 1: Sufficient funds (Core Principle 11)
	if wallet.AvailableUSD < check.RequiredMargin {
		check.Passed = false
		check.Errors = append(check.Errors, fmt.Sprintf(
			"Insufficient funds: need $%.2f, available $%.2f",
			check.RequiredMargin, wallet.AvailableUSD))
	}

	// Check 2: Position limits (Core Principle 5)
	user, err := s.store.GetUser(userID)
	if err != nil {
		check.Passed = false
		check.Errors = append(check.Errors, "User not found")
		return check
	}

	currentExposure := s.store.GetUserExposure(userID)
	newExposure := currentExposure + check.RequiredMargin
	if newExposure > user.PositionLimitUSD {
		check.Passed = false
		check.Errors = append(check.Errors, fmt.Sprintf(
			"Position limit exceeded: current $%.2f + order $%.2f > limit $%.2f",
			currentExposure, check.RequiredMargin, user.PositionLimitUSD))
	}

	// Check 3: Rate limiting (Core Principle 4)
	if s.isRateLimited(userID) {
		check.Passed = false
		check.Errors = append(check.Errors, "Order rate limit exceeded. Please wait.")
	}

	// Check 4: Trading halt (Core Principle 4)
	if s.store.IsTradingHalted(marketTicker) {
		check.Passed = false
		check.Errors = append(check.Errors, "Trading is currently halted for this market")
	}

	// Warning: Approaching position limit
	if newExposure > user.PositionLimitUSD*0.8 {
		check.Warnings = append(check.Warnings, fmt.Sprintf(
			"Approaching position limit (%.0f%% utilized)",
			(newExposure/user.PositionLimitUSD)*100))
	}

	return check
}

// isRateLimited checks if user is submitting orders too quickly.
// Core Principle 4: Prevents potential manipulation through rapid-fire orders.
func (s *SurveillanceEngine) isRateLimited(userID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-time.Minute)

	// Get order timestamps for this user
	timestamps := s.orderCounts[userID]

	// Filter to only recent orders
	var recent []time.Time
	for _, ts := range timestamps {
		if ts.After(cutoff) {
			recent = append(recent, ts)
		}
	}

	// Add current timestamp
	recent = append(recent, now)
	s.orderCounts[userID] = recent

	return len(recent) > s.maxOrdersPerMinute
}

// =============================================================================
// POST-TRADE SURVEILLANCE
// Core Principle 4: Detection of manipulation
// =============================================================================

// AnalyzeTradePattern checks for suspicious trading patterns.
// This is a stub - production would use ML/statistical analysis.
func (s *SurveillanceEngine) AnalyzeTradePattern(userID, marketTicker string, orders []models.Order) []models.ComplianceAlert {
	var alerts []models.ComplianceAlert

	// Pattern 1: Wash trading detection (stub)
	// Core Principle 4: Same user buying/selling to create false volume
	if s.detectWashTrading(orders) {
		alert := s.store.CreateComplianceAlert(userID, marketTicker, "wash_trade", "high",
			"Potential wash trading detected: rapid buy/sell pattern")
		alerts = append(alerts, *alert)
	}

	// Pattern 2: Spoofing detection (stub)
	// Core Principle 4: Placing orders with intent to cancel
	if s.detectSpoofing(orders) {
		alert := s.store.CreateComplianceAlert(userID, marketTicker, "spoofing", "high",
			"Potential spoofing detected: large orders placed and cancelled")
		alerts = append(alerts, *alert)
	}

	// Pattern 3: Layering detection (stub)
	// Core Principle 4: Multiple orders at different prices to influence
	if s.detectLayering(orders) {
		alert := s.store.CreateComplianceAlert(userID, marketTicker, "layering", "medium",
			"Potential layering detected: stacked orders at multiple price levels")
		alerts = append(alerts, *alert)
	}

	return alerts
}

// detectWashTrading identifies potential wash trades.
// Stub implementation - production uses statistical analysis.
func (s *SurveillanceEngine) detectWashTrading(orders []models.Order) bool {
	// In production: Check for offsetting trades within short time windows
	// from same user or related accounts
	if len(orders) < 2 {
		return false
	}

	// Simplified check: Look for rapid buy/sell pairs
	for i := 0; i < len(orders)-1; i++ {
		for j := i + 1; j < len(orders); j++ {
			if orders[i].Side != orders[j].Side &&
				orders[i].MarketTicker == orders[j].MarketTicker &&
				orders[j].CreatedAt.Sub(orders[i].CreatedAt) < time.Minute {
				return true
			}
		}
	}
	return false
}

// detectSpoofing identifies potential spoofing behavior.
// Stub implementation.
func (s *SurveillanceEngine) detectSpoofing(orders []models.Order) bool {
	// In production: Check for large orders that get cancelled
	// before execution, especially when price moves
	cancelledLarge := 0
	for _, order := range orders {
		if order.Status == models.OrderStatusCancelled && order.Quantity > 100 {
			cancelledLarge++
		}
	}
	return cancelledLarge > 3
}

// detectLayering identifies potential layering behavior.
// Stub implementation.
func (s *SurveillanceEngine) detectLayering(orders []models.Order) bool {
	// In production: Check for multiple orders at incrementing prices
	// that get cancelled after price moves
	priceCount := make(map[int]int)
	for _, order := range orders {
		if order.Status == models.OrderStatusOpen {
			priceCount[order.PriceCents]++
		}
	}
	return len(priceCount) > 5
}

// =============================================================================
// POSITION LIMIT MANAGEMENT
// Core Principle 5: Position Limits
// =============================================================================

// PositionLimitConfig defines limits per user tier.
type PositionLimitConfig struct {
	Tier         string  `json:"tier"`
	MaxPositionUSD float64 `json:"max_position_usd"`
	MaxOrderSize   int     `json:"max_order_size"`
	DailyVolumeUSD float64 `json:"daily_volume_usd"`
}

// DefaultPositionLimits returns tiered limits.
// Core Principle 5: Speculative position limits.
func DefaultPositionLimits() []PositionLimitConfig {
	return []PositionLimitConfig{
		{Tier: "basic", MaxPositionUSD: 25000, MaxOrderSize: 500, DailyVolumeUSD: 10000},
		{Tier: "standard", MaxPositionUSD: 100000, MaxOrderSize: 2000, DailyVolumeUSD: 50000},
		{Tier: "professional", MaxPositionUSD: 500000, MaxOrderSize: 10000, DailyVolumeUSD: 250000},
	}
}

// CheckPositionLimit validates against configured limits.
// Core Principle 5: Prevents excessive concentration.
func (s *SurveillanceEngine) CheckPositionLimit(userID, marketTicker string, additionalExposure float64) error {
	user, err := s.store.GetUser(userID)
	if err != nil {
		return err
	}

	currentExposure := s.store.GetUserExposure(userID)
	totalExposure := currentExposure + additionalExposure

	if totalExposure > user.PositionLimitUSD {
		return fmt.Errorf("position limit exceeded: $%.2f > $%.2f",
			totalExposure, user.PositionLimitUSD)
	}

	return nil
}

// =============================================================================
// EMERGENCY CONTROLS
// Core Principle 4: Emergency authority
// =============================================================================

// HaltTrading initiates an emergency trading halt.
// Core Principle 4: DCM must have emergency authority.
func (s *SurveillanceEngine) HaltTrading(marketTicker, reason, initiatedBy string) *models.EmergencyHalt {
	return s.store.InitiateEmergencyHalt(marketTicker, reason, initiatedBy)
}

// ResumeTrading lifts an emergency halt.
func (s *SurveillanceEngine) ResumeTrading(marketTicker string) error {
	return s.store.LiftEmergencyHalt(marketTicker)
}

// =============================================================================
// RECORDKEEPING
// Core Principle 18: Recordkeeping and Reporting
// Must maintain records for minimum 5 years.
// =============================================================================

// ComplianceReport generates audit data for regulators.
type ComplianceReport struct {
	GeneratedAt   time.Time               `json:"generated_at"`
	PeriodStart   time.Time               `json:"period_start"`
	PeriodEnd     time.Time               `json:"period_end"`
	TotalUsers    int                     `json:"total_users"`
	TotalOrders   int                     `json:"total_orders"`
	TotalVolume   float64                 `json:"total_volume_usd"`
	Alerts        []models.ComplianceAlert `json:"alerts"`
	Halts         []models.EmergencyHalt  `json:"halts"`
	AuditEntries  []models.AuditEntry     `json:"audit_entries"`
}

// GenerateComplianceReport creates a regulatory report.
// Core Principle 18: Required for CFTC reporting.
func (s *SurveillanceEngine) GenerateComplianceReport(start, end time.Time) *ComplianceReport {
	report := &ComplianceReport{
		GeneratedAt: time.Now().UTC(),
		PeriodStart: start,
		PeriodEnd:   end,
	}

	// In production: Query database for comprehensive data
	// This stub shows the structure required

	report.AuditEntries = s.store.GetAuditLog("", start, 10000)

	return report
}
