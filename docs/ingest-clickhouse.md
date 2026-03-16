# ClickHouse

Pipe Pearcut's stdout events directly into [ClickHouse](https://clickhouse.com) — a self-hosted, open-source analytics database built for append-only data like assignment events.

## Prerequisites

- Pearcut running with `--events=stdout`
- ClickHouse server running ([quick start](https://clickhouse.com/docs/getting-started/quick-start/oss))
- `clickhouse-client` available

## Create the table

```sql
CREATE TABLE assignment_events
(
    type       String,
    user_id    String,
    experiment String,
    variant    String,
    timestamp  DateTime64(3, 'UTC')
)
ENGINE = MergeTree()
ORDER BY (experiment, timestamp);
```

## Pipe events

```bash
./pearcut --events=stdout \
  | clickhouse-client --query="INSERT INTO assignment_events FORMAT JSONEachRow"
```

Each JSON line from stdout is inserted as a row.

## Verify

```bash
# Trigger a few assignments
curl -s "localhost:8080/api/v1/assign?experiment=checkout-flow&user_id=user-42"

# Query
clickhouse-client --query="SELECT * FROM assignment_events LIMIT 10"
```

## Example query

```sql
SELECT experiment, variant, count() AS assignments
FROM assignment_events
WHERE type = 'assignment'
GROUP BY experiment, variant
ORDER BY assignments DESC
```
