import apiClient from './client'
import type { Workflow, WorkflowConfig, HealthCheckSchedule, NotificationConfig, HealthCheckReport, ComparisonResponse, HealthCheckSnapshot } from '@/types'

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

  // Run health check
  runHealthCheck(id: string, config?: any) {
    return apiClient.post<{ message: string; workflow_id: string }>(`/workflows/${id}/monitoring/run`, config || {})
  },

  // Get health check reports for a workflow
  getHealthChecks(id: string, limit?: number) {
    return apiClient.get<{ workflow_id: string; reports: any[]; total: number }>(`/workflows/${id}/monitoring`, {
      params: { limit: limit || 10 }
    })
  },

  // Get specific health check report
  getHealthCheckReport(reportId: string) {
    return apiClient.get<any>(`/monitoring/${reportId}`)
  },

  // Phase 2: Schedule Management
  getSchedule(workflowId: string) {
    return apiClient.get<HealthCheckSchedule>(`/workflows/${workflowId}/schedule`)
  },

  createSchedule(workflowId: string, data: Partial<HealthCheckSchedule>) {
    return apiClient.post<HealthCheckSchedule>(`/workflows/${workflowId}/schedule`, data)
  },

  deleteSchedule(workflowId: string) {
    return apiClient.delete(`/workflows/${workflowId}/schedule`)
  },

  testNotification(workflowId: string, config: NotificationConfig) {
    return apiClient.post(`/workflows/${workflowId}/monitoring/run`, config)
  },

  // Baseline Management
  setBaseline(reportId: string) {
    return apiClient.post(`/monitoring/${reportId}/set-baseline`)
  },

  getBaseline(workflowId: string) {
    return apiClient.get<HealthCheckReport>(`/workflows/${workflowId}/baseline`)
  },

  compareWithBaseline(reportId: string) {
    return apiClient.get<ComparisonResponse>(`/monitoring/${reportId}/compare`)
  },

  // Snapshot API methods
  getSnapshotsByReport(reportId: string) {
    return apiClient.get<{ report_id: string; snapshots: HealthCheckSnapshot[]; total: number }>(`/monitoring/${reportId}/snapshots`)
  },

  getSnapshot(snapshotId: string) {
    return apiClient.get<HealthCheckSnapshot>(`/snapshots/${snapshotId}`)
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
