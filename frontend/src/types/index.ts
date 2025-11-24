export interface Workflow {
  id: string
  name: string
  description: string
  status: 'draft' | 'active' | 'inactive'
  config: WorkflowConfig
  created_at: string
  updated_at: string
}

export interface WorkflowConfig {
  start_urls: string[]
  max_depth: number
  rate_limit_delay: number
  headers?: Record<string, string>
  storage: StorageConfig
  // NEW: Phase-based format
  phases?: WorkflowPhase[]
  // LEGACY: Old format (for backward compatibility)
  url_discovery?: Node[]
  data_extraction?: Node[]
}

export interface WorkflowPhase {
  id: string
  type: 'discovery' | 'extraction' | 'processing' | 'custom'
  name?: string
  nodes: Node[]
  url_filter?: URLFilter
  transition?: PhaseTransition
}

export interface URLFilter {
  markers?: string[]
  patterns?: string[]
  depth?: number
}

export interface PhaseTransition {
  condition: string
  next_phase?: string
  params?: Record<string, any>
}

export interface Node {
  id: string
  type: string
  name: string
  params: Record<string, any>
  dependencies?: string[]
}

export interface StorageConfig {
  type: 'database' | 'file' | 'webhook'
  path?: string
  webhook_url?: string
}

export interface Execution {
  id: string
  workflow_id: string
  status: 'pending' | 'running' | 'paused' | 'completed' | 'failed' | 'stopped'
  started_at: string
  completed_at?: string
  error?: string
  stats?: ExecutionStats
  workflow_name?: string
  workflow_config?: WorkflowConfig
}

export interface ExecutionStats {
  total_urls: number
  pending: number
  processing: number
  completed: number
  failed: number
  items_extracted: number
}

export interface ExtractedData {
  id: string
  execution_id: string
  url: string
  schema: string
  data: Record<string, any>
  extracted_at: string
}

// Health Check types
export interface HealthCheckReport {
  id: string
  workflow_id: string
  workflow_name?: string
  status: 'running' | 'healthy' | 'degraded' | 'failed'
  started_at: string
  completed_at?: string
  duration_ms?: number
  phase_results?: Record<string, PhaseValidationResult>
  summary?: HealthCheckSummary
  config?: HealthCheckConfig
}

export interface PhaseValidationResult {
  phase_id: string
  phase_name: string
  node_results: NodeValidationResult[]
  navigation_error?: string
  has_critical_issues: boolean
}

export interface NodeValidationResult {
  node_id: string
  node_name: string
  node_type: string
  status: 'pass' | 'fail' | 'warning' | 'skip'
  metrics: Record<string, any>
  issues: ValidationIssue[]
  duration_ms: number
}

export interface ValidationIssue {
  severity: string
  code: string
  message: string
  selector?: string
  expected?: any
  actual?: any
  suggestion?: string
}

export interface HealthCheckSummary {
  total_phases: number
  total_nodes: number
  passed_nodes: number
  failed_nodes: number
  warning_nodes: number
  critical_issues: ValidationIssue[]
}

export interface HealthCheckConfig {
  max_urls_per_phase: number
  max_pagination_pages: number
  max_depth: number
  timeout_seconds: number
  skip_data_storage: boolean
}

// Phase 2: Scheduling and Notifications
export interface HealthCheckSchedule {
  id: string
  workflow_id: string
  schedule: string
  enabled: boolean
  last_run_at?: string
  next_run_at?: string
  notification_config?: NotificationConfig
  created_at: string
  updated_at: string
}

export interface NotificationConfig {
  slack?: SlackConfig
  only_on_failure: boolean
  only_on_change?: boolean
}

export interface SlackConfig {
  webhook_url: string
  channel?: string
}

export interface BaselineComparison {
  metric: string
  baseline: any
  current: any
  change_percent?: number
  status: 'improved' | 'degraded' | 'unchanged'
}

export interface ComparisonResponse {
  current: HealthCheckReport
  baseline: HealthCheckReport
  comparisons: BaselineComparison[]
}

