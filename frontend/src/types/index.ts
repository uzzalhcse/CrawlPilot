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
