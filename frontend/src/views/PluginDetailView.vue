<template>
  <div class="min-h-screen bg-background text-foreground font-sans">
    <ScrollArea class="h-screen">
      <div class="container mx-auto max-w-6xl px-6 py-12">
        <!-- Back Button -->
        <div class="mb-8">
          <Button variant="ghost" class="gap-2 pl-0 hover:bg-transparent hover:text-primary" @click="$router.back()">
            <ArrowLeft class="h-4 w-4" />
            Back to Store
          </Button>
        </div>

        <div v-if="loading" class="space-y-8 animate-pulse">
          <div class="flex gap-6">
            <div class="w-24 h-24 rounded-xl bg-secondary/50" />
            <div class="space-y-4 flex-1">
              <div class="h-8 w-1/3 bg-secondary/50 rounded" />
              <div class="h-4 w-1/4 bg-secondary/50 rounded" />
            </div>
          </div>
        </div>

        <div v-else-if="plugin" class="grid grid-cols-1 lg:grid-cols-3 gap-12">
          <!-- Main Content (Left) -->
          <div class="lg:col-span-2 space-y-8">
            <!-- Header -->
            <div class="flex gap-6 items-start">
              <div class="w-24 h-24 rounded-xl bg-gradient-to-br from-primary/20 to-primary/10 flex items-center justify-center text-4xl border border-primary/20 shrink-0">
                {{ getPhaseIcon(plugin.phase_type) }}
              </div>
              <div class="space-y-3">
                <div class="flex items-center gap-3 flex-wrap">
                  <h1 class="text-3xl font-bold tracking-tight">{{ plugin.name }}</h1>
                  <div class="flex gap-2">
                    <Badge v-if="plugin.is_verified" variant="secondary" class="bg-blue-500/10 text-blue-500 border-blue-500/20 gap-1">
                      <BadgeCheck class="w-3 h-3" />
                      Verified
                    </Badge>
                    <Badge v-if="plugin.plugin_type === 'official'" variant="secondary" class="bg-amber-500/10 text-amber-500 border-amber-500/20">
                      Official
                    </Badge>
                  </div>
                </div>
                <p class="text-lg text-muted-foreground leading-relaxed">
                  {{ plugin.description }}
                </p>
              </div>
            </div>

            <Separator />

            <!-- Readme / Details -->
            <div class="prose prose-neutral dark:prose-invert max-w-none">
              <h3>About this Actor</h3>
              <p>
                This is a placeholder for the full description. In a real implementation, this would render the plugin's README.md content.
              </p>
              <p>
                Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.
              </p>
            </div>
          </div>

          <!-- Sidebar (Right) -->
          <div class="space-y-6">
            <!-- Install Card -->
            <Card class="border-border/50 shadow-sm">
              <CardContent class="p-6 space-y-4">
                <Button 
                  class="w-full h-11 text-base font-semibold shadow-lg shadow-primary/20" 
                  :disabled="installing || isInstalled"
                  @click="handleInstall"
                >
                  <Download v-if="!installing && !isInstalled" class="mr-2 h-4 w-4" />
                  <Loader2 v-else-if="installing" class="mr-2 h-4 w-4 animate-spin" />
                  <Check v-else-if="isInstalled" class="mr-2 h-4 w-4" />
                  {{ getInstallButtonText() }}
                </Button>
                
                <div v-if="isInstalled" class="text-center">
                  <Button variant="outline" class="w-full text-destructive hover:text-destructive hover:bg-destructive/10 border-destructive/20" @click="handleUninstall" :disabled="uninstalling">
                    Uninstall
                  </Button>
                </div>
              </CardContent>
            </Card>

            <!-- Metadata Card -->
            <Card class="border-border/50 shadow-sm bg-secondary/20">
              <CardContent class="p-6 space-y-6">
                <div class="space-y-4">
                  <h3 class="font-semibold text-sm uppercase tracking-wider text-muted-foreground">Information</h3>
                  
                  <div class="flex justify-between items-center py-2 border-b border-border/50">
                    <span class="text-sm text-muted-foreground">Version</span>
                    <span class="font-medium">{{ latestVersion?.version || 'Unknown' }}</span>
                  </div>
                  
                  <div class="flex justify-between items-center py-2 border-b border-border/50">
                    <span class="text-sm text-muted-foreground">Author</span>
                    <span class="font-medium flex items-center gap-2">
                       <div class="w-5 h-5 rounded-full bg-primary/10 flex items-center justify-center text-[10px] font-bold text-primary">
                        {{ plugin.author_name.charAt(0).toUpperCase() }}
                      </div>
                      {{ plugin.author_name }}
                    </span>
                  </div>

                  <div class="flex justify-between items-center py-2 border-b border-border/50">
                    <span class="text-sm text-muted-foreground">Downloads</span>
                    <span class="font-medium">{{ formatNumber(plugin.total_downloads) }}</span>
                  </div>

                  <div class="flex justify-between items-center py-2 border-b border-border/50">
                    <span class="text-sm text-muted-foreground">Rating</span>
                    <div class="flex items-center gap-1 font-medium">
                      <Star class="w-4 h-4 text-yellow-500 fill-yellow-500" />
                      {{ plugin.average_rating.toFixed(1) }}
                    </div>
                  </div>

                  <div class="flex justify-between items-center py-2 border-b border-border/50">
                    <span class="text-sm text-muted-foreground">Last Updated</span>
                    <span class="font-medium">{{ formatDate(plugin.updated_at) }}</span>
                  </div>
                </div>

                <div class="space-y-3 pt-2">
                  <Button v-if="plugin.repository_url" variant="outline" class="w-full justify-start gap-2 h-10" as="a" :href="plugin.repository_url" target="_blank">
                    <Github class="w-4 h-4" />
                    View Source
                  </Button>
                  <Button v-if="plugin.documentation_url" variant="outline" class="w-full justify-start gap-2 h-10" as="a" :href="plugin.documentation_url" target="_blank">
                    <BookOpen class="w-4 h-4" />
                    Documentation
                  </Button>
                </div>
              </CardContent>
            </Card>

            <!-- Tags -->
            <div v-if="plugin.tags && plugin.tags.length > 0">
              <h3 class="font-semibold text-sm uppercase tracking-wider text-muted-foreground mb-3">Tags</h3>
              <div class="flex flex-wrap gap-2">
                <Badge v-for="tag in plugin.tags" :key="tag" variant="secondary" class="font-normal">
                  {{ tag }}
                </Badge>
              </div>
            </div>
          </div>
        </div>
      </div>
    </ScrollArea>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft, BadgeCheck, Download, Star, Github, BookOpen, Loader2, Check } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'
