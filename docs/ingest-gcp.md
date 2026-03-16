# GCP Cloud Logging

Cloud Run automatically captures everything your container writes to stdout and sends it to Cloud Logging as structured log entries. No agent, no sidecar, no configuration — if Pearcut runs on Cloud Run with `--events=stdout`, your assignment events are already in Cloud Logging.

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

<!-- TODO: screenshot of Logs Explorer with filter applied -->

```
resource.type="cloud_run_revision"
resource.labels.service_name="pearcut"
jsonPayload.type="assignment"
```

Replace `pearcut` with your Cloud Run service name. The `jsonPayload.type="assignment"` condition selects only assignment events.

## Verify

```bash
# Trigger an assignment
curl -s "https://your-service.run.app/api/v1/assign?experiment=checkout-flow&user_id=user-42"

# Check Cloud Logging
gcloud logging read \
  'resource.type="cloud_run_revision" jsonPayload.type="assignment"' \
  --project=your-project \
  --limit=5 \
  --format=json
```

## Bonus: sink to BigQuery

Cloud Logging can stream matching entries directly into BigQuery for SQL analysis. The sink auto-creates a table with this schema:

```
timestamp               TIMESTAMP       Cloud Logging ingestion time
jsonPayload             RECORD
  ├── type              STRING          Event type ("assignment")
  ├── user_id           STRING          Assigned user
  ├── experiment        STRING          Experiment slug
  ├── variant           STRING          Assigned variant
  └── timestamp         STRING          RFC 3339 assignment timestamp
```

Create a log sink:

```bash
gcloud logging sinks create pearcut-events \
  bigquery.googleapis.com/projects/your-project/datasets/pearcut \
  --log-filter='resource.type="cloud_run_revision" resource.labels.service_name="pearcut" jsonPayload.type="assignment"'
```

<!-- TODO: screenshot of BigQuery query results -->

Then query:

```sql
SELECT
  jsonPayload.experiment AS experiment,
  jsonPayload.variant AS variant,
  COUNT(*) AS assignments
FROM `your-project.pearcut.cloud_run_revision`
WHERE jsonPayload.type = "assignment"
GROUP BY experiment, variant
ORDER BY assignments DESC
```
