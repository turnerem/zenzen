# CLAUDE.md - Development Guide for ZenZen

> **Purpose**: This document helps Claude (and developers) work efficiently on ZenZen by documenting architecture decisions, common patterns, gotchas, and workflow best practices.

## Project Overview

**ZenZen** is a terminal-based work log application for tracking achievements and time estimations.

**Core Value Proposition:**
- Track completed work with timestamps and descriptions
- Compare estimated vs actual time to understand optimism bias
- Build a portfolio of achievements for performance reviews
- Access logs from anywhere via cloud sync and mobile API

**Tech Stack:**
- **Language**: Go 1.21+
- **TUI**: Bubble Tea (charmbracelet)
- **Database**: PostgreSQL (local + optional cloud via Neon/AWS RDS)
- **API**: Chi router with dual auth (API keys + AWS Cognito)
- **Logging**: Go slog (structured JSON for production)

## Architecture Principles

### 1. Clean Layered Architecture

```
┌─────────────────────────────────────────────────────────┐
│  Presentation Layer (tui.go, api/)                      │
│  - Bubble Tea TUI models and views                      │
│  - Chi HTTP handlers                                    │
└────────────────────┬────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────┐
│  Service Layer (service/)                               │
│  - Business logic (Notes, SyncService)                  │
│  - Timestamp management for user edits                  │
└────────────────────┬────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────┐
│  Storage Layer (storage/)                               │
│  - PostgreSQL persistence                               │
│  - Schema management                                    │
│  - NO timestamp modification (preserves sync data)      │
└─────────────────────────────────────────────────────────┘
```

**Key Principle**: Each layer has a clear responsibility. Never skip layers (e.g., TUI should never call storage directly).

### 2. Data Flow for User Edits vs Sync

**Critical Distinction** (this caused bugs in the past):

**User Edit Flow:**
```go
// In service/service.go
func (l *Notes) SaveEntry(entry core.Entry) error {
    // Service layer SETS timestamp for user edits
    entry.LastModifiedTimestamp = time.Now()

    l.Entries[entry.ID] = entry
    return l.store.SaveEntry(entry) // Storage just saves
}
```

**Sync Flow:**
```go
// In service/sync.go
func (s *SyncService) performSync() {
    // Sync NEVER modifies timestamps
    // It compares LastModifiedTimestamp to determine which is newer
    if localEntry.LastModifiedTimestamp.After(cloudEntry.LastModifiedTimestamp) {
        s.cloud.SaveEntry(localEntry) // Preserves original timestamp
    }
}
```

**Storage Layer:**
```go
// In storage/sql.go
func (s *SQLStorage) SaveEntry(entry core.Entry) error {
    // Storage layer NEVER modifies timestamps
    // Just saves whatever it receives
    // Note: LastModifiedTimestamp should be set by the caller
}
```

**Why This Matters:**
- If storage sets timestamps, sync ping-pongs entries forever
- Service layer is the source of truth for "this was a user edit"
- Storage layer is dumb - just persists what it's given

## Critical Architectural Decisions

### Decision 1: Last-Write-Wins Sync

**Problem**: How to handle conflicts when same entry edited locally and in cloud?

**Chosen Solution**: Last-Write-Wins using `LastModifiedTimestamp`

**Alternatives Considered:**
1. ❌ **Manual Conflict Resolution**: Show user a diff and ask them to choose
   - **Rejected**: Too complex for a simple logging tool, breaks background sync

2. ❌ **Operational Transforms (CRDT)**: Merge changes automatically
   - **Rejected**: Overkill for this use case, complex to implement correctly

3. ✅ **Last-Write-Wins**: Most recent timestamp wins
   - **Chosen**: Simple, predictable, works for single-user scenario
   - **Tradeoff**: Can lose changes if editing on multiple devices simultaneously (rare for work logs)

**Implementation Details:**
```go
// service/sync.go
if localEntry.LastModifiedTimestamp.After(cloudEntry.LastModifiedTimestamp) {
    s.cloud.SaveEntry(localEntry) // Local is newer, push to cloud
} else if cloudEntry.LastModifiedTimestamp.After(localEntry.LastModifiedTimestamp) {
    s.local.SaveEntry(cloudEntry) // Cloud is newer, pull to local
}
// If equal, no sync needed
```

### Decision 2: Mode-Based Log Routing

