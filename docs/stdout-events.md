# Stdout event streaming

When started with `--events=stdout`, Pearcut emits one JSON line to stdout for every assignment.

## Event format

```json
{"type":"assignment","user_id":"user-42","experiment":"checkout-flow","variant":"control","timestamp":"2025-01-15T10:30:00Z"}
```

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Event type (always `"assignment"` for now) |
| `user_id` | string | The assigned user |
| `experiment` | string | Experiment slug |
| `variant` | string | Variant the user was assigned to |
| `timestamp` | string | RFC 3339 timestamp of the assignment |

Events are emitted for both single (`/api/v1/assign`) and bulk (`/api/v1/assign/bulk`) assignments.

## Stream separation

Pearcut writes **events to stdout** and **application logs to stderr**. This follows the Unix convention (data on stdout, diagnostics on stderr) and lets you pipe events directly without filtering:

```bash
# Events go into the pipe, logs appear in the terminal
./pearcut --events=stdout | your-pipeline

# Capture logs separately if needed
./pearcut --events=stdout 2>pearcut.log | your-pipeline
```

## Quick start

```bash
# Start Pearcut with stdout events
./pearcut --events=stdout

# In another terminal, trigger an assignment
curl -s "localhost:8080/api/v1/assign?experiment=checkout-flow&user_id=user-42"

# You'll see a JSON line appear in the first terminal
```

## Buffering

Events are published asynchronously through a 4096-event buffer. If the buffer fills up (e.g. the downstream pipe is too slow), new events are dropped and a warning is logged to stderr. On shutdown, the buffer is drained before the process exits.

## Ingestion guides

- [Fly.io](ingest-fly.md) — native log shipping with fly-log-shipper
- [GCP Cloud Logging](ingest-gcp.md) — automatic on Cloud Run
- [ClickHouse](ingest-clickhouse.md) — self-hosted analytics, pipe directly
- [Vector](ingest-vector.md) — universal log router, any destination
