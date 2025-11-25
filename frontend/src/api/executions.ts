import apiClient from './client'
import type { Execution, ExecutionStats, ExtractedData } from '@/types'

export interface ExecutionTimeline {
  node_id: string
  node_name: string
  node_type: string
  started_at: string
  completed_at: string
  duration_ms: number
  status: string
  error?: string
  timestamp?: string // Alias for started_at
  urls_discovered?: number
  items_extracted?: number
}

export interface ExecutionHierarchy {
  url: string
  url_type: string
  depth: number
  parent_url?: string
  discovered_by?: string
  children?: ExecutionHierarchy[]
}

export interface PerformanceMetrics {
  node_type: string
  node_name?: string
  total_executions: number
  executions?: number // Alias for total_executions
  successful: number
  failed: number
  success_rate?: number // Computed or returned
  avg_duration_ms: number
  min_duration_ms: number
  max_duration_ms: number
  total_urls_discovered?: number
  total_items_extracted?: number
}

export interface ItemWithHierarchy {
  id: string
  url: string
  schema: string
  data: Record<string, any>
  hierarchy: {
    depth: number
    parent_url?: string
    discovered_by?: string
  }
}

export interface Bottleneck {
  node_execution_id: string
  node_id: string
  node_name: string
  url: string
  duration_ms: number
  status: string
  error?: string
  started_at: string
}

export interface NodeTreeNode {
  id: string
  node_id: string
  node_type: string
  status: string
  started_at: string
  completed_at?: string
  duration_ms?: number
  urls_discovered: number
  items_extracted: number
  error_message?: string
  parent_node_execution_id?: string
  children?: NodeTreeNode[]
}

export interface NodeTreeStats {
  total_nodes: number
  completed_nodes: number
  failed_nodes: number
  max_depth: number
  total_urls_found: number
  total_items_found: number
}

export interface NodeTreeResponse {
  execution_id: string
  tree: NodeTreeNode[]
  stats: NodeTreeStats
}

export const executionsApi = {
  // List all executions
  list(params?: { workflow_id?: string; status?: string; limit?: number; offset?: number }) {
    return apiClient.get<{ executions: Execution[]; total: number; limit: number; offset: number }>('/executions', { params })
  },

  // Get execution by ID
  getById(id: string) {
    return apiClient.get<Execution>(`/executions/${id}`)
  },

  // Get execution statistics
  getStats(id: string) {
    return apiClient.get<ExecutionStats>(`/executions/${id}/stats`)
  },

  // Get extracted data
  getData(id: string, params?: { limit?: number; offset?: number }) {
    return apiClient.get<{ items: ExtractedData[]; total: number; limit: number; offset: number }>(`/executions/${id}/data`, { params })
  },

  // Stop execution
  stop(id: string) {
    return apiClient.delete(`/executions/${id}`)
  },

  // Pause execution
  pause(id: string) {
    return apiClient.post(`/executions/${id}/pause`)
  },

  // Resume execution
  resume(id: string) {
    return apiClient.post(`/executions/${id}/resume`)
  },

  // Get execution timeline
  getTimeline(id: string) {
    return apiClient.get<ExecutionTimeline[]>(`/executions/${id}/timeline`)
  },

  // Get URL hierarchy
  getHierarchy(id: string) {
    return apiClient.get<ExecutionHierarchy[]>(`/executions/${id}/hierarchy`)
  },

  // Get performance metrics
  getPerformance(id: string) {
    return apiClient.get<PerformanceMetrics[]>(`/executions/${id}/performance`)
  },

  // Get items with hierarchy
  getItemsWithHierarchy(id: string) {
    return apiClient.get<ItemWithHierarchy[]>(`/executions/${id}/items-with-hierarchy`)
  },

  // Get bottlenecks
  getBottlenecks(id: string, threshold?: number) {
    return apiClient.get<Bottleneck[]>(`/executions/${id}/bottlenecks`, {
      params: { threshold }
    })
  },

  // Get node execution tree
  getNodeTree(id: string) {
    return apiClient.get<NodeTreeResponse>(`/executions/${id}/node-tree`)
  }
}
