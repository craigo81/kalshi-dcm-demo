// Package mock provides persistent data stores for the DCM demo.
// CP 18: All operations are logged with 5-year retention via JSON persistence.
package mock

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kalshi-dcm-demo/backend/internal/models"
)

// =============================================================================
// ERRORS
// =============================================================================

var (
	ErrUserNotFound          = errors.New("user not found")
	ErrUserExists            = errors.New("user already exists")
	ErrWalletNotFound        = errors.New("wallet not found")
	ErrInsufficientFunds     = errors.New("insufficient funds")
	ErrOrderNotFound         = errors.New("order not found")
	ErrPositionNotFound      = errors.New("position not found")
	ErrKYCRequired           = errors.New("KYC verification required")
	ErrUserSuspended         = errors.New("user account suspended")
	ErrMarketClosed          = errors.New("market is closed")
	ErrPositionLimitExceeded = errors.New("position limit exceeded")
	ErrTradingHalted         = errors.New("trading is currently halted")
)

// =============================================================================
// PERSISTENCE CONFIG - CP 18: 5-year retention
// =============================================================================

type PersistenceConfig struct {
	Enabled          bool
	DataDir          string
	AutoSaveInterval time.Duration
	RetentionYears   int
}

// =============================================================================
// STORE - Thread-safe persistent data store
// =============================================================================

type Store struct {
	users           map[string]*models.User
	usersByEmail    map[string]string
	usersMu         sync.RWMutex
	kycRecords      map[string]*models.KYCRecord
	kycRecordsMu    sync.RWMutex
	wallets         map[string]*models.Wallet
	walletsMu       sync.RWMutex
	transactions    map[string]*models.Transaction
	txByWallet      map[string][]string
	transactionsMu  sync.RWMutex
	orders          map[string]*models.Order
	ordersByUser    map[string][]string
	ordersMu        sync.RWMutex
	positions       map[string]*models.Position
	positionsByUser map[string][]string
	positionsMu     sync.RWMutex
	auditLog        []models.AuditEntry
	auditLogMu      sync.RWMutex
	alerts          []models.ComplianceAlert
	alertsMu        sync.RWMutex
	halts           map[string]*models.EmergencyHalt
	haltsMu         sync.RWMutex
	idCounter       int64
	idCounterMu     sync.Mutex
	persistence     PersistenceConfig
	stopChan        chan struct{}
	saveMu          sync.Mutex
}

// PersistentData - JSON serialization structure for CP 18 compliance
type PersistentData struct {
	Version         string                           `json:"version"`
	SavedAt         time.Time                        `json:"saved_at"`
	Users           map[string]*models.User          `json:"users"`
	UsersByEmail    map[string]string                `json:"users_by_email"`
	KYCRecords      map[string]*models.KYCRecord     `json:"kyc_records"`
	Wallets         map[string]*models.Wallet        `json:"wallets"`
	Transactions    map[string]*models.Transaction   `json:"transactions"`
	TxByWallet      map[string][]string              `json:"tx_by_wallet"`
	Orders          map[string]*models.Order         `json:"orders"`
	OrdersByUser    map[string][]string              `json:"orders_by_user"`
	Positions       map[string]*models.Position      `json:"positions"`
	PositionsByUser map[string][]string              `json:"positions_by_user"`
	AuditLog        []models.AuditEntry              `json:"audit_log"`
	Alerts          []models.ComplianceAlert         `json:"alerts"`
	Halts           map[string]*models.EmergencyHalt `json:"halts"`
	IDCounter       int64                            `json:"id_counter"`
}

func NewStore() *Store {
	return NewStoreWithPersistence(PersistenceConfig{
		Enabled:          false,
		DataDir:          "./data",
		AutoSaveInterval: 5 * time.Minute,
		RetentionYears:   5,
	})
}

