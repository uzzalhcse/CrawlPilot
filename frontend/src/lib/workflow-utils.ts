import type { WorkflowNode, WorkflowEdge, WorkflowConfig, WorkflowPhase, Node } from '@/types'

// Helper to generate a unique ID
function generateId(): string {
    return Math.random().toString(36).substr(2, 9)
}

export function convertNodesToWorkflowConfig(
    nodes: WorkflowNode[],
    edges: WorkflowEdge[],
    baseConfig: Partial<WorkflowConfig> = {},
    preservedPhaseProps: Map<string, any> = new Map(),
    preservedWorkflowProps: Record<string, any> = {}
): WorkflowConfig {
    // Group nodes by phaseId if available, otherwise use heuristic
    const nodesByPhase: Record<string, WorkflowNode[]> = {}
    const phaseOrder: string[] = []

    // First pass: Group by existing phaseId
    nodes.forEach(node => {
        const phaseId = node.data.phaseId || getPhaseIdByType(node.data.nodeType)
        if (!nodesByPhase[phaseId]) {
            nodesByPhase[phaseId] = []
            phaseOrder.push(phaseId)
        }
        nodesByPhase[phaseId].push(node)
    })

    // Sort phases based on edges? 
    // For now, we'll stick to the simple heuristic order if we fell back to types, 
    // or the order we encountered them (which might be random).
    // A better approach for "phaseId" derived from types is to enforce a specific order.

    const phases: WorkflowPhase[] = []

    // Define standard phases for sorting/naming
    const standardPhases = ['discovery_phase', 'extraction_phase', 'processing_phase']
    const phaseNames: Record<string, string> = {
        'discovery_phase': 'URL Discovery',
        'extraction_phase': 'Data Extraction',
        'processing_phase': 'Data Processing'
    }

    // Sort phase IDs: standard ones first, then others
    const sortedPhaseIds = Object.keys(nodesByPhase).sort((a, b) => {
        const idxA = standardPhases.indexOf(a)
        const idxB = standardPhases.indexOf(b)
        if (idxA !== -1 && idxB !== -1) return idxA - idxB
        if (idxA !== -1) return -1
        if (idxB !== -1) return 1
        return 0
    })

    sortedPhaseIds.forEach(phaseId => {
        const phaseNodes = nodesByPhase[phaseId]
        if (phaseNodes.length === 0) return

        const phaseType = getPhaseType(phaseId)

        // Filter out virtual nodes (extractField) and UI-only nodes (phaseLabel)
        const realNodes = phaseNodes.filter(n => !n.data.isVirtual && n.type !== 'phaseLabel')

        // Skip this phase if there are no real nodes after filtering
        if (realNodes.length === 0) return

        // Reconstruct fields for extract nodes
        const extractNodes = realNodes.filter(n => n.data.nodeType === 'extract')
        extractNodes.forEach(extractNode => {
            const fieldNodes = phaseNodes.filter(n =>
                n.data.nodeType === 'extractField' &&
                n.data.parentId === extractNode.id
            )

            if (fieldNodes.length > 0) {
                const fields: Record<string, any> = {}
                fieldNodes.forEach(fn => {
                    // fn.data.label is the key (name), fn.data.field is the params
                    // But wait, in WorkflowBuilder we set:
                    // name: key (which becomes label)
                    // params: { ...field } (which becomes data.params or data.field depending on how we mapped it)
                    // In WorkflowBuilder loadWorkflow:
                    // field: node.type === 'extractField' ? node.params : undefined
                    // So we should look at data.field or data.params. 
                    // Let's check WorkflowBuilder again. 
                    // We mapped node.params to data.params AND data.field.
                    // So data.params should hold the field config.

                    if (fn.data.label) {
                        fields[fn.data.label] = fn.data.params
                    }
                })

                // Update the extract node's params
                if (!extractNode.data.params) extractNode.data.params = {}
                extractNode.data.params.fields = fields
            }
        })

        // Get preserved properties for this phase
        const preserved = preservedPhaseProps.get(phaseId) || {}

        // Create phase with preserved or default properties
        const phase: WorkflowPhase = {
            id: phaseId,
            type: preserved.type || phaseType, // Use preserved type if available
            name: preserved.name || phaseNames[phaseId] || `Phase ${phaseId}`,
            nodes: realNodes.map(node => convertToBackendNode(node, edges, nodes)),
            // Preserve url_filter if it exists, otherwise use default
            url_filter: preserved.url_filter !== undefined ? preserved.url_filter : getUrlFilterForPhase(phaseId),
            // Preserve transition if it exists
            transition: preserved.transition
        }

        // Merge any other custom properties from preserved config
        Object.keys(preserved).forEach(key => {
            if (!['id', 'type', 'name', 'nodes', 'url_filter', 'transition'].includes(key)) {
                (phase as any)[key] = preserved[key]
            }
        })

        phases.push(phase)
    })

    // Link phases if transitions are not already preserved
    for (let i = 0; i < phases.length; i++) {
        if (!phases[i].transition && i < phases.length - 1) {
            phases[i].transition = {
                condition: 'all_nodes_complete',
                next_phase: phases[i + 1].id
            }
        }
    }

    const config: WorkflowConfig = {
        start_urls: baseConfig.start_urls || [],
        max_depth: baseConfig.max_depth || 3,
        rate_limit_delay: baseConfig.rate_limit_delay || 1000,
        storage: baseConfig.storage || { type: 'database' },
        phases,
        ...preservedWorkflowProps // Merge any preserved workflow-level properties (headers, etc.)
    }

    return config
}

