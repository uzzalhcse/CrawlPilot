import type {
    Plugin,
    PluginVersion,
    PluginInstallation,
    PluginReview,
    PluginCategory,
    PluginFilters,
    PluginSearchResult
} from '@/types'

const API_BASE = '/api/v1'

class PluginAPI {
    // Plugin CRUD
    async listPlugins(filters?: PluginFilters): Promise<Plugin[]> {
        const params = new URLSearchParams()
        if (filters) {
            if (filters.q) params.append('q', filters.q)
            if (filters.category) params.append('category', filters.category)
            if (filters.phase_type) params.append('phase_type', filters.phase_type)
            if (filters.plugin_type) params.append('plugin_type', filters.plugin_type)
            if (filters.verified !== undefined) params.append('verified', String(filters.verified))
            if (filters.sort_by) params.append('sort_by', filters.sort_by)
            if (filters.sort_order) params.append('sort_order', filters.sort_order)
            if (filters.limit) params.append('limit', String(filters.limit))
            if (filters.offset) params.append('offset', String(filters.offset))
        }

        const url = `${API_BASE}/plugins${params.toString() ? `?${params.toString()}` : ''}`
        const response = await fetch(url)
        if (!response.ok) throw new Error('Failed to fetch plugins')
        const data = await response.json()
        return data || [] // Return empty array if null
    }

    async getPlugin(slug: string): Promise<Plugin> {
        const response = await fetch(`${API_BASE}/plugins/${slug}`)
        if (!response.ok) throw new Error('Failed to fetch plugin')
        return response.json()
    }