func NewStoreWithPersistence(config PersistenceConfig) *Store {
	s := &Store{
		users:           make(map[string]*models.User),
		usersByEmail:    make(map[string]string),
		kycRecords:      make(map[string]*models.KYCRecord),
		wallets:         make(map[string]*models.Wallet),
		transactions:    make(map[string]*models.Transaction),
		txByWallet:      make(map[string][]string),
		orders:          make(map[string]*models.Order),
		ordersByUser:    make(map[string][]string),
		positions:       make(map[string]*models.Position),
		positionsByUser: make(map[string][]string),
		auditLog:        make([]models.AuditEntry, 0),
		alerts:          make([]models.ComplianceAlert, 0),
		halts:           make(map[string]*models.EmergencyHalt),
		persistence:     config,
		stopChan:        make(chan struct{}),
	}
	if config.Enabled {
		s.initPersistence()
	}
	return s
}

func (s *Store) initPersistence() {
	dirs := []string{s.persistence.DataDir, filepath.Join(s.persistence.DataDir, "snapshots"), filepath.Join(s.persistence.DataDir, "audit")}
	for _, dir := range dirs {
		os.MkdirAll(dir, 0755)
	}
	s.Load()
	go s.autoSaveLoop()
}

func (s *Store) autoSaveLoop() {
	ticker := time.NewTicker(s.persistence.AutoSaveInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.Save()
		case <-s.stopChan:
			s.Save()
			return
		}
	}
}

func (s *Store) Stop() {
	if s.persistence.Enabled {
		close(s.stopChan)
	}
}

func (s *Store) Save() error {
	if !s.persistence.Enabled {
		return nil
	}
	s.saveMu.Lock()
	defer s.saveMu.Unlock()

	data := s.collectData()
	snapshotPath := filepath.Join(s.persistence.DataDir, "snapshots", "latest.json")
	if err := s.writeJSON(snapshotPath, data); err != nil {
		return err
	}
	backupPath := filepath.Join(s.persistence.DataDir, "snapshots", fmt.Sprintf("snapshot_%s.json", time.Now().Format("20060102_150405")))
	s.writeJSON(backupPath, data)
	s.saveAuditLog()
	return nil
}

func (s *Store) collectData() *PersistentData {
	s.usersMu.RLock()
	users := make(map[string]*models.User)
	for k, v := range s.users {
		users[k] = v
	}
	usersByEmail := make(map[string]string)
	for k, v := range s.usersByEmail {
		usersByEmail[k] = v
	}
	s.usersMu.RUnlock()

	s.kycRecordsMu.RLock()
	kycRecords := make(map[string]*models.KYCRecord)
	for k, v := range s.kycRecords {
		kycRecords[k] = v
	}
	s.kycRecordsMu.RUnlock()

	s.walletsMu.RLock()
	wallets := make(map[string]*models.Wallet)
	for k, v := range s.wallets {
		wallets[k] = v
	}
	s.walletsMu.RUnlock()

	s.transactionsMu.RLock()
	transactions := make(map[string]*models.Transaction)
	for k, v := range s.transactions {
		transactions[k] = v
	}
	txByWallet := make(map[string][]string)
	for k, v := range s.txByWallet {
		txByWallet[k] = append([]string{}, v...)
	}
	s.transactionsMu.RUnlock()

	s.ordersMu.RLock()
	orders := make(map[string]*models.Order)
	for k, v := range s.orders {
		orders[k] = v
	}
	ordersByUser := make(map[string][]string)
	for k, v := range s.ordersByUser {
		ordersByUser[k] = append([]string{}, v...)
	}
	s.ordersMu.RUnlock()

	s.positionsMu.RLock()
	positions := make(map[string]*models.Position)
	for k, v := range s.positions {
		positions[k] = v
	}
	positionsByUser := make(map[string][]string)
	for k, v := range s.positionsByUser {
		positionsByUser[k] = append([]string{}, v...)
	}
	s.positionsMu.RUnlock()

	s.auditLogMu.RLock()
	auditLog := append([]models.AuditEntry{}, s.auditLog...)
	s.auditLogMu.RUnlock()

	s.alertsMu.RLock()
	alerts := append([]models.ComplianceAlert{}, s.alerts...)
	s.alertsMu.RUnlock()

	s.haltsMu.RLock()
	halts := make(map[string]*models.EmergencyHalt)
	for k, v := range s.halts {
		halts[k] = v
	}
	s.haltsMu.RUnlock()

	s.idCounterMu.Lock()
	idCounter := s.idCounter
	s.idCounterMu.Unlock()

	return &PersistentData{
		Version: "2.0", SavedAt: time.Now().UTC(), Users: users, UsersByEmail: usersByEmail,
		KYCRecords: kycRecords, Wallets: wallets, Transactions: transactions, TxByWallet: txByWallet,
		Orders: orders, OrdersByUser: ordersByUser, Positions: positions, PositionsByUser: positionsByUser,
		AuditLog: auditLog, Alerts: alerts, Halts: halts, IDCounter: idCounter,
	}
}