**Problem**: Logs interfere with TUI display when written to stdout

**Chosen Solution**: Route logs based on execution mode

| Mode | Destination | Format | Reason |
|------|-------------|--------|--------|
| TUI | `zenzen.log` file | JSON | Keeps display clean, queryable logs |
| API | stdout | JSON | CloudWatch-ready, production monitoring |
| Sync/Setup | stdout | Text | Human-readable for CLI commands |

**Implementation:**
```go
// logger/logger.go
func SetupLogger(mode string) (*os.File, error) {
    switch mode {
    case "tui":
        logFile, _ := os.OpenFile("zenzen.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
        Logger = slog.New(slog.NewJSONHandler(logFile, &slog.HandlerOptions{Level: slog.LevelInfo}))

    case "api":
        Logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

    case "sync", "setup":
        Logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
    }
}
```

**Why JSON for Production:**
- Easy to parse and query in CloudWatch
- Structured fields enable metrics and alerts
- Future-proof for log aggregation tools

### Decision 3: Dual Authentication (API Key + Cognito)

**Problem**: Need simple auth for development but production-ready auth for deployment

**Chosen Solution**: Support both API keys and AWS Cognito JWT

**Alternatives Considered:**
1. ❌ **API Keys Only**: Simple but not scalable for multi-user
2. ❌ **Cognito Only**: Too complex for local development
3. ✅ **Dual Mode**: Falls back from Cognito → API key
   - Best of both worlds
   - Easy local dev, production-ready

**Implementation:**
```go
// api/server.go authMiddleware
if s.cognito != nil {
    bearerToken := extractBearerToken(r)
    if bearerToken != "" {
        _, err := s.cognito.ValidateToken(bearerToken)
        if err == nil {
            logger.Info("authenticated", "method", "cognito")
            next.ServeHTTP(w, r)
            return
        }
    }
}

// Fall back to API key
apiKey := r.Header.Get("X-API-Key")
if apiKey == s.apiKey {
    logger.Info("authenticated", "method", "api_key")
    next.ServeHTTP(w, r)
}
```

### Decision 4: TUI State Management

**Problem**: How to manage entry display order and selection?

**Chosen Solution**: Map for fast lookup + Ordered IDs array for display

```go
type Model struct {
    entries    map[string]core.Entry  // Fast O(1) lookup by ID
    orderedIDs []string               // Display order (sorted by timestamp)
    selectedIndex int                 // Index into orderedIDs
}
```

**Why This Works:**
- Map provides O(1) access for editing specific entries
- Array provides stable ordering for TUI navigation
- Sorted by `StartedAtTimestamp` (most recent first)

**Alternative Considered:**
- ❌ **Array Only**: Would need O(n) search to find entries by ID
- ❌ **Map Only**: No stable ordering, hard to navigate

### Decision 5: Entry ID Generation

**Problem**: How to generate unique IDs for new entries?

**Current Solution**: Unix nanosecond timestamp
```go
newID := fmt.Sprintf("%d", time.Now().UnixNano())
```

**Why This Works:**
- Guaranteed unique for single user creating entries
- Sortable (newer IDs are lexicographically larger)
- Simple, no dependencies

**Alternatives Considered:**
1. ❌ **UUID**: More robust but overkill, adds dependency
2. ❌ **Auto-increment**: Would require database sequence, breaks offline mode
3. ✅ **Timestamp**: Good enough for single-user scenario

**Future Consideration**: If multi-user concurrent creation becomes a thing, switch to UUIDs.

## Common Development Tasks

### Adding a New Field to Entry

**Files to Modify:**
1. `core/entry.go` - Add field to struct
2. `storage/sql.go` - Update CREATE TABLE and INSERT/SELECT queries
3. `tui.go` - Add input field if editable
4. `api/handlers.go` - Include in JSON responses if needed

**Example: Adding "Priority" field**

```go
// 1. core/entry.go
type Entry struct {
    // ... existing fields
    Priority string `json:"priority"`
}

// 2. storage/sql.go
CREATE TABLE IF NOT EXISTS entries (
    -- ... existing columns
    priority TEXT DEFAULT ''
)

INSERT INTO entries (..., priority) VALUES (..., $8)

// 3. tui.go - if user-editable
type Model struct {
    priorityInput textinput.Model
}

// 4. Update save logic to include priority
```

