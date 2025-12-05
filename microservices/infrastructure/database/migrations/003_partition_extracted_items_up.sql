-- Migration: 003_partition_extracted_items
-- Description: Convert extracted_items to a partitioned table for high-throughput workloads
-- This improves INSERT/SELECT performance and enables instant partition drops for cleanup
-- 
-- IMPORTANT: Run this during a maintenance window if data exists in extracted_items
-- At 10K URLs/sec, this table will grow to ~100M rows/day

-- Step 1: Rename existing table to backup (if exists with data)
ALTER TABLE IF EXISTS extracted_items RENAME TO extracted_items_old;

-- Step 2: Create partitioned table (range by extracted_at, daily partitions)
CREATE TABLE extracted_items (
    id UUID DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL,  -- Removed FK for partitioning compatibility
    workflow_id UUID NOT NULL,   -- Removed FK for partitioning compatibility
    task_id VARCHAR(255) NOT NULL,
    url TEXT NOT NULL,
    data JSONB NOT NULL,
    extracted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Composite primary key required for partitioning
    PRIMARY KEY (id, extracted_at)
) PARTITION BY RANGE (extracted_at);

-- Step 3: Create default partition for any data that doesn't match specific ranges
CREATE TABLE extracted_items_default PARTITION OF extracted_items DEFAULT;

-- Step 4: Create initial partitions dynamically (7 days ahead from current date)
-- This uses DO block to create partitions based on CURRENT_DATE
DO $$
DECLARE
    partition_date DATE;
    partition_name TEXT;
    start_date DATE;
    end_date DATE;
BEGIN
    -- Create partitions for today + next 7 days
    FOR i IN 0..7 LOOP
        partition_date := CURRENT_DATE + i;
        partition_name := 'extracted_items_y' || TO_CHAR(partition_date, 'YYYY') ||
                          'm' || TO_CHAR(partition_date, 'MM') ||
                          'd' || TO_CHAR(partition_date, 'DD');
        start_date := partition_date;
        end_date := partition_date + 1;
        
        -- Check if partition exists before creating
        IF NOT EXISTS (
            SELECT 1 FROM pg_class WHERE relname = partition_name
        ) THEN
            EXECUTE format(
                'CREATE TABLE %I PARTITION OF extracted_items FOR VALUES FROM (%L) TO (%L)',
                partition_name, start_date, end_date
            );
            RAISE NOTICE 'Created partition: %', partition_name;
        END IF;
    END LOOP;
END $$;

-- Step 5: Create indexes (automatically created on each partition)
CREATE INDEX idx_extracted_items_execution ON extracted_items (execution_id);
CREATE INDEX idx_extracted_items_workflow ON extracted_items (workflow_id);
CREATE INDEX idx_extracted_items_task ON extracted_items (task_id);
CREATE INDEX idx_extracted_items_date ON extracted_items (extracted_at DESC);
-- GIN index for JSONB queries
CREATE INDEX idx_extracted_items_data ON extracted_items USING GIN (data);

-- Step 6: Function to auto-create future partitions
CREATE OR REPLACE FUNCTION create_extracted_items_partition()
RETURNS void AS $$
DECLARE
    partition_date DATE;
    partition_name TEXT;
    start_date DATE;
    end_date DATE;
BEGIN
    -- Create partitions for next 7 days
    FOR i IN 0..7 LOOP
        partition_date := CURRENT_DATE + i;
        partition_name := 'extracted_items_y' || TO_CHAR(partition_date, 'YYYY') ||
                          'm' || TO_CHAR(partition_date, 'MM') ||
                          'd' || TO_CHAR(partition_date, 'DD');
        start_date := partition_date;
        end_date := partition_date + 1;
        
        -- Check if partition exists
        IF NOT EXISTS (
            SELECT 1 FROM pg_class WHERE relname = partition_name
        ) THEN
            EXECUTE format(
                'CREATE TABLE %I PARTITION OF extracted_items FOR VALUES FROM (%L) TO (%L)',
                partition_name, start_date, end_date
            );
            RAISE NOTICE 'Created partition: %', partition_name;
        END IF;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Step 7: Function to archive and drop old partitions
