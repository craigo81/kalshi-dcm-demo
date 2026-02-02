# Kalshi DCM Demo - CFTC Compliant Binary Contracts Platform

A complete demonstration of a CFTC-compliant trading platform for binary event contracts, routing to Kalshi as the Designated Contract Market (DCM). This demo implements key CFTC Core Principles for DCM intermediaries (e.g., Introducing Broker).

## 🏛️ CFTC Core Principles Implemented

| Core Principle | Implementation |
|----------------|----------------|
| **CP 2** - Compliance with CEA Rules | All trading flows comply with CEA Section 5(d) requirements |
| **CP 3** - Contracts Not Readily Susceptible to Manipulation | Risk classification for markets (low-risk economic binaries prioritized) |
| **CP 4** - Prevention of Market Disruption | Emergency halt system, wash trade detection, spoofing alerts |
| **CP 5** - Position Limits | Per-user exposure limits with real-time monitoring |
| **CP 9** - Execution of Transactions | Fair order routing with pre-trade checks |
| **CP 11** - Financial Integrity | 100% collateralization requirement for all orders |
| **CP 13** - Financial Resources | Segregated customer funds tracking |
| **CP 17** - Fitness Standards | KYC/AML verification, US residency checks |
| **CP 18** - Recordkeeping | Complete audit trail with 5-year retention support |

## 🚀 Quick Start

### Prerequisites

