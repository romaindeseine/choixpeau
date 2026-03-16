# Fly.io

On Fly.io, everything your app writes to stdout is automatically captured via [NATS](https://nats.io) and available via `fly logs`. To export these logs to an external service, Fly provides [fly-log-shipper](https://github.com/superfly/fly-log-shipper) — a pre-packaged app based on [Vector](https://vector.dev) under the hood.

```
Pearcut (stdout) → Fly.io (NATS) → fly-log-shipper (Vector) → destination
```

## Prerequisites

- Pearcut deployed on Fly.io with `--events=stdout`
- `flyctl` CLI authenticated

## Verify logs are flowing

```bash
fly logs --app pearcut
```

You should see JSON lines for each assignment event.

## Export with fly-log-shipper

Deploy the log shipper as a separate Fly app in your organization:

```bash
fly launch --image flyio/log-shipper:latest --no-public-ips
```

After generating `fly.toml`, update the internal port to `8686` (Vector's health check port).

Set the required secrets:

```bash
fly secrets set \
  ORG=your-org \
  ACCESS_TOKEN=$(fly tokens create readonly personal)
```

Then configure your destination. The [fly-log-shipper README](https://github.com/superfly/fly-log-shipper) covers supported destinations, required secrets, and how to provide a custom Vector configuration.

> Analytics services like BigQuery or Athena are not direct destinations — they require an intermediate storage layer (GCS, S3). See the [Vector ingestion guide](ingest-vector.md) for examples.

## Filter only Pearcut logs

By default, the log shipper captures logs from all apps in your org. To restrict to Pearcut, set the `SUBJECT` secret using the NATS subject pattern:

```bash
fly secrets set SUBJECT="logs.pearcut.>"
```

This captures all Pearcut logs across regions and instances.
