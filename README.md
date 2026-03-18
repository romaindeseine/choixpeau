# Pearcut

Deterministic A/B cohort assignment. One binary, one SQLite file, zero dependencies.

[![CI](https://github.com/romaindeseine/pearcut/actions/workflows/ci.yml/badge.svg)](https://github.com/romaindeseine/pearcut/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/romaindeseine/pearcut)](https://github.com/romaindeseine/pearcut/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## Quickstart

```bash
# Download the latest release
curl -L https://github.com/romaindeseine/pearcut/releases/download/v0.1.0/pearcut_0.1.0_linux_amd64.tar.gz | tar xz

# Start the server
./pearcut

# Create an experiment
curl -s -X POST localhost:8080/admin/v1/experiments \
  -d '{"slug":"checkout-flow","status":"running","variants":[{"name":"control","weight":50},{"name":"new_checkout","weight":50}]}'

# Assign a user
curl -s -X POST localhost:8080/api/v1/assign \
  -d '{"experiment":"checkout-flow","user_id":"user-42"}'
```

## API Reference

### Assign

```bash
curl -X POST localhost:8080/api/v1/assign \
  -d '{"experiment":"checkout-flow","user_id":"user-42"}'
```

```json
{"experiment":"checkout-flow","variant":"control","user_id":"user-42"}
```

Pass `"attributes": {"country": "FR"}` for audience targeting. Returns **204 No Content** if the user doesn't match targeting rules.

### Bulk assign

Assigns a user to all running experiments (or a subset).

```bash
curl -X POST localhost:8080/api/v1/assign/bulk \
  -d '{"user_id":"user-42"}'
```

```json
{"user_id":"user-42","assignments":{"checkout-flow":"control","onboarding":"variant_b"}}
```

Pass `"experiments": ["checkout-flow"]` to restrict to specific experiments. Pass `"attributes": {...}` for targeting.

### Admin

Full CRUD on experiments. Example — create an experiment:

```bash
curl -X POST localhost:8080/admin/v1/experiments \
  -d '{
    "slug": "checkout-flow",
    "status": "running",
    "variants": [
      {"name": "control", "weight": 50},
      {"name": "new_checkout", "weight": 50}
    ],
    "overrides": {"user-vip": "new_checkout"},
    "seed": "checkout-flow-v2",
    "targeting_rules": [
      {"attribute": "country", "operator": "in", "values": ["FR", "BE"]}
    ],
    "exclusion_percentage": 80,
    "description": "Test the new checkout flow",
    "tags": ["checkout", "q1-2026"],
    "owner": "team-growth",
    "hypothesis": "New checkout improves conversion rate"
  }'
```

Status must be one of: `draft`, `running`, `paused`, `stopped`. Seed is optional (defaults to slug). Targeting rules use operators `in` and `not_in` with AND logic; overrides bypass targeting. Metadata fields (`description`, `tags`, `owner`, `hypothesis`) are all optional.

| Method   | Endpoint                              | Description         |
|----------|---------------------------------------|---------------------|
| `GET`    | `/admin/v1/experiments`               | List experiments (`?status=`, `?tags=`, `?page=`, `?per_page=`) |
| `GET`    | `/admin/v1/experiments/{slug}`        | Get one experiment  |
| `POST`   | `/admin/v1/experiments`               | Create experiment   |
| `PUT`    | `/admin/v1/experiments/{slug}`        | Update experiment   |
| `DELETE` | `/admin/v1/experiments/{slug}`        | Delete experiment   |

## Configuration

| Flag | Default | Description |
|------|---------|-------------|
| `--http` | `0.0.0.0:8080` | Listen address (host:port) |
| `--db` | `pearcut.db` | Path to SQLite database file |
| `--events` | `noop` | Event publisher (`noop`) |

## Docker

There is no official Docker image, but you can use this minimal one:

```dockerfile
FROM alpine:latest

ARG VERSION=0.1.0

RUN apk add --no-cache ca-certificates

ADD https://github.com/romaindeseine/pearcut/releases/download/v${VERSION}/pearcut_${VERSION}_linux_amd64.tar.gz /tmp/pearcut.tar.gz
RUN tar xzf /tmp/pearcut.tar.gz -C / && rm /tmp/pearcut.tar.gz

EXPOSE 8080

CMD ["/pearcut", "--db=/data/pearcut.db"]
```

## Deployment

The key concern is persisting the SQLite file across restarts.

### Fly.io

```bash
fly volumes create pearcut_data --size 1
```

```toml
# fly.toml
[mounts]
  source = "pearcut_data"
  destination = "/data"
```

Pass `--db=/data/pearcut.db` in your Dockerfile or Procfile.

### Cloud Run

```bash
gcloud run deploy pearcut \
  --image=your-image \
  --add-volume=name=data,type=cloud-storage,bucket=your-bucket \
  --add-volume-mount=volume=data,mount-path=/data \
  --args="--db=/data/pearcut.db"
```

## How it works

1. Hash `seed + user_id` with MurmurHash3 (32-bit)
2. Map the hash to a bucket in `[0, total_weight)`
3. Walk cumulative weights to find the matching variant

```
seed: "checkout-flow", user_id: "user-42"

MurmurHash3("checkout-flow/user-42") → 2847103 % 100 → 47

  control [0–50)  ← 47 lands here
  new_checkout [50–100)
```

Same input always produces the same variant — no database lookup needed.

## License

This project is licensed under the terms of the [MIT](LICENSE) license.