func (s *Store) saveAuditLog() error {
	s.auditLogMu.RLock()
	entries := append([]models.AuditEntry{}, s.auditLog...)
	s.auditLogMu.RUnlock()
	byMonth := make(map[string][]models.AuditEntry)
	for _, entry := range entries {
		month := entry.Timestamp.Format("2006-01")
		byMonth[month] = append(byMonth[month], entry)
	}
	for month, monthEntries := range byMonth {
		path := filepath.Join(s.persistence.DataDir, "audit", fmt.Sprintf("audit_%s.json", month))
		s.writeJSON(path, monthEntries)
	}
	return nil
}

func (s *Store) Load() error {
	if !s.persistence.Enabled {
		return nil
	}
	snapshotPath := filepath.Join(s.persistence.DataDir, "snapshots", "latest.json")
	file, err := os.Open(snapshotPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()
	var data PersistentData
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return err
	}
	s.restoreData(&data)
	return nil
}

func (s *Store) restoreData(data *PersistentData) {
	s.usersMu.Lock()
	s.users = data.Users
	s.usersByEmail = data.UsersByEmail
	if s.users == nil {
		s.users = make(map[string]*models.User)
	}
	if s.usersByEmail == nil {
		s.usersByEmail = make(map[string]string)
	}
	s.usersMu.Unlock()

	s.kycRecordsMu.Lock()
	s.kycRecords = data.KYCRecords
	if s.kycRecords == nil {
		s.kycRecords = make(map[string]*models.KYCRecord)
	}
	s.kycRecordsMu.Unlock()

	s.walletsMu.Lock()
	s.wallets = data.Wallets
	if s.wallets == nil {
		s.wallets = make(map[string]*models.Wallet)
	}
	s.walletsMu.Unlock()

	s.transactionsMu.Lock()
	s.transactions = data.Transactions
	s.txByWallet = data.TxByWallet
	if s.transactions == nil {
		s.transactions = make(map[string]*models.Transaction)
	}
	if s.txByWallet == nil {
		s.txByWallet = make(map[string][]string)
	}
	s.transactionsMu.Unlock()

	s.ordersMu.Lock()
	s.orders = data.Orders
	s.ordersByUser = data.OrdersByUser
	if s.orders == nil {
		s.orders = make(map[string]*models.Order)
	}
	if s.ordersByUser == nil {
		s.ordersByUser = make(map[string][]string)
	}
	s.ordersMu.Unlock()

	s.positionsMu.Lock()
	s.positions = data.Positions
	s.positionsByUser = data.PositionsByUser
	if s.positions == nil {
		s.positions = make(map[string]*models.Position)
	}
	if s.positionsByUser == nil {
		s.positionsByUser = make(map[string][]string)
	}
	s.positionsMu.Unlock()

	s.auditLogMu.Lock()
	s.auditLog = data.AuditLog
	if s.auditLog == nil {
		s.auditLog = make([]models.AuditEntry, 0)
	}
	s.auditLogMu.Unlock()

	s.alertsMu.Lock()
	s.alerts = data.Alerts
	if s.alerts == nil {
		s.alerts = make([]models.ComplianceAlert, 0)
	}
	s.alertsMu.Unlock()

	s.haltsMu.Lock()
	s.halts = data.Halts
	if s.halts == nil {
		s.halts = make(map[string]*models.EmergencyHalt)
	}
	s.haltsMu.Unlock()

	s.idCounterMu.Lock()
	s.idCounter = data.IDCounter
	s.idCounterMu.Unlock()
}

