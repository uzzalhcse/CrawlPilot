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
  url_discovery: Node[]
  data_extraction: Node[]
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
  status: 'pending' | 'running' | 'completed' | 'failed' | 'stopped'
  started_at: string
  completed_at?: string
  error?: string
  stats?: ExecutionStats
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
  created_at: string
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
}

export interface WorkflowEdge {
  id: string
  source: string
  target: string
  type?: string
  animated?: boolean
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
  | 'wait_for'
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
  type: 'text' | 'number' | 'boolean' | 'select' | 'textarea' | 'array'
  required?: boolean
  defaultValue?: any
  options?: { label: string; value: string }[]
  placeholder?: string
  description?: string
}
