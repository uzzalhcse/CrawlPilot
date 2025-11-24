import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { executionsApi } from '@/api/executions'
import { workflowsApi } from '@/api/workflows'
import type { Execution, ExecutionStats, ExtractedData } from '@/types'
import type {
  ExecutionTimeline,
  ExecutionHierarchy,
  PerformanceMetrics,
  ItemWithHierarchy,
  Bottleneck
} from '@/api/executions'

export const useExecutionsStore = defineStore('executions', () => {
  const executions = ref<Execution[]>([])
  const currentExecution = ref<Execution | null>(null)
  const executionStats = ref<ExecutionStats | null>(null)
  const extractedData = ref<ExtractedData[]>([])
  const timeline = ref<ExecutionTimeline[]>([])
  const hierarchy = ref<ExecutionHierarchy[]>([])
  const performance = ref<PerformanceMetrics[]>([])
  const itemsWithHierarchy = ref<ItemWithHierarchy[]>([])
  const bottlenecks = ref<Bottleneck[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  // Pagination State
  const extractedDataTotal = ref(0)
  const extractedDataLimit = ref(50)
  const extractedDataOffset = ref(0)

  // Computed
  const runningExecutions = computed(() =>
    executions.value.filter(e => e.status === 'running')
  )

  const completedExecutions = computed(() =>
    executions.value.filter(e => e.status === 'completed')
  )

  const failedExecutions = computed(() =>
    executions.value.filter(e => e.status === 'failed')
  )

  // Actions
  async function fetchAllExecutions(params?: { workflow_id?: string; status?: string; limit?: number; offset?: number }) {
    loading.value = true
    error.value = null
    try {
      const response = await executionsApi.list(params)
      executions.value = response.data.executions || []
      return executions.value
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to fetch executions'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function fetchExecutionById(id: string) {
    loading.value = true
    error.value = null
    try {
      const response = await executionsApi.getById(id)
      // Backend now returns full WorkflowExecution object
      currentExecution.value = response.data as any
      return currentExecution.value
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to fetch execution'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function fetchExecutionStats(id: string) {
    loading.value = true
    error.value = null
    try {
      const response = await executionsApi.getStats(id)
      // Backend returns { execution_id, stats, pending_count }
      const stats = response.data.stats || {}
      const pending = response.data.pending_count || 0

      executionStats.value = {
        total_urls: (stats.completed || 0) + (stats.processing || 0) + pending,
        pending: pending,
        processing: stats.processing || 0,
        completed: stats.completed || 0,
        failed: stats.failed || 0,
        items_extracted: stats.items_extracted || 0
      } as any

      return executionStats.value
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to fetch execution stats'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function fetchExtractedData(id: string, params?: { limit?: number; offset?: number }) {
    loading.value = true
    error.value = null
    try {
      const response = await executionsApi.getData(id, params)
      // Backend returns { execution_id, items, total, limit, offset }
      extractedData.value = response.data.items || []
      extractedDataTotal.value = response.data.total || 0
      extractedDataLimit.value = response.data.limit || 50
      extractedDataOffset.value = response.data.offset || 0
      return extractedData.value
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to fetch extracted data'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function stopExecution(id: string) {
    loading.value = true
    error.value = null
    try {
      await executionsApi.stop(id)
      if (currentExecution.value?.id === id) {
        currentExecution.value.status = 'stopped'
      }
      const index = executions.value.findIndex(e => e.id === id)
      if (index !== -1) {
        executions.value[index].status = 'stopped'
      }
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to stop execution'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function pauseExecution(id: string) {
    loading.value = true
    error.value = null
    try {
      await executionsApi.pause(id)
      if (currentExecution.value?.id === id) {
        currentExecution.value.status = 'paused'
      }
      const index = executions.value.findIndex(e => e.id === id)
      if (index !== -1) {
        executions.value[index].status = 'paused'
      }
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to pause execution'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function resumeExecution(id: string) {
    loading.value = true
    error.value = null
    try {
      await executionsApi.resume(id)
      if (currentExecution.value?.id === id) {
        currentExecution.value.status = 'running'
      }
      const index = executions.value.findIndex(e => e.id === id)
      if (index !== -1) {
        executions.value[index].status = 'running'
      }
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to resume execution'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function fetchTimeline(id: string) {
    loading.value = true
    error.value = null
    try {
      const response = await executionsApi.getTimeline(id)
      // Backend returns { execution_id, timeline, summary }
      timeline.value = response.data.timeline || []
      return timeline.value
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to fetch timeline'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function fetchHierarchy(id: string) {
    loading.value = true
    error.value = null
    try {
      const response = await executionsApi.getHierarchy(id)
      hierarchy.value = response.data
      return response.data
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to fetch hierarchy'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function fetchPerformance(id: string) {
    loading.value = true
    error.value = null
    try {
      const response = await executionsApi.getPerformance(id)
      // Backend returns { execution_id, node_metrics, summary }
      performance.value = response.data.node_metrics || []
      return performance.value
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to fetch performance'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function fetchItemsWithHierarchy(id: string) {
    loading.value = true
    error.value = null
    try {
      const response = await executionsApi.getItemsWithHierarchy(id)
      itemsWithHierarchy.value = response.data
      return response.data
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to fetch items with hierarchy'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function fetchBottlenecks(id: string, threshold: number = 5000) {
    loading.value = true
    error.value = null
    try {
      const response = await executionsApi.getBottlenecks(id, threshold)
      bottlenecks.value = response.data
      return response.data
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to fetch bottlenecks'
      throw e
    } finally {
      loading.value = false
    }
  }

  function clearError() {
    error.value = null
  }

  function clearCurrentExecution() {
    currentExecution.value = null
    executionStats.value = null
    extractedData.value = []
    timeline.value = []
    hierarchy.value = []
    performance.value = []
    itemsWithHierarchy.value = []
    bottlenecks.value = []
    extractedDataTotal.value = 0
    extractedDataLimit.value = 50
    extractedDataOffset.value = 0
  }

  return {
    executions,
    currentExecution,
    executionStats,
    extractedData,
    extractedDataTotal,
    extractedDataLimit,
    extractedDataOffset,
    timeline,
    hierarchy,
    performance,
    itemsWithHierarchy,
    bottlenecks,
    loading,
    error,
    runningExecutions,
    completedExecutions,
    failedExecutions,
    fetchAllExecutions,
    fetchExecutionById,
    fetchExecutionStats,
    fetchExtractedData,
    stopExecution,
    pauseExecution,
    resumeExecution,
    fetchTimeline,
    fetchHierarchy,
    fetchPerformance,
    fetchItemsWithHierarchy,
    fetchBottlenecks,
    clearError,
    clearCurrentExecution
  }
})