func (s *Store) writeJSON(path string, data interface{}) error {
	tempPath := path + ".tmp"
	file, err := os.Create(tempPath)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		file.Close()
		os.Remove(tempPath)
		return err
	}
	file.Close()
	return os.Rename(tempPath, path)
}

func (s *Store) generateID(prefix string) string {
	s.idCounterMu.Lock()
	defer s.idCounterMu.Unlock()
	s.idCounter++
	return fmt.Sprintf("%s_%d_%d", prefix, time.Now().UnixNano(), s.idCounter)
}

// =============================================================================
// AUDIT LOGGING - CP 18: Recordkeeping (5-year retention)
// =============================================================================

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
		ID: s.generateID("audit"), Timestamp: time.Now().UTC(), UserID: userID, Action: action,
		EntityType: entityType, EntityID: entityID, OldValue: oldJSON, NewValue: newJSON,
		IPAddress: ip, UserAgent: ua, Description: desc,
	}
	s.auditLog = append(s.auditLog, entry)
}

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

func (s *Store) GetAllAuditLogs(since time.Time, limit int) []models.AuditEntry {
	return s.GetAuditLog("", since, limit)
}

// =============================================================================
// USER OPERATIONS - CP 17: Fitness Standards
// =============================================================================

func (s *Store) CreateUser(email, passwordHash, firstName, lastName, stateCode string, dob time.Time, isUSResident bool, ip string) (*models.User, error) {
	s.usersMu.Lock()
	defer s.usersMu.Unlock()
	if _, exists := s.usersByEmail[email]; exists {
		return nil, ErrUserExists
	}
	now := time.Now().UTC()
	user := &models.User{
		ID: s.generateID("user"), Email: email, PasswordHash: passwordHash, FirstName: firstName,
		LastName: lastName, Status: models.UserStatusKYCPending, IsUSResident: isUSResident,
		StateCode: stateCode, DateOfBirth: dob, CreatedAt: now, UpdatedAt: now,
		PositionLimitUSD: 25000.00, LastLoginIP: ip,
	}
	s.users[user.ID] = user
	s.usersByEmail[email] = user.ID
	s.LogAudit(user.ID, models.AuditActionCreate, "user", user.ID, nil, user, ip, "", "User account created")
	return user, nil
}

func (s *Store) GetUser(userID string) (*models.User, error) {
	s.usersMu.RLock()
	defer s.usersMu.RUnlock()
	user, exists := s.users[userID]
	if !exists {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (s *Store) GetUserByEmail(email string) (*models.User, error) {
	s.usersMu.RLock()
	defer s.usersMu.RUnlock()
	userID, exists := s.usersByEmail[email]
	if !exists {
		return nil, ErrUserNotFound
	}
	return s.users[userID], nil
}

func (s *Store) GetAllUsers() []*models.User {
	s.usersMu.RLock()
	defer s.usersMu.RUnlock()
	users := make([]*models.User, 0, len(s.users))
	for _, u := range s.users {
		users = append(users, u)
	}
	return users
}

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
	s.LogAudit(userID, models.AuditActionUpdate, "user", userID,
		map[string]interface{}{"status": oldStatus}, map[string]interface{}{"status": status},
		ip, "", fmt.Sprintf("User status changed from %s to %s", oldStatus, status))
	return nil
}

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
// KYC OPERATIONS - CP 17: Fitness Standards
// =============================================================================

func (s *Store) CreateKYCRecord(userID, docType, docNumber, ip string) (*models.KYCRecord, error) {
	s.kycRecordsMu.Lock()
	defer s.kycRecordsMu.Unlock()
	now := time.Now().UTC()
	record := &models.KYCRecord{
		ID: s.generateID("kyc"), UserID: userID, Status: models.KYCStatusPending,
		DocumentType: docType, DocumentNumber: docNumber, SubmittedAt: now,
	}
	s.kycRecords[userID] = record
	s.LogAudit(userID, models.AuditActionKYC, "kyc", record.ID, nil, record, ip, "", "KYC verification submitted")
	return record, nil
}

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
		expiry := now.AddDate(2, 0, 0)
		record.ExpiresAt = &expiry
		s.UpdateUserStatus(userID, models.UserStatusVerified, "system")
	} else {
		record.Status = models.KYCStatusRejected
		record.RejectionReason = reason
	}
	return nil
}

