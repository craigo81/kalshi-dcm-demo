# Surveillance & Risk Operator Dashboard

CFTC Core Principle 4 compliant surveillance dashboard for DCM operators.

## Features

- **Real-time monitoring** - WebSocket-powered live updates
- **Alert management** - View, filter, and resolve compliance alerts
- **Trading halts** - Market-specific or global trading suspension
- **User surveillance** - Monitor position limits and exposure
- **Activity feed** - Live audit trail of system events

## Quick Start

```bash
# Install dependencies
go mod tidy

# Run the server
go run cmd/server/main.go

# Access dashboard
open http://localhost:3001
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/health` | Health check |
| `GET` | `/api/stats` | Dashboard statistics |
| `GET` | `/api/alerts` | List alerts (filter: status, severity) |
| `POST` | `/api/alerts/{id}/resolve` | Resolve an alert |
| `GET` | `/api/users` | List users with surveillance data |
| `POST` | `/api/users/{id}/suspend` | Suspend a user |
| `GET` | `/api/markets` | List markets with halt status |
| `POST` | `/api/markets/{ticker}/halt` | Halt a market |
| `POST` | `/api/markets/{ticker}/resume` | Resume a market |
| `POST` | `/api/halt` | Global trading halt |
| `POST` | `/api/resume` | Resume all trading |
| `WS` | `/ws` | Real-time WebSocket updates |

## WebSocket Events

| Event | Description |
|-------|-------------|
| `initial_state` | Full state on connection |
| `stats_update` | Periodic stats refresh |
| `alert_resolved` | Alert was resolved |
| `market_halted` | Market trading halted |
| `market_resumed` | Market trading resumed |
| `global_halt` | All trading halted |
| `global_resume` | All trading resumed |
| `user_suspended` | User account suspended |

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `3001` | Server port |
| `BACKEND_API_URL` | `http://localhost:8080/api/v1` | Main DCM API |

## Core Principle Compliance

### CP 4: Prevention of Market Disruption

- Real-time surveillance for manipulation patterns
- Emergency halt capabilities (per-market and global)
- Alert severity classification and escalation
- Audit trail of all operator actions

### CP 5: Position Limits

- User exposure monitoring dashboard
- Position limit utilization visualization
- Alerts when users approach limits

### CP 18: Recordkeeping

- Activity feed with timestamps
- Alert resolution history
- All actions logged for audit

## Demo Data

The dashboard initializes with demo data for testing:
- Sample alerts (high, medium, low severity)
- Mock users with various exposure levels
- Example markets with activity

## Integration

This dashboard connects to the main DCM demo backend (`localhost:8080`).
In production, configure `BACKEND_API_URL` to point to your actual backend.

```bash
BACKEND_API_URL=https://api.yourdcm.com/v1 go run cmd/server/main.go
```