**Testing Checklist:**
- [ ] Test data creation still works
- [ ] Sync preserves new field
- [ ] API returns new field
- [ ] TUI can edit new field (if applicable)

### Modifying the TUI

**Key Components:**
- `Model` - State (entries, inputs, view mode)
- `Update()` - Handle keyboard input and state transitions
- `View()` - Render current state
- `render*View()` - Specific view renderers (list, detail, edit)

**State Machine:**
```
list view ←→ edit view
     ↓
  detail view (currently unused, kept for future)
```

**Common Patterns:**

**Adding a new keyboard shortcut:**
```go
// In handleKey()
case "x": // Your new key
    if m.view == "list" {
        // Do something
    }
```

**Adding a new input field:**
```go
// 1. Add to Model
type Model struct {
    myInput textinput.Model
}

// 2. Initialize in NewModel()
myInput := textinput.New()
myInput.Placeholder = "Enter value"

// 3. Handle in Update() tab cycling
m.focusIndex = (m.focusIndex + 1) % 5 // Increment total fields

// 4. Render in renderEditView()
content = append(content, labelStyle.Render("my field:"))
content = append(content, m.myInput.View())
```

### Changing Database Schema

**Process:**
1. Update `storage/sql.go` CREATE TABLE statement
2. Add migration logic if needed (currently no migrations, just recreate)
3. Update INSERT/SELECT queries
4. Update `scanEntry()` helper
5. Test with fresh database: `DROP DATABASE zenzen; CREATE DATABASE zenzen;`

**No Migrations Yet**: Current approach is "drop and recreate for development"
- **Future**: Add proper migrations with versioning for production

### Testing Sync Locally

**Setup Two Databases:**
```bash
# In psql
CREATE DATABASE zenzen_local;
CREATE DATABASE zenzen_cloud;

# In config.yaml
database:
  local_connection: "postgres://user@localhost/zenzen_local?sslmode=disable"
  cloud_connection: "postgres://user@localhost/zenzen_cloud?sslmode=disable"

sync:
  enabled: true
  interval: "10s"  # Short interval for testing
```

**Test Scenarios:**
1. **Create entry locally** → Run TUI → Check cloud DB has entry
2. **Create entry in cloud** → Run TUI → Check local DB receives entry
3. **Edit locally** → Verify newer timestamp wins
4. **Edit in cloud** → Verify newer timestamp wins
5. **Conflict test** → Edit same entry both places, older edit should be lost

**Validation Queries:**
```sql
-- Check sync happened
SELECT id, title, last_modified FROM entries ORDER BY last_modified DESC;

-- Compare local vs cloud
-- (run on both databases, timestamps should match after sync)
```

### Adding a New API Endpoint

**Steps:**
1. Add route in `api/server.go` `setupRoutes()`
2. Implement handler in `api/handlers.go`
3. Use structured logging for errors/success
4. Test with curl or test script

**Example: Add "Mark Complete" endpoint**

```go
// 1. api/server.go
func (s *Server) setupRoutes() {
    s.router.Route("/api/v1", func(r chi.Router) {
        r.Get("/entries", s.handleGetEntries)
        r.Put("/entries/{id}/complete", s.handleMarkComplete) // NEW
    })
}

// 2. api/handlers.go
func (s *Server) handleMarkComplete(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")

    entry, err := s.store.GetEntry(id)
    if err != nil {
        logger.Error("get_entry_failed", "id", id, "error", err.Error())
        http.Error(w, "Entry not found", http.StatusNotFound)
        return
    }

    entry.EndedAtTimestamp = time.Now()
    entry.LastModifiedTimestamp = time.Now()

    if err := s.store.SaveEntry(entry); err != nil {
        logger.Error("mark_complete_failed", "id", id, "error", err.Error())
        http.Error(w, "Failed to update entry", http.StatusInternalServerError)
        return
    }

    logger.Info("entry_completed", "id", id)
    respondJSON(w, entry)
}
```

**Test:**
```bash
curl -X PUT \
  -H "X-API-Key: $ZENZEN_API_KEY" \
  http://localhost:8080/api/v1/entries/123/complete
```

## Important Gotchas

### 1. ⚠️ NEVER Set Timestamps in Storage Layer

