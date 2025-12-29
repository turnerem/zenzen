# ZenZen

> A terminal-based work log for tracking achievements and time estimations

Track your work, log your achievements, and build awareness of your estimation accuracy over time. ZenZen helps you understand your optimism bias by comparing estimated vs actual time to completion.

Perfect for justifying your value to employers with a detailed log of completed work.

## Features

✅ **Terminal UI (TUI)** - Beautiful, keyboard-driven interface using Bubble Tea
✅ **PostgreSQL Storage** - Reliable local database with optional cloud sync
✅ **Cloud Sync** - Background synchronization to Neon/AWS RDS
✅ **REST API** - HTTP API for mobile and web access
✅ **Dual Authentication** - API keys or AWS Cognito JWT tokens
✅ **Tag Autocomplete** - Smart tag suggestions while typing
✅ **Duration Tracking** - Compare estimated vs actual time
✅ **Estimation Bias** - Track your optimism/pessimism over time

## Quick Start

```bash
# 1. Setup database and config
cp config.example.yaml config.yaml
# Edit config.yaml with your PostgreSQL connection

# 2. Create test data
go run . setup

# 3. Launch TUI
go run .
```

**TUI Controls:**
- `↑/↓` - Navigate entries
- `Enter` - View/edit entry
- `Tab` - Switch fields (tags, estimated, body)
- `Ctrl+S` - Save
- `Ctrl+D` - Delete
- `Ctrl+C` - Exit

**TUI Logging:**
Logs are written to `zenzen.log` to keep the display clean. View logs with:
```bash
tail -f zenzen.log
```

See [QUICK_START.md](QUICK_START.md) for detailed setup instructions.

## Architecture

### Components

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   TUI       │────▶│   Service    │────▶│  Storage    │
│  (Bubble    │     │   Layer      │     │ (PostgreSQL)│
│   Tea)      │     │              │     │             │
└─────────────┘     └──────────────┘     └─────────────┘
                            │
                    ┌───────┴────────┐
                    ▼                ▼
              ┌──────────┐    ┌──────────┐
              │   Sync   │────│  Cloud   │
              │ Service  │    │ Database │
              └──────────┘    └──────────┘
                                    │
                                    ▼
                            ┌──────────────┐
                            │  REST API    │
                            │ (Chi Router) │
                            └──────────────┘
                                    │
                            ┌───────┴────────┐
                            ▼                ▼
                      ┌──────────┐    ┌──────────┐
                      │ API Key  │    │ Cognito  │
                      │   Auth   │    │   JWT    │
                      └──────────┘    └──────────┘
```

### Tech Stack

- **Language**: Go 1.21+
- **TUI**: [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- **Database**: PostgreSQL (pgx driver)
- **API**: Chi router
- **Auth**: API keys + AWS Cognito (optional)
- **Cloud**: Neon / AWS RDS

## Usage Modes

### 1. Local TUI (Default)

```bash
go run .
```

Work completely offline with local PostgreSQL storage.

### 2. Cloud Sync

```bash
# Edit config.yaml:
# sync:
#   enabled: true
#   interval: "60s"

go run .
```

Automatic background sync to cloud database every 60 seconds.

### 3. API Server

```bash
export ZENZEN_API_KEY=$(openssl rand -hex 32)
go run . api
```

RESTful API for mobile/web access.

### 4. One-Time Sync

```bash
go run . sync-now
```

Manually trigger sync between local and cloud databases.

## Data Model

```go
type Entry struct {
    ID                    string        // UUID
    Title                 string        // Entry title
    Tags                  []string      // Work categories
    StartedAtTimestamp    time.Time     // When work began
    EndedAtTimestamp      time.Time     // When work finished
    LastModifiedTimestamp time.Time     // Last edit time
    EstimatedDuration     time.Duration // Initial estimate
    Body                  string        // Description/notes
}
```

**Duration formats**: `1h30m`, `2d`, `1w3d`, `45m`

## Configuration

Create `config.yaml`:

```yaml
database:
  # Local database (required)
  local_connection: "postgres://user@localhost:5432/zenzen?sslmode=disable"

  # Cloud database (optional)
  cloud_connection: "postgres://user:pass@cloud-host/zenzen?sslmode=require"

sync:
  # Enable background sync
  enabled: false

  # Sync interval
  interval: "60s"
