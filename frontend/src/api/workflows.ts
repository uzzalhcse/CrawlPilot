import apiClient from './client'
import type { Workflow, WorkflowConfig } from '@/types'

export interface CreateWorkflowRequest {
  name: string
  description: string
  config: WorkflowConfig
}

export interface UpdateWorkflowRequest {
  name?: string
  description?: string
  status?: 'draft' | 'active' | 'inactive'
  config?: WorkflowConfig
}

export interface ListWorkflowsParams {
  status?: 'draft' | 'active' | 'inactive'
  limit?: number
  offset?: number
}

export const workflowsApi = {
  // List all workflows
  list(params?: ListWorkflowsParams) {
    return apiClient.get<{ count: number; workflows: Workflow[] }>('/workflows', { params })
  },

  // Get workflow by ID
  getById(id: string) {
    return apiClient.get<Workflow>(`/workflows/${id}`)
  },

  // Create new workflow
  create(data: CreateWorkflowRequest) {
    return apiClient.post<Workflow>('/workflows', data)
  },

  // Update workflow
  update(id: string, data: UpdateWorkflowRequest) {
    return apiClient.put<Workflow>(`/workflows/${id}`, data)
  },

  // Update workflow status
  updateStatus(id: string, status: 'draft' | 'active' | 'inactive') {
    return apiClient.patch<Workflow>(`/workflows/${id}/status`, { status })
  },

  // Delete workflow
  delete(id: string) {
    return apiClient.delete(`/workflows/${id}`)
  },

  // Execute workflow
  execute(id: string) {
    return apiClient.post<{ execution_id: string }>(`/workflows/${id}/execute`)
  }
}