**Bad:**
```go
// storage/sql.go - DON'T DO THIS
func (s *SQLStorage) SaveEntry(entry core.Entry) error {
    entry.LastModifiedTimestamp = time.Now() // ❌ BREAKS SYNC
    // ...
}
```

**Why**: Storage layer must preserve timestamps for sync to work. Only service layer should set timestamps for user edits.

### 2. ⚠️ Tag Autocomplete Requires Rebuilding Available Tags

When saving an entry with new tags:
```go
// tui.go - After saving
m.availableTags = m.collectAllTags() // Rebuild tag list
```

Forgetting this means new tags won't appear in autocomplete until restart.

### 3. ⚠️ TUI Logs Go to File, Not Terminal

When debugging TUI:
```bash
# Terminal 1
go run .

# Terminal 2
tail -f zenzen.log  # Watch logs here
```

Don't add `fmt.Printf` or `log.Println` - they'll mess up the TUI display.

### 4. ⚠️ Entry Creation Requires All Timestamps

```go
newEntry := core.Entry{
    ID:                    newID,
    StartedAtTimestamp:    time.Now(),        // ✅ Required
    LastModifiedTimestamp: time.Now(),        // ✅ Required
    EndedAtTimestamp:      time.Time{},       // Zero = in progress
}
```

Missing timestamps can cause sync issues or display bugs.

### 5. ⚠️ API Authentication Checks Order Matters

```go
// Cognito checked FIRST
if s.cognito != nil && bearerToken != "" {
    // Validate JWT
}

// API key checked as FALLBACK
if apiKey == s.apiKey {
    // Allow access
}
```

Reversing this order would make API keys take precedence over JWT tokens.

### 6. ⚠️ Bubble Tea Update() Must Return tea.Cmd

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmd tea.Cmd

    // ... handle input

    return m, cmd  // ✅ Always return both
}
```

Forgetting `tea.Cmd` return value breaks Bubble Tea's event loop.

## Code Patterns and Conventions

### Structured Logging

**Always use structured logging with named events:**

```go
// ✅ Good
logger.Info("sync_completed", "synced_count", 5, "duration_ms", 123)
logger.Error("database_connection_failed", "error", err.Error())

// ❌ Bad
logger.Info("Sync completed with 5 entries")  // Hard to query
log.Printf("Error: %v", err)  // Not structured
```

**Event Naming Convention:**
- Use `snake_case` for event names
- Past tense for completed actions: `entry_saved`, `sync_completed`
- Present tense for errors: `connection_failed`, `validation_error`
- Include relevant fields: `"entry_id", id, "error", err.Error()`

### Error Handling

**Always log before returning errors:**

```go
func (s *Service) DoThing() error {
    if err := s.store.GetData(); err != nil {
        logger.Error("get_data_failed", "error", err.Error())
        return fmt.Errorf("failed to get data: %w", err)
    }
    return nil
}
```

**Use `%w` for error wrapping** (enables `errors.Is()` and `errors.As()`).

### Database Queries

**Always use squirrel for SQL building:**

```go
// ✅ Good - Safe from SQL injection
query := squirrel.Select("*").
    From("entries").
    Where(squirrel.Eq{"id": id})

// ❌ Bad - SQL injection risk
query := fmt.Sprintf("SELECT * FROM entries WHERE id = '%s'", id)
```

### JSON Responses (API)

**Use helper function for consistency:**

```go
func respondJSON(w http.ResponseWriter, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(data)
}

// Usage
respondJSON(w, entries)
```

## File Structure Reference

```
zenzen/
├── main.go                  # Entry point, mode routing (tui/api/sync/setup)
├── tui.go                   # Bubble Tea TUI implementation
├── ui_minimal.go            # TUI rendering logic
├── create_test_data.go      # Test data generation
│
├── api/
│   ├── server.go            # HTTP server, routing, middleware
│   ├── handlers.go          # API endpoint implementations
│   └── cognito.go           # AWS Cognito JWT validation
│
├── core/
│   └── entry.go             # Entry domain model
│
├── service/
│   ├── service.go           # Notes CRUD operations
│   └── sync.go              # Background sync service
│
├── storage/
│   ├── sql.go               # PostgreSQL implementation
│   └── file_system.go       # (Legacy) File storage
│
├── logger/
│   └── logger.go            # Structured logging setup
│
├── config/
│   └── config.go            # Configuration loading (YAML)
│
└── docs/
    ├── README.md            # User-facing documentation
    ├── QUICK_START.md       # Setup guide
    ├── ARCHITECTURE.md      # Architecture overview
    ├── LOGGING.md           # Logging strategy
    ├── API.md               # API documentation
    ├── CLOUD_SETUP.md       # Cloud database setup
    ├── COGNITO_SETUP.md     # Cognito authentication setup
    └── CLAUDE.md            # This file (development guide)
