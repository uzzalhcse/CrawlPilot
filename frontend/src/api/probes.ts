import apiClient from './client'

// Types
export interface ProbeResult {
    id: string
    execution_id: string
    workflow_id: string
    status: 'healthy' | 'degraded' | 'broken'
    duration_ms: number
    phases: PhaseProbeResult[]
    created_at: string
}

export interface PhaseProbeResult {
    phase_id: string
    phase_name: string
    status: 'passed' | 'failed' | 'skipped'
    duration_ms: number
    sample_url: string
    nodes: NodeProbeResult[]
}

export interface NodeProbeResult {
    node_id: string
    node_name: string
    node_type: string
    status: 'passed' | 'failed' | 'skipped'
    duration_ms: number
    element_count: number
    expected_element_count?: number
    links_found: number
    expected_links_found?: number
    selector?: string
    selector_context?: string  // HTML around the selector for debugging
    error?: string
    snapshot?: ProbeSnapshot   // Full snapshot object, not just ID
    deviation?: string
    fields?: FieldProbeResult[]  // Field-level results for extract nodes
}

export interface ProbeSnapshot {
    dom_path: string
    screenshot_path: string
    page_url: string
    page_title: string
    captured_at: number
    dom_size: number
    image_width: number
    image_height: number
}

export interface FieldProbeResult {
    name: string
    selector?: string
    status: 'ok' | 'empty' | 'error'
    value?: string
    error?: string
}

export interface ProbeStats {
    total: number
    healthy: number
    degraded: number
    broken: number
    by_workflow: Record<string, number>
}

export interface AutoFix {
    id: string
    workflow_id: string
    node_id: string
    fix_type: 'update_selector' | 'update_field_selector' | 'skip_node'
    old_selector?: string
    new_selector?: string
    reasoning: string
    confidence: number
    status: 'applied' | 'pending' | 'rejected'
    auto_applied: boolean
    created_at: string
}

// API Functions
export const probesApi = {
    // Get probe results for a workflow
    getProbeResults(workflowId: string, limit: number = 10) {
        return apiClient.get<{ results: ProbeResult[]; count: number }>(
            `/workflows/${workflowId}/probe/results`,
            { params: { limit } }
        )
    },

    // Get latest probe result for a workflow
    getLatestProbeResult(workflowId: string) {
        return apiClient.get<ProbeResult>(`/workflows/${workflowId}/probe/latest`)
    },

    // Get sample URLs for a workflow
    getSampleURLs(workflowId: string) {
        return apiClient.get<{ workflow_id: string; sample_urls: Record<string, string> }>(
            `/workflows/${workflowId}/probe/sample-urls`
        )
    },

    // Run a probe check for a workflow
    runProbe(workflowId: string) {
        return apiClient.post<{ execution_id: string; workflow_id: string; status: string }>(
            `/workflows/${workflowId}/probe`
        )
    },

    // Get all recent probes across workflows
    getRecentProbes(limit: number = 20) {
        return apiClient.get<{ probes: ProbeResult[]; total: number }>(
            '/probes/recent',
            { params: { limit } }
        )
    },

    // Get probe statistics
    getProbeStats() {
        return apiClient.get<ProbeStats>('/probes/stats')
    },

    // Get auto-fixes list
    getAutoFixes(params?: { status?: string; limit?: number }) {
        return apiClient.get<{ fixes: AutoFix[]; total: number }>(
            '/probes/auto-fixes',
            { params }
        )
    },

    // Approve an auto-fix
    approveAutoFix(fixId: string) {
        return apiClient.post<{ success: boolean }>(`/probes/auto-fixes/${fixId}/approve`)
    },

    // Reject an auto-fix
    rejectAutoFix(fixId: string) {
        return apiClient.post<{ success: boolean }>(`/probes/auto-fixes/${fixId}/reject`)
    },

    // Get snapshot details
    getSnapshot(snapshotId: string) {
        return apiClient.get<{ id: string; screenshot_url: string; dom_content: string }>(
            `/snapshots/${snapshotId}`
        )
    },

    // Get screenshot URL
    getScreenshotUrl(snapshotId: string) {
        return `/api/v1/snapshots/${snapshotId}/screenshot`
    },

    // Get DOM URL
    getDOMUrl(snapshotId: string) {
        return `/api/v1/snapshots/${snapshotId}/dom`
    }
}
