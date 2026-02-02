// Package mock provides in-memory data stores for the DCM demo.
// This simulates database operations without requiring actual DB infrastructure.
// Core Principle 18: All operations are logged for audit trail compliance.
package mock

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/kalshi-dcm-demo/backend/internal/models"
)

// =============================================================================
// ERRORS
// =============================================================================

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserExists         = errors.New("user already exists")
	ErrWalletNotFound     = errors.New("wallet not found")
	ErrInsufficientFunds  = errors.New("insufficient funds")
	ErrOrderNotFound      = errors.New("order not found")
	ErrPositionNotFound   = errors.New("position not found")
	ErrKYCRequired        = errors.New("KYC verification required")
	ErrUserSuspended      = errors.New("user account suspended")
	ErrMarketClosed       = errors.New("market is closed")
	ErrPositionLimitExceeded = errors.New("position limit exceeded")
	ErrTradingHalted      = errors.New("trading is currently halted")
)

// =============================================================================
// STORE - Thread-safe in-memory data store
// =============================================================================

type Store struct {
	// User data
	users      map[string]*models.User
	usersByEmail map[string]string // email -> userID
	usersMu    sync.RWMutex

	// KYC records
	kycRecords   map[string]*models.KYCRecord // userID -> KYC
	kycRecordsMu sync.RWMutex

	// Wallets
	wallets   map[string]*models.Wallet // userID -> Wallet
	walletsMu sync.RWMutex

	// Transactions
	transactions   map[string]*models.Transaction
	txByWallet     map[string][]string // walletID -> []txID
	transactionsMu sync.RWMutex

	// Orders
	orders     map[string]*models.Order
	ordersByUser map[string][]string // userID -> []orderID
	ordersMu   sync.RWMutex

	// Positions
	positions     map[string]*models.Position
	positionsByUser map[string][]string // userID -> []positionID
	positionsMu   sync.RWMutex

	// Audit log
	auditLog   []models.AuditEntry
	auditLogMu sync.RWMutex

	// Compliance
	alerts       []models.ComplianceAlert
	alertsMu     sync.RWMutex
	halts        map[string]*models.EmergencyHalt // marketTicker or "GLOBAL" -> halt
	haltsMu      sync.RWMutex

	// ID counters
	idCounter   int64
	idCounterMu sync.Mutex
}

// NewStore creates a new in-memory store instance.
func NewStore() *Store {
	return &Store{
		users:         make(map[string]*models.User),
		usersByEmail:  make(map[string]string),
		kycRecords:    make(map[string]*models.KYCRecord),
		wallets:       make(map[string]*models.Wallet),
		transactions:  make(map[string]*models.Transaction),
		txByWallet:    make(map[string][]string),
		orders:        make(map[string]*models.Order),
		ordersByUser:  make(map[string][]string),
		positions:     make(map[string]*models.Position),
		positionsByUser: make(map[string][]string),
		auditLog:      make([]models.AuditEntry, 0),
		alerts:        make([]models.ComplianceAlert, 0),
		halts:         make(map[string]*models.EmergencyHalt),
	}
}

// generateID creates a unique ID for entities.
func (s *Store) generateID(prefix string) string {
	s.idCounterMu.Lock()
	defer s.idCounterMu.Unlock()
	s.idCounter++
	return fmt.Sprintf("%s_%d_%d", prefix, time.Now().UnixNano(), s.idCounter)
}

// =============================================================================
// AUDIT LOGGING
// Core Principle 18: Recordkeeping and Reporting
// All records must be retained for minimum 5 years.
// =============================================================================

// LogAudit creates an immutable audit entry.
func (s *Store) LogAudit(userID string, action models.AuditAction, entityType, entityID string, oldVal, newVal interface{}, ip, ua, desc string) {
	s.auditLogMu.Lock()
	defer s.auditLogMu.Unlock()

	var oldJSON, newJSON string
	if oldVal != nil {
		if b, err := json.Marshal(oldVal); err == nil {
			oldJSON = string(b)
		}
	}
	if newVal != nil {
		if b, err := json.Marshal(newVal); err == nil {
			newJSON = string(b)
		}
	}

	entry := models.AuditEntry{
		ID:          s.generateID("audit"),
		Timestamp:   time.Now().UTC(),
		UserID:      userID,
		Action:      action,
		EntityType:  entityType,
		EntityID:    entityID,
		OldValue:    oldJSON,
		NewValue:    newJSON,
		IPAddress:   ip,
		UserAgent:   ua,
		Description: desc,
	}
	s.auditLog = append(s.auditLog, entry)
}