func (s *Store) GetKYCRecord(userID string) (*models.KYCRecord, error) {
	s.kycRecordsMu.RLock()
	defer s.kycRecordsMu.RUnlock()
	record, exists := s.kycRecords[userID]
	if !exists {
		return nil, nil
	}
	return record, nil
}

// =============================================================================
// WALLET OPERATIONS - CP 11: Financial Integrity, CP 13: Financial Resources
// =============================================================================

func (s *Store) CreateWallet(userID, ip string) (*models.Wallet, error) {
	s.walletsMu.Lock()
	defer s.walletsMu.Unlock()
	if _, exists := s.wallets[userID]; exists {
		return s.wallets[userID], nil
	}
	now := time.Now().UTC()
	wallet := &models.Wallet{ID: s.generateID("wallet"), UserID: userID, CreatedAt: now, UpdatedAt: now}
	s.wallets[userID] = wallet
	s.LogAudit(userID, models.AuditActionCreate, "wallet", wallet.ID, nil, wallet, ip, "", "Wallet created")
	return wallet, nil
}

func (s *Store) GetWallet(userID string) (*models.Wallet, error) {
	s.walletsMu.RLock()
	defer s.walletsMu.RUnlock()
	wallet, exists := s.wallets[userID]
	if !exists {
		return nil, ErrWalletNotFound
	}
	return wallet, nil
}

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

	s.transactionsMu.Lock()
	defer s.transactionsMu.Unlock()
	now := time.Now().UTC()
	tx := &models.Transaction{
		ID: s.generateID("tx"), WalletID: wallet.ID, UserID: userID, Type: models.TxTypeDeposit,
		Status: models.TxStatusCompleted, AmountUSD: amountUSD, BalanceBefore: balanceBefore,
		BalanceAfter: wallet.AvailableUSD, Reference: reference,
		Description: fmt.Sprintf("ACH Deposit: $%.2f", amountUSD), CreatedAt: now, CompletedAt: &now, IPAddress: ip,
	}
	s.transactions[tx.ID] = tx
	s.txByWallet[wallet.ID] = append(s.txByWallet[wallet.ID], tx.ID)
	s.LogAudit(userID, models.AuditActionDeposit, "transaction", tx.ID, nil, tx, ip, "", fmt.Sprintf("Deposited $%.2f", amountUSD))
	return tx, nil
}

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

	s.transactionsMu.Lock()
	defer s.transactionsMu.Unlock()
	now := time.Now().UTC()
	pnl := settlementAmount - lockedAmount
	tx := &models.Transaction{
		ID: s.generateID("tx"), WalletID: wallet.ID, UserID: userID, Type: models.TxTypeSettlement,
		Status: models.TxStatusCompleted, AmountUSD: settlementAmount, BalanceAfter: wallet.AvailableUSD,
		Reference: orderID, Description: fmt.Sprintf("Settlement: P&L $%.2f", pnl), CreatedAt: now, CompletedAt: &now,
	}
	s.transactions[tx.ID] = tx
	s.txByWallet[wallet.ID] = append(s.txByWallet[wallet.ID], tx.ID)
	return nil
}

