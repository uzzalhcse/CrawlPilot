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
