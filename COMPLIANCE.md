# CFTC Core Principles Compliance Documentation

## Kalshi DCM Demo - Regulatory Compliance Guide

This document outlines how the Kalshi DCM Demo implements CFTC Core Principles for Designated Contract Markets (DCMs) as specified in the Commodity Exchange Act (CEA) Section 5(d).

---

## Core Principles Implementation Summary

| Core Principle | Status | Implementation Files |
|----------------|--------|---------------------|
| CP 2 - Compliance with CEA Rules | ✅ Implemented | `handlers.go`, `router.go` |
| CP 3 - Contracts Not Susceptible to Manipulation | ✅ Implemented | `models.go`, `client.go` |
| CP 4 - Prevention of Market Disruption | ✅ Implemented | `surveillance.go`, `store.go` |
| CP 5 - Position Limits | ✅ Implemented | `surveillance.go`, `store.go` |
| CP 9 - Execution of Transactions | ✅ Implemented | `handlers.go`, `mock_auth.go` |
| CP 11 - Financial Integrity | ✅ Implemented | `store.go`, `handlers.go` |
| CP 13 - Financial Resources | ✅ Implemented | `models.go`, `store.go` |
| CP 17 - Fitness Standards | ✅ Implemented | `handlers.go`, `jwt.go` |
| CP 18 - Recordkeeping | ✅ Implemented | `store.go`, `models.go` |

---

## Detailed Compliance Mapping

### Core Principle 2: Compliance with CEA Rules

**Requirement:** DCM must establish and enforce rules for trading on the facility.

**Implementation:**
- All trading flows validate against CEA Section 5(d) requirements
- US residency verification during signup (`handlers.go:118-123`)
- State restriction checks (`handlers.go:126-133`)
- Age verification (18+) (`handlers.go:143-147`)

```go
// handlers.go - CP 2 Compliance
if !req.IsUSResident {
    respondError(w, http.StatusForbidden,
        "Trading is only available to US residents", "US_RESIDENCY_REQUIRED")
    return
}
```

---

### Core Principle 3: Contracts Not Readily Susceptible to Manipulation

**Requirement:** Only list contracts that are not readily susceptible to manipulation.

**Implementation:**
- Risk classification for all markets (`models.go`)
- Focus on economic binaries with objective resolution sources
- Settlement based on verifiable government data sources

```go
// models.go - Risk Classification
type MarketRiskLevel string
const (
    RiskLevelLow    MarketRiskLevel = "low"     // Economic indicators (Fed, BLS)
    RiskLevelMedium MarketRiskLevel = "medium"  // Weather, sports
    RiskLevelHigh   MarketRiskLevel = "high"    // Political, subjective
)
```

**Resolution Sources by Category:**
| Category | Primary Source | Secondary | Tertiary |
|----------|---------------|-----------|----------|
| FED Rates | federalreserve.gov | Reuters | Bloomberg |
| CPI Data | bls.gov | Reuters | Trading Economics |
| GDP Data | bea.gov | Reuters | Trading Economics |
| Unemployment | bls.gov | Reuters | Trading Economics |

---

### Core Principle 4: Prevention of Market Disruption

**Requirement:** Prevent manipulation, price distortion, and market disruptions.

**Implementation:**
- Real-time market surveillance (`surveillance.go`)
- Wash trade detection
- Spoofing pattern identification
- Layering detection
- Emergency trading halt capability

```go
// surveillance.go - Pattern Detection
func (s *SurveillanceEngine) AnalyzeTradePattern(userID, ticker string, orders []models.Order) []string {
    // Wash Trading: Offsetting positions in short timeframe
    // Spoofing: Large orders cancelled before fill
    // Layering: Multiple orders at different price levels
}
```

**Emergency Halt Triggers:**
- Unusual price volatility (>25% in 15 minutes)
- System outages
- External manipulation alerts
- Regulatory intervention requests

---

### Core Principle 5: Position Limits and Accountability

