import { ref, onUnmounted } from 'vue'
import { useExecutionsStore } from '@/stores/executions'

export interface ExecutionEvent {
    type: string
    execution_id: string
    timestamp: string
    data: any
}

export interface LogEntry {
    id: string
    timestamp: string
    message: string
    level: 'info' | 'warn' | 'error' | 'debug'
    metadata?: any
}

export function useExecutionStream(executionId: string) {
    const executionsStore = useExecutionsStore()
    const isConnected = ref(false)
    const logs = ref<LogEntry[]>([])
    const currentPhase = ref<string | null>(null)
    const activeNodes = ref<Set<string>>(new Set())

    let eventSource: EventSource | null = null

    const connect = () => {
        if (eventSource) return

        const url = `${import.meta.env.VITE_API_URL || 'http://localhost:8080'}/api/v1/executions/${executionId}/stream`
        eventSource = new EventSource(url)

        eventSource.onopen = () => {
            isConnected.value = true
            console.log('SSE Connected')
        }

        eventSource.onerror = (error) => {
            console.error('SSE Error:', error)
            isConnected.value = false
            // EventSource automatically tries to reconnect, but we might want to handle it explicitly
        }

        // Generic event handler
        eventSource.onmessage = (event) => {
            // This catches "message" type events, but we use custom event types
            console.log('SSE Message:', event.data)
        }

        // Specific event listeners
        eventSource.addEventListener('execution_started', (e: MessageEvent) => {
            const event = JSON.parse(e.data)
            const data = event.data
            logs.value.push(createLogEntry('Execution started', 'info', data))
            executionsStore.fetchExecutionById(executionId) // Refresh full state
        })

        eventSource.addEventListener('execution_completed', (e: MessageEvent) => {
            const event = JSON.parse(e.data)
            const data = event.data
            logs.value.push(createLogEntry('Execution completed', 'info', data))
            executionsStore.fetchExecutionById(executionId)
            activeNodes.value.clear()
        })

        eventSource.addEventListener('execution_failed', (e: MessageEvent) => {
            const event = JSON.parse(e.data)
            const data = event.data
            logs.value.push(createLogEntry(`Execution failed: ${data.error}`, 'error', data))
            executionsStore.fetchExecutionById(executionId)
            activeNodes.value.clear()
        })

        eventSource.addEventListener('phase_started', (e: MessageEvent) => {
            const event = JSON.parse(e.data)
            const data = event.data
            currentPhase.value = data.phase_id
            logs.value.push(createLogEntry(`Phase started: ${data.phase_name || data.phase_id}`, 'info', data))
        })

        eventSource.addEventListener('phase_completed', (e: MessageEvent) => {
            const event = JSON.parse(e.data)
            const data = event.data
            logs.value.push(createLogEntry(`Phase completed: ${data.phase_id}`, 'info', data))
        })

        eventSource.addEventListener('phase_failed', (e: MessageEvent) => {
            const event = JSON.parse(e.data)
            const data = event.data
            logs.value.push(createLogEntry(`Phase failed: ${data.error}`, 'error', data))
        })

        eventSource.addEventListener('node_started', (e: MessageEvent) => {
            const event = JSON.parse(e.data)
            const data = event.data
            activeNodes.value.add(data.node_id)
            logs.value.push(createLogEntry(`Node started: ${data.node_name || data.node_id}`, 'debug', data))
        })

        eventSource.addEventListener('node_completed', (e: MessageEvent) => {
            const event = JSON.parse(e.data)
            const data = event.data
            activeNodes.value.delete(data.node_id)
            logs.value.push(createLogEntry(`Node completed: ${data.node_id}`, 'debug', data))
        })

        eventSource.addEventListener('node_failed', (e: MessageEvent) => {
            const event = JSON.parse(e.data)
            const data = event.data
            activeNodes.value.delete(data.node_id)
            logs.value.push(createLogEntry(`Node failed: ${data.error}`, 'warn', data))
        })

        eventSource.addEventListener('url_discovered', (e: MessageEvent) => {
            const event = JSON.parse(e.data)
            const data = event.data
            logs.value.push(createLogEntry(`Discovered URL: ${data.url}`, 'info', data))
        })

        eventSource.addEventListener('item_extracted', (e: MessageEvent) => {
            const event = JSON.parse(e.data)
            const data = event.data
            logs.value.push(createLogEntry(`Item extracted`, 'info', data))
            // Optimistically update stats if possible, or trigger a fetch
        })

        eventSource.addEventListener('stats_updated', (e: MessageEvent) => {
            const event = JSON.parse(e.data)
            const data = event.data
            // Update store stats directly if possible
            if (executionsStore.executionStats) {
                Object.assign(executionsStore.executionStats, data.stats)
            }
        })
    }

    const disconnect = () => {
        if (eventSource) {
            eventSource.close()
            eventSource = null
            isConnected.value = false
        }
    }

    onUnmounted(() => {
        disconnect()
    })

    return {
        connect,
        disconnect,
        isConnected,
        logs,
        currentPhase,
        activeNodes
    }
}

function createLogEntry(message: string, level: LogEntry['level'], metadata?: any): LogEntry {
    return {
        id: Math.random().toString(36).substring(7),
        timestamp: new Date().toISOString(),
        message,
        level,
        metadata
    }
}
