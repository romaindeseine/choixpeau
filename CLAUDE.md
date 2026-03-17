# Pearcut

Open-source, minimalist A/B cohort assignment as a single downloadable binary. Written in Go, backed by SQLite.

## Conventions

- Codebase language: English (code, comments, commits, docs)
- Flat `package pearcut` at root, one file per responsibility, `cmd/pearcut/` for standalone binary
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

- **Experiment** — an A/B test identified by a unique `slug`. Has a status (`draft` → `running` → `paused` → `stopped`), a list of variants, optional overrides, and optional metadata (`description`, `tags`, `owner`, `hypothesis`).
- **Variant** — one option within an experiment (e.g. `control`, `new_checkout`). Defined by a `name` and a `weight` (relative traffic allocation).
- **Assignment** — the result of an assignment: maps a `user_id` to a `variant` for a given `experiment`.
- **Override** — forced assignment of a `user_id` to a specific variant, takes priority over hash.
- **Seed** — salt used for deterministic hashing (defaults to the experiment slug).

## Code structure

Flat layout — all Go files at the root in `package pearcut`, one file per responsibility:

- `model.go` — domain structs (Experiment, Variant, Assignment, AssignmentEvent), interfaces (Store, Engine, EventPublisher)
- `errors.go` — sentinel errors
- `validate.go` — validation methods on Experiment
- `engine.go` — assignment engine (lookup, overrides, hash-based variant selection, event publishing)
- `assign.go` — HTTP handlers for assign and bulk assign endpoints
- `admin.go` — admin handlers (CRUD experiments under `/admin/v1`)
- `server.go` — Server struct, NewServer, RegisterRoutes, writeJSON helper, health handler
- `publisher.go` — NoopPublisher, StdoutPublisher (EventPublisher implementations)
- `sqlite_store.go` — SQLite-backed Store implementation
- `cached_store.go` — in-memory cache wrapping a Store (warm-up on startup, reads from cache, writes refresh cache)
- `cmd/pearcut/main.go` — standalone binary entrypoint

## Build & Run

```bash
go build -o pearcut ./cmd/pearcut
./pearcut --http=0.0.0.0:8080 --db=pearcut.db --events=noop
```

### CLI flags

| Flag | Default | Description |
|------|---------|-------------|
| `--http` | `0.0.0.0:8080` | Listen address (host:port) |
| `--db` | `pearcut.db` | Path to SQLite database file |
| `--events` | `noop` | Event publisher (`noop`, `stdout`) |

## Tests

```bash
go test ./...
```