**Requirement:** Establish position limits or position accountability for speculators.

**Implementation:**
- Per-user position limits (`models.go:User.PositionLimitUSD`)
- Real-time exposure monitoring (`store.go:GetUserExposure`)
- Pre-trade position validation (`surveillance.go:ValidateOrder`)
- Position limit alerts (`store.go:CreateComplianceAlert`)

```go
// store.go - Position Limit Enforcement
currentExposure := s.GetUserExposure(userID)
if currentExposure+collateralUSD > user.PositionLimitUSD {
    s.CreateComplianceAlert(userID, marketTicker, "position_limit", "high", ...)
    return nil, ErrPositionLimitExceeded
}
```

**Default Limits:**
| User Tier | Position Limit | Max Single Order |
|-----------|---------------|------------------|
| Standard | $25,000 | $5,000 |
| Verified+ | $100,000 | $25,000 |
| Institutional | $250,000 | $50,000 |

---

### Core Principle 9: Execution of Transactions

**Requirement:** Provide a competitive, open, and efficient market.

**Implementation:**
- Fair order routing with pre-trade checks (`handlers.go:PlaceOrder`)
- Price-time priority in mock execution (`mock_auth.go:PlaceOrder`)
- Transparent execution reporting
- Order status visibility

```go
// mock_auth.go - Fair Execution Simulation
if req.Type == "market" {
    if req.Side == "yes" {
        fillPrice = marketAsk  // Buy yes at ask
    } else {
        fillPrice = marketBid  // Buy no at bid
    }
    status = "filled"
}
```

**Execution Quality Metrics:**
- Fill rate
- Price improvement
- Latency (simulated < 500ms)

---

### Core Principle 11: Financial Integrity of Transactions

**Requirement:** Establish systems to ensure financial integrity.

**Implementation:**
- 100% collateralization for all orders (`store.go:CreateOrder`)
- Pre-trade margin validation
- Segregated funds tracking
- Real-time balance management

```go
// store.go - 100% Collateralization
// For binary contracts: collateral = quantity * price (YES) or quantity * (100-price) (NO)
if side == models.OrderSideYes {
    collateralCents = quantity * priceCents
} else {
    collateralCents = quantity * (100 - priceCents)
}
```

**Financial Integrity Flow:**
1. Pre-trade balance check
2. Funds locked during order
3. Settlement releases/transfers funds
4. Immutable transaction records

---

### Core Principle 13: Financial Resources

**Requirement:** Adequate financial, operational, and managerial resources.

**Implementation:**
- Segregated customer funds (`models.go:Wallet`)
- Fund tracking with audit trails
- Deposit/withdrawal transaction logging
- Balance reconciliation support

```go
// models.go - Segregated Funds
type Wallet struct {
    AvailableUSD   float64   // Available for trading
    LockedUSD      float64   // Locked in open orders
    TotalDeposited float64   // Lifetime deposits (CP 18)
    TotalWithdrawn float64   // Lifetime withdrawals
}
```

---

### Core Principle 17: Fitness Standards

**Requirement:** Establish and enforce fitness standards for participants.

**Implementation:**
- KYC/AML verification (`handlers.go:SubmitKYC`)
- US residency confirmation
- Age verification (18+)
- Document verification
- Status-based trading access

```go
// handlers.go - Fitness Verification Flow
1. Signup → US residency check, age check
2. KYC Submit → Document type, document number
3. KYC Review → Auto-approval (demo) / Manual review (prod)
4. Verified → Trading enabled
```

**User Status Lifecycle:**
```
kyc_pending → verified → [suspended] → [banned]
                 ↓
            trading enabled
```

**Prohibited Actions:**
- Unverified users cannot place orders
- Suspended users cannot trade
- Banned users cannot access account

---

### Core Principle 18: Recordkeeping and Reporting

**Requirement:** Maintain records of all activities for 5+ years.

