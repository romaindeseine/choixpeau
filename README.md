# Choixpeau

Deterministic A/B cohort assignment. One binary, one SQLite file, zero dependencies.

[![CI](https://github.com/romaindeseine/choixpeau/actions/workflows/ci.yml/badge.svg)](https://github.com/romaindeseine/choixpeau/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/romaindeseine/choixpeau)](https://github.com/romaindeseine/choixpeau/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## Quickstart

```bash
# Download the latest release
curl -L https://github.com/romaindeseine/choixpeau/releases/download/v0.1.0/choixpeau_0.1.0_linux_amd64.tar.gz | tar xz

# Start the server
./choixpeau

# Create an experiment
curl -s -X POST localhost:8080/admin/v1/experiments \
  -d '{"slug":"checkout-flow","status":"running","variants":[{"name":"control","weight":50},{"name":"new_checkout","weight":50}]}'

# Assign a user
curl -s "localhost:8080/api/v1/assign?experiment=checkout-flow&user_id=user-42"
```

## API Reference

### Assign

```bash
curl "localhost:8080/api/v1/assign?experiment=checkout-flow&user_id=user-42"
```

```json
{"experiment":"checkout-flow","variant":"control","user_id":"user-42"}
```

### Bulk assign

Assigns a user to all running experiments (or a subset).

```bash
curl -X POST localhost:8080/api/v1/assign/bulk \
  -d '{"user_id":"user-42"}'
```

```json
{"user_id":"user-42","assignments":{"checkout-flow":"control","onboarding":"variant_b"}}
```

Pass `"experiments": ["checkout-flow"]` to restrict to specific experiments.

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
    "seed": "checkout-flow-v2"
  }'
```

Status must be one of: `draft`, `running`, `paused`, `stopped`. Seed is optional (defaults to slug).

| Method   | Endpoint                              | Description         |
|----------|---------------------------------------|---------------------|
| `GET`    | `/admin/v1/experiments`               | List experiments (optional `?status=` filter) |
| `GET`    | `/admin/v1/experiments/{slug}`        | Get one experiment  |
| `POST`   | `/admin/v1/experiments`               | Create experiment   |
| `PUT`    | `/admin/v1/experiments/{slug}`        | Update experiment   |
| `DELETE` | `/admin/v1/experiments/{slug}`        | Delete experiment   |

## Configuration

| Variable  | Default         | Description                    |
|-----------|-----------------|--------------------------------|
| `PORT`    | `8080`          | Server port                    |
| `DB_PATH` | `choixpeau.db`  | Path to SQLite database file   |

## Docker

There is no official Docker image, but you can use this minimal one:

```dockerfile
FROM alpine:latest

ARG VERSION=0.1.0

RUN apk add --no-cache ca-certificates

ADD https://github.com/romaindeseine/choixpeau/releases/download/v${VERSION}/choixpeau_${VERSION}_linux_amd64.tar.gz /tmp/choixpeau.tar.gz
RUN tar xzf /tmp/choixpeau.tar.gz -C / && rm /tmp/choixpeau.tar.gz

EXPOSE 8080

CMD ["/choixpeau"]
```

## Deployment

The key concern is persisting the SQLite file across restarts.

### Fly.io

```bash
fly volumes create choixpeau_data --size 1
```

```toml
# fly.toml
[mounts]
  source = "choixpeau_data"
  destination = "/data"

[env]
  DB_PATH = "/data/choixpeau.db"
```

### Cloud Run

```bash
gcloud run deploy choixpeau \
  --image=your-image \
  --add-volume=name=data,type=cloud-storage,bucket=your-bucket \
  --add-volume-mount=volume=data,mount-path=/data \
  --set-env-vars=DB_PATH=/data/choixpeau.db
```

### AWS ECS

Attach an EFS file system to your task definition to persist the database:

```json
{
  "volumes": [{"name": "data", "efsVolumeConfiguration": {"fileSystemId": "fs-xxxxx"}}],
  "containerDefinitions": [{
    "mountPoints": [{"sourceVolume": "data", "containerPath": "/data"}],
    "environment": [{"name": "DB_PATH", "value": "/data/choixpeau.db"}]
  }]
}
```

## How it works

Choixpeau hashes `seed + user_id` with MurmurHash3 (32-bit) to produce a deterministic bucket in `[0, total_weight)`.
The bucket maps to a variant based on cumulative weights.
Same input always yields the same variant — no database lookup needed for assignment.
Overrides bypass hashing entirely, forcing a specific variant for a given user.
The seed defaults to the experiment slug but can be changed to re-shuffle assignments.

## License

[MIT](LICENSE)