```

**Environment variables** (override config.yaml):
- `ZENZEN_DB_CONNECTION` - Local database
- `ZENZEN_CLOUD_DB_CONNECTION` - Cloud database
- `ZENZEN_SYNC_ENABLED` - Enable/disable sync
- `ZENZEN_API_KEY` - API authentication key

## API Endpoints

**Health Check:**
```bash
GET /health
```

**List Entries:**
```bash
GET /api/v1/entries
```

**Get Entry:**
```bash
GET /api/v1/entries/{id}
```

**Authentication:**
```bash
# API Key
curl -H "X-API-Key: your-key" http://localhost:8080/api/v1/entries

# Cognito JWT
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/entries
```

See [API.md](API.md) for full documentation.

## Cloud Deployment

### Neon (Free, Recommended)

1. Sign up: https://neon.tech
2. Create project: `zenzen`
3. Copy connection string
4. Update `config.yaml`

See [CLOUD_SETUP.md](CLOUD_SETUP.md) for details.

### AWS RDS

Free tier for 12 months, then ~$15/month.

See [CLOUD_SETUP.md](CLOUD_SETUP.md) for setup guide.

## Authentication

### API Keys (Simple)

```bash
# Generate key
openssl rand -hex 32

# Use it
export ZENZEN_API_KEY=your-generated-key
go run . api
```

Best for: Personal use, single user

### AWS Cognito (Advanced)

```bash
export COGNITO_REGION="eu-west-2"
export COGNITO_USER_POOL_ID="eu-west-2_abc123XYZ"
export COGNITO_CLIENT_ID="1a2b3c4d5e6f7g8h9i0j1k2l3m"
go run . api
```

Best for: Multi-user apps, learning industry standards

See [COGNITO_SETUP.md](COGNITO_SETUP.md) for setup guide.

## Development

### Setup

```bash
# Install dependencies
go mod download

# Create test database
psql postgres -c "CREATE DATABASE zenzen;"

# Create test data
go run . setup
```

### Testing

```bash
# Run all tests
go test ./...

# Run specific package
go test ./storage

# Test API
./test-api.sh
```

### Building

```bash
# Build binary
go build -o zenzen

# Run binary
./zenzen
```

## Project Structure

```
zenzen/
├── api/                    # REST API server
│   ├── server.go           # HTTP server setup
│   ├── handlers.go         # Endpoint handlers
│   └── cognito.go          # AWS Cognito auth
├── config/                 # Configuration
│   └── config.go           # Config loading
├── core/                   # Domain models
│   └── entry.go            # Entry struct
├── service/                # Business logic
│   ├── service.go          # Notes CRUD operations
│   └── sync.go             # Cloud sync service
├── storage/                # Data persistence
│   └── sql.go              # PostgreSQL implementation
├── main.go                 # Application entry point
├── tui.go                  # Terminal UI
├── ui_minimal.go           # UI rendering
├── config.yaml             # Your config (gitignored)
├── config.example.yaml     # Config template
├── API.md                  # API documentation
├── ARCHITECTURE.md         # Architecture guide
├── CLOUD_SETUP.md          # Cloud database guide
├── COGNITO_SETUP.md        # Cognito auth guide
├── QUICK_START.md          # Getting started guide
└── test-api.sh             # API test script
```

## Documentation

- **[QUICK_START.md](QUICK_START.md)** - Fast setup guide
- **[ARCHITECTURE.md](ARCHITECTURE.md)** - Architecture and design patterns
- **[API.md](API.md)** - REST API reference
- **[LOGGING.md](LOGGING.md)** - Structured logging and monitoring
- **[CLOUD_SETUP.md](CLOUD_SETUP.md)** - Cloud database setup
- **[COGNITO_SETUP.md](COGNITO_SETUP.md)** - AWS Cognito authentication

## Roadmap

**Phase 1: ✅ Complete**
- Terminal UI with PostgreSQL
- Cloud database support
- Background sync service

**Phase 2: ✅ Complete**
- REST API server
- API key authentication
- AWS Cognito JWT authentication

**Phase 3: Planned**
- Mobile web UI
- Write endpoints (POST/PUT/DELETE)
- AWS Lambda deployment
- Charts and analytics

## Philosophy

ZenZen helps you:

1. **Track achievements** - Build a portfolio of completed work
2. **Understand estimation bias** - Compare estimated vs actual time
3. **Improve over time** - Learn from patterns in your estimates
4. **Justify your value** - Concrete evidence of work completed

The name "ZenZen" (全然) means "not at all" or "completely" in Japanese - a reminder that estimation is an art, not a science.

## License

MIT

## Author

Built with Go, Bubble Tea, and probably more optimism than necessary about how long it would take.
