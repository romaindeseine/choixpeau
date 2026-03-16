# Vector

[Vector](https://vector.dev) is an open-source log router that can read Pearcut's stdout and ship events to many destinations. It's also what powers Fly.io's [fly-log-shipper](ingest-fly.md) under the hood.

```
Pearcut (stdout) → Vector → destination (GCS, S3, etc.)
```

## Prerequisites

- Pearcut running with `--events=stdout`
- Vector installed ([quickstart](https://vector.dev/docs/setup/quickstart/))

## Basic configuration

Create a `vector.toml` that reads from stdin and prints to the console to verify the pipeline:

```toml
[sources.pearcut]
type = "stdin"

[sinks.console]
type = "console"
inputs = ["pearcut"]
encoding.codec = "text"
```

Run it:

```bash
./pearcut --events=stdout | vector --config vector.toml
```

## Sink examples

Replace the `[sinks.console]` section with your destination.

### GCP Cloud Storage → BigQuery

Store events as JSON files in GCS, then query them with BigQuery.

**Vector sink:**

```toml
[sinks.gcs]
type = "gcp_cloud_storage"
inputs = ["pearcut"]
bucket = "your-bucket"
key_prefix = "pearcut/events/%Y/%m/%d"
encoding.codec = "json"
```

See [GCS sink reference](https://vector.dev/docs/reference/configuration/sinks/gcp_cloud_storage/) for authentication options.

**BigQuery external table:**

```sql
CREATE EXTERNAL TABLE `your-project.pearcut.assignment_events`
(
  type STRING,
  user_id STRING,
  experiment STRING,
  variant STRING,
  timestamp TIMESTAMP
)
OPTIONS (
  format = 'JSON',
  uris = ['gs://your-bucket/pearcut/events/*']
);
```

```sql
SELECT experiment, variant, COUNT(*) AS assignments
FROM `your-project.pearcut.assignment_events`
WHERE type = 'assignment'
GROUP BY experiment, variant;
```

### AWS S3 → Athena

Store events as JSON files in S3, then query them with Athena.

**Vector sink:**

```toml
[sinks.s3]
type = "aws_s3"
inputs = ["pearcut"]
bucket = "your-bucket"
key_prefix = "pearcut/events/%Y/%m/%d"
region = "us-east-1"
encoding.codec = "json"
```

See [AWS S3 sink reference](https://vector.dev/docs/reference/configuration/sinks/aws_s3/) for authentication options.

**Athena external table:**

```sql
CREATE EXTERNAL TABLE assignment_events (
  type STRING,
  user_id STRING,
  experiment STRING,
  variant STRING,
  `timestamp` TIMESTAMP
)
ROW FORMAT SERDE 'org.openx.data.jsonserde.JsonSerDe'
LOCATION 's3://your-bucket/pearcut/events/';
```

```sql
SELECT experiment, variant, COUNT(*) AS assignments
FROM assignment_events
WHERE type = 'assignment'
GROUP BY experiment, variant;
```

See the full list of [Vector sinks](https://vector.dev/docs/reference/configuration/sinks/) for more destinations.
