# Choixpeau

Minimalist A/B cohort assignment API, written in Go.

## Conventions

- Codebase language: English (code, comments, commits, docs)
- Flat `package choixpeau` at root, one file per responsibility, `cmd/choixpeau/` for standalone binary
- Format with `gofmt`

### Error handling

- Idiomatic Go: return `error`, wrap with `fmt.Errorf("context: %w", err)`
- Sentinel errors for domain cases (`ErrExperimentNotFound`, etc.)
- Error messages: lowercase, no trailing punctuation, no redundant prefix

### Logging

- `log/slog` with `JSONHandler` (structured key/value pairs, stdlib)
- Log HTTP requests and errors
- Emojis only for lifecycle logs: 🚀 startup, ✅ connected, ⚠️ fallback, ❌ fatal

### Testing

- Table-driven tests: group related cases in a single `Test*` function using a `[]struct` slice and `t.Run`

## Domain vocabulary

- **Experiment** — an A/B test identified by a unique `slug`. Has a status (`draft` → `running` → `paused` → `stopped`), a list of variants, and optional overrides.
- **Variant** — one option within an experiment (e.g. `control`, `new_checkout`). Defined by a `name` and a `weight` (relative traffic allocation).
- **Assignment** — the result of an assignment: maps a `user_id` to a `variant` for a given `experiment`.
- **Override** — forced assignment of a `user_id` to a specific variant, takes priority over hash.
- **Seed** — salt used for deterministic hashing (defaults to the experiment slug).

## Code structure

Flat layout — all Go files at the root in `package choixpeau`, one file per responsibility:

- `model.go` — domain structs (Experiment, Variant, Assignment), interfaces (Store, Engine)
- `errors.go` — sentinel errors
- `validate.go` — validation methods on Experiment
- `engine.go` — assignment engine (lookup, overrides, hash-based variant selection)
- `assign.go` — HTTP handlers for assign and bulk assign endpoints
- `admin.go` — admin handlers (CRUD experiments under `/admin/v1`)
- `server.go` — Server struct, NewServer, RegisterRoutes, writeJSON helper, health handler
- `sqlite_store.go` — SQLite-backed Store implementation
- `cached_store.go` — in-memory cache wrapping a Store (warm-up on startup, reads from cache, writes refresh cache)
- `cmd/choixpeau/main.go` — standalone binary entrypoint

## Build & Run

```bash
go build -o choixpeau ./cmd/choixpeau
PORT=8080 DB_PATH=choixpeau.db ./choixpeau
```

Server listens on `:8080`.

### Environment variables

- `PORT` — server port (default: `8080`)
- `DB_PATH` — path to SQLite database file (default: `choixpeau.db`)

## Tests

```bash
go test ./...
```
