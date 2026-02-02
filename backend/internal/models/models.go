// Package models defines data structures for the CFTC-compliant DCM demo.
// These models support Core Principle 18 (Recordkeeping) by providing
// structured, auditable data types for all trading activity.
package models

import (
	"time"
)

// =============================================================================
// USER & AUTHENTICATION MODELS
// Core Principle 17: Fitness Standards - User eligibility tracking
// =============================================================================

type UserStatus string

const (
	UserStatusPending    UserStatus = "pending"
	UserStatusKYCPending UserStatus = "kyc_pending"
	UserStatusVerified   UserStatus = "verified"
	UserStatusSuspended  UserStatus = "suspended"
	UserStatusBanned     UserStatus = "banned"
)

// User represents a platform participant.
// CFTC Core Principle 17: Maintains fitness standards for market participants.
type User struct {
	ID            string     `json:"id"`
	Email         string     `json:"email"`
	PasswordHash  string     `json:"-"` // Never expose in JSON
	FirstName     string     `json:"first_name"`
	LastName      string     `json:"last_name"`
	Status        UserStatus `json:"status"`
	IsUSResident  bool       `json:"is_us_resident"`
	StateCode     string     `json:"state_code"` // 2-letter state code
	DateOfBirth   time.Time  `json:"date_of_birth"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	LastLoginAt   *time.Time `json:"last_login_at,omitempty"`
	KYCVerifiedAt *time.Time `json:"kyc_verified_at,omitempty"`

	// CFTC Compliance Fields
	// Core Principle 5: Position Limits
	PositionLimitUSD float64 `json:"position_limit_usd"`
	// Core Principle 18: Recordkeeping - IP tracking for audit
	LastLoginIP string `json:"last_login_ip,omitempty"`
}

// =============================================================================
// KYC/AML MODELS
// Core Principle 17: Fitness Standards
// Core Principle 18: Recordkeeping - 5-year retention required
// =============================================================================

type KYCStatus string

const (
	KYCStatusNotStarted KYCStatus = "not_started"
	KYCStatusPending    KYCStatus = "pending"
	KYCStatusApproved   KYCStatus = "approved"
	KYCStatusRejected   KYCStatus = "rejected"
	KYCStatusExpired    KYCStatus = "expired"
)

// KYCRecord tracks identity verification for AML compliance.
// Required by CEA Section 5(d) and Core Principle 17.
type KYCRecord struct {
	ID               string    `json:"id"`
	UserID           string    `json:"user_id"`
	Status           KYCStatus `json:"status"`
	DocumentType     string    `json:"document_type"` // drivers_license, passport, state_id
	DocumentNumber   string    `json:"-"`             // Encrypted, never expose
	SubmittedAt      time.Time `json:"submitted_at"`
	ReviewedAt       *time.Time `json:"reviewed_at,omitempty"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
	RejectionReason  string    `json:"rejection_reason,omitempty"`
	ReviewerNotes    string    `json:"-"` // Internal only

	// Core Principle 18: Recordkeeping
	AuditTrail []AuditEntry `json:"audit_trail,omitempty"`
}

// =============================================================================
// WALLET & FUNDS MODELS
// Core Principle 11: Financial Integrity - 100% collateralization
// Core Principle 13: Financial Resources - Segregated funds
// =============================================================================

type TransactionType string

const (
	TxTypeDeposit    TransactionType = "deposit"
	TxTypeWithdrawal TransactionType = "withdrawal"
	TxTypeTrade      TransactionType = "trade"
	TxTypeSettlement TransactionType = "settlement"
	TxTypeFee        TransactionType = "fee"
	TxTypeRefund     TransactionType = "refund"
)

type TransactionStatus string

const (
	TxStatusPending   TransactionStatus = "pending"
	TxStatusCompleted TransactionStatus = "completed"
	TxStatusFailed    TransactionStatus = "failed"
	TxStatusReversed  TransactionStatus = "reversed"
)

