// Package compliance provides CFTC Core Principle 4 surveillance testing.
package compliance

import (
	"testing"
	"time"

	"github.com/kalshi-dcm-demo/backend/internal/mock"
	"github.com/kalshi-dcm-demo/backend/internal/models"
)

// =============================================================================
// TEST FIXTURES
// =============================================================================

func setupTestEngine() *SurveillanceEngine {
	store := mock.NewStore()
	return NewSurveillanceEngine(store)
}

func createTestOrder(side models.OrderSide, qty, price int, createdAt time.Time) models.Order {
	return models.Order{
		ID:           "test_order",
		UserID:       "user_123",
		MarketTicker: "FED-RATE-MAR",
		Side:         side,
		Quantity:     qty,
		PriceCents:   price,
		CreatedAt:    createdAt,
	}
}

// =============================================================================
// POSITION LIMIT TESTS
// Core Principle 5: Position Limits
// =============================================================================

func TestValidateOrder_PassesWithinLimits(t *testing.T) {
	engine := setupTestEngine()

	// Create user with position limit
	store := engine.store
	store.CreateUser(
		"test@example.com",
		"password",
		"Test",
		"User",
		"NY",
		time.Now().AddDate(-30, 0, 0),
		true,
		"127.0.0.1",
	)

	// Validate a small order
	check := engine.ValidateOrder("user_123", "FED-RATE-MAR", models.OrderSideYes, 10, 50)

	if !check.Passed {
		t.Errorf("Expected order to pass, got errors: %v", check.Errors)
	}
}

func TestValidateOrder_CalculatesCorrectMargin(t *testing.T) {
	engine := setupTestEngine()

	// YES side: margin = quantity * price
	checkYes := engine.ValidateOrder("user_123", "FED-RATE-MAR", models.OrderSideYes, 100, 65)
	expectedYesMargin := float64(100*65) / 100.0 // 65.00 USD
	if checkYes.RequiredMargin != expectedYesMargin {
		t.Errorf("YES margin: expected %.2f, got %.2f", expectedYesMargin, checkYes.RequiredMargin)
	}

	// NO side: margin = quantity * (100 - price)
	checkNo := engine.ValidateOrder("user_123", "FED-RATE-MAR", models.OrderSideNo, 100, 65)
	expectedNoMargin := float64(100*35) / 100.0 // 35.00 USD
	if checkNo.RequiredMargin != expectedNoMargin {
		t.Errorf("NO margin: expected %.2f, got %.2f", expectedNoMargin, checkNo.RequiredMargin)
	}
}

func TestValidateOrder_RejectsExcessiveQuantity(t *testing.T) {
	engine := setupTestEngine()

	// Try to place order for 10,000 contracts (should fail)
	check := engine.ValidateOrder("user_123", "FED-RATE-MAR", models.OrderSideYes, 10000, 50)

	if check.Passed {
		t.Error("Expected order to fail due to quantity limit")
	}

	foundQuantityError := false
	for _, err := range check.Errors {
		if err == "Quantity exceeds maximum allowed (1000)" {
			foundQuantityError = true
			break
		}
	}
	if !foundQuantityError {
		t.Errorf("Expected quantity error, got: %v", check.Errors)
	}
}

func TestValidateOrder_RejectsInvalidPrice(t *testing.T) {
	engine := setupTestEngine()

	// Price must be 1-99
	testCases := []struct {
		price    int
		expected bool // should pass
	}{
		{0, false},
		{1, true},
		{50, true},
		{99, true},
		{100, false},
		{-5, false},
	}

	for _, tc := range testCases {
		check := engine.ValidateOrder("user_123", "FED-RATE-MAR", models.OrderSideYes, 10, tc.price)
		if check.Passed != tc.expected {
			t.Errorf("Price %d: expected passed=%v, got passed=%v", tc.price, tc.expected, check.Passed)
		}
	}
}

// =============================================================================
// WASH TRADE DETECTION TESTS
// Core Principle 4: Prevention of Market Disruption
// =============================================================================

