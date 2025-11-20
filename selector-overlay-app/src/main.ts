import { createApp } from 'vue'
import App from './App.vue'
import './assets/styles.css'

// Ensure body has position relative for absolute positioning to work
if (document.body.style.position !== 'relative' && document.body.style.position !== 'absolute') {
  document.body.style.position = 'relative'
}

// Create and mount the app
const app = createApp(App)

// Create container if it doesn't exist
let container = document.getElementById('crawlify-selector-overlay')
if (!container) {
  container = document.createElement('div')
  container.id = 'crawlify-selector-overlay'
  container.style.position = 'absolute'
  container.style.top = '0'
  container.style.left = '0'
  container.style.width = '100%'
  container.style.minHeight = '100vh'
  container.style.pointerEvents = 'none'
  container.style.zIndex = '999999'
  document.body.appendChild(container)
}

const instance = app.mount(container)

// Expose API to parent window/Go backend
window.__crawlifyGetSelections = () => {
  return (instance as any).getSelections?.() || []
}

// Store app instance for external access
window.__crawlifyApp = instance

// Export for module usage
export { app, instance }
