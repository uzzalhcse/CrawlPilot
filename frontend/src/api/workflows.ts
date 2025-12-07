import apiClient from './client'
import type { Workflow, WorkflowConfig } from '@/types'

export interface CreateWorkflowRequest {
  name: string
  description: string
  status?: 'draft' | 'active' | 'inactive'
  browser_profile_id?: string
  config: WorkflowConfig
}

export interface UpdateWorkflowRequest {
  name?: string
  description?: string
  status?: 'draft' | 'active' | 'inactive'
  browser_profile_id?: string
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
  },

  // Snapshot API methods (used by Probes)
  getSnapshot(snapshotId: string) {
    return apiClient.get<any>(`/snapshots/${snapshotId}`)
  },

  getScreenshotUrl(snapshotId: string) {
    return `/api/v1/snapshots/${snapshotId}/screenshot`
  },

  getDOMUrl(snapshotId: string) {
    return `/api/v1/snapshots/${snapshotId}/dom`
  },

  deleteSnapshot(snapshotId: string) {
    return apiClient.delete(`/snapshots/${snapshotId}`)
  },

  // AI Auto-fix methods
  analyzeSnapshot(snapshotId: string) {
    return apiClient.post(`/snapshots/${snapshotId}/analyze`)
  },

  getSuggestions(snapshotId: string) {
    return apiClient.get(`/snapshots/${snapshotId}/suggestions`)
  },

  approveSuggestion(suggestionId: string) {
    return apiClient.post(`/suggestions/${suggestionId}/approve`)
  },

  rejectSuggestion(suggestionId: string) {
    return apiClient.post(`/suggestions/${suggestionId}/reject`)
  },

  applySuggestion(suggestionId: string) {
    return apiClient.post(`/suggestions/${suggestionId}/apply`)
  },

  revertSuggestion(suggestionId: string) {
    return apiClient.post(`/suggestions/${suggestionId}/revert`)
  }
}