func TestDetectWashTrading_IdentifiesOpposingTrades(t *testing.T) {
	engine := setupTestEngine()
	now := time.Now()

	// Create opposing trades within 60 seconds
	orders := []models.Order{
		createTestOrder(models.OrderSideYes, 100, 50, now),
		createTestOrder(models.OrderSideNo, 100, 50, now.Add(30*time.Second)),
	}

	alerts := engine.AnalyzeTradePattern("user_123", "FED-RATE-MAR", orders)

	foundWashAlert := false
	for _, alert := range alerts {
		if alert == "Potential wash trading detected: opposing positions within 60 seconds" {
			foundWashAlert = true
			break
		}
	}

	if !foundWashAlert {
		t.Error("Expected wash trading alert to be detected")
	}
}

func TestDetectWashTrading_IgnoresLegitimateHedges(t *testing.T) {
	engine := setupTestEngine()
	now := time.Now()

	// Trades more than 60 seconds apart should not trigger
	orders := []models.Order{
		createTestOrder(models.OrderSideYes, 100, 50, now),
		createTestOrder(models.OrderSideNo, 100, 50, now.Add(5*time.Minute)),
	}

	alerts := engine.AnalyzeTradePattern("user_123", "FED-RATE-MAR", orders)

	for _, alert := range alerts {
		if alert == "Potential wash trading detected: opposing positions within 60 seconds" {
			t.Error("Should not detect wash trading for trades 5 minutes apart")
		}
	}
}

// =============================================================================
// SPOOFING DETECTION TESTS
// Core Principle 4: Prevention of Market Disruption
// =============================================================================

func TestDetectSpoofing_IdentifiesLargeCancelledOrders(t *testing.T) {
	engine := setupTestEngine()
	now := time.Now()

	// Large order cancelled quickly
	order := createTestOrder(models.OrderSideYes, 500, 50, now)
	order.Status = models.OrderStatusCancelled
	cancelTime := now.Add(5 * time.Second)
	order.CancelledAt = &cancelTime

	orders := []models.Order{order}
	alerts := engine.AnalyzeTradePattern("user_123", "FED-RATE-MAR", orders)

	foundSpoofAlert := false
	for _, alert := range alerts {
		if alert == "Potential spoofing: large order cancelled within 10 seconds" {
			foundSpoofAlert = true
			break
		}
	}

	if !foundSpoofAlert {
		t.Error("Expected spoofing alert to be detected")
	}
}

func TestDetectSpoofing_IgnoresNormalCancellations(t *testing.T) {
	engine := setupTestEngine()
	now := time.Now()

	// Small order cancelled (not suspicious)
	order := createTestOrder(models.OrderSideYes, 10, 50, now)
	order.Status = models.OrderStatusCancelled
	cancelTime := now.Add(5 * time.Second)
	order.CancelledAt = &cancelTime

	orders := []models.Order{order}
	alerts := engine.AnalyzeTradePattern("user_123", "FED-RATE-MAR", orders)

	for _, alert := range alerts {
		if alert == "Potential spoofing: large order cancelled within 10 seconds" {
			t.Error("Should not detect spoofing for small orders")
		}
	}
}

// =============================================================================
// LAYERING DETECTION TESTS
// Core Principle 4: Prevention of Market Disruption
// =============================================================================

func TestDetectLayering_IdentifiesStackedOrders(t *testing.T) {
	engine := setupTestEngine()
	now := time.Now()

	// Multiple orders at different price levels (layering pattern)
	orders := []models.Order{
		{ID: "1", Side: models.OrderSideYes, PriceCents: 50, Status: models.OrderStatusOpen, CreatedAt: now},
		{ID: "2", Side: models.OrderSideYes, PriceCents: 51, Status: models.OrderStatusOpen, CreatedAt: now},
		{ID: "3", Side: models.OrderSideYes, PriceCents: 52, Status: models.OrderStatusOpen, CreatedAt: now},
		{ID: "4", Side: models.OrderSideYes, PriceCents: 53, Status: models.OrderStatusOpen, CreatedAt: now},
		{ID: "5", Side: models.OrderSideYes, PriceCents: 54, Status: models.OrderStatusOpen, CreatedAt: now},
		{ID: "6", Side: models.OrderSideYes, PriceCents: 55, Status: models.OrderStatusOpen, CreatedAt: now},
	}

	alerts := engine.AnalyzeTradePattern("user_123", "FED-RATE-MAR", orders)

	foundLayeringAlert := false
	for _, alert := range alerts {
		if alert == "Potential layering: 6 open orders at different price levels" {
			foundLayeringAlert = true
			break
		}
	}

	if !foundLayeringAlert {
		t.Errorf("Expected layering alert, got: %v", alerts)
	}
}

