import apiClient from './client'

// Types
export interface ScrapeRequest {
    url: string
    driver: string
    profile_id?: string
    output_format: string
    timeout?: number
    wait_for_selector?: string
}

export interface ScrapeResponse {
    scrape_id: string
    status: string
    url: string
}

export interface ScrapeResult {
    id: string
    status: 'pending' | 'running' | 'completed' | 'failed'
    content?: string
    screenshot?: string
    content_type: string
    duration_ms: number
    error?: string
    url: string
}

export interface Driver {
    id: string
    name: string
    description: string
}

export interface Profile {
    id: string
    name: string
}

// API Functions
export async function submitScrape(request: ScrapeRequest): Promise<ScrapeResponse> {
    const response = await apiClient.post<ScrapeResponse>('/scrape', request)
    return response.data
}

export async function getScrapeResult(scrapeId: string): Promise<ScrapeResult> {
    const response = await apiClient.get<ScrapeResult>(`/scrape/${scrapeId}`)
    return response.data
}

export async function getDrivers(): Promise<Driver[]> {
    const response = await apiClient.get<{ drivers: Driver[] }>('/scrape/drivers')
    return response.data.drivers
}

export async function getProfiles(): Promise<Profile[]> {
    const response = await apiClient.get<{ profiles: Profile[] }>('/scrape/profiles')
    return response.data.profiles
}

// Polling helper for result
export async function pollScrapeResult(
    scrapeId: string,
    onUpdate: (result: ScrapeResult) => void,
    maxAttempts = 60,
    intervalMs = 1000
): Promise<ScrapeResult> {
    let attempts = 0

    while (attempts < maxAttempts) {
        const result = await getScrapeResult(scrapeId)
        onUpdate(result)

        if (result.status === 'completed' || result.status === 'failed') {
            return result
        }

        await new Promise(resolve => setTimeout(resolve, intervalMs))
        attempts++
    }

    throw new Error('Scrape timed out')
}