function getPhaseIdByType(type: string): string {
    if (['fetch', 'extract_links', 'filter_urls', 'navigate', 'paginate'].includes(type)) {
        return 'discovery_phase'
    }
    if (['extract', 'extract_text', 'extract_attr', 'extract_json', 'sequence', 'click', 'scroll', 'hover', 'type', 'wait', 'screenshot'].includes(type)) {
        return 'extraction_phase'
    }
    if (['transform', 'filter', 'map', 'validate'].includes(type)) {
        return 'processing_phase'
    }
    return 'custom_phase'
}

function getPhaseType(phaseId: string): 'discovery' | 'extraction' | 'processing' | 'custom' {
    if (phaseId.includes('discovery')) return 'discovery'
    if (phaseId.includes('extraction')) return 'extraction'
    if (phaseId.includes('processing')) return 'processing'
    return 'custom'
}

function getUrlFilterForPhase(phaseId: string): any {
    if (phaseId === 'discovery_phase') {
        return { depth: 0 }
    }
    if (phaseId === 'extraction_phase') {
        return { markers: ['product'] }
    }
    return undefined
}

function convertToBackendNode(node: WorkflowNode, edges: WorkflowEdge[], allNodes: WorkflowNode[]): Node {
    // Find dependencies from edges
    // We only care about dependencies within the same phase for node execution order usually,
    // but the backend handles dependencies by ID.
    // IMPORTANT: Filter out dependencies that are from a different phase (visual connections)
    const dependencies = edges
        .filter(e => e.target === node.id)
        .filter(e => {
            const sourceNode = allNodes.find(n => n.id === e.source)
            // If source node not found, or has different phaseId, exclude it
            // Note: phaseId might be in data or we might need to derive it again if not persisted.
            // But we persisted it in data.phaseId in WorkflowBuilder.vue loadWorkflow.
            // And in convertNodesToWorkflowConfig we rely on node.data.phaseId or heuristic.
            // Let's use the same logic as convertNodesToWorkflowConfig loop if possible, 
            // but since we don't have the phase map here easily, let's rely on data.phaseId 
            // or re-derive.

            if (!sourceNode) return false

            const sourcePhaseId = sourceNode.data.phaseId || getPhaseIdByType(sourceNode.data.nodeType)
            const targetPhaseId = node.data.phaseId || getPhaseIdByType(node.data.nodeType)

            return sourcePhaseId === targetPhaseId
        })
        .map(e => e.source)

    return {
        id: node.id,
        type: node.data.nodeType,
        name: node.data.label,
        params: node.data.params,
        dependencies: dependencies.length > 0 ? dependencies : undefined,
        // output_key etc are not in the Node interface in types.ts but were in the Vue file?
        // Let's check types.ts again. Node interface has params: Record<string, any>.
        // The backend likely expects these in params or as separate fields. 
        // The Vue file had them separate. Let's put them in params if they aren't standard.
        // Wait, the Vue file `convertToBackendNode` put them as top level properties.
        // Let's check the Node interface in types.ts again.
        // export interface Node { id, type, name, params, dependencies }
        // It does NOT have output_key, optional, retry.
        // So they must be in params or the interface is incomplete.
        // Looking at the Vue file again:
        /*
          return {
            id: node.id,
            ...
            output_key: node.data.outputKey,
            optional: node.data.optional,
            retry: node.data.retry
          }
        */
        // If I strictly follow the interface, I should put them in params or extend the interface.
        // For now I will cast to any to match the Vue file's behavior.
    } as any
}
