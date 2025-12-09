import apiClient from './client'

export interface Schedule {
    id: string
    workflow_id: string
    name: string
    cron_expression: string
    timezone: string
    is_enabled: boolean
    next_run_at?: string
    last_run_at?: string
    last_execution_id?: string
    created_at: string
    updated_at: string
    workflow_name?: string
}

export interface CreateScheduleRequest {
    workflow_id: string
    name: string
    cron_expression: string
    timezone?: string
}

export interface UpdateScheduleRequest {
    name?: string
    cron_expression?: string
    timezone?: string
    is_enabled?: boolean
}

export interface ListSchedulesParams {
    workflow_id?: string
    is_enabled?: boolean
    limit?: number
    offset?: number
}

export interface ScheduleStats {
    total: number
    active: number
    paused: number
}

export interface CronPreset {
    name: string
    expression: string
    description: string
}

export const schedulesApi = {
    // List all schedules
    list(params?: ListSchedulesParams) {
        return apiClient.get<{
            schedules: Schedule[]
            count: number
            stats: ScheduleStats
        }>('/schedules', { params })
    },

    // Get schedule by ID
    getById(id: string) {
        return apiClient.get<Schedule>(`/schedules/${id}`)
    },

    // Create new schedule
    create(data: CreateScheduleRequest) {
        return apiClient.post<Schedule>('/schedules', data)
    },

    // Update schedule
    update(id: string, data: UpdateScheduleRequest) {
        return apiClient.put<Schedule>(`/schedules/${id}`, data)
    },

    // Delete schedule
    delete(id: string) {
        return apiClient.delete(`/schedules/${id}`)
    },

    // Toggle schedule enabled/disabled
    toggle(id: string) {
        return apiClient.patch<Schedule>(`/schedules/${id}/toggle`)
    },

    // Run schedule now
    runNow(id: string) {
        return apiClient.post<{ execution_id: string; schedule_id: string; workflow_id: string }>(
            `/schedules/${id}/run`
        )
    },

    // Get cron presets
    getPresets() {
        return apiClient.get<{ presets: CronPreset[] }>('/schedules/presets')
    }
}
