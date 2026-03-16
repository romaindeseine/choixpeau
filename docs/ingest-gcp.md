# GCP Cloud Logging

[Cloud Run](https://cloud.google.com/run) automatically captures everything your container writes to stdout and sends it to [Cloud Logging](https://cloud.google.com/logging) as structured log entries. No agent, no sidecar, no configuration — if Pearcut runs on Cloud Run with `--events=stdout`, your assignment events are already in Cloud Logging.

```
Pearcut (stdout) → Cloud Run (automatic capture) → Cloud Logging
```

## Prerequisites

- Pearcut running on Cloud Run with `--events=stdout`
- `gcloud` CLI authenticated

## How it works

When your Pearcut container emits a JSON line to stdout, Cloud Run intercepts it and creates a structured log entry in Cloud Logging. Each JSON field becomes a queryable `jsonPayload` attribute — no parsing needed.

## Filter assignment events

In the Logs Explorer (Cloud Console) or via `gcloud`, use this filter to isolate assignment events:

```
resource.type="cloud_run_revision"
resource.labels.service_name="your-service"
jsonPayload.type="assignment"
```

Replace `your-service` with your Cloud Run service name.

## Verify

```bash
# Trigger an assignment
curl -s "https://your-service-url.run.app/api/v1/assign?experiment=checkout-flow&user_id=user-42"

# Check Cloud Logging
gcloud logging read \
  'resource.type="cloud_run_revision" jsonPayload.type="assignment"' \
  --project=your-project \
  --limit=5 \
  --format=json
```

## Bonus: sink to BigQuery via GCS

Cloud Logging can export matching entries to [Cloud Storage](https://cloud.google.com/storage) as JSON files. You can then query them from [BigQuery](https://cloud.google.com/bigquery) using an external table.

```
Cloud Logging → sink → GCS (JSON files) → BigQuery external table
```

### 1. Create a GCS bucket

```bash
gcloud storage buckets create gs://your-bucket \
  --location=your-region
```

### 2. Create the log sink

```bash
gcloud logging sinks create your-sink-name \
  storage.googleapis.com/your-bucket \
  --log-filter='resource.type="cloud_run_revision" resource.labels.service_name="your-service" jsonPayload.type="assignment"'
```

### 3. Grant write access

The sink uses a dedicated service account to write to GCS. Find it in the sink details:

```bash
gcloud logging sinks describe your-sink-name --project=your-project
```

Look for the `writerIdentity` field, then grant it the Storage Object Creator role on the bucket:

```bash
gcloud storage buckets add-iam-policy-binding gs://your-bucket \
  --member="serviceAccount:WRITER_IDENTITY" \
  --role="roles/storage.objectCreator"
```

Replace `WRITER_IDENTITY` with the service account from `writerIdentity` (e.g. `service-123456@gcp-sa-logging.iam.gserviceaccount.com`).

### 4. Create a BigQuery external table

Cloud Logging exports files in its own JSON format, which includes many metadata fields (`insertId`, `resource`, `logName`, etc.). The schema below keeps only what's needed for analytics: the ingestion `timestamp` and the event fields under `jsonPayload`.

```bash
bq mk --dataset --location=your-region your-project:your-dataset
```

```sql
CREATE EXTERNAL TABLE `your-project.your-dataset.assignment_events`
(
  timestamp TIMESTAMP,
  jsonPayload STRUCT<
    type STRING,
    user_id STRING,
    experiment STRING,
    variant STRING,
    timestamp STRING
  >
)
OPTIONS (
  format = 'JSON',
  uris = ['gs://your-bucket/*']
);
```

### 5. Query

```sql
SELECT
  jsonPayload.experiment AS experiment,
  jsonPayload.variant AS variant,
  COUNT(*) AS assignments
FROM `your-project.your-dataset.assignment_events`
GROUP BY experiment, variant
ORDER BY assignments DESC
```
