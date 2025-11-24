<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useThemeStore } from '@/stores/theme'
import {
  LayoutDashboard,
  Workflow,
  PlayCircle,
  BarChart3,
  Activity,
  Settings,
  Moon,
  Sun,
  Menu,
  X
} from 'lucide-vue-next'

const router = useRouter()
const themeStore = useThemeStore()
const isSidebarOpen = ref(true)

const menuItems = [
  { icon: LayoutDashboard, label: 'Dashboard', route: '/' },
  { icon: Workflow, label: 'Workflows', route: '/workflows' },
  { icon: PlayCircle, label: 'Executions', route: '/executions' },
  { icon: Activity, label: 'Health Checks', route: '/health-checks' },
  { icon: BarChart3, label: 'Analytics', route: '/analytics' }
]

const isActive = (route: string) => {
  if (route === '/') {
    return router.currentRoute.value.path === '/'
  }
  return router.currentRoute.value.path.startsWith(route)
}

const toggleSidebar = () => {
  isSidebarOpen.value = !isSidebarOpen.value
}

onMounted(() => {
  themeStore.initTheme()
})
</script>

<template>
  <div class="flex h-screen overflow-hidden bg-background">
    <!-- Sidebar -->
    <aside
      :class="[
        'fixed left-0 top-0 z-40 h-screen transition-transform duration-300',
        isSidebarOpen ? 'translate-x-0' : '-translate-x-full',
        'w-64 bg-sidebar border-r border-sidebar-border'
      ]"
    >
      <div class="flex h-full flex-col">
        <!-- Logo -->
        <div class="flex h-16 items-center justify-between border-b border-sidebar-border px-6">
          <div class="flex items-center gap-2">
            <div class="flex h-8 w-8 items-center justify-center rounded-lg bg-primary">
              <Workflow class="h-5 w-5 text-primary-foreground" />
            </div>
            <span class="text-xl font-bold text-sidebar-foreground">Crawlify</span>
          </div>
        </div>

        <!-- Navigation -->
        <nav class="flex-1 space-y-1 px-3 py-4">
          <RouterLink
            v-for="item in menuItems"
            :key="item.route"
            :to="item.route"
            :class="[
              'flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors',
              isActive(item.route)
                ? 'bg-sidebar-accent text-sidebar-accent-foreground'
                : 'text-sidebar-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground'
            ]"
          >
            <component :is="item.icon" class="h-5 w-5" />
            <span>{{ item.label }}</span>
          </RouterLink>
        </nav>

        <!-- Footer -->
        <div class="border-t border-sidebar-border p-4">
          <button
            @click="themeStore.toggleTheme"
            class="flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-sidebar-foreground transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
          >
            <Moon v-if="!themeStore.isDark" class="h-5 w-5" />
            <Sun v-else class="h-5 w-5" />
            <span>{{ themeStore.isDark ? 'Light Mode' : 'Dark Mode' }}</span>
          </button>
        </div>
      </div>
    </aside>

    <!-- Main Content -->
    <div :class="['flex flex-1 flex-col', isSidebarOpen ? 'ml-64' : 'ml-0']">
      <!-- Top Bar -->
      <header class="sticky top-0 z-30 flex h-16 items-center gap-4 border-b bg-background px-6">
        <button
          @click="toggleSidebar"
          class="rounded-lg p-2 hover:bg-accent"
        >
          <Menu v-if="!isSidebarOpen" class="h-5 w-5" />
          <X v-else class="h-5 w-5" />
        </button>

        <div class="flex flex-1 items-center justify-between">
          <h1 class="text-xl font-semibold">
            {{ router.currentRoute.value.meta.title || 'Dashboard' }}
          </h1>

          <div class="flex items-center gap-4">
            <!-- Add user menu or other actions here -->
          </div>
        </div>
      </header>

      <!-- Page Content -->
      <main class="flex-1 overflow-y-auto p-6">
        <RouterView />
      </main>
    </div>
  </div>
</template>
