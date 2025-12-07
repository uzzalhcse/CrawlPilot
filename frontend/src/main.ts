import { createApp } from 'vue'
import { createPinia } from 'pinia'
import VueApexCharts from 'vue3-apexcharts'
import App from './App.vue'
import router from './router'
import './assets/index.css'

const pinia = createPinia()
const app = createApp(App)

app.use(pinia)
app.use(router)
app.use(VueApexCharts)

app.mount('#app')

// Initialize theme after app is mounted
import { useThemeStore } from './stores/theme'
const themeStore = useThemeStore()
themeStore.initTheme()
