import apiClient from './client'

// Domain Strategy Types
export interface DomainStrategy {
    id: string
    domain: string
    recommended_tier: number        // 0=direct, 1=datacenter, 2=residential, 3=mobile
    tier_confidence: number         // 0.00-1.00
    session_requirement: number     // 0=unknown, 1=required, 2=optional, 3=not_needed
    detected_cookies?: string[]
    optimal_rotation_count?: number
    rotation_confidence: number
    adaptive_delay_ms: number
    max_concurrent_requests?: number
    total_requests: number
    total_successes: number
    total_failures: number
    success_rate: number
    learning_status: 'new' | 'learning' | 'stable' | 'needs_review'
    sample_size: number
    last_block_at?: string
    last_success_at?: string
    tier_attempts?: string          // JSON string of tier history
    created_at: string
    updated_at: string
}

export interface DomainStrategyStats {
    total: number
    new: number
    learning: number
    stable: number
    needs_review: number
    avg_success_rate: number
}

// API Functions
export async function getDomainStrategies(params?: {
    status?: string
    limit?: number
    offset?: number
}): Promise<{ strategies: DomainStrategy[]; total: number; stats: DomainStrategyStats }> {
    const response = await apiClient.get('/recovery/domains', { params })
    return response.data
}

export async function getDomainStrategy(domain: string): Promise<DomainStrategy> {
    const response = await apiClient.get(`/recovery/domains/${encodeURIComponent(domain)}`)
    return response.data
}

export async function getDomainStrategyStats(): Promise<DomainStrategyStats> {
    const response = await apiClient.get('/recovery/domains/stats')
    return response.data
}

export async function clearDomainStrategy(domain: string): Promise<{ success: boolean; message: string }> {
    const response = await apiClient.delete(`/recovery/domains/${encodeURIComponent(domain)}`)
    return response.data
}

export async function clearAllDomainStrategies(): Promise<{ success: boolean; deleted: number; message: string }> {
    const response = await apiClient.delete('/recovery/domains', { data: { confirm: true } })
    return response.data
}

// Create a new domain strategy
export interface CreateDomainStrategyRequest {
    domain: string
    recommended_tier: number
}

export async function createDomainStrategy(data: CreateDomainStrategyRequest): Promise<DomainStrategy> {
    const response = await apiClient.post('/recovery/domains', data)
    return response.data
}

// Update an existing domain strategy
export interface UpdateDomainStrategyRequest {
    recommended_tier?: number
    adaptive_delay_ms?: number
    max_concurrent_requests?: number
    learning_status?: string
    session_requirement?: number
}

export async function updateDomainStrategy(domain: string, data: UpdateDomainStrategyRequest): Promise<DomainStrategy> {
    const response = await apiClient.patch(`/recovery/domains/${encodeURIComponent(domain)}`, data)
    return response.data
}