// =============================================================================
// EMERGENCY HALT TESTS
// Core Principle 4: Prevention of Market Disruption
// =============================================================================

func TestEmergencyHalt_HaltsTrading(t *testing.T) {
	engine := setupTestEngine()

	halt := engine.HaltTrading("FED-RATE-MAR", "Unusual volatility", "admin@dcm.com")

	if halt == nil {
		t.Fatal("Expected halt to be created")
	}

	if !halt.IsActive {
		t.Error("Halt should be active")
	}

	if halt.MarketTicker != "FED-RATE-MAR" {
		t.Errorf("Expected ticker FED-RATE-MAR, got %s", halt.MarketTicker)
	}

	// Verify trading is halted
	if !engine.IsHalted("FED-RATE-MAR") {
		t.Error("Trading should be halted for FED-RATE-MAR")
	}
}

func TestEmergencyHalt_GlobalHalt(t *testing.T) {
	engine := setupTestEngine()

	// Empty ticker = global halt
	halt := engine.HaltTrading("", "System maintenance", "admin@dcm.com")

	if halt == nil {
		t.Fatal("Expected global halt to be created")
	}

	// Any market should be halted
	if !engine.IsHalted("ANY-MARKET") {
		t.Error("Global halt should affect all markets")
	}
}

func TestEmergencyHalt_ResumeTrading(t *testing.T) {
	engine := setupTestEngine()

	// Halt then resume
	engine.HaltTrading("FED-RATE-MAR", "Test halt", "admin@dcm.com")

	if !engine.IsHalted("FED-RATE-MAR") {
		t.Error("Trading should be halted")
	}

	engine.ResumeTrading("FED-RATE-MAR")

	if engine.IsHalted("FED-RATE-MAR") {
		t.Error("Trading should be resumed")
	}
}

// =============================================================================
// PRE-TRADE CHECK INTEGRATION TESTS
// Core Principle 11: Financial Integrity
// =============================================================================

func TestPreTradeCheck_VerifiesCollateral(t *testing.T) {
	engine := setupTestEngine()

	// Small order should pass
	smallCheck := engine.ValidateOrder("user_123", "FED-RATE-MAR", models.OrderSideYes, 10, 50)
	if !smallCheck.Passed {
		t.Error("Small order should pass pre-trade check")
	}

	// Verify margin is 100% collateralized
	// 10 contracts * 50 cents = 500 cents = $5.00
	expectedMargin := 5.0
	if smallCheck.RequiredMargin != expectedMargin {
		t.Errorf("Expected margin $%.2f, got $%.2f", expectedMargin, smallCheck.RequiredMargin)
	}
}

// =============================================================================
// CONCURRENT ACCESS TESTS
// =============================================================================

func TestSurveillance_ConcurrentAccess(t *testing.T) {
	engine := setupTestEngine()

	// Simulate concurrent validations
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			engine.ValidateOrder("user_123", "FED-RATE-MAR", models.OrderSideYes, 10, 50)
			engine.AnalyzeTradePattern("user_123", "FED-RATE-MAR", []models.Order{})
			engine.IsHalted("FED-RATE-MAR")
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// =============================================================================
// BENCHMARK TESTS
// =============================================================================

func BenchmarkValidateOrder(b *testing.B) {
	engine := setupTestEngine()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.ValidateOrder("user_123", "FED-RATE-MAR", models.OrderSideYes, 10, 50)
	}
}

func BenchmarkAnalyzeTradePattern(b *testing.B) {
	engine := setupTestEngine()
	orders := make([]models.Order, 100)
	now := time.Now()
	for i := range orders {
		orders[i] = createTestOrder(models.OrderSideYes, 10, 50, now.Add(time.Duration(i)*time.Minute))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.AnalyzeTradePattern("user_123", "FED-RATE-MAR", orders)
	}
}
