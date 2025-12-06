import apiClient from './client'

// Types
export interface Incident {
    id: string
    execution_id: string
    task_id: string
    url: string
    domain: string
    error_pattern: string
    error_message: string
    status_code: number
    page_content: string
    recovery_attempts: RecoveryAttempt[]
    status: 'open' | 'in_progress' | 'resolved' | 'ignored'
    priority: 'low' | 'medium' | 'high' | 'critical'
    assigned_to: string | null
    resolution: string | null
    created_at: string
    updated_at: string
    resolved_at: string | null
}

export interface RecoveryAttempt {
    action: string
    params: Record<string, any>
    source: string
    timestamp: string
    success: boolean
}

export interface IncidentStats {
    total: number
    by_status: Record<string, number>
    by_priority: Record<string, number>
    by_pattern: Record<string, number>
}

export interface DomainStats {
    domain: string
    total_incidents: number
    open_incidents: number
    last_incident: string
}

// API Functions
export async function getIncidents(params?: {
    status?: string
    priority?: string
    limit?: number
    offset?: number
}): Promise<{ incidents: Incident[]; total: number; limit: number; offset: number }> {
    const response = await apiClient.get('/incidents/', { params })
    return response.data
}

export async function getIncident(id: string): Promise<Incident> {
    const response = await apiClient.get(`/incidents/${id}`)
    return response.data
}

export async function updateIncidentStatus(
    id: string,
    status: Incident['status'],
    resolution?: string
): Promise<{ success: boolean; id: string; status: string }> {
    const response = await apiClient.patch(`/incidents/${id}/status`, { status, resolution })
    return response.data
}

export async function assignIncident(
    id: string,
    userId: string
): Promise<{ success: boolean; id: string; assigned_to: string }> {
    const response = await apiClient.patch(`/incidents/${id}/assign`, { user_id: userId })
    return response.data
}

export async function resolveIncident(
    id: string,
    resolution?: string
): Promise<{ success: boolean; id: string; status: string; resolution: string }> {
    const response = await apiClient.post(`/incidents/${id}/resolve`, { resolution })
    return response.data
}

export async function getIncidentStats(): Promise<IncidentStats> {
    const response = await apiClient.get('/incidents/stats')
    return response.data
}

export async function getDomainStats(): Promise<{ domains: DomainStats[] }> {
    const response = await apiClient.get('/incidents/domains')
    return response.data
}