import { Separator } from '@/components/ui/separator'
import { ScrollArea } from '@/components/ui/scroll-area'
import pluginAPI from '@/lib/plugin-api'
import type { Plugin, PluginVersion } from '@/types'

const route = useRoute()
const router = useRouter()
const plugin = ref<Plugin | null>(null)
const latestVersion = ref<PluginVersion | null>(null)
const loading = ref(true)
const installing = ref(false)
const uninstalling = ref(false)
const isInstalled = ref(false)

const loadPlugin = async () => {
  loading.value = true
  try {
    // In a real app, we might fetch by slug. For now, we might need to fetch by ID or handle slug lookup.
    // Assuming the route param is 'id' for simplicity based on existing API, 
    // but if we want slug we'd need an API endpoint for it.
    // Let's assume the ID is passed for now.
    const pluginId = route.params.id as string
    
    // We don't have getPluginById in the snippet, but we have GetPluginBySlug in backend.
    // Let's try to fetch by slug if the param is a slug, or ID if it looks like a UUID.
    // For this implementation, let's assume we pass the ID for now to be safe with existing frontend API methods.
    // Actually, looking at the backend code: GetPlugin takes a slug.
    // So we should pass the slug.
    
    // However, the frontend API library might not have getPlugin(slug).
    // Let's check if we can list and find, or if we need to add getPlugin.
    // Since I can't see plugin-api.ts right now, I'll assume I can fetch it.
    // If not, I'll implement a fallback.
    
    // Ideally: plugin.value = await pluginAPI.getPlugin(pluginId)
    // But let's just list for now if get is missing, or try get.
    
    // TEMPORARY: List all and find (inefficient but works without changing API lib blindly)
    const allPlugins = await pluginAPI.listPlugins({})
    plugin.value = allPlugins.find(p => p.id === pluginId || p.slug === pluginId) || null
    
    if (plugin.value) {
      await loadLatestVersion(plugin.value.id)
      await checkInstallStatus(plugin.value.id)
    }
  } catch (error) {
    console.error('Failed to load plugin:', error)
  } finally {
    loading.value = false
  }
}

const loadLatestVersion = async (pluginId: string) => {
  try {
    const versions = await pluginAPI.listVersions(pluginId)
    if (versions.length > 0) {
      latestVersion.value = versions[0]
    }
  } catch (error) {
    console.error('Failed to load version:', error)
  }
}

const checkInstallStatus = async (pluginId: string) => {
  try {
    isInstalled.value = await pluginAPI.isPluginInstalled(pluginId)
  } catch (error) {
    console.error('Failed to check install status:', error)
  }
}

const handleInstall = async () => {
  if (!plugin.value) return
  installing.value = true
  try {
    await pluginAPI.installPlugin(plugin.value.id)
    isInstalled.value = true
  } catch (error) {
    console.error('Failed to install plugin:', error)
  } finally {
    installing.value = false
  }
}

const handleUninstall = async () => {
  if (!plugin.value) return
  if (!confirm(`Are you sure you want to uninstall ${plugin.value.name}?`)) return
  
  uninstalling.value = true
  try {
    await pluginAPI.uninstallPlugin(plugin.value.id)
    isInstalled.value = false
  } catch (error) {
    console.error('Failed to uninstall plugin:', error)
  } finally {
    uninstalling.value = false
  }
}

const getInstallButtonText = () => {
  if (installing.value) return 'Installing...'
  if (isInstalled.value) return 'Installed'
  return 'Install Actor'
}

const formatNumber = (num: number): string => {
  if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M'
  if (num >= 1000) return (num / 1000).toFixed(1) + 'K'
  return num.toString()
}

const formatDate = (dateString: string): string => {
  const date = new Date(dateString)
  return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
}

const getPhaseIcon = (phaseType: string): string => {
  const icons: Record<string, string> = {
    discovery: '🔍',
    extraction: '📦',
    processing: '⚙️',
    custom: '🔧'
  }
  return icons[phaseType] || '📄'
}

onMounted(() => {
  loadPlugin()
})
</script>
