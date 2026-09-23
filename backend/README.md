# Backend

## Stack

- Go
- `net/http`
- `database/sql`
- SQLite via `modernc.org/sqlite`

## Requirements

Go `1.25.0` or compatible, as defined by `go.mod`.

## Run

```bash
go run ./cmd/server
```

## Environment

- `DATABASE_PATH` — SQLite path; defaults to `./data/app.db`.
- `SEED_DEMO_DATA` — set to `true` to create deterministic demo data.
- `FRONTEND_ORIGIN` — allowed frontend CORS origin; defaults to `http://localhost:5173`.

See `.env.example` for the default values. The application does not load dotenv files automatically.

## Tests

```bash
go test ./...
```

## Demo mode

Run with `SEED_DEMO_DATA=true` to create five demo tasks, teams, and proposals. Repeated runs reuse the deterministic demo records instead of creating duplicates.

## Main API routes

- `GET /api/health`
- `POST /api/tasks`
- `GET /api/tasks`
- `GET /api/tasks/{id}`
- `PUT /api/tasks/{id}`
- `GET /api/tasks/{id}/rating`
- `POST /api/tasks/{id}/confirm`
- `POST /api/tasks/{id}/publish`
- `POST /api/tasks/{id}/proposals`
- `GET /api/tasks/{id}/proposals`
- `PATCH /api/proposals/{id}/status`