```

## Testing Strategy

### Current State
- Unit tests for `core/`, `service/`, `storage/`
- No integration tests yet
- No TUI tests (hard to test Bubble Tea apps)
- Manual API testing with curl/scripts

### Running Tests
```bash
# All tests
go test ./...

# Specific package
go test ./service -v

# With coverage
go test ./... -cover
```

### Test Data
```bash
# Create test data in database
go run . setup

# Manually verify
psql zenzen -c "SELECT id, title FROM entries;"
```

## Development Workflow

### Making Changes Safely

1. **Read relevant files first** - Understand existing code before modifying
2. **Check for similar patterns** - Maintain consistency
3. **Test incrementally** - Build after each logical change
4. **Use structured logging** - Don't use `fmt.Printf` or `log.Println`
5. **Update documentation** - Keep README/docs in sync with code changes

### Before Committing

- [ ] `go build` succeeds
- [ ] `go test ./...` passes
- [ ] Manual smoke test (run the TUI, create/edit entry)
- [ ] Check logs are structured JSON (if in TUI mode)
- [ ] Update relevant documentation if needed

### Debugging Checklist

**TUI Issues:**
- [ ] Check `zenzen.log` for errors
- [ ] Verify timestamps are set correctly
- [ ] Check if entries are in `orderedIDs` array
- [ ] Verify `selectedIndex` is valid

**Sync Issues:**
- [ ] Check both databases have the schema
- [ ] Verify timestamps are preserved (not modified by storage)
- [ ] Check sync interval is reasonable
- [ ] Look for sync errors in logs

**API Issues:**
- [ ] Verify API key is set correctly
- [ ] Check CORS headers if calling from browser
- [ ] Verify database connection string
- [ ] Check structured logs in stdout

## Future Considerations

### Potential Improvements

1. **Database Migrations**: Add versioned schema migrations instead of "drop and recreate"
2. **Write Endpoints**: Implement POST/PUT/DELETE for API (currently read-only)
3. **Conflict UI**: Show user when sync conflicts occur (currently silent Last-Write-Wins)
4. **UUID Entry IDs**: More robust than timestamps for multi-user scenarios
5. **Completed Date Field**: Separate "ended at" (when work stopped) from "completed at" (when marked done)
6. **Tags as Separate Table**: Many-to-many relationship for better querying
7. **AWS Lambda Deployment**: Serverless API deployment guide
8. **Mobile Web UI**: HTML/JS frontend for mobile access

### Known Limitations

1. **Single User Assumption**: Timestamp-based IDs and Last-Write-Wins sync assume single user
2. **No Real-time Sync**: Sync happens on interval, not on every change
3. **No Conflict Resolution UI**: Older edits are silently overwritten
4. **No Undo**: Deleting an entry is permanent
5. **Limited Search**: No full-text search in TUI

## Quick Reference Commands

```bash
# Run TUI (default)
go run .

# Create test data
go run . setup

# One-time sync
go run . sync-now

# Start API server
export ZENZEN_API_KEY=$(openssl rand -hex 32)
go run . api

# Run tests
go test ./...

# Build binary
go build -o zenzen

# Watch TUI logs (separate terminal)
tail -f zenzen.log

# Test API endpoint
curl -H "X-API-Key: $ZENZEN_API_KEY" \
  http://localhost:8080/api/v1/entries
```

## Getting Help

1. **Check this file first** - Most common tasks are documented
2. **Read relevant docs**:
   - User setup: `QUICK_START.md`
   - Architecture: `ARCHITECTURE.md`
   - API: `API.md`
   - Logging: `LOGGING.md`
3. **Check test files** - Often show usage examples
4. **Read source code comments** - Most complex logic is commented

---

**Last Updated**: 2025-12-29

**Maintainer**: Emily Turner (@turnerem)

**Status**: Active Development (Phase 2 Complete - TUI + API + Sync)
