import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { workflowsApi } from '@/api/workflows'
import type { Workflow } from '@/types'

export const useWorkflowsStore = defineStore('workflows', () => {
  const workflows = ref<Workflow[]>([])
  const currentWorkflow = ref<Workflow | null>(null)
  const recentExecutions = ref<any[]>([]) // Track recently started executions
  const loading = ref(false)
  const error = ref<string | null>(null)

  // Computed
  const activeWorkflows = computed(() =>
    workflows.value.filter(w => w.status === 'active')
  )

  const draftWorkflows = computed(() =>
    workflows.value.filter(w => w.status === 'draft')
  )

  const inactiveWorkflows = computed(() =>
    workflows.value.filter(w => w.status === 'inactive')
  )

  // Actions
  async function fetchWorkflows(params?: { status?: string; limit?: number; offset?: number }) {
    loading.value = true
    error.value = null
    try {
      const response = await workflowsApi.list(params)
      // Backend returns { count, workflows: [...] }
      workflows.value = response.data.workflows || []
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to fetch workflows'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function fetchWorkflowById(id: string) {
    loading.value = true
    error.value = null
    try {
      const response = await workflowsApi.getById(id)
      currentWorkflow.value = response.data
      return response.data
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to fetch workflow'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function createWorkflow(data: { name: string; description: string; status?: 'draft' | 'active'; config: any }) {
    loading.value = true
    error.value = null
    try {
      // Extract browser_profile_id from config if present
      const payload = {
        ...data,
        browser_profile_id: data.config?.browser_profile_id
      }
      const response = await workflowsApi.create(payload)
      workflows.value.unshift(response.data)
      return response.data
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to create workflow'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function updateWorkflow(id: string, data: any) {
    loading.value = true
    error.value = null
    try {
      // Extract browser_profile_id from config if present
      const payload = {
        ...data,
        browser_profile_id: data.config?.browser_profile_id
      }
      const response = await workflowsApi.update(id, payload)
      const index = workflows.value.findIndex(w => w.id === id)
      if (index !== -1) {
        workflows.value[index] = response.data
      }
      if (currentWorkflow.value?.id === id) {
        currentWorkflow.value = response.data
      }
      return response.data
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to update workflow'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function updateWorkflowStatus(id: string, status: 'draft' | 'active' | 'inactive') {
    loading.value = true
    error.value = null
    try {
      const response = await workflowsApi.updateStatus(id, status)
      const index = workflows.value.findIndex(w => w.id === id)
      if (index !== -1) {
        workflows.value[index] = response.data
      }
      return response.data
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to update workflow status'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function deleteWorkflow(id: string) {
    loading.value = true
    error.value = null
    try {
      await workflowsApi.delete(id)
      workflows.value = workflows.value.filter(w => w.id !== id)
      if (currentWorkflow.value?.id === id) {
        currentWorkflow.value = null
      }
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to delete workflow'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function executeWorkflow(id: string) {
    loading.value = true
    error.value = null
    try {
      const response = await workflowsApi.execute(id)
      const workflow = workflows.value.find(w => w.id === id)

      // Track the execution
      if (workflow) {
        recentExecutions.value.unshift({
          id: response.data.execution_id,
          workflow_id: id,
          workflow_name: workflow.name,
          status: 'running',
          started_at: new Date().toISOString(),
          stats: {
            total_urls: 0,
            pending: 0,
            processing: 0,
            completed: 0,
            failed: 0,
            items_extracted: 0
          }
        })

        // Keep only last 50 executions
        if (recentExecutions.value.length > 50) {
          recentExecutions.value = recentExecutions.value.slice(0, 50)
        }
      }

      return response.data
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to execute workflow'
      throw e
    } finally {
      loading.value = false
    }
  }

  function clearError() {
    error.value = null
  }

  return {
    workflows,
    currentWorkflow,
    recentExecutions,
    loading,
    error,
    activeWorkflows,
    draftWorkflows,
    inactiveWorkflows,
    fetchWorkflows,
    fetchWorkflowById,
    createWorkflow,
    updateWorkflow,
    updateWorkflowStatus,
    deleteWorkflow,
    executeWorkflow,
    clearError
  }
})
