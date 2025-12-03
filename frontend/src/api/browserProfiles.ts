import apiClient from './client'

export interface BrowserProfile {
    id: string
    name: string
    description?: string
    status: 'active' | 'inactive' | 'archived' | 'running'
    folder?: string
    tags?: string[]
    browser_type: 'chromium' | 'firefox' | 'webkit'
    executable_path?: string
    cdp_endpoint?: string
    launch_args?: string[]
    user_agent: string
    platform: string
    screen_width: number
    screen_height: number
    timezone?: string
    locale?: string
    languages?: string[]
    webgl_vendor?: string
    webgl_renderer?: string
    canvas_noise: boolean
    hardware_concurrency: number
    device_memory: number
    fonts?: string[]
    do_not_track: boolean
    disable_webrtc: boolean
    geolocation_latitude?: number
    geolocation_longitude?: number
    geolocation_accuracy?: number
    proxy_enabled: boolean
    proxy_type?: string
    proxy_server?: string
    proxy_username?: string
    proxy_password?: string
    cookies?: any
    local_storage?: any
    session_storage?: any
    indexed_db?: any
    clear_on_close: boolean
    usage_count: number
    last_used_at?: string
    created_at: string
    updated_at: string
}

export interface CreateBrowserProfileRequest {
    name: string
    description?: string
    browser_type: 'chromium' | 'firefox' | 'webkit'
    folder?: string
    tags?: string[]
    executable_path?: string
    cdp_endpoint?: string
    launch_args?: string[]
    user_agent?: string
    platform?: string
    screen_width: number
    screen_height: number
    timezone?: string
    locale?: string
    languages?: string[]
    webgl_vendor?: string
    webgl_renderer?: string
    canvas_noise?: boolean
    hardware_concurrency?: number
    device_memory?: number
    fonts?: string[]
    do_not_track?: boolean
    disable_webrtc?: boolean
    geolocation_latitude?: number
    geolocation_longitude?: number
    geolocation_accuracy?: number
    proxy_enabled?: boolean
    proxy_type?: string
    proxy_server?: string
    proxy_username?: string
    proxy_password?: string
    clear_on_close?: boolean
}

export interface UpdateBrowserProfileRequest extends Partial<CreateBrowserProfileRequest> {
    status?: 'active' | 'inactive' | 'archived' | 'running'
}

export interface ListBrowserProfilesParams {
    status?: string
    folder?: string
    limit?: number
    offset?: number
}

export interface BrowserType {
    type: string
    name: string
    description: string
    icon: string
}

export interface Fingerprint {
    UserAgent: string
    Platform: string
    ScreenWidth: number
    ScreenHeight: number
    Timezone: string
    Locale: string
    Languages: string[]
    WebGLVendor: string
    WebGLRenderer: string
    HardwareConcurrency: number
    DeviceMemory: number
    Fonts: string[]
}

export const browserProfilesApi = {
    // List all profiles
    list(params?: ListBrowserProfilesParams) {
        return apiClient.get<{ count: number; profiles: BrowserProfile[] }>('/profiles', { params })
    },

    // Get profile by ID
    getById(id: string) {
        return apiClient.get<BrowserProfile>(`/profiles/${id}`)
    },

    // Create new profile
    create(data: CreateBrowserProfileRequest) {
        return apiClient.post<BrowserProfile>('/profiles', data)
    },

    // Update profile
    update(id: string, data: UpdateBrowserProfileRequest) {
        return apiClient.put<BrowserProfile>(`/profiles/${id}`, data)
    },

    // Delete profile
    delete(id: string) {
        return apiClient.delete(`/profiles/${id}`)
    },

    // Duplicate profile
    duplicate(id: string) {
        return apiClient.post<BrowserProfile>(`/profiles/${id}/duplicate`)
    },

    // Generate random fingerprint
    generateFingerprint(browserType: 'chromium' | 'firefox' | 'webkit') {
        return apiClient.post<Fingerprint>('/profiles/generate-fingerprint', { browser_type: browserType })
    },

    // Get available browser types
    getBrowserTypes() {
        return apiClient.get<{ browser_types: BrowserType[] }>('/profiles/browser-types')
    },

    // Get all folders
    getFolders() {
        return apiClient.get<{ folders: string[] }>('/profiles/folders')
    },

    // Test browser configuration
    testBrowserConfig(data: { browser_type: string; executable_path?: string; cdp_endpoint?: string }) {
        return apiClient.post<{ success: boolean; message: string; error?: string }>('/profiles/test-browser-config', data)
    },

    // Test profile (launch and close)
    testProfile(id: string) {
        return apiClient.post<{ success: boolean; message: string; error?: string }>(`/profiles/${id}/test`)
    },

    // Launch profile (standalone)
    launchProfile(id: string, headless: boolean = false) {
        return apiClient.post<{ success: boolean; message: string; id: string }>(`/profiles/${id}/launch`, null, {
            params: { headless }
        })
    },

    // Stop profile
    stopProfile(id: string) {
        return apiClient.post<{ success: boolean; message: string }>(`/profiles/${id}/stop`)
    }
}