// Phase 1: Diagnostic Snapshots
export interface HealthCheckSnapshot {
  id: string
  report_id: string
  node_id: string
  phase_name: string
  created_at: string
  url: string
  page_title?: string
  status_code?: number
  screenshot_path?: string
  dom_snapshot_path?: string
  console_logs?: ConsoleLog[]
  selector_type?: string
  selector_value?: string
  elements_found?: number
  error_message?: string
  metadata?: Record<string, any>
}

// Phase 2: AI Fix Suggestions
export interface FixSuggestion {
  id: string
  snapshot_id: string
  workflow_id: string
  node_id: string
  suggested_selector: string
  alternative_selectors?: string[]
  suggested_node_config?: Record<string, any>
  fix_explanation: string
  confidence_score: number
  status: 'pending' | 'approved' | 'rejected' | 'applied' | 'reverted'
  reviewed_by?: string
  reviewed_at?: string
  applied_at?: string
  reverted_at?: string
  ai_model: string
  verification_result?: VerificationResult
  created_at: string
  updated_at: string
}

export interface VerificationResult {
  is_valid: boolean
  elements_found: number
  data_preview: string[]
  error_message?: string
}

export interface ConsoleLog {
  type: string
  message: string
  timestamp: string
  source?: string
}

// Vue Flow related types
export interface WorkflowNode {
  id: string
  type: string
  position: { x: number; y: number }
  data: NodeData
}

export interface NodeData {
  label: string
  nodeType: NodeType
  params: Record<string, any>
  dependencies?: string[]
  outputKey?: string
  optional?: boolean
  retry?: RetryConfig
  // Nested node support
  phaseId?: string
  parentId?: string
  level?: number
  branch?: 'true' | 'false'
  // Execution status
  status?: 'pending' | 'running' | 'completed' | 'failed'
  startTime?: string
  endTime?: string
  result?: any
  error?: string
  logs?: string[]
}

export interface WorkflowEdge {
  id: string
  source: string
  target: string
  type?: string
  animated?: boolean
  style?: Record<string, any>
}

export interface RetryConfig {
  max_retries: number
  delay: number
}

export type NodeType =
  // URL Discovery
  | 'fetch'
  | 'extract_links'
  | 'filter_urls'
  | 'navigate'
  | 'paginate'
  // Interaction
  | 'click'
  | 'scroll'
  | 'type'
  | 'hover'
  | 'wait'

  | 'screenshot'
  // Extraction
  | 'extract'
  | 'extract_text'
  | 'extract_attr'
  | 'extract_json'
  // Transformation
  | 'transform'
  | 'filter'
  | 'map'
  | 'validate'
  // Control Flow
  | 'sequence'
  | 'conditional'
  | 'loop'
  | 'parallel'

export interface NodeCategory {
  name: string
  icon: string
  nodes: NodeTemplate[]
}

export interface NodeTemplate {
  type: NodeType
  label: string
  description: string
  category: string
  defaultParams: Record<string, any>
  paramSchema: ParamField[]
}

export interface ParamField {
  key: string
  label: string
  type: 'text' | 'number' | 'boolean' | 'select' | 'textarea' | 'array' | 'field_array' | 'nested_field_array' | 'sequence_steps'
  required?: boolean
  defaultValue?: any
  options?: { label: string; value: string }[]
  placeholder?: string
  description?: string
  arrayItemSchema?: ParamField[]
}

// Field configuration for extraction
export interface FieldConfig {
  selector?: string
  type?: 'text' | 'attr' | 'html' | 'href' | 'src'
  attribute?: string
  multiple?: boolean
  limit?: number
  transform?: string | TransformConfig[]
  default_value?: any
  fields?: Record<string, FieldConfig>
  extractions?: ExtractionPair[]
}

export interface ExtractionPair {
  key_selector: string
  value_selector: string
  key_type: 'text' | 'attr' | 'html' | 'href' | 'src'
  value_type: 'text' | 'attr' | 'html' | 'href' | 'src'
  key_attribute?: string
  value_attribute?: string
  transform?: string | TransformConfig[]
  limit?: number
}

export interface TransformConfig {
  type: string
  params?: Record<string, any>
}