// Wallet represents a user's segregated funds account.
// Core Principle 13: Customer funds must be segregated.
type Wallet struct {
	ID              string    `json:"id"`
	UserID          string    `json:"user_id"`
	AvailableUSD    float64   `json:"available_usd"`    // Available for trading
	LockedUSD       float64   `json:"locked_usd"`       // Locked in open positions
	PendingUSD      float64   `json:"pending_usd"`      // Pending deposits/withdrawals
	TotalDeposited  float64   `json:"total_deposited"`  // Lifetime deposits
	TotalWithdrawn  float64   `json:"total_withdrawn"`  // Lifetime withdrawals
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Transaction records all fund movements for audit trail.
// Core Principle 18: 5-year recordkeeping requirement.
type Transaction struct {
	ID          string            `json:"id"`
	WalletID    string            `json:"wallet_id"`
	UserID      string            `json:"user_id"`
	Type        TransactionType   `json:"type"`
	Status      TransactionStatus `json:"status"`
	AmountUSD   float64           `json:"amount_usd"`
	BalanceBefore float64         `json:"balance_before"`
	BalanceAfter  float64         `json:"balance_after"`
	Reference   string            `json:"reference,omitempty"` // Order ID, ACH ref, etc.
	Description string            `json:"description"`
	CreatedAt   time.Time         `json:"created_at"`
	CompletedAt *time.Time        `json:"completed_at,omitempty"`

	// Core Principle 18: Audit metadata
	IPAddress   string `json:"ip_address,omitempty"`
	UserAgent   string `json:"user_agent,omitempty"`
}

// =============================================================================
// MARKET & ORDER MODELS
// Core Principle 2: Compliance with CEA Rules
// Core Principle 3: Contracts Not Readily Susceptible to Manipulation
// Core Principle 9: Execution of Transactions
// =============================================================================

type OrderSide string

const (
	OrderSideYes OrderSide = "yes"
	OrderSideNo  OrderSide = "no"
)

type OrderType string

const (
	OrderTypeLimit  OrderType = "limit"
	OrderTypeMarket OrderType = "market"
)

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusOpen      OrderStatus = "open"
	OrderStatusFilled    OrderStatus = "filled"
	OrderStatusPartial   OrderStatus = "partial"
	OrderStatusCancelled OrderStatus = "cancelled"
	OrderStatusRejected  OrderStatus = "rejected"
	OrderStatusExpired   OrderStatus = "expired"
)

// Order represents a trading order for a binary contract.
// Core Principle 9: Fair and equitable execution.
type Order struct {
	ID              string      `json:"id"`
	UserID          string      `json:"user_id"`
	MarketTicker    string      `json:"market_ticker"`
	EventTicker     string      `json:"event_ticker"`
	Side            OrderSide   `json:"side"`
	Type            OrderType   `json:"type"`
	Status          OrderStatus `json:"status"`
	Quantity        int         `json:"quantity"`         // Number of contracts
	FilledQuantity  int         `json:"filled_quantity"`
	PriceCents      int         `json:"price_cents"`      // 1-99 cents
	FilledPriceCents int        `json:"filled_price_cents,omitempty"`
	CollateralUSD   float64     `json:"collateral_usd"`   // Locked funds
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
	FilledAt        *time.Time  `json:"filled_at,omitempty"`
	ExpiresAt       *time.Time  `json:"expires_at,omitempty"`

	// Core Principle 4: Prevention of Market Disruption
	// Surveillance metadata
	SubmitIP        string `json:"submit_ip,omitempty"`
	DeviceFingerprint string `json:"-"` // For manipulation detection

	// Core Principle 18: Recordkeeping
	AuditTrail []AuditEntry `json:"-"`
}

