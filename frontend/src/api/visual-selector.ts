import apiClient from './client'

export interface StartSessionRequest {
    url: string
    workflow_id?: string
    driver?: string      // playwright or camoufox (default: playwright)
    profile_id?: string  // browser profile ID for fingerprint/proxy
    existing_fields?: Record<string, any>
}

export interface VisualSelectorSession {
    session_id: string
    status: 'launching' | 'running' | 'completed' | 'failed' | 'closed'
    url?: string
    workflow_id?: string
    fields?: Record<string, any>
    error?: string
    created_at?: string
    completed_at?: string
}

export const visualSelectorApi = {
    // Start a new visual selector session
    startSession(data: StartSessionRequest) {
        return apiClient.post<{ session_id: string; status: string }>('/visual-selector/start', data)
    },

    // Get session result (poll this until status is 'completed' or 'failed')
    getResult(sessionId: string) {
        return apiClient.get<VisualSelectorSession>(`/visual-selector/${sessionId}/result`)
    },

    // Close a session manually
    closeSession(sessionId: string) {
        return apiClient.delete(`/visual-selector/${sessionId}`)
    }
}
