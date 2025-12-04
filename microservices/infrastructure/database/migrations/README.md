# Database Migrations

This directory contains database migration files for the Crawlify microservices.

## Naming Convention

Migrations follow the pattern: `{version}_{description}_{direction}.sql`

- **version**: Sequential number (001, 002, 003, etc.)
- **description**: Brief description of the migration (snake_case)
- **direction**: `up` (apply) or `down` (rollback)

## Current Migrations

### 001_initial_schema
- **Up**: Creates all core tables (workflows, workflow_executions, extracted_items_metadata, task_history)
- **Down**: Drops all tables and related database objects

## Running Migrations

```bash
# Apply all migrations (docker-compose)
make migrate-up

# Rollback all migrations (docker-compose)
make migrate-down

# Fresh install (drop and recreate)
make migrate-fresh

# Check migration status
make migrate-status

# Standalone PostgreSQL
make migrate-up-standalone
make migrate-down-standalone
```

## Adding New Migrations

1. Create a new numbered migration pair:
   - `00X_description_up.sql` - Changes to apply
   - `00X_description_down.sql` - Rollback changes

2. Update the Makefile if needed to include new migrations

3. Test both up and down migrations locally before committing
