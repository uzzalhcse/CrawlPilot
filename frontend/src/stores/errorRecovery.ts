import { defineStore } from 'pinia'
import axios from 'axios'
import { toast } from 'vue-sonner'

const API_BASE = 'http://localhost:8080/api/v1'

export interface Condition {
    field: string
    operator: string
    value: any
}

export interface Action {
    type: string
    parameters: Record<string, any>
    condition?: Condition
}

export interface RuleContext {
    domain_pattern: string
    variables: Record<string, any>
    max_retries: number
    timeout_multiplier?: number
}

// Recovery Rule - matches backend schema
export interface ContextAwareRule {
    id: string
    name: string
    description: string
    priority: number
    enabled: boolean
    pattern: string
    conditions: Record<string, any>
    action: string
    action_params: Record<string, any>
    max_retries: number
    retry_delay: number
    is_learned: boolean
    learned_from?: string
    success_count: number
    failure_count: number
    created_at: string
    updated_at: string
}

export interface ErrorRecoveryConfig {
    key: string
    value: any
}

export const useErrorRecoveryStore = defineStore('errorRecovery', {
    state: () => ({
        rules: [] as ContextAwareRule[],
        config: {} as Record<string, any>,
        loading: false,
        selectedRule: null as ContextAwareRule | null,
    }),

    actions: {
        async fetchRules() {
            this.loading = true
            try {
                const response = await axios.get(`${API_BASE}/recovery/rules`)
                this.rules = response.data?.rules || []
            } catch (error: any) {
                toast.error('Failed to fetch rules', {
                    description: error.response?.data?.error || error.message
                })
                throw error
            } finally {
                this.loading = false
            }
        },

        async createRule(rule: Partial<ContextAwareRule>) {
            this.loading = true
            try {
                const response = await axios.post(`${API_BASE}/recovery/rules`, rule)
                this.rules.push(response.data)
                toast.success('Rule created successfully')
                return response.data
            } catch (error: any) {
                toast.error('Failed to create rule', {
                    description: error.response?.data?.error || error.message
                })
                throw error
            } finally {
                this.loading = false
            }
        },

        async updateRule(id: string, rule: Partial<ContextAwareRule>) {
            this.loading = true
            try {
                const response = await axios.put(`${API_BASE}/recovery/rules/${id}`, rule)
                const index = this.rules.findIndex(r => r.id === id)
                if (index !== -1) {
                    this.rules[index] = response.data
                }
                toast.success('Rule updated successfully')
                return response.data
            } catch (error: any) {
                toast.error('Failed to update rule', {
                    description: error.response?.data?.error || error.message
                })
                throw error
            } finally {
                this.loading = false
            }
        },

        async deleteRule(id: string) {
            this.loading = true
            try {
                await axios.delete(`${API_BASE}/recovery/rules/${id}`)
                this.rules = this.rules.filter(r => r.id !== id)
                toast.success('Rule deleted successfully')
            } catch (error: any) {
                toast.error('Failed to delete rule', {
                    description: error.response?.data?.error || error.message
                })
                throw error
            } finally {
                this.loading = false
            }
        },

        async fetchAllConfigs() {
            try {
                const response = await axios.get(`${API_BASE}/recovery/config`)
                // Convert array of {key, value} to object
                const configMap: Record<string, any> = {}
                if (response.data?.configs) {
                    response.data.configs.forEach((c: any) => {
                        configMap[c.key] = c.value
                    })
                }
                this.config = configMap
                return configMap
            } catch (error: any) {
                toast.error('Failed to fetch configurations', {
                    description: error.response?.data?.error || error.message
                })
                throw error
            }
        },

        async fetchConfig(key: string) {
            try {
                const response = await axios.get(`${API_BASE}/recovery/config/${key}`)
                this.config[key] = response.data.value
                return response.data.value
            } catch (error: any) {
                if (error.response?.status !== 404) {
                    toast.error('Failed to fetch config', {
                        description: error.response?.data?.error || error.message
                    })
                }
                throw error
            }
        },

        async updateConfig(key: string, value: any) {
            try {
                await axios.put(`${API_BASE}/recovery/config/item/${key}`, { value })
                this.config[key] = value
                toast.success('Configuration updated successfully')
            } catch (error: any) {
                toast.error('Failed to update configuration', {
                    description: error.response?.data?.error || error.message
                })
                throw error
            }
        },

        async updateMultipleConfigs(updates: Record<string, any>) {
            try {
                await axios.put(`${API_BASE}/recovery/config`, { updates })
                // Update local state
                Object.keys(updates).forEach(key => {
                    this.config[key] = updates[key]
                })
                toast.success('Settings saved successfully')
            } catch (error: any) {
                toast.error('Failed to save settings', {
                    description: error.response?.data?.error || error.message
                })
                throw error
            }
        },

        selectRule(rule: ContextAwareRule | null) {
            this.selectedRule = rule
        },
    },

    getters: {
        rulesSortedByPriority: (state) => {
            return [...state.rules].sort((a, b) => b.priority - a.priority)
        },

        predefinedRules: (state) => {
            return state.rules.filter(r => !r.is_learned)
        },

        learnedRules: (state) => {
            return state.rules.filter(r => r.is_learned)
        },

        customRules: (state) => {
            // Custom rules are non-learned rules (manually created)
            return state.rules.filter(r => !r.is_learned)
        },
    },
})
