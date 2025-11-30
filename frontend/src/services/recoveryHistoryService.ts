import apiClient from '@/api/client'

export interface RecoveryAction {
    type: string
    parameters: Record<string, any>
}

export interface RecoveryHistoryRecord {
    id: string
    execution_id: string
    workflow_id: string

    // Error Details
    error_type: string
    error_message: string
    status_code?: number
    url: string
    domain: string
    node_id: string
    phase_id: string

    // Pattern Analysis
    pattern_detected: boolean
    pattern_type?: string
    activation_reason?: string
    error_rate?: number

    // Recovery Solution
    rule_id?: string
    rule_name?: string
    solution_type: 'rule' | 'ai' | 'none'
    confidence?: number

    // Actions Applied
    actions_applied?: RecoveryAction[]

    // Outcome
    recovery_attempted: boolean
    recovery_successful: boolean
    retry_count: number
    time_to_recovery_ms: number

    // Context
    request_context?: Record<string, any>

    // Timestamps
    detected_at: string
    recovered_at?: string
    created_at: string
}

export interface RecoveryStats {
    total_attempts: number
    successful_recoveries: number
    success_rate: number
    avg_recovery_time_ms: number
    by_error_type: Record<string, { count: number; success_rate: number }>
    by_rule: Record<string, { count: number; success_rate: number; avg_time_ms: number }>
    by_domain: Record<string, { count: number; success_rate: number }>
    timeline?: Array<{ timestamp: string; attempts: number; successes: number }>
}

export interface RecoveryStatsFilter {
    time_range?: string  // e.g., '24h', '7d'
    workflow_id?: string
    domain?: string
    error_type?: string
}

class RecoveryHistoryService {
    /**
     * Get recovery history for a specific execution
     */
    async getExecutionHistory(executionId: string): Promise<RecoveryHistoryRecord[]> {
        const response = await apiClient.get(`/executions/${executionId}/recovery-history`)
        return response.data.events || []
    }

    /**
     * Get recovery history for a specific workflow
     */
    async getWorkflowHistory(workflowId: string, limit = 50): Promise<RecoveryHistoryRecord[]> {
        const response = await apiClient.get(`/workflows/${workflowId}/recovery-history`, {
            params: { limit }
        })
        return response.data.events || []
    }

    /**
     * Get recent recovery events across all workflows
     */
    async getRecentHistory(limit = 100): Promise<RecoveryHistoryRecord[]> {
        const response = await apiClient.get('/error-recovery/history/recent', {
            params: { limit }
        })
        return response.data.events || []
    }

    /**
     * Get aggregated recovery statistics
     */
    async getStats(filter?: RecoveryStatsFilter): Promise<RecoveryStats> {
        const response = await apiClient.get('/error-recovery/history/stats', {
            params: filter
        })
        return response.data.stats
    }
}

export default new RecoveryHistoryService()