- **Go 1.22+** - [Install Go](https://go.dev/dl/)
- **Node.js 18+** - [Install Node.js](https://nodejs.org/)

### Backend Setup

```bash
cd backend

# Install dependencies
go mod tidy

# Run the server
go run cmd/server/main.go

# Server runs at http://localhost:8080
```

### Frontend Setup

```bash
cd frontend

# Install dependencies
npm install

# Run development server
npm run dev

# App runs at http://localhost:3000
```

### Surveillance Dashboard Setup

```bash
cd surveillance-app

# Install dependencies
go mod tidy

# Run the server
go run cmd/server/main.go

# Dashboard runs at http://localhost:3001
```

## 📁 Project Structure

```
kalshi-dcm-demo/
├── backend/                          # Go REST API (port 8080)
│   ├── cmd/server/main.go           # Entry point
│   └── internal/
│       ├── api/                      # HTTP handlers & routing
│       │   ├── handlers.go          # All API endpoints
│       │   └── router.go            # Route definitions
│       ├── auth/                     # JWT authentication
│       │   └── jwt.go               # Token generation/validation
│       ├── compliance/              # CFTC compliance engine
│       │   ├── surveillance.go      # Market surveillance, position limits
│       │   └── surveillance_test.go # Unit tests
│       ├── config/                  # Configuration management
│       │   └── config.go            # Multi-exchange config
│       ├── kalshi/                  # Kalshi API client
│       │   ├── client.go            # Real market data integration
│       │   └── mock_auth.go         # Mock authenticated endpoints
│       ├── mock/                    # In-memory data store
│       │   └── store.go             # Users, wallets, orders, positions
│       ├── models/                  # Data structures
│       │   └── models.go            # All entity definitions
│       ├── persistence/             # File-based persistence
│       │   └── persistence.go       # Snapshot & audit archival
│       └── ws/                      # WebSocket support
│           └── hub.go               # Real-time market updates
│
├── frontend/                        # React + TypeScript (port 3000)
│   └── src/
│       ├── api/client.ts            # API client
│       ├── context/AuthContext.tsx  # Auth state management
│       ├── components/
│       │   ├── auth/                # Login, Signup forms
│       │   ├── kyc/                 # KYC verification
│       │   ├── wallet/              # Wallet management
│       │   ├── trading/             # Market cards, trade form
│       │   ├── portfolio/           # Positions display
│       │   └── layout/              # Navbar
│       └── pages/
│           └── Dashboard.tsx        # Main trading interface
│
├── surveillance-app/                # Operator Dashboard (port 3001)
│   ├── cmd/server/main.go          # Entry point
│   ├── static/index.html           # Dashboard UI
│   └── README.md                   # Surveillance app docs
│
└── COMPLIANCE.md                    # CFTC Core Principles documentation
```

## 🔍 Surveillance & Risk Dashboard

The surveillance app provides a real-time operator dashboard for compliance officers to monitor trading activity, manage alerts, and control trading halts per CFTC Core Principle 4.

### Features

| Feature | Description |
|---------|-------------|
| **Real-time Stats** | Active users, open positions, volume, alert counts |
| **Alert Management** | View, filter, and resolve compliance alerts by severity |
| **Trading Halts** | Per-market or global emergency halt controls |
| **User Surveillance** | Monitor position limits, exposure, and utilization |
| **Live Activity Feed** | WebSocket-powered audit trail of system events |

### Surveillance API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/stats` | Dashboard statistics |
| `GET` | `/api/alerts` | List alerts (filter: `status`, `severity`) |
| `POST` | `/api/alerts/{id}/resolve` | Resolve an alert |
| `GET` | `/api/users` | List users with surveillance data |
| `POST` | `/api/users/{id}/suspend` | Suspend a user |
| `GET` | `/api/markets` | List markets with halt status |
| `POST` | `/api/markets/{ticker}/halt` | Halt a specific market |
| `POST` | `/api/markets/{ticker}/resume` | Resume a halted market |
| `POST` | `/api/halt` | **Global trading halt** |
| `POST` | `/api/resume` | Resume all trading |
| `WS` | `/ws` | Real-time WebSocket updates |

### Alert Severity Levels

- **Critical** - Immediate action required, potential manipulation detected
- **High** - Position limit breaches, suspicious patterns
- **Medium** - Unusual activity requiring review
- **Low** - Informational, no action needed

## 🔌 API Endpoints

### Public Endpoints (No Auth)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/v1/health` | Health check |
| `POST` | `/api/v1/auth/signup` | Register new user |
| `POST` | `/api/v1/auth/login` | Authenticate user |
| `GET` | `/api/v1/markets` | List Kalshi markets |
| `GET` | `/api/v1/markets/{ticker}` | Get market details |
| `GET` | `/api/v1/markets/{ticker}/orderbook` | Get orderbook |
| `GET` | `/api/v1/events` | List events |
| `GET` | `/api/v1/series` | List series |

### Authenticated Endpoints (Requires JWT)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/v1/profile` | Get user profile |
| `GET` | `/api/v1/kyc` | Get KYC status |
| `POST` | `/api/v1/kyc` | Submit KYC verification |
| `GET` | `/api/v1/wallet` | Get wallet balance |
| `POST` | `/api/v1/wallet/deposit` | Deposit funds (mock) |
| `GET` | `/api/v1/wallet/transactions` | Transaction history |
| `GET` | `/api/v1/audit` | Audit trail |

### Verified User Endpoints (Requires KYC)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/v1/orders/check` | Pre-trade compliance check |
| `POST` | `/api/v1/orders` | Place trading order |
| `GET` | `/api/v1/orders` | Order history |
| `GET` | `/api/v1/positions` | Open positions |
| `GET` | `/api/v1/portfolio` | Portfolio summary |

### WebSocket

| Endpoint | Description |
|----------|-------------|
| `WS /ws` | Real-time market updates |

## 🔐 User Flow

### 1. Sign Up
- Email/password registration
- US residency confirmation (required)
- State selection
- Age verification (18+)

### 2. KYC Verification (Core Principle 17)
- Document type selection (Driver's License, Passport, State ID)
- Document number submission
- Auto-approval in demo (simulates verification service)

### 3. Deposit Funds (Core Principle 13)
- Mock ACH deposit
- Funds tracked as segregated
- Transaction history maintained

### 4. Browse Markets (Core Principle 3)
- Real Kalshi market data
- Risk classification (low/medium/high)
- Economic binaries prioritized

### 5. Place Order (Core Principles 9, 11)
- Pre-trade margin check (100% collateralization)
- Position limit validation
- Order submission and mock fill

### 6. Monitor Positions (Core Principle 5)
- Real-time position tracking
- P&L calculation
- Exposure utilization

## ⚠️ Demo Limitations

This is a **DEMO APPLICATION**. The following are mocked/simulated:

- ❌ **No real funds** - All deposits are simulated
- ❌ **No real trades** - Orders don't route to Kalshi's authenticated API
- ❌ **Mock KYC** - Auto-approves after delay
- ❌ **In-memory storage** - Data resets on server restart
- ❌ **No encryption** - Passwords are hashed but no TLS enforcement

### For Production, You Need:

- [ ] Database integration (PostgreSQL, etc.)
- [ ] Real KYC/AML service integration (Jumio, Onfido, etc.)
- [ ] Kalshi authenticated API credentials
- [ ] TLS/HTTPS enforcement
- [ ] Proper secrets management
- [ ] Security audit and penetration testing
- [ ] Legal review for regulatory compliance
- [ ] Rate limiting and DDoS protection

## 🔧 Configuration

### Backend Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server port |
| `KALSHI_API_URL` | `https://api.elections.kalshi.com/trade-api/v2` | Kalshi API base URL |

### Frontend Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `VITE_API_URL` | `/api/v1` | Backend API URL (proxied) |

## 📊 Compliance Features

### Market Surveillance (CP 4)

```go
// Detects potential manipulation patterns
surveillance.AnalyzeTradePattern(userID, marketTicker, orders)

// Patterns detected:
// - Wash trading (offsetting trades)
// - Spoofing (large cancelled orders)
// - Layering (stacked price levels)
```

### Position Limits (CP 5)

```go
// Pre-trade position limit check
check := surveillance.ValidateOrder(userID, ticker, side, qty, price)

// Returns:
// - passed: bool
// - errors: []string
// - warnings: []string
// - required_margin: float64
```

### Emergency Halt (CP 4)

```go
// Initiate trading halt
halt := surveillance.HaltTrading(marketTicker, reason, initiatedBy)

// Resume trading
surveillance.ResumeTrading(marketTicker)
```

### Audit Trail (CP 18)

```go
// All actions are logged
store.LogAudit(userID, action, entityType, entityID, oldVal, newVal, ip, ua, desc)

// Retrievable for 5+ years
entries := store.GetAuditLog(userID, since, limit)
```

## 📝 License

MIT

---

**⚠️ DISCLAIMER**: This demo is for educational and demonstration purposes only. It is not intended for production use or real trading. Consult with legal and compliance experts before operating any trading platform.

Built for demonstrating CFTC-compliant binary contracts trading platform architecture.