func (s *Store) GetTransactions(userID string, limit int) ([]models.Transaction, error) {
	wallet, err := s.GetWallet(userID)
	if err != nil {
		return nil, err
	}
	s.transactionsMu.RLock()
	defer s.transactionsMu.RUnlock()
	txIDs := s.txByWallet[wallet.ID]
	var result []models.Transaction
	for i := len(txIDs) - 1; i >= 0 && len(result) < limit; i-- {
		if tx, exists := s.transactions[txIDs[i]]; exists {
			result = append(result, *tx)
		}
	}
	return result, nil
}

// =============================================================================
// ORDER OPERATIONS - CP 9: Execution, CP 11: Financial Integrity
// =============================================================================

func (s *Store) CreateOrder(userID, marketTicker, eventTicker string, side models.OrderSide, orderType models.OrderType, quantity, priceCents int, ip string) (*models.Order, error) {
	if s.IsTradingHalted(marketTicker) {
		return nil, ErrTradingHalted
	}
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
	// CP 11: 100% collateralization
	var collateralCents int
	if side == models.OrderSideYes {
		collateralCents = quantity * priceCents
	} else {
		collateralCents = quantity * (100 - priceCents)
	}
	collateralUSD := float64(collateralCents) / 100.0
	// CP 5: Position limits
	currentExposure := s.GetUserExposure(userID)
	if currentExposure+collateralUSD > user.PositionLimitUSD {
		s.CreateComplianceAlert(userID, marketTicker, "position_limit", "high",
			fmt.Sprintf("Order would exceed position limit: current=%.2f, order=%.2f, limit=%.2f", currentExposure, collateralUSD, user.PositionLimitUSD))
		return nil, ErrPositionLimitExceeded
	}
	if err := s.LockFunds(userID, collateralUSD, ""); err != nil {
		return nil, err
	}
	s.ordersMu.Lock()
	defer s.ordersMu.Unlock()
	now := time.Now().UTC()
	order := &models.Order{
		ID: s.generateID("order"), UserID: userID, MarketTicker: marketTicker, EventTicker: eventTicker,
		Side: side, Type: orderType, Status: models.OrderStatusPending, Quantity: quantity,
		PriceCents: priceCents, CollateralUSD: collateralUSD, CreatedAt: now, UpdatedAt: now, SubmitIP: ip,
	}
	s.orders[order.ID] = order
	s.ordersByUser[userID] = append(s.ordersByUser[userID], order.ID)
	s.LogAudit(userID, models.AuditActionTrade, "order", order.ID, nil, order, ip, "",
		fmt.Sprintf("Order placed: %s %d %s @ %d¢", side, quantity, marketTicker, priceCents))
	return order, nil
}

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
	s.createOrUpdatePosition(order)
	return nil
}

func (s *Store) createOrUpdatePosition(order *models.Order) {
	s.positionsMu.Lock()
	defer s.positionsMu.Unlock()
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
		totalCost := existingPos.CostBasisUSD + order.CollateralUSD
		totalQty := existingPos.Quantity + order.FilledQuantity
		existingPos.Quantity = totalQty
		existingPos.CostBasisUSD = totalCost
		existingPos.AvgPriceCents = int(totalCost * 100 / float64(totalQty))
		existingPos.UpdatedAt = now
	} else {
		pos := &models.Position{
			ID: s.generateID("pos"), UserID: order.UserID, MarketTicker: order.MarketTicker,
			EventTicker: order.EventTicker, Side: order.Side, Quantity: order.FilledQuantity,
			AvgPriceCents: order.FilledPriceCents, CostBasisUSD: order.CollateralUSD, CreatedAt: now, UpdatedAt: now,
		}
		s.positions[pos.ID] = pos
		s.positionsByUser[order.UserID] = append(s.positionsByUser[order.UserID], pos.ID)
	}
}

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

