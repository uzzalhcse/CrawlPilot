<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useThemeStore } from '@/stores/theme'
import {
  Workflow,
  PlayCircle,
  Settings,
  ChevronDown,
  Search,
  Bell,
  ChevronLeft,
  ChevronRight,
  Home,
  Store,
  FileText,
  Calendar,
  Server,
  HardDrive,
  HelpCircle,
    GlobeIcon,
} from 'lucide-vue-next'

const router = useRouter()
const themeStore = useThemeStore()
const isCollapsed = ref(false)

const menuItems = [
  { icon: Home, label: 'Home', route: '/' },
  { icon: Store, label: 'Store', route: '/store' },
  { icon: Workflow, label: 'Workflows', route: '/workflows', badge: null },
  { icon: PlayCircle, label: 'Executions', route: '/executions' },
  { icon: Calendar, label: 'Monitoring', route: '/monitoring' },
  { icon: Calendar, label: 'Schedules', route: '/schedules' },
  { icon: FileText, label: 'Logs', route: '/logs' },
]

const antiBotItems = [
  { icon: GlobeIcon, label: 'Browsers', route: '/browsers' },
]

const bottomItems = [
  { icon: Server, label: 'Proxy', route: '/proxy' },
  { icon: HardDrive, label: 'Storage', route: '/storage' },
  { icon: Settings, label: 'Settings', route: '/settings' },
  { icon: HelpCircle, label: 'Help', route: '/help' },
]

const devSectionOpen = ref(true)

const isActive = (route: string) => {
  if (route === '/') {
    return router.currentRoute.value.path === '/'
  }
  return router.currentRoute.value.path.startsWith(route)
}

