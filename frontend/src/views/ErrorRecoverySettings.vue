<template>
  <PageLayout>
    <!-- Header -->
    <PageHeader 
      title="Error Recovery System" 
      description="Configure intelligent error recovery with AI-powered fallback"
      :show-help-icon="true"
    >
      <template #actions>
        <Button v-if="activeTab === 'rules'" @click="openRuleEditor(null)" variant="default" size="sm">
          <Plus class="w-4 h-4 mr-2" />
          Add Rule
        </Button>
      </template>
    </PageHeader>

    <!-- Stats -->
    <StatsBar :stats="rulesStats" />

    <!-- Tabs -->
    <TabBar :tabs="tabs" v-model="activeTab" />

    <!-- Filters -->
    <FilterBar 
      v-if="activeTab === 'rules'"
      search-placeholder="Search rules..." 
      :search-value="searchQuery"
      @update:search-value="searchQuery = $event"
    >
      <template #filters>
        <Select v-model="filterType">
          <SelectTrigger class="w-[140px] h-9">
            <div class="flex items-center gap-2">
              <SlidersHorizontal class="w-4 h-4" />
              <SelectValue placeholder="Rule type" />
            </div>
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Rules</SelectItem>
            <SelectItem value="predefined">Predefined</SelectItem>
            <SelectItem value="learned">AI Learned</SelectItem>
            <SelectItem value="custom">Custom</SelectItem>
          </SelectContent>
        </Select>
      </template>
    </FilterBar>

    <!-- Content -->
    <div class="flex-1 overflow-auto">
      <div v-if="loading" class="flex items-center justify-center py-12">
        <Loader2 class="h-8 w-8 animate-spin text-primary" />
      </div>

      <!-- Rules Tab -->
      <div v-else-if="activeTab === 'rules'">
        <div v-if="filteredRules.length === 0" class="py-12 text-center">
          <p class="text-muted-foreground">No rules found</p>
        </div>
        <div v-else class="grid gap-3 p-6">
          <div
            v-for="rule in filteredRules"
            :key="rule.id"
            class="group bg-card border rounded-lg p-4 hover:border-primary/50 transition-colors cursor-pointer"
            @click="openRuleEditor(rule)"
          >
            <div class="flex items-start justify-between gap-4">
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2 mb-1">
                  <h3 class="text-sm font-medium truncate">{{ rule.name }}</h3>
                  <Badge 
                    variant="outline"
                    :class="rule.is_learned 
                      ? 'bg-green-500/10 text-green-600 dark:text-green-400 border-green-500/20' 
                      : 'bg-purple-500/10 text-purple-600 dark:text-purple-400 border-purple-500/20'"
                    class="text-xs px-1.5 py-0"
                  >
                    {{ rule.is_learned ? 'AI Learned' : 'Manual' }}
                  </Badge>
                  <Badge variant="outline" class="text-xs px-1.5 py-0">
                    P{{ rule.priority }}
                  </Badge>
                  <Badge v-if="!rule.enabled" variant="outline" class="text-xs px-1.5 py-0 bg-yellow-500/10 text-yellow-600">
                    Disabled
                  </Badge>
                </div>
                <p class="text-xs text-muted-foreground line-clamp-1 mb-2">{{ rule.description || rule.pattern }}</p>
                
                <div class="flex flex-wrap gap-3 text-xs text-muted-foreground">
                  <span>{{ rule.action }}</span>
                  <span>{{ rule.success_count }} successes</span>
                  <span>{{ rule.failure_count }} failures</span>
                  <span>Max {{ rule.max_retries }} retries</span>
                </div>
              </div>
              
              <div class="flex items-center gap-1" @click.stop>
                <Button @click="openRuleEditor(rule)" size="sm" variant="ghost" class="h-8 w-8 p-0">
                  <Pencil class="h-4 w-4" />
                </Button>
                <Button @click="confirmDelete(rule)" size="sm" variant="ghost" class="h-8 w-8 p-0 text-destructive hover:text-destructive">
                  <Trash2 class="h-4 w-4" />
                </Button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Settings Tab -->
      <div v-else-if="activeTab === 'settings'" class="p-6">
        <GeneralSettings />
      </div>
    </div>

    <!-- Rule Editor Modal -->
    <RuleEditor
      v-if="showRuleEditor"
      :rule="selectedRule"
      @close="showRuleEditor = false"
      @save="handleSaveRule"
    />
  </PageLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useErrorRecoveryStore, type ContextAwareRule } from '@/stores/errorRecovery'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import PageLayout from '@/components/layout/PageLayout.vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import StatsBar from '@/components/layout/StatsBar.vue'
import TabBar from '@/components/layout/TabBar.vue'
import FilterBar from '@/components/layout/FilterBar.vue'
import RuleEditor from '@/components/error-recovery/RuleEditor.vue'
import GeneralSettings from '@/components/error-recovery/GeneralSettings.vue'
import { Plus, Pencil, Trash2, Loader2, SlidersHorizontal } from 'lucide-vue-next'

const store = useErrorRecoveryStore()
const activeTab = ref('rules')
const filterType = ref('all')
const searchQuery = ref('')
const showRuleEditor = ref(false)
const selectedRule = ref<ContextAwareRule | null>(null)
const loading = ref(false)

const tabs = [
  { id: 'rules', label: 'Rules' },
  { id: 'settings', label: 'Settings' }
]

const rulesStats = computed(() => [
  { label: 'Total Rules', value: store.rules.length },
  { label: 'Predefined', value: store.predefinedRules.length, color: 'text-purple-600 dark:text-purple-400' },
  { label: 'AI Learned', value: store.learnedRules.length, color: 'text-green-600 dark:text-green-400' },
  { label: 'Custom', value: store.customRules.length, color: 'text-blue-600 dark:text-blue-400' }
])

const filteredRules = computed(() => {
  let rules = store.rulesSortedByPriority
  if (filterType.value === 'learned') {
    rules = rules.filter(r => r.is_learned)
  } else if (filterType.value === 'predefined' || filterType.value === 'custom') {
    rules = rules.filter(r => !r.is_learned)
  }
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    rules = rules.filter(r => r.name.toLowerCase().includes(query) || r.description?.toLowerCase().includes(query))
  }
  return rules
})

function openRuleEditor(rule: ContextAwareRule | null) {
  selectedRule.value = rule
  showRuleEditor.value = true
}

async function handleSaveRule(rule: Partial<ContextAwareRule>) {
  try {
    if (selectedRule.value?.id) {
      await store.updateRule(selectedRule.value.id, rule)
    } else {
      await store.createRule(rule)
    }
    showRuleEditor.value = false
  } catch (error) {
    // Error handled by store
  }
}

async function confirmDelete(rule: ContextAwareRule) {
  if (confirm(`Delete rule "${rule.name}"?`)) {
    await store.deleteRule(rule.id)
  }
}

onMounted(async () => {
  loading.value = true
  await store.fetchRules()
  loading.value = false
})
</script>
