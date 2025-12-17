ALTER TABLE workflow_auto_fixes
ADD COLUMN execution_id UUID;

CREATE INDEX idx_workflow_auto_fixes_execution_id ON workflow_auto_fixes(execution_id);