// GetAuditLog retrieves audit entries with optional filters.
func (s *Store) GetAuditLog(userID string, since time.Time, limit int) []models.AuditEntry {
	s.auditLogMu.RLock()
	defer s.auditLogMu.RUnlock()

	var results []models.AuditEntry
	for i := len(s.auditLog) - 1; i >= 0 && len(results) < limit; i-- {
		entry := s.auditLog[i]
		if entry.Timestamp.Before(since) {
			continue
		}
		if userID != "" && entry.UserID != userID {
			continue
		}
		results = append(results, entry)
	}
	return results
}

// =============================================================================
// USER OPERATIONS
// Core Principle 17: Fitness Standards
// =============================================================================

// CreateUser registers a new user.
func (s *Store) CreateUser(email, passwordHash, firstName, lastName, stateCode string, dob time.Time, isUSResident bool, ip string) (*models.User, error) {
	s.usersMu.Lock()
	defer s.usersMu.Unlock()

	// Check if email exists
	if _, exists := s.usersByEmail[email]; exists {
		return nil, ErrUserExists
	}

	now := time.Now().UTC()
	user := &models.User{
		ID:              s.generateID("user"),
		Email:           email,
		PasswordHash:    passwordHash,
		FirstName:       firstName,
		LastName:        lastName,
		Status:          models.UserStatusKYCPending,
		IsUSResident:    isUSResident,
		StateCode:       stateCode,
		DateOfBirth:     dob,
		CreatedAt:       now,
		UpdatedAt:       now,
		PositionLimitUSD: 25000.00, // Default position limit per Core Principle 5
		LastLoginIP:     ip,
	}

	s.users[user.ID] = user
	s.usersByEmail[email] = user.ID

	// Core Principle 18: Log user creation
	s.LogAudit(user.ID, models.AuditActionCreate, "user", user.ID, nil, user, ip, "", "User account created")

	return user, nil
}

