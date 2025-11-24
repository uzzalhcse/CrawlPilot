import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { workflowsApi } from '@/api/workflows'
import type { HealthCheckSchedule, NotificationConfig } from '@/types'

export const useScheduleStore = defineStore('schedule', () => {
    const schedules = ref<Map<string, HealthCheckSchedule>>(new Map())
    const loading = ref(false)
    const error = ref<string | null>(null)

    // Get schedule for a workflow
    const getSchedule = computed(() => {
        return (workflowId: string) => schedules.value.get(workflowId)
    })

    // Fetch schedule from API
    async function fetchSchedule(workflowId: string) {
        loading.value = true
        error.value = null
        try {
            const response = await workflowsApi.getSchedule(workflowId)
            schedules.value.set(workflowId, response.data)
            return response.data
        } catch (err: any) {
            if (err.response?.status !== 404) {
                error.value = 'Failed to fetch schedule'
                throw err
            }
            return null
        } finally {
            loading.value = false
        }
    }

    // Create or update schedule
    async function saveSchedule(workflowId: string, data: Partial<HealthCheckSchedule>) {
        loading.value = true
        error.value = null
        try {
            const response = await workflowsApi.createSchedule(workflowId, data)
            schedules.value.set(workflowId, response.data)
            return response.data
        } catch (err) {
            error.value = 'Failed to save schedule'
            throw err
        } finally {
            loading.value = false
        }
    }

    // Delete schedule
    async function deleteSchedule(workflowId: string) {
        loading.value = true
        error.value = null
        try {
            await workflowsApi.deleteSchedule(workflowId)
            schedules.value.delete(workflowId)
        } catch (err) {
            error.value = 'Failed to delete schedule'
            throw err
        } finally {
            loading.value = false
        }
    }

    // Test notification
    async function testNotification(workflowId: string, config: NotificationConfig) {
        loading.value = true
        error.value = null
        try {
            await workflowsApi.testNotification(workflowId, config)
        } catch (err) {
            error.value = 'Failed to send test notification'
            throw err
        } finally {
            loading.value = false
        }
    }

    return {
        schedules,
        loading,
        error,
        getSchedule,
        fetchSchedule,
        saveSchedule,
        deleteSchedule,
        testNotification
    }
})
