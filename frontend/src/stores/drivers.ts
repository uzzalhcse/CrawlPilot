import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { getDrivers, type Driver } from '@/api/scraper'

export const useDriversStore = defineStore('drivers', () => {
    const drivers = ref<Driver[]>([])
    const loading = ref(false)
    const error = ref<string | null>(null)
    const fetched = ref(false)

    // Computed: Drivers formatted for select dropdowns
    const driverOptions = computed(() =>
        drivers.value.map(d => ({
            value: d.id,
            label: d.name
        }))
    )

    // Computed: Browser drivers only (exclude http)
    const browserDrivers = computed(() =>
        drivers.value.filter(d => d.id !== 'http')
    )

    // Computed: For workflow default driver selection
    const workflowDriverOptions = computed(() =>
        drivers.value.map(d => ({
            value: d.id,
            label: d.name,
            description: d.description
        }))
    )

    // Actions
    async function fetchDrivers() {
        // Skip if already fetched
        if (fetched.value && drivers.value.length > 0) {
            return drivers.value
        }

        loading.value = true
        error.value = null
        try {
            const result = await getDrivers()
            drivers.value = result
            fetched.value = true
            return result
        } catch (e: any) {
            error.value = e.response?.data?.error || 'Failed to fetch drivers'
            // Fallback to defaults if API fails
            drivers.value = [
                { id: 'http', name: 'HTTP Client (Fast)', description: 'Fast HTTP requests without JavaScript' },
                { id: 'playwright', name: 'Playwright', description: 'Full browser automation' },
                { id: 'camoufox', name: 'Camoufox (Stealth)', description: 'Anti-detection browser' }
            ]
            fetched.value = true
            return drivers.value
        } finally {
            loading.value = false
        }
    }

    function getDriverById(id: string): Driver | undefined {
        return drivers.value.find(d => d.id === id)
    }

    function getDriverName(id: string): string {
        const driver = getDriverById(id)
        return driver?.name || id
    }

    return {
        drivers,
        loading,
        error,
        fetched,
        driverOptions,
        browserDrivers,
        workflowDriverOptions,
        fetchDrivers,
        getDriverById,
        getDriverName
    }
})
