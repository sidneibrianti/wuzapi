# WuzAPI AI Agent Instructions

## Architecture Overview

**WuzAPI** is a Go REST API that provides WhatsApp integration via the `go.mau.fi/whatsmeow` library. It's designed as a multi-tenant service where each user (WhatsApp number) gets isolated clients.

### Core Components

- **Server**: Main HTTP server struct in `main.go` with `db`, `router`, `exPath` fields
- **MyClient**: WhatsApp client wrapper in `wmiau.go` containing `*whatsmeow.Client`, event handlers, and user context
- **Handlers**: HTTP endpoint handlers in `handlers.go` following pattern `func (s *server) HandlerName() http.HandlerFunc`
- **Routes**: Gorilla Mux routing with middleware chains in `routes.go`

### Critical Data Flow Patterns

1. **Authentication**: All user endpoints require `Authorization` header with user token, validated via `userinfocache` (go-cache)
2. **Client Management**: Global `clientsArray` manages active WhatsApp connections, indexed by userID
3. **Event Processing**: WhatsApp events flow through `MyClient.eventHandler` → webhooks/RabbitMQ → external systems
4. **Database**: Dual support for PostgreSQL (production) and SQLite (development), auto-detected via env vars

## Development Workflows

### Local Development

```bash
# Build and run locally
go build .
./wuzapi -logtype=console -color=true

# With custom admin token
./wuzapi -admintoken=your_token_here
```

### Docker Development

```bash
# Full stack with PostgreSQL
docker compose up -d

# View logs
docker compose logs -f wuzapi-server
```

### Testing Endpoints

- Swagger UI: `http://localhost:8080/api`
- Login interface: `http://localhost:8080/login`
- Dashboard: `http://localhost:8080/dashboard`

## Project-Specific Conventions

### Handler Pattern

```go
func (s *server) HandlerName() http.HandlerFunc {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 1. Parse request struct
        type requestStruct struct {
            Field string `json:"field"`
        }

        // 2. Get client via getWAClient helper
        mycli, err := s.getWAClient(r)

        // 3. Use whatsmeow APIs
        result, err := mycli.WAClient.SomeMethod()

        // 4. Return with s.Respond helper
        s.Respond(w, r, http.StatusOK, result)
    })
}
```

### WhatsApp Integration Patterns

- **AppState**: Use `appstate.BuildLabelEdit()` for label management, `appstate.BuildLabelChat()` for chat operations
- **Event Subscriptions**: Events listed in `constants.go` as `supportedEventTypes`, filtered per user
- **Media Handling**: S3 integration with configurable delivery (base64, S3 URLs, or both)

### Database Patterns

- **Migrations**: Versioned in `migrations.go`, applied on startup
- **User Management**: Admin-only CRUD via `/admin/users` endpoints
- **Configuration**: Proxy and S3 configs stored per-user in JSON columns

## Critical Integration Points

### WhatsApp Protocol

- Library: `go.mau.fi/whatsmeow` - direct WebSocket communication (no browser automation)
- **WARNING**: Protocol changes can break connections, requires library updates
- Connection states: Connecting → Connected → Disconnected/LoggedOut

### External Services

- **RabbitMQ**: Optional global event distribution via `RABBITMQ_URL` env var
- **S3 Compatible**: AWS S3, MinIO, etc. for media storage with retention policies
- **Webhooks**: Per-user HTTP callbacks for WhatsApp events

### Security Considerations

- **Admin Token**: Required for user management (`WUZAPI_ADMIN_TOKEN`)
- **User Tokens**: Per-user authentication for all WhatsApp operations
- **Rate Limiting**: Implement carefully - WhatsApp can ban numbers for spam

## Common Gotchas

1. **Session Management**: WhatsApp sessions persist in database, clients auto-reconnect
2. **Media Downloads**: Use `-skipmedia` flag to avoid downloading large files during development
3. **Docker Builds**: Multi-stage build required for CGO dependencies (SQLite, image processing)
4. **Environment Variables**: `.env` file loading happens before flag parsing
5. **Timezone**: Set `TZ` env var for consistent timestamps across containers

## File Structure Patterns

- `main.go`: Server initialization, flag parsing, graceful shutdown
- `handlers.go`: All HTTP handlers (~5000 lines, organized by feature)
- `wmiau.go`: WhatsApp client lifecycle and event handling
- `routes.go`: HTTP routing with Alice middleware chains
- `db.go`: Database abstraction layer (PostgreSQL/SQLite)
- `constants.go`: Event types and configuration constants
- `static/`: Web UI, Swagger docs, dashboard assets

## When Adding New Features

1. **New Endpoints**: Add route in `routes.go`, handler in `handlers.go` following existing patterns
2. **WhatsApp Events**: Add to `supportedEventTypes` in `constants.go`
3. **Database Changes**: Add migration to `migrations.go` with version increment
4. **API Documentation**: Update `static/api/spec.yml` for Swagger UI
5. **Multi-User**: Always consider user isolation and token validation