    async createPlugin(plugin: Partial<Plugin>): Promise<Plugin> {
        const response = await fetch(`${API_BASE}/plugins`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(plugin)
        })
        if (!response.ok) throw new Error('Failed to create plugin')
        return response.json()
    }

    async updatePlugin(id: string, plugin: Partial<Plugin>): Promise<Plugin> {
        const response = await fetch(`${API_BASE}/plugins/${id}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(plugin)
        })
        if (!response.ok) throw new Error('Failed to update plugin')
        return response.json()
    }

    async deletePlugin(id: string): Promise<void> {
        const response = await fetch(`${API_BASE}/plugins/${id}`, {
            method: 'DELETE'
        })
        if (!response.ok) throw new Error('Failed to delete plugin')
    }

    // Version Management
    async publishVersion(pluginId: string, version: Partial<PluginVersion>): Promise<PluginVersion> {
        const response = await fetch(`${API_BASE}/plugins/${pluginId}/versions`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(version)
        })
        if (!response.ok) throw new Error('Failed to publish version')
        return response.json()
    }

    async listVersions(pluginId: string): Promise<PluginVersion[]> {
        const response = await fetch(`${API_BASE}/plugins/${pluginId}/versions`)
        if (!response.ok) throw new Error('Failed to fetch versions')
        return response.json()
    }

    async getVersion(pluginId: string, versionId: string): Promise<PluginVersion> {
        const response = await fetch(`${API_BASE}/plugins/${pluginId}/versions/${versionId}`)
        if (!response.ok) throw new Error('Failed to fetch version')
        return response.json()
    }

    // Installation
    async installPlugin(pluginId: string, workspaceId?: string): Promise<PluginInstallation> {
        const headers: HeadersInit = { 'Content-Type': 'application/json' }
        if (workspaceId) {
            headers['X-Workspace-ID'] = workspaceId
        }

        const response = await fetch(`${API_BASE}/plugins/${pluginId}/install`, {
            method: 'POST',
            headers
        })
        if (!response.ok) throw new Error('Failed to install plugin')
        return response.json()
    }

    async uninstallPlugin(pluginId: string, workspaceId?: string): Promise<void> {
        const headers: HeadersInit = {}
        if (workspaceId) {
            headers['X-Workspace-ID'] = workspaceId
        }

        const response = await fetch(`${API_BASE}/plugins/${pluginId}/uninstall`, {
            method: 'POST',
            headers
        })
        if (!response.ok) throw new Error('Failed to uninstall plugin')
    }

    async listInstalledPlugins(workspaceId?: string): Promise<Plugin[]> {
        const headers: HeadersInit = {}
        if (workspaceId) {
            headers['X-Workspace-ID'] = workspaceId
        }

        const response = await fetch(`${API_BASE}/plugins/installed`, { headers })
        if (!response.ok) throw new Error('Failed to fetch installed plugins')
        return response.json()
    }

    // Reviews
    async createReview(pluginId: string, review: Partial<PluginReview>): Promise<PluginReview> {
        const response = await fetch(`${API_BASE}/plugins/${pluginId}/reviews`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(review)
        })
        if (!response.ok) throw new Error('Failed to create review')
        return response.json()
    }

    async listReviews(pluginId: string, limit = 20, offset = 0): Promise<PluginReview[]> {
        const response = await fetch(
            `${API_BASE}/plugins/${pluginId}/reviews?limit=${limit}&offset=${offset}`
        )
        if (!response.ok) throw new Error('Failed to fetch reviews')
        return response.json()
    }

    // Discovery & Search
    async getCategories(): Promise<PluginCategory[]> {
        const response = await fetch(`${API_BASE}/plugins/categories`)
        if (!response.ok) throw new Error('Failed to fetch categories')
        return response.json()
    }

    async searchPlugins(query: string, limit = 20): Promise<Plugin[]> {
        const response = await fetch(
            `${API_BASE}/plugins/search?q=${encodeURIComponent(query)}&limit=${limit}`
        )
        if (!response.ok) throw new Error('Failed to search plugins')
        return response.json()
    }

    async getPopularPlugins(limit = 10): Promise<Plugin[]> {
        const response = await fetch(`${API_BASE}/plugins/popular?limit=${limit}`)
        if (!response.ok) throw new Error('Failed to fetch popular plugins')
        return response.json()
    }

    // Helper: Check if plugin is installed
    async isPluginInstalled(pluginId: string, workspaceId?: string): Promise<boolean> {
        try {
            const installedPlugins = await this.listInstalledPlugins(workspaceId)
            return installedPlugins.some(p => p.id === pluginId)
        } catch {
            return false
        }
    }

    // Code Management
    async getPluginSource(pluginId: string): Promise<{ files: Record<string, string> }> {
        const response = await fetch(`${API_BASE}/plugins/${pluginId}/code`)
        if (!response.ok) throw new Error('Failed to fetch plugin source')
        return response.json()
    }

    async updatePluginSource(pluginId: string, files: Record<string, string>): Promise<void> {
        const response = await fetch(`${API_BASE}/plugins/${pluginId}/code`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ files })
        })
        if (!response.ok) throw new Error('Failed to update plugin source')
    }

    async buildPlugin(pluginId: string): Promise<{ build_id: string; message: string }> {
        const response = await fetch(`${API_BASE}/plugins/${pluginId}/build`, {
            method: 'POST'
        })
        if (!response.ok) throw new Error('Failed to trigger build')
        return response.json()
    }

    async getBuildStatus(buildId: string): Promise<any> {
        const response = await fetch(`${API_BASE}/builds/${buildId}/status`)
        if (!response.ok) throw new Error('Failed to fetch build status')
        return response.json()
    }

    async scaffoldPlugin(config: {
        name: string
        slug: string
        description: string
        phase_type: string
        author_name: string
        author_email: string
        category?: string
        tags?: string[]
    }): Promise<Plugin> {
        const response = await fetch(`${API_BASE}/plugins/scaffold`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(config)
        })
        if (!response.ok) throw new Error('Failed to scaffold plugin')
        return response.json()
    }

    async getPluginReadme(pluginId: string): Promise<{ content: string }> {
        const response = await fetch(`${API_BASE}/plugins/${pluginId}/readme`)
        if (!response.ok) throw new Error('Failed to fetch plugin readme')
        return response.json()
    }
}

export const pluginAPI = new PluginAPI()
export default pluginAPI
