# Quickstart: SQLite to PostgreSQL Migration

**Feature**: 013-sqlite-postgres-import

## Prerequisites

1. SQLite database with memory data (e.g., `~/.brains/memories.db`)
2. PostgreSQL database with memories table initialized
3. `brains` CLI tool installed

## Basic Migration

### Step 1: Preview the Import

```bash
brains db import --from ~/.brains/memories.db --dry-run
```

This shows what will be imported without making changes.

### Step 2: Run the Import

```bash
brains db import \
  --from ~/.brains/memories.db \
  --to "postgres://user:password@localhost:5432/brains"
```

### Step 3: Verify

```bash
brains memory list --db-type postgres
```

## Incremental Imports

After the initial import, subsequent runs only import new data:

```bash
# Add new memories to SQLite (via normal usage)
brains memory set "new-memory" "content" --db-type sqlite

# Run import again - only new items are transferred
brains db import --from ~/.brains/memories.db
```

## Common Scenarios

### Development to Production

```bash
# Local development with SQLite
export BRAINS_BACKEND=sqlite

# Push to production PostgreSQL
brains db import \
  --from ~/.brains/memories.db \
  --to "$PRODUCTION_POSTGRES_URL" \
  --verbose
```

### CI/CD Pipeline

```bash
# In deployment script
RESULT=$(brains db import --from backup.db --format json)
IMPORTED=$(echo "$RESULT" | jq '.result.imported')
echo "Imported $IMPORTED items"
```

### Scheduled Sync

```bash
# Cron job for regular sync
0 * * * * /usr/local/bin/brains db import \
  --from /data/memories.db \
  --to "postgres://..." \
  >> /var/log/brains-sync.log 2>&1
```

## Troubleshooting

### "Database is locked"

Another process is using the SQLite database. Wait for it to finish or stop the other process.

### "Connection refused"

PostgreSQL is not running or connection URL is incorrect. Verify:
```bash
psql "postgres://user:password@localhost:5432/brains" -c "SELECT 1"
```

### "Items already exist"

This is normal - the tool skips items that were already imported. Use `--verbose` to see which items are skipped.

### Partial Failure

If import fails partway through, simply re-run it. Already-imported items will be skipped automatically.

## Performance Tips

1. **Batch Size**: Default is 100. For large imports, try `--batch-size 500`
2. **Network**: Run import from a machine close to PostgreSQL server
3. **Timing**: Import during off-peak hours to avoid locking conflicts
