import apiClient from './client'

// Types
export interface Proxy {
    id: string
    proxy_id: string
    server: string
    username: string
    password: string
    proxy_address: string
    port: number
    valid: boolean
    last_verified: string
    country_code: string
    city_name: string
    asn_name: string
    asn_number: number
    confidence_high: boolean
    proxy_type: string
    failure_count: number
    success_count: number
    last_used: string
    is_healthy: boolean
    created_at: string
    updated_at: string
}

export interface ProxyStats {
    total: number
    healthy: number
    unhealthy: number
    by_country: Record<string, number>
    by_type: Record<string, number>
}

export interface CreateProxyRequest {
    server: string
    username?: string
    password?: string
    proxy_address: string
    port: number
    proxy_type?: string
    country_code?: string
}

// API Functions
export async function getProxies(): Promise<{ proxies: Proxy[]; total: number }> {
    const response = await apiClient.get('/recovery/proxies')
    return response.data
}

export async function createProxy(proxy: CreateProxyRequest): Promise<Proxy> {
    const response = await apiClient.post('/recovery/proxies', proxy)
    return response.data
}

export async function updateProxy(id: string, proxy: Partial<CreateProxyRequest>): Promise<Proxy> {
    const response = await apiClient.put(`/recovery/proxies/${id}`, proxy)
    return response.data
}

export async function deleteProxy(id: string): Promise<void> {
    await apiClient.delete(`/recovery/proxies/${id}`)
}

export async function toggleProxy(id: string, enabled: boolean): Promise<{ id: string; is_healthy: boolean }> {
    const response = await apiClient.patch(`/recovery/proxies/${id}/toggle`, { is_healthy: enabled })
    return response.data
}

export async function getProxyStats(): Promise<ProxyStats> {
    const response = await apiClient.get('/recovery/proxies/stats')
    return response.data
}

// Recovery Config API
export interface RecoveryConfig {
    key: string
    value: any
    category: string
    description: string
    updated_at: string
}

export async function getRecoveryConfigs(): Promise<{ configs: RecoveryConfig[] }> {
    const response = await apiClient.get('/recovery/config')
    return response.data
}

export async function updateRecoveryConfig(key: string, value: any): Promise<RecoveryConfig> {
    const response = await apiClient.put(`/recovery/config/item/${key}`, { value })
    return response.data
}

export async function updateMultipleConfigs(updates: Record<string, any>): Promise<void> {
    await apiClient.put('/recovery/config', { updates })
}

// Recovery Attempts Types
export interface RecoveryAttempt {
    id: string
    execution_id: string
    task_id: string
    workflow_id: string
    url: string
    domain: string
    error_pattern: string
    error_message: string
    status_code: number
    action: string
    source: string // rule, ai, default, none, pending
    rule_id: string
    ai_reasoning: string
    status: 'detected' | 'pending' | 'success' | 'failed'
    retry_delay_ms: number
    duration_ms: number
    proxy_id: string
    proxy_tier: number
    tier_from: number
    tier_to: number
    confidence: number
    trigger_reason: string
    retry_count: number
    created_at: string
    updated_at: string
}

export interface RecoveryAttemptStats {
    total: number
    pending: number
    success: number
    failed: number
    success_rate: number
    from_rules: number
    from_ai: number
    from_default: number
    last_hour: number
    last_24h: number
    by_pattern: PatternStats[]
}

export interface PatternStats {
    pattern: string
    total: number
    success: number
    failed: number
    success_rate: number
}

// Recovery Attempts API Functions
export async function getRecoveryAttempts(params?: {
    execution_id?: string
    status?: string
    pattern?: string
    limit?: number
    offset?: number
}): Promise<{ attempts: RecoveryAttempt[]; total: number; limit: number; offset: number }> {
    const response = await apiClient.get('/recovery/attempts', { params })
    return response.data
}

export async function getRecoveryAttemptStats(): Promise<RecoveryAttemptStats> {
    const response = await apiClient.get('/recovery/attempts/stats')
    return response.data
}

// External Proxy API Import
export interface ExternalProxyResponse {
    data: {
        data: ExternalProxy[]
        message: string
        meta: {
            limit: number
            page: number
            total: number
            total_pages: number
        }
    }
    success: boolean
}

export interface ExternalProxy {
    id: string
    proxy_id: string
    server: string
    username: string
    password: string
    proxy_address: string
    port: number
    valid: boolean
    last_verification: string
    country_code: string
    city_name: string
    asn_name: string
    asn_number: number
    high_country_confidence: boolean
    proxy_type: string
}

export async function importProxiesFromApi(apiUrl: string): Promise<{ imported: number; failed: number }> {
    // Fetch proxies from external API
    const response = await fetch(apiUrl)
    if (!response.ok) {
        throw new Error(`Failed to fetch proxies from API: ${response.statusText}`)
    }

    const data: ExternalProxyResponse = await response.json()

    if (!data.success || !data.data?.data) {
        throw new Error('Invalid API response format')
    }

    let imported = 0
    let failed = 0

    // Import each proxy
    for (const externalProxy of data.data.data) {
        try {
            await createProxy({
                server: externalProxy.server,
                username: externalProxy.username || '',
                password: externalProxy.password || '',
                proxy_address: externalProxy.proxy_address,
                port: externalProxy.port,
                proxy_type: externalProxy.proxy_type || 'static',
                country_code: externalProxy.country_code || ''
            })
            imported++
        } catch (error) {
            console.error(`Failed to import proxy ${externalProxy.server}:`, error)
            failed++
        }
    }

    return { imported, failed }
}
