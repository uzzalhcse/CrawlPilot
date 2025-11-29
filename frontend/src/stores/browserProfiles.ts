import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { browserProfilesApi } from '@/api/browserProfiles'
import type { BrowserProfile, CreateBrowserProfileRequest, UpdateBrowserProfileRequest } from '@/api/browserProfiles'

export const useBrowserProfilesStore = defineStore('browserProfiles', () => {
    const profiles = ref<BrowserProfile[]>([])
    const currentProfile = ref<BrowserProfile | null>(null)
    const loading = ref(false)
    const error = ref<string | null>(null)

    // Computed
    const activeProfiles = computed(() =>
        profiles.value.filter(p => p.status === 'active')
    )

    const inactiveProfiles = computed(() =>
        profiles.value.filter(p => p.status === 'inactive')
    )

    const profilesByType = computed(() => {
        return {
            chromium: profiles.value.filter(p => p.browser_type === 'chromium'),
            firefox: profiles.value.filter(p => p.browser_type === 'firefox'),
            webkit: profiles.value.filter(p => p.browser_type === 'webkit')
        }
    })

    const recentlyUsedProfiles = computed(() =>
        [...profiles.value]
            .filter(p => p.last_used_at)
            .sort((a, b) => {
                const dateA = a.last_used_at ? new Date(a.last_used_at).getTime() : 0
                const dateB = b.last_used_at ? new Date(b.last_used_at).getTime() : 0
                return dateB - dateA
            })
            .slice(0, 10)
    )

    // Actions
    async function fetchProfiles(params?: { status?: string; folder?: string; limit?: number; offset?: number }) {
        loading.value = true
        error.value = null
        try {
            const response = await browserProfilesApi.list(params)
            profiles.value = response.data.profiles || []
        } catch (e: any) {
            error.value = e.response?.data?.error || 'Failed to fetch browser profiles'
            throw e
        } finally {
            loading.value = false
        }
    }

    async function fetchProfileById(id: string) {
        loading.value = true
        error.value = null
        try {
            const response = await browserProfilesApi.getById(id)
            currentProfile.value = response.data
            return response.data
        } catch (e: any) {
            error.value = e.response?.data?.error || 'Failed to fetch profile'
            throw e
        } finally {
            loading.value = false
        }
    }

    async function createProfile(data: CreateBrowserProfileRequest) {
        loading.value = true
        error.value = null
        try {
            const response = await browserProfilesApi.create(data)
            profiles.value.unshift(response.data)
            return response.data
        } catch (e: any) {
            error.value = e.response?.data?.error || 'Failed to create profile'
            throw e
        } finally {
            loading.value = false
        }
    }

    async function updateProfile(id: string, data: UpdateBrowserProfileRequest) {
        loading.value = true
        error.value = null
        try {
            const response = await browserProfilesApi.update(id, data)
            const index = profiles.value.findIndex(p => p.id === id)
            if (index !== -1) {
                profiles.value[index] = response.data
            }
            if (currentProfile.value?.id === id) {
                currentProfile.value = response.data
            }
            return response.data
        } catch (e: any) {
            error.value = e.response?.data?.error || 'Failed to update profile'
            throw e
        } finally {
            loading.value = false
        }
    }

    async function deleteProfile(id: string) {
        loading.value = true
        error.value = null
        try {
            await browserProfilesApi.delete(id)
            profiles.value = profiles.value.filter(p => p.id !== id)
            if (currentProfile.value?.id === id) {
                currentProfile.value = null
            }
        } catch (e: any) {
            error.value = e.response?.data?.error || 'Failed to delete profile'
            throw e
        } finally {
            loading.value = false
        }
    }

    async function duplicateProfile(id: string) {
        loading.value = true
        error.value = null
        try {
            const response = await browserProfilesApi.duplicate(id)
            profiles.value.unshift(response.data)
            return response.data
        } catch (e: any) {
            error.value = e.response?.data?.error || 'Failed to duplicate profile'
            throw e
        } finally {
            loading.value = false
        }
    }

    async function testProfile(id: string) {
        loading.value = true
        error.value = null
        try {
            const response = await browserProfilesApi.testProfile(id)
            return response.data
        } catch (e: any) {
            error.value = e.response?.data?.error || 'Failed to test profile'
            throw e
        } finally {
            loading.value = false
        }
    }

    async function launchProfile(id: string, headless: boolean = false) {
        loading.value = true
        error.value = null
        try {
            const response = await browserProfilesApi.launchProfile(id, headless)
            return response.data
        } catch (e: any) {
            error.value = e.response?.data?.error || 'Failed to launch profile'
            throw e
        } finally {
            loading.value = false
        }
    }

    async function stopProfile(id: string) {
        loading.value = true
        error.value = null
        try {
            const response = await browserProfilesApi.stopProfile(id)
            return response.data
        } catch (e: any) {
            error.value = e.response?.data?.error || 'Failed to stop profile'
            throw e
        } finally {
            loading.value = false
        }
    }

    async function generateFingerprint(browserType: 'chromium' | 'firefox' | 'webkit') {
        loading.value = true
        error.value = null
        try {
            const response = await browserProfilesApi.generateFingerprint(browserType)
            return response.data
        } catch (e: any) {
            error.value = e.response?.data?.error || 'Failed to generate fingerprint'
            throw e
        } finally {
            loading.value = false
        }
    }

    function clearError() {
        error.value = null
    }

    return {
        profiles,
        currentProfile,
        loading,
        error,
        activeProfiles,
        inactiveProfiles,
        profilesByType,
        recentlyUsedProfiles,
        fetchProfiles,
        fetchProfileById,
        createProfile,
        updateProfile,
        deleteProfile,
        duplicateProfile,
        testProfile,
        launchProfile,
        stopProfile,
        generateFingerprint,
        clearError
    }
})