**Implementation:**
- Immutable audit log (`store.go:LogAudit`)
- All CRUD operations logged
- IP address tracking
- User agent recording
- Timestamp preservation (UTC)

```go
// models.go - Audit Entry Structure
type AuditEntry struct {
    ID          string      // Unique identifier
    Timestamp   time.Time   // UTC timestamp
    UserID      string      // Acting user
    Action      AuditAction // create, update, delete, login, trade, etc.
    EntityType  string      // user, order, wallet, etc.
    EntityID    string      // Entity identifier
    OldValue    string      // JSON of previous state
    NewValue    string      // JSON of new state
    IPAddress   string      // Client IP
    UserAgent   string      // Client UA
    Description string      // Human-readable
}
```

**Audit Actions Tracked:**
| Action | Description | Retention |
|--------|-------------|-----------|
| `create` | Entity creation | 5 years |
| `update` | Entity modification | 5 years |
| `delete` | Entity deletion | 5 years |
| `login` | User authentication | 5 years |
| `trade` | Order placement | 5 years |
| `kyc` | KYC submission/review | 5 years |
| `deposit` | Fund deposit | 5 years |
| `withdrawal` | Fund withdrawal | 5 years |
| `halt` | Trading halt | 5 years |

---

## Compliance Alert Categories

### Alert Types
1. **position_limit** - User approaching or exceeding position limits
2. **wash_trade** - Potential wash trading detected
3. **spoofing** - Potential spoofing behavior
4. **layering** - Potential layering detected
5. **unusual_activity** - Anomalous trading patterns

### Alert Severity Levels
- **low** - Informational, no action required
- **medium** - Review recommended
- **high** - Immediate review required
- **critical** - Trading halted pending review

---

## Emergency Procedures

### Trading Halt Process
1. **Detection** - Surveillance engine flags issue
2. **Alert** - Compliance team notified
3. **Halt Initiation** - `InitiateEmergencyHalt()` called
4. **User Notification** - Orders rejected with `TRADING_HALTED`
5. **Investigation** - Root cause analysis
6. **Resolution** - `LiftEmergencyHalt()` or extended halt
7. **Reporting** - Incident documented in audit log

### Halt Scope
- **Market-specific** - Single ticker halted
- **Global** - All trading halted

---

## Data Retention Policy

Per CFTC regulations and Core Principle 18:

| Data Type | Retention Period | Storage |
|-----------|-----------------|---------|
| Audit Logs | 5 years minimum | File-based + archive |
| User Records | Account lifetime + 5 years | Database |
| Order History | 5 years | Database |
| KYC Documents | 5 years after verification | Secure storage |
| Transaction Records | 5 years | Database |
| Compliance Alerts | 5 years | Database |

---

## API Compliance Annotations

All API handlers include inline compliance annotations:

```go
// PlaceOrder submits a trading order (mock).
// Core Principle 9: Fair and equitable execution.
// Core Principle 11: Pre-trade margin check.
func (h *Handler) PlaceOrder(w http.ResponseWriter, r *http.Request) {
    // Implementation with CP 9/11 checks
}
```

---

## Production Checklist

Before deploying to production:

- [ ] Integrate real KYC/AML service (Jumio, Onfido)
- [ ] Connect to Kalshi authenticated API
- [ ] Implement TLS/HTTPS
- [ ] Set up secrets management (Vault, AWS Secrets)
- [ ] Configure production database (PostgreSQL)
- [ ] Enable audit log archival
- [ ] Conduct security audit
- [ ] Complete legal review
- [ ] CFTC registration/exemption confirmed
- [ ] Rate limiting configured
- [ ] DDoS protection enabled
- [ ] Monitoring and alerting set up

---

## Contact

For compliance questions regarding this demo implementation:
- Documentation: This file (`COMPLIANCE.md`)
- Code: Review inline annotations in source files
- Architecture: See `README.md`

---

**Disclaimer:** This is a demonstration implementation for educational purposes. Production deployment requires full regulatory review, legal compliance verification, and security audits.
