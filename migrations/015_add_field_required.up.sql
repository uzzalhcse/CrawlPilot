-- Add field_required column to track whether a field is required or optional
ALTER TABLE health_check_snapshots 
ADD COLUMN field_required BOOLEAN DEFAULT true;

-- Update existing snapshots to have field_required = true (backward compatible)
UPDATE health_check_snapshots 
SET field_required = true 
WHERE field_required IS NULL;