// Position represents a user's holdings in a market.
// Core Principle 5: Position Limits enforcement.
type Position struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	MarketTicker  string    `json:"market_ticker"`
	EventTicker   string    `json:"event_ticker"`
	Side          OrderSide `json:"side"`
	Quantity      int       `json:"quantity"`
	AvgPriceCents int       `json:"avg_price_cents"`
	CostBasisUSD  float64   `json:"cost_basis_usd"`
	CurrentValue  float64   `json:"current_value_usd"`
	UnrealizedPnL float64   `json:"unrealized_pnl_usd"`
	RealizedPnL   float64   `json:"realized_pnl_usd"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	ClosedAt      *time.Time `json:"closed_at,omitempty"`
}

// =============================================================================
// KALSHI MARKET MODELS (from public API)
// Core Principle 3: Contracts not readily susceptible to manipulation
// =============================================================================

type MarketStatus string

const (
	MarketStatusOpen    MarketStatus = "open"
	MarketStatusClosed  MarketStatus = "closed"
	MarketStatusSettled MarketStatus = "settled"
)

// KalshiMarket represents a binary event contract from Kalshi.
type KalshiMarket struct {
	Ticker          string       `json:"ticker"`
	EventTicker     string       `json:"event_ticker"`
	SeriesTicker    string       `json:"series_ticker"`
	Title           string       `json:"title"`
	Subtitle        string       `json:"subtitle"`
	Status          MarketStatus `json:"status"`
	Category        string       `json:"category"`
	YesBid          int          `json:"yes_bid"`
	YesAsk          int          `json:"yes_ask"`
	NoBid           int          `json:"no_bid"`
	NoAsk           int          `json:"no_ask"`
	LastPrice       int          `json:"last_price"`
	Volume          int64        `json:"volume"`
	Volume24H       int64        `json:"volume_24h"`
	OpenInterest    int64        `json:"open_interest"`
	OpenTime        time.Time    `json:"open_time"`
	CloseTime       time.Time    `json:"close_time"`
	ExpirationTime  time.Time    `json:"expiration_time"`
	SettlementValue *int         `json:"settlement_value,omitempty"`
	Result          string       `json:"result,omitempty"`

	// Core Principle 3: Risk classification
	RiskCategory    string `json:"risk_category,omitempty"` // low, medium, high
}

// =============================================================================
// COMPLIANCE & AUDIT MODELS
// Core Principle 4: Prevention of Market Disruption
// Core Principle 18: Recordkeeping and Reporting
// =============================================================================

type AuditAction string

const (
	AuditActionCreate   AuditAction = "create"
	AuditActionUpdate   AuditAction = "update"
	AuditActionDelete   AuditAction = "delete"
	AuditActionLogin    AuditAction = "login"
	AuditActionLogout   AuditAction = "logout"
	AuditActionTrade    AuditAction = "trade"
	AuditActionKYC      AuditAction = "kyc"
	AuditActionDeposit  AuditAction = "deposit"
	AuditActionWithdraw AuditAction = "withdraw"
	AuditActionSuspend  AuditAction = "suspend"
	AuditActionHalt     AuditAction = "halt"
)

// AuditEntry provides immutable audit trail for compliance.
// Core Principle 18: Must retain for 5 years minimum.
type AuditEntry struct {
	ID          string      `json:"id"`
	Timestamp   time.Time   `json:"timestamp"`
	UserID      string      `json:"user_id,omitempty"`
	Action      AuditAction `json:"action"`
	EntityType  string      `json:"entity_type"` // user, order, position, etc.
	EntityID    string      `json:"entity_id"`
	OldValue    string      `json:"old_value,omitempty"` // JSON of previous state
	NewValue    string      `json:"new_value,omitempty"` // JSON of new state
	IPAddress   string      `json:"ip_address,omitempty"`
	UserAgent   string      `json:"user_agent,omitempty"`
	Description string      `json:"description"`
}

// ComplianceAlert for market surveillance.
// Core Principle 4: Capacity to detect and prevent manipulation.
type ComplianceAlert struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"` // wash_trade, spoofing, position_limit, etc.
	Severity    string    `json:"severity"` // low, medium, high, critical
	UserID      string    `json:"user_id,omitempty"`
	MarketTicker string   `json:"market_ticker,omitempty"`
	Description string    `json:"description"`
	Evidence    string    `json:"evidence"` // JSON data
	Status      string    `json:"status"`   // open, investigating, resolved, escalated
	CreatedAt   time.Time `json:"created_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
	ResolvedBy  string    `json:"resolved_by,omitempty"`
	Notes       string    `json:"notes,omitempty"` // Resolution notes
}

// EmergencyHalt tracks market-wide or market-specific trading halts.
// Core Principle 4: Emergency authority.
type EmergencyHalt struct {
	ID           string     `json:"id"`
	MarketTicker string     `json:"market_ticker,omitempty"` // Empty = market-wide
	Reason       string     `json:"reason"`
	InitiatedBy  string     `json:"initiated_by"`
	StartedAt    time.Time  `json:"started_at"`
	EndsAt       *time.Time `json:"ends_at,omitempty"`
	IsActive     bool       `json:"is_active"`
}
