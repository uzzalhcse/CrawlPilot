import { createApp } from 'vue'
import './style.css'
import App from './App.vue'

declare global {
    interface Window {
        __selectFlowApp: any
    }
}

// Ensure body has position relative for absolute positioning to work
if (getComputedStyle(document.body).position === 'static') {
    document.body.style.position = 'relative'
}

// Create container if it doesn't exist
let container = document.getElementById('selectflow-overlay')
if (!container) {
    container = document.createElement('div')
    container.id = 'selectflow-overlay'
    container.style.position = 'absolute'
    container.style.top = '0'
    container.style.left = '0'
    container.style.width = '100%'
    container.style.minHeight = '100vh'
    container.style.pointerEvents = 'none' // Let clicks pass through by default
    container.style.zIndex = '999999'
    document.body.appendChild(container)
}

const app = createApp(App)
const instance = app.mount(container)

// Store app instance for external access
window.__selectFlowApp = instance

export { app, instance }