const toggleSidebar = () => {
  isCollapsed.value = !isCollapsed.value
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
        'fixed left-0 top-0 z-40 h-screen transition-all duration-300 bg-sidebar border-r border-sidebar-border flex flex-col',
        isCollapsed ? 'w-14' : 'w-56'
      ]"
    >
      <!-- User Section -->
      <div v-if="!isCollapsed" class="p-3 border-b border-sidebar-border">
        <button class="w-full flex items-center gap-2 p-2 rounded-lg hover:bg-sidebar-accent transition-colors">
          <div class="w-8 h-8 rounded-full bg-primary/20 flex items-center justify-center text-sm font-medium text-primary">
            U
          </div>
          <div class="flex-1 text-left min-w-0">
            <div class="text-sm font-medium truncate">Uzzal</div>
            <div class="text-xs text-muted-foreground truncate">Personal</div>
          </div>
          <ChevronDown class="w-4 h-4 text-muted-foreground shrink-0" />
        </button>
      </div>

      <!-- Collapsed User Section -->
      <div v-else class="p-2 border-b border-sidebar-border flex justify-center">
        <div class="w-8 h-8 rounded-full bg-primary/20 flex items-center justify-center text-sm font-medium text-primary">
          U
        </div>
      </div>

      <!-- Search Bar -->
      <div v-if="!isCollapsed" class="p-3 border-b border-sidebar-border">
        <div class="relative">
          <Search class="absolute left-2.5 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <input
            type="text"
            placeholder="Search..."
            class="w-full pl-8 pr-12 py-1.5 text-sm bg-background border border-sidebar-border rounded-md focus:outline-none focus:ring-1 focus:ring-primary"
          />
          <kbd class="absolute right-2 top-1/2 -translate-y-1/2 px-1.5 py-0.5 text-[10px] font-mono bg-muted text-muted-foreground rounded border border-sidebar-border">
            ⌘K
          </kbd>
        </div>
      </div>

      <!-- Notification Icon (Collapsed) -->
      <div v-else class="p-2 border-b border-sidebar-border flex justify-center">
        <button class="p-2 hover:bg-sidebar-accent rounded-lg transition-colors">
          <Bell class="w-4 h-4 text-muted-foreground" />
        </button>
      </div>

      <!-- Navigation -->
      <nav class="flex-1 overflow-y-auto py-2 scrollbar-hide">
        <div class="px-2 space-y-0.5">
          <RouterLink
            v-for="item in menuItems"
            :key="item.route"
            :to="item.route"
            :class="[
              'group flex items-center gap-3 px-2.5 py-2 rounded-lg text-sm font-medium transition-colors',
              isActive(item.route)
                ? 'bg-sidebar-accent text-foreground'
                : 'text-sidebar-foreground hover:bg-sidebar-accent hover:text-foreground',
              isCollapsed && 'justify-center'
            ]"
          >
            <component :is="item.icon" class="w-4 h-4 shrink-0" />
            <span v-if="!isCollapsed">{{ item.label }}</span>
          </RouterLink>
        </div>

        <!-- Development Section -->
        <div class="mt-4">
          <button
            v-if="!isCollapsed"
            @click="devSectionOpen = !devSectionOpen"
            class="w-full flex items-center justify-between px-4 py-2 text-xs font-semibold text-muted-foreground uppercase tracking-wider hover:text-foreground transition-colors"
          >
            <span>AntiBot</span>
            <ChevronRight :class="['w-3 h-3 transition-transform', devSectionOpen && 'rotate-90']" />
          </button>
          
          <div v-if="devSectionOpen || isCollapsed" class="px-2 space-y-0.5 mt-1">
            <RouterLink
              v-for="item in antiBotItems"
              :key="item.route"
              :to="item.route"
              :class="[
                'group flex items-center gap-3 px-2.5 py-2 rounded-lg text-sm font-medium transition-colors',
                isActive(item.route)
                  ? 'bg-sidebar-accent text-foreground'
                  : 'text-sidebar-foreground hover:bg-sidebar-accent hover:text-foreground',
                isCollapsed && 'justify-center'
              ]"
            >
              <component :is="item.icon" class="w-4 h-4 shrink-0" />
              <span v-if="!isCollapsed">{{ item.label }}</span>
            </RouterLink>
          </div>
        </div>

        <!-- Bottom Items -->
        <div class="mt-4 px-2 space-y-0.5">
          <RouterLink
            v-for="item in bottomItems"
            :key="item.route"
            :to="item.route"
            :class="[
              'group flex items-center gap-3 px-2.5 py-2 rounded-lg text-sm font-medium transition-colors',
              isActive(item.route)
                ? 'bg-sidebar-accent text-foreground'
                : 'text-sidebar-foreground hover:bg-sidebar-accent hover:text-foreground',
              isCollapsed && 'justify-center'
            ]"
          >
            <component :is="item.icon" class="w-4 h-4 shrink-0" />
            <span v-if="!isCollapsed">{{ item.label }}</span>
          </RouterLink>
        </div>
      </nav>

      <!-- Footer -->
      <div class="border-t border-sidebar-border">

        <!-- Logo and Toggle -->
        <div class="p-3 flex items-center justify-between">
          <div v-if="!isCollapsed" class="flex items-center gap-2">
            <Workflow class="w-5 h-5 text-primary" />
            <span class="text-sm font-bold">Crawlify</span>
          </div>
          <button
            @click="toggleSidebar"
            class="p-2 hover:bg-sidebar-accent rounded-lg transition-colors"
            :class="isCollapsed && 'mx-auto'"
          >
            <ChevronLeft v-if="!isCollapsed" class="w-4 h-4" />
            <ChevronRight v-else class="w-4 h-4" />
          </button>
        </div>
      </div>
    </aside>

    <!-- Main Content -->
    <div :class="['flex flex-1 flex-col transition-all duration-300 min-w-0', isCollapsed ? 'ml-14' : 'ml-56']">
      <!-- Page Content -->
      <main class="flex-1 overflow-y-auto bg-background">
        <RouterView />
      </main>
    </div>
  </div>
</template>

<style scoped>
/* Hide scrollbar but keep functionality */
.scrollbar-hide {
  scrollbar-width: none; /* Firefox */
  -ms-overflow-style: none; /* IE and Edge */
}

.scrollbar-hide::-webkit-scrollbar {
  display: none; /* Chrome, Safari, Opera */
}
</style>
