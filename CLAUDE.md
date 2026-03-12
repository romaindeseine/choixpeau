# Choixpeau

Minimalist A/B cohort assignment API, written in Go.

## Conventions

- Codebase language: English (code, comments, commits, docs)
- Go standard library only — no framework
- Flat file architecture: all Go source files live at the root in `package main` — no sub-packages

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

Flat layout — all Go files at the root in `package main`, one file per responsibility:

- `main.go` — entrypoint, HTTP server, handlers
- `model.go` — domain structs (Experiment, Variant, Assignment)

## Build & Run

```bash
go build -o choixpeau .
./choixpeau
```

Server listens on `:8080`.

## Tests

```bash
go test ./...
```