CREATE OR REPLACE FUNCTION drop_old_extracted_items_partitions(retention_days INTEGER DEFAULT 7)
RETURNS void AS $$
DECLARE
    partition_record RECORD;
    cutoff_date DATE;
BEGIN
    cutoff_date := CURRENT_DATE - retention_days;
    
    FOR partition_record IN
        SELECT c.relname AS partition_name
        FROM pg_class c
        JOIN pg_inherits i ON c.oid = i.inhrelid
        JOIN pg_class p ON i.inhparent = p.oid
        WHERE p.relname = 'extracted_items'
          AND c.relname LIKE 'extracted_items_y%'
          AND c.relname != 'extracted_items_default'
    LOOP
        -- Extract date from partition name (format: extracted_items_yYYYYmMMdDD)
        DECLARE
            partition_date DATE;
        BEGIN
            partition_date := TO_DATE(
                SUBSTRING(partition_record.partition_name FROM 'y(\d{4})m(\d{2})d(\d{2})'),
                'YYYYMMDD'
            );
            
            IF partition_date < cutoff_date THEN
                EXECUTE format('DROP TABLE %I', partition_record.partition_name);
                RAISE NOTICE 'Dropped old partition: %', partition_record.partition_name;
            END IF;
        EXCEPTION WHEN OTHERS THEN
            RAISE NOTICE 'Could not parse date from: %', partition_record.partition_name;
        END;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Step 8: Migrate existing data (if any) from old table
-- This can take a while for large tables - consider batching in production
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'extracted_items_old') THEN
        INSERT INTO extracted_items (id, execution_id, workflow_id, task_id, url, data, extracted_at)
        SELECT id, execution_id, workflow_id, task_id, url, data, extracted_at
        FROM extracted_items_old;
        
        -- Drop old table after successful migration
        DROP TABLE extracted_items_old;
        RAISE NOTICE 'Migrated data from extracted_items_old and dropped old table';
    END IF;
END $$;

-- Table comments
COMMENT ON TABLE extracted_items IS 'Partitioned table for extracted data. Partitioned by extracted_at (daily).';
COMMENT ON FUNCTION create_extracted_items_partition() IS 'Creates partitions for the next 7 days. Run daily via pg_cron.';
COMMENT ON FUNCTION drop_old_extracted_items_partitions(INTEGER) IS 'Drops partitions older than retention_days. Default 7 days.';

-- Usage instructions:
-- 
-- 1. Create future partitions (run daily):
--    SELECT create_extracted_items_partition();
--
-- 2. Drop old partitions (run daily, keeps 7 days by default):
--    SELECT drop_old_extracted_items_partitions(7);
--
-- 3. With pg_cron (recommended):
--    SELECT cron.schedule('create-partitions', '0 0 * * *', 'SELECT create_extracted_items_partition()');
--    SELECT cron.schedule('cleanup-partitions', '0 1 * * *', 'SELECT drop_old_extracted_items_partitions(7)');

-- Step 9: Cascade delete triggers (replaces FK constraints removed for partitioning)
-- These ensure extracted_items are cleaned up when executions/workflows are deleted

CREATE OR REPLACE FUNCTION cleanup_extracted_items_on_execution_delete()
RETURNS TRIGGER AS $$
BEGIN
    DELETE FROM extracted_items WHERE execution_id = OLD.id;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION cleanup_extracted_items_on_workflow_delete()
RETURNS TRIGGER AS $$
BEGIN
    DELETE FROM extracted_items WHERE workflow_id = OLD.id;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

-- Trigger on workflow_executions delete
CREATE TRIGGER trg_cleanup_extracted_items_execution
    BEFORE DELETE ON workflow_executions
    FOR EACH ROW EXECUTE FUNCTION cleanup_extracted_items_on_execution_delete();

-- Trigger on workflows delete
CREATE TRIGGER trg_cleanup_extracted_items_workflow
    BEFORE DELETE ON workflows
    FOR EACH ROW EXECUTE FUNCTION cleanup_extracted_items_on_workflow_delete();

COMMENT ON FUNCTION cleanup_extracted_items_on_execution_delete() IS 'Cascade delete extracted_items when execution is deleted';
COMMENT ON FUNCTION cleanup_extracted_items_on_workflow_delete() IS 'Cascade delete extracted_items when workflow is deleted';