// GetUser retrieves a user by ID.
func (s *Store) GetUser(userID string) (*models.User, error) {
	s.usersMu.RLock()
	defer s.usersMu.RUnlock()

	user, exists := s.users[userID]
	if !exists {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// GetUserByEmail retrieves a user by email.
func (s *Store) GetUserByEmail(email string) (*models.User, error) {
	s.usersMu.RLock()
	defer s.usersMu.RUnlock()

	userID, exists := s.usersByEmail[email]
	if !exists {
		return nil, ErrUserNotFound
	}
	return s.users[userID], nil
}

// UpdateUserStatus changes user status (verified, suspended, banned).
func (s *Store) UpdateUserStatus(userID string, status models.UserStatus, ip string) error {
	s.usersMu.Lock()
	defer s.usersMu.Unlock()

	user, exists := s.users[userID]
	if !exists {
		return ErrUserNotFound
	}

	oldStatus := user.Status
	user.Status = status
	user.UpdatedAt = time.Now().UTC()

	if status == models.UserStatusVerified {
		now := time.Now().UTC()
		user.KYCVerifiedAt = &now
	}

	// Core Principle 18: Log status change
	s.LogAudit(userID, models.AuditActionUpdate, "user", userID,
		map[string]interface{}{"status": oldStatus},
		map[string]interface{}{"status": status},
		ip, "", fmt.Sprintf("User status changed from %s to %s", oldStatus, status))

	return nil
}

// RecordLogin updates last login info.
func (s *Store) RecordLogin(userID, ip string) error {
	s.usersMu.Lock()
	defer s.usersMu.Unlock()

	user, exists := s.users[userID]
	if !exists {
		return ErrUserNotFound
	}

	now := time.Now().UTC()
	user.LastLoginAt = &now
	user.LastLoginIP = ip

	s.LogAudit(userID, models.AuditActionLogin, "user", userID, nil, nil, ip, "", "User logged in")
	return nil
}

// =============================================================================
// KYC OPERATIONS
// Core Principle 17: Fitness Standards
// =============================================================================

// CreateKYCRecord initiates KYC verification.
func (s *Store) CreateKYCRecord(userID, docType, docNumber, ip string) (*models.KYCRecord, error) {
	s.kycRecordsMu.Lock()
	defer s.kycRecordsMu.Unlock()

	now := time.Now().UTC()
	record := &models.KYCRecord{
		ID:             s.generateID("kyc"),
		UserID:         userID,
		Status:         models.KYCStatusPending,
		DocumentType:   docType,
		DocumentNumber: docNumber,
		SubmittedAt:    now,
	}

	s.kycRecords[userID] = record

	s.LogAudit(userID, models.AuditActionKYC, "kyc", record.ID, nil, record, ip, "", "KYC verification submitted")
	return record, nil
}

// MockKYCApproval simulates KYC approval (demo only).
// In production, this would integrate with identity verification services.
func (s *Store) MockKYCApproval(userID string, approved bool, reason string) error {
	s.kycRecordsMu.Lock()
	defer s.kycRecordsMu.Unlock()

	record, exists := s.kycRecords[userID]
	if !exists {
		return ErrUserNotFound
	}

	now := time.Now().UTC()
	record.ReviewedAt = &now

	if approved {
		record.Status = models.KYCStatusApproved
		expiry := now.AddDate(2, 0, 0) // 2 year expiry
		record.ExpiresAt = &expiry
		// Update user status
		s.UpdateUserStatus(userID, models.UserStatusVerified, "system")
	} else {
		record.Status = models.KYCStatusRejected
		record.RejectionReason = reason
	}

	return nil
}

// GetKYCRecord retrieves KYC status for a user.
func (s *Store) GetKYCRecord(userID string) (*models.KYCRecord, error) {
	s.kycRecordsMu.RLock()
	defer s.kycRecordsMu.RUnlock()

	record, exists := s.kycRecords[userID]
	if !exists {
		return nil, nil // No record yet
	}
	return record, nil
}

// =============================================================================
// WALLET OPERATIONS
// Core Principle 11: Financial Integrity (100% collateralization)
// Core Principle 13: Financial Resources (Segregated funds)
// =============================================================================

// CreateWallet creates a segregated funds wallet for a user.
func (s *Store) CreateWallet(userID, ip string) (*models.Wallet, error) {
	s.walletsMu.Lock()
	defer s.walletsMu.Unlock()

	// Check if wallet already exists
	if _, exists := s.wallets[userID]; exists {
		return s.wallets[userID], nil
	}

	now := time.Now().UTC()
	wallet := &models.Wallet{
		ID:        s.generateID("wallet"),
		UserID:    userID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.wallets[userID] = wallet
	s.LogAudit(userID, models.AuditActionCreate, "wallet", wallet.ID, nil, wallet, ip, "", "Wallet created")
	return wallet, nil
}

// GetWallet retrieves a user's wallet.
func (s *Store) GetWallet(userID string) (*models.Wallet, error) {
	s.walletsMu.RLock()
	defer s.walletsMu.RUnlock()

	wallet, exists := s.wallets[userID]
	if !exists {
		return nil, ErrWalletNotFound
	}
	return wallet, nil
}

// Deposit adds funds to wallet (mock ACH).
// Core Principle 13: Funds are segregated and tracked.
func (s *Store) Deposit(userID string, amountUSD float64, reference, ip string) (*models.Transaction, error) {
	s.walletsMu.Lock()
	defer s.walletsMu.Unlock()

	wallet, exists := s.wallets[userID]
	if !exists {
		return nil, ErrWalletNotFound
	}

	balanceBefore := wallet.AvailableUSD
	wallet.AvailableUSD += amountUSD
	wallet.TotalDeposited += amountUSD
	wallet.UpdatedAt = time.Now().UTC()

	// Create transaction record
	s.transactionsMu.Lock()
	defer s.transactionsMu.Unlock()

	now := time.Now().UTC()
	tx := &models.Transaction{
		ID:            s.generateID("tx"),
		WalletID:      wallet.ID,
		UserID:        userID,
		Type:          models.TxTypeDeposit,
		Status:        models.TxStatusCompleted,
		AmountUSD:     amountUSD,
		BalanceBefore: balanceBefore,
		BalanceAfter:  wallet.AvailableUSD,
		Reference:     reference,
		Description:   fmt.Sprintf("ACH Deposit: $%.2f", amountUSD),
		CreatedAt:     now,
		CompletedAt:   &now,
		IPAddress:     ip,
	}

	s.transactions[tx.ID] = tx
	s.txByWallet[wallet.ID] = append(s.txByWallet[wallet.ID], tx.ID)

	s.LogAudit(userID, models.AuditActionDeposit, "transaction", tx.ID, nil, tx, ip, "",
		fmt.Sprintf("Deposited $%.2f", amountUSD))

	return tx, nil
}

// LockFunds locks funds for an order (pre-trade margin).
// Core Principle 11: 100% collateralization required.
func (s *Store) LockFunds(userID string, amountUSD float64, orderID string) error {
	s.walletsMu.Lock()
	defer s.walletsMu.Unlock()

	wallet, exists := s.wallets[userID]
	if !exists {
		return ErrWalletNotFound
	}

	if wallet.AvailableUSD < amountUSD {
		return ErrInsufficientFunds
	}

	wallet.AvailableUSD -= amountUSD
	wallet.LockedUSD += amountUSD
	wallet.UpdatedAt = time.Now().UTC()

	return nil
}

// UnlockFunds releases locked funds (order cancelled/expired).
func (s *Store) UnlockFunds(userID string, amountUSD float64, orderID string) error {
	s.walletsMu.Lock()
	defer s.walletsMu.Unlock()

	wallet, exists := s.wallets[userID]
	if !exists {
		return ErrWalletNotFound
	}

	wallet.LockedUSD -= amountUSD
	wallet.AvailableUSD += amountUSD
	wallet.UpdatedAt = time.Now().UTC()

	return nil
}

// SettleFunds processes settlement (win/loss).
func (s *Store) SettleFunds(userID string, lockedAmount, settlementAmount float64, orderID, ip string) error {
	s.walletsMu.Lock()
	defer s.walletsMu.Unlock()

	wallet, exists := s.wallets[userID]
	if !exists {
		return ErrWalletNotFound
	}

	wallet.LockedUSD -= lockedAmount
	wallet.AvailableUSD += settlementAmount
	wallet.UpdatedAt = time.Now().UTC()

	// Create settlement transaction
	s.transactionsMu.Lock()
	defer s.transactionsMu.Unlock()

	now := time.Now().UTC()
	pnl := settlementAmount - lockedAmount
	tx := &models.Transaction{
		ID:            s.generateID("tx"),
		WalletID:      wallet.ID,
		UserID:        userID,
		Type:          models.TxTypeSettlement,
		Status:        models.TxStatusCompleted,
		AmountUSD:     settlementAmount,
		BalanceAfter:  wallet.AvailableUSD,
		Reference:     orderID,
		Description:   fmt.Sprintf("Settlement: P&L $%.2f", pnl),
		CreatedAt:     now,
		CompletedAt:   &now,
	}

	s.transactions[tx.ID] = tx
	s.txByWallet[wallet.ID] = append(s.txByWallet[wallet.ID], tx.ID)

	return nil
}

// GetTransactions retrieves transaction history.
func (s *Store) GetTransactions(userID string, limit int) ([]models.Transaction, error) {
	wallet, err := s.GetWallet(userID)
	if err != nil {
		return nil, err
	}

	s.transactionsMu.RLock()
	defer s.transactionsMu.RUnlock()

	txIDs := s.txByWallet[wallet.ID]
	var result []models.Transaction

	// Return most recent first
	for i := len(txIDs) - 1; i >= 0 && len(result) < limit; i-- {
		if tx, exists := s.transactions[txIDs[i]]; exists {
			result = append(result, *tx)
		}
	}
	return result, nil
}

// =============================================================================
// ORDER OPERATIONS
// Core Principle 9: Execution of Transactions
// Core Principle 11: Financial Integrity
// =============================================================================

// CreateOrder places a new order with pre-trade compliance checks.
func (s *Store) CreateOrder(userID, marketTicker, eventTicker string, side models.OrderSide, orderType models.OrderType, quantity, priceCents int, ip string) (*models.Order, error) {
	// Check for trading halts (Core Principle 4)
	if s.IsTradingHalted(marketTicker) {
		return nil, ErrTradingHalted
	}

	// Check user status
	user, err := s.GetUser(userID)
	if err != nil {
		return nil, err
	}
	if user.Status == models.UserStatusSuspended || user.Status == models.UserStatusBanned {
		return nil, ErrUserSuspended
	}
	if user.Status != models.UserStatusVerified {
		return nil, ErrKYCRequired
	}

	// Core Principle 11: Calculate required collateral (100% margin)
	// For binary contracts: collateral = quantity * price (for YES) or quantity * (100-price) (for NO)
	var collateralCents int
	if side == models.OrderSideYes {
		collateralCents = quantity * priceCents
	} else {
		collateralCents = quantity * (100 - priceCents)
	}
	collateralUSD := float64(collateralCents) / 100.0

	// Core Principle 5: Check position limits
	currentExposure := s.GetUserExposure(userID)
	if currentExposure+collateralUSD > user.PositionLimitUSD {
		s.CreateComplianceAlert(userID, marketTicker, "position_limit", "high",
			fmt.Sprintf("Order would exceed position limit: current=%.2f, order=%.2f, limit=%.2f",
				currentExposure, collateralUSD, user.PositionLimitUSD))
		return nil, ErrPositionLimitExceeded
	}

	// Lock funds
	if err := s.LockFunds(userID, collateralUSD, ""); err != nil {
		return nil, err
	}

	s.ordersMu.Lock()
	defer s.ordersMu.Unlock()

	now := time.Now().UTC()
	order := &models.Order{
		ID:            s.generateID("order"),
		UserID:        userID,
		MarketTicker:  marketTicker,
		EventTicker:   eventTicker,
		Side:          side,
		Type:          orderType,
		Status:        models.OrderStatusPending,
		Quantity:      quantity,
		PriceCents:    priceCents,
		CollateralUSD: collateralUSD,
		CreatedAt:     now,
		UpdatedAt:     now,
		SubmitIP:      ip,
	}

	s.orders[order.ID] = order
	s.ordersByUser[userID] = append(s.ordersByUser[userID], order.ID)

	s.LogAudit(userID, models.AuditActionTrade, "order", order.ID, nil, order, ip, "",
		fmt.Sprintf("Order placed: %s %d %s @ %d¢", side, quantity, marketTicker, priceCents))

	return order, nil
}

// MockFillOrder simulates order execution.
func (s *Store) MockFillOrder(orderID string, fillPrice int) error {
	s.ordersMu.Lock()
	defer s.ordersMu.Unlock()

	order, exists := s.orders[orderID]
	if !exists {
		return ErrOrderNotFound
	}

	now := time.Now().UTC()
	order.Status = models.OrderStatusFilled
	order.FilledQuantity = order.Quantity
	order.FilledPriceCents = fillPrice
	order.FilledAt = &now
	order.UpdatedAt = now

	// Create or update position
	s.createOrUpdatePosition(order)

	return nil
}

// createOrUpdatePosition manages positions after fills.
func (s *Store) createOrUpdatePosition(order *models.Order) {
	s.positionsMu.Lock()
	defer s.positionsMu.Unlock()

	// Find existing position for this market/side
	var existingPos *models.Position
	for _, posID := range s.positionsByUser[order.UserID] {
		pos := s.positions[posID]
		if pos.MarketTicker == order.MarketTicker && pos.Side == order.Side && pos.ClosedAt == nil {
			existingPos = pos
			break
		}
	}

	now := time.Now().UTC()
	if existingPos != nil {
		// Update existing position
		totalCost := existingPos.CostBasisUSD + order.CollateralUSD
		totalQty := existingPos.Quantity + order.FilledQuantity
		existingPos.Quantity = totalQty
		existingPos.CostBasisUSD = totalCost
		existingPos.AvgPriceCents = int(totalCost * 100 / float64(totalQty))
		existingPos.UpdatedAt = now
	} else {
		// Create new position
		pos := &models.Position{
			ID:            s.generateID("pos"),
			UserID:        order.UserID,
			MarketTicker:  order.MarketTicker,
			EventTicker:   order.EventTicker,
			Side:          order.Side,
			Quantity:      order.FilledQuantity,
			AvgPriceCents: order.FilledPriceCents,
			CostBasisUSD:  order.CollateralUSD,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		s.positions[pos.ID] = pos
		s.positionsByUser[order.UserID] = append(s.positionsByUser[order.UserID], pos.ID)
	}
}

// GetOrders retrieves orders for a user.
func (s *Store) GetOrders(userID string, status *models.OrderStatus, limit int) ([]models.Order, error) {
	s.ordersMu.RLock()
	defer s.ordersMu.RUnlock()

	orderIDs := s.ordersByUser[userID]
	var result []models.Order

	for i := len(orderIDs) - 1; i >= 0 && len(result) < limit; i-- {
		order := s.orders[orderIDs[i]]
		if status != nil && order.Status != *status {
			continue
		}
		result = append(result, *order)
	}
	return result, nil
}

// GetPositions retrieves open positions for a user.
func (s *Store) GetPositions(userID string) ([]models.Position, error) {
	s.positionsMu.RLock()
	defer s.positionsMu.RUnlock()

	posIDs := s.positionsByUser[userID]
	var result []models.Position

	for _, posID := range posIDs {
		pos := s.positions[posID]
		if pos.ClosedAt == nil {
			result = append(result, *pos)
		}
	}
	return result, nil
}

// GetUserExposure calculates total locked funds + open position value.
// Core Principle 5: Position limits monitoring.
func (s *Store) GetUserExposure(userID string) float64 {
	wallet, err := s.GetWallet(userID)
	if err != nil {
		return 0
	}
	return wallet.LockedUSD
}

// =============================================================================
// COMPLIANCE OPERATIONS
// Core Principle 4: Prevention of Market Disruption
// =============================================================================

// CreateComplianceAlert logs a surveillance alert.
func (s *Store) CreateComplianceAlert(userID, marketTicker, alertType, severity, description string) *models.ComplianceAlert {
	s.alertsMu.Lock()
	defer s.alertsMu.Unlock()

	alert := models.ComplianceAlert{
		ID:           s.generateID("alert"),
		Type:         alertType,
		Severity:     severity,
		UserID:       userID,
		MarketTicker: marketTicker,
		Description:  description,
		Status:       "open",
		CreatedAt:    time.Now().UTC(),
	}

	s.alerts = append(s.alerts, alert)
	return &alert
}

// InitiateEmergencyHalt stops trading.
// Core Principle 4: Emergency authority.
func (s *Store) InitiateEmergencyHalt(marketTicker, reason, initiatedBy string) *models.EmergencyHalt {
	s.haltsMu.Lock()
	defer s.haltsMu.Unlock()

	key := marketTicker
	if key == "" {
		key = "GLOBAL"
	}

	halt := &models.EmergencyHalt{
		ID:           s.generateID("halt"),
		MarketTicker: marketTicker,
		Reason:       reason,
		InitiatedBy:  initiatedBy,
		StartedAt:    time.Now().UTC(),
		IsActive:     true,
	}

	s.halts[key] = halt

	s.LogAudit("system", models.AuditActionHalt, "halt", halt.ID, nil, halt, "", "",
		fmt.Sprintf("Emergency halt initiated: %s - %s", key, reason))

	return halt
}

// IsTradingHalted checks if trading is halted for a market.
func (s *Store) IsTradingHalted(marketTicker string) bool {
	s.haltsMu.RLock()
	defer s.haltsMu.RUnlock()

	// Check global halt
	if halt, exists := s.halts["GLOBAL"]; exists && halt.IsActive {
		return true
	}

	// Check market-specific halt
	if halt, exists := s.halts[marketTicker]; exists && halt.IsActive {
		return true
	}

	return false
}

// LiftEmergencyHalt resumes trading.
func (s *Store) LiftEmergencyHalt(marketTicker string) error {
	s.haltsMu.Lock()
	defer s.haltsMu.Unlock()

	key := marketTicker
	if key == "" {
		key = "GLOBAL"
	}

	if halt, exists := s.halts[key]; exists {
		halt.IsActive = false
		now := time.Now().UTC()
		halt.EndsAt = &now
	}

	return nil
}