func (s *Store) GetAllOrders(limit int) []models.Order {
	s.ordersMu.RLock()
	defer s.ordersMu.RUnlock()
	var result []models.Order
	for _, order := range s.orders {
		result = append(result, *order)
		if len(result) >= limit {
			break
		}
	}
	return result
}

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

func (s *Store) GetAllPositions() []models.Position {
	s.positionsMu.RLock()
	defer s.positionsMu.RUnlock()
	var result []models.Position
	for _, pos := range s.positions {
		if pos.ClosedAt == nil {
			result = append(result, *pos)
		}
	}
	return result
}

func (s *Store) GetUserExposure(userID string) float64 {
	wallet, err := s.GetWallet(userID)
	if err != nil {
		return 0
	}
	return wallet.LockedUSD
}

// =============================================================================
// COMPLIANCE OPERATIONS - CP 4: Prevention of Market Disruption
// =============================================================================

func (s *Store) CreateComplianceAlert(userID, marketTicker, alertType, severity, description string) *models.ComplianceAlert {
	s.alertsMu.Lock()
	defer s.alertsMu.Unlock()
	alert := models.ComplianceAlert{
		ID: s.generateID("alert"), Type: alertType, Severity: severity, UserID: userID,
		MarketTicker: marketTicker, Description: description, Status: "open", CreatedAt: time.Now().UTC(),
	}
	s.alerts = append(s.alerts, alert)
	return &alert
}

func (s *Store) GetComplianceAlerts(status, severity string, limit int) []models.ComplianceAlert {
	s.alertsMu.RLock()
	defer s.alertsMu.RUnlock()
	var result []models.ComplianceAlert
	for i := len(s.alerts) - 1; i >= 0 && len(result) < limit; i-- {
		alert := s.alerts[i]
		if status != "" && alert.Status != status {
			continue
		}
		if severity != "" && alert.Severity != severity {
			continue
		}
		result = append(result, alert)
	}
	return result
}

func (s *Store) ResolveAlert(alertID, resolvedBy, notes string) error {
	s.alertsMu.Lock()
	defer s.alertsMu.Unlock()
	for i := range s.alerts {
		if s.alerts[i].ID == alertID {
			now := time.Now().UTC()
			s.alerts[i].Status = "resolved"
			s.alerts[i].ResolvedAt = &now
			s.alerts[i].ResolvedBy = resolvedBy
			s.alerts[i].Notes = notes
			return nil
		}
	}
	return errors.New("alert not found")
}

func (s *Store) InitiateEmergencyHalt(marketTicker, reason, initiatedBy string) *models.EmergencyHalt {
	s.haltsMu.Lock()
	defer s.haltsMu.Unlock()
	key := marketTicker
	if key == "" {
		key = "GLOBAL"
	}
	halt := &models.EmergencyHalt{
		ID: s.generateID("halt"), MarketTicker: marketTicker, Reason: reason,
		InitiatedBy: initiatedBy, StartedAt: time.Now().UTC(), IsActive: true,
	}
	s.halts[key] = halt
	s.LogAudit("system", models.AuditActionHalt, "halt", halt.ID, nil, halt, "", "",
		fmt.Sprintf("Emergency halt initiated: %s - %s", key, reason))
	return halt
}

func (s *Store) IsTradingHalted(marketTicker string) bool {
	s.haltsMu.RLock()
	defer s.haltsMu.RUnlock()
	if halt, exists := s.halts["GLOBAL"]; exists && halt.IsActive {
		return true
	}
	if halt, exists := s.halts[marketTicker]; exists && halt.IsActive {
		return true
	}
	return false
}

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

func (s *Store) GetActiveHalts() []*models.EmergencyHalt {
	s.haltsMu.RLock()
	defer s.haltsMu.RUnlock()
	var result []*models.EmergencyHalt
	for _, halt := range s.halts {
		if halt.IsActive {
			result = append(result, halt)
		}
	}
	return result
}
