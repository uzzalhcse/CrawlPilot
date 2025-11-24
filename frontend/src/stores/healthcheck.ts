import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { workflowsApi } from '@/api/workflows'
import type { HealthCheckReport } from '@/types'

export const useHealthCheckStore = defineStore('healthcheck', () => {
    // State
    const reports = ref<HealthCheckReport[]>([])
    const currentReport = ref<HealthCheckReport | null>(null)
    const loading = ref(false)
    const error = ref<string | null>(null)

    // Computed
    const hasReports = computed(() => reports.value.length > 0)
    const latestReport = computed(() => reports.value[0] || null)

    // Actions
    async function runHealthCheck(workflowId: string, config?: any) {
        loading.value = true
        error.value = null
        try {
            const response = await workflowsApi.runHealthCheck(workflowId, config)
            return response.data
        } catch (e: any) {
            error.value = e.response?.data?.error || 'Failed to run health check'
            throw e
        } finally {
            loading.value = false
        }
    }

    async function fetchHealthChecks(workflowId: string, limit = 10) {
        loading.value = true
        error.value = null
        try {
            const response = await workflowsApi.getHealthChecks(workflowId, limit)
            reports.value = response.data.reports || []
        } catch (e: any) {
            error.value = e.response?.data?.error || 'Failed to fetch health checks'
            reports.value = []
        } finally {
            loading.value = false
        }
    }

    async function fetchHealthCheckReport(reportId: string) {
        loading.value = true
        error.value = null
        try {
            const response = await workflowsApi.getHealthCheckReport(reportId)
            currentReport.value = response.data
            return response.data
        } catch (e: any) {
            error.value = e.response?.data?.error || 'Failed to fetch report'
            currentReport.value = null
            throw e
        } finally {
            loading.value = false
        }
    }

    function clearCurrentReport() {
        currentReport.value = null
    }

    function clearError() {
        error.value = null
    }

    return {
        // State
        reports,
        currentReport,
        loading,
        error,
        // Computed
        hasReports,
        latestReport,
        // Actions
        runHealthCheck,
        fetchHealthChecks,
        fetchHealthCheckReport,
        clearCurrentReport,
        clearError
    }
})
