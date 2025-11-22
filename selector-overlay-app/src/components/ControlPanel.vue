<template>
  <div class="fixed top-4 right-4 bg-white rounded-2xl shadow-2xl w-[440px] lg:w-[480px] max-h-[92vh] flex flex-col pointer-events-auto z-[1000000] border-2 border-gray-100 overflow-hidden backdrop-blur-sm" @click.stop>
    <!-- Header (Fixed) with better gradient -->
    <div class="flex-shrink-0 px-6 pt-5 pb-4 border-b-2 border-gray-100 bg-gradient-to-br from-blue-50 via-indigo-50 to-purple-50">
      <div class="flex items-start justify-between">
        <div class="flex-1">
          <div class="flex items-center gap-3">
            <div class="flex items-center gap-2">
              <span class="text-2xl">🎯</span>
              <h2 class="text-xl font-bold text-gray-900 tracking-tight">Element Selector</h2>
            </div>
            <Button
              v-if="props.detailedViewField"
              @click="emit('closeDetailedView')"
              variant="ghost"
              size="sm"
              class="h-8 w-8 p-0 hover:bg-white/80 transition-all"
              title="Back to list (ESC)"
            >
              <span class="text-xl">←</span>
            </Button>
          </div>
          <p class="text-sm text-gray-600 mt-1.5 font-medium">
            {{ props.detailedViewField ? '✨ Configure field details' : '👆 Click elements on the page to select' }}
          </p>
        </div>
      </div>
      
      <!-- Keyboard hints - More visible and attractive -->
      <div v-if="!props.detailedViewField" class="mt-3 flex items-center gap-4 text-xs">
        <div class="flex items-center gap-1.5 text-gray-700">
          <kbd class="px-2 py-1 bg-white border-2 border-gray-300 rounded-md text-gray-800 font-mono font-bold shadow-sm">ESC</kbd>
          <span class="font-medium">Clear</span>
        </div>
        <div class="flex items-center gap-1.5 text-gray-700">
          <kbd class="px-2 py-1 bg-white border-2 border-gray-300 rounded-md text-gray-800 font-mono font-bold shadow-sm">↵</kbd>
          <span class="font-medium">Add Field</span>
        </div>
      </div>
    </div>

    <!-- Scrollable Content Area with better styling -->
    <ScrollArea class="flex-1">
      <div class="px-6 pb-6">
        <!-- Tab Navigation -->
        <Tabs v-if="!props.detailedViewField" v-model="activeTab" class="w-full mt-4">
          <TabsList class="grid w-full grid-cols-2">
            <TabsTrigger value="regular" class="flex items-center gap-1.5">
              <span>📄</span>
              <span class="text-xs">Single/Multiple</span>
            </TabsTrigger>
            <TabsTrigger value="key-value" class="flex items-center gap-1.5">
              <span>🔗</span>
              <span class="text-xs">Key-Value</span>
            </TabsTrigger>
          </TabsList>

          <!-- Tab Content - Regular Mode with improved design -->
          <TabsContent value="regular" class="space-y-4 mt-4">
            <div>
              <Label for="field-name" class="text-sm font-semibold mb-2 flex items-center gap-2">
                <span>📝</span>
                <span>Field Name</span>
              </Label>
              <Input
                id="field-name"
                :model-value="props.fieldName"
                @update:model-value="emit('update:fieldName', $event)"
                @keydown.enter="canAddField && emit('addField')"
                type="text"
                placeholder="e.g., title, price, description"
                class="mt-1.5 h-11 text-base"
                autofocus
              />
            </div>

            <!-- Multiple Value Option with better styling -->
            <Card class="bg-gradient-to-br from-blue-50 to-indigo-50 border-2 border-blue-200 shadow-sm hover:shadow-md transition-shadow">
              <CardContent class="p-4">
                <label class="flex items-start gap-3 cursor-pointer group">
                  <input
                    type="checkbox"
                    v-model="extractMultiple"
                    class="mt-0.5 w-5 h-5 text-blue-600 rounded-md border-gray-300 focus:ring-2 focus:ring-blue-500 cursor-pointer"
                  />
                  <div class="flex-1">
                    <span class="text-sm font-semibold text-gray-900 group-hover:text-blue-700 transition-colors">📋 Extract Multiple Values</span>
                    <p class="text-xs text-gray-600 mt-1 leading-relaxed">Extract an array of values from all matching elements on the page</p>
                  </div>
                </label>
              </CardContent>
            </Card>

            <div>
              <Label for="extract-type" class="text-sm font-semibold mb-2 flex items-center gap-2">
                <span>🎨</span>
                <span>Extract Type</span>
              </Label>
              <Select
                :model-value="props.fieldType"
                @update:model-value="emit('update:fieldType', $event)"
              >
                <SelectTrigger id="extract-type" class="mt-1.5 h-11">
                  <SelectValue placeholder="Select extraction type" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="text">📝 Text Content</SelectItem>
                  <SelectItem value="attribute">🏷️ Attribute</SelectItem>
                  <SelectItem value="html">📄 HTML</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div v-if="props.fieldType === 'attribute'" class="animate-in slide-in-from-top-2 duration-300">
              <Label for="attribute-name" class="text-sm font-semibold mb-2 flex items-center gap-2">
                <span>🏷️</span>
                <span>Attribute Name</span>
              </Label>
              <Input
                id="attribute-name"
                :model-value="props.fieldAttribute"
                @update:model-value="emit('update:fieldAttribute', $event)"
                type="text"
                placeholder="e.g., href, src, data-id"
                class="mt-1.5 h-11 text-base font-mono"
              />
            </div>

            <!-- Validation Message with better styling -->
            <div v-if="props.hoveredElementValidation" class="text-sm animate-in fade-in duration-200">
              <Alert
                :variant="props.hoveredElementValidation.isValid ? 'default' : 'destructive'"
                class="py-3 shadow-sm"
              >
                <span class="text-lg mr-2">{{ props.hoveredElementValidation.isValid ? '✅' : '❌' }}</span>
                <AlertDescription class="font-semibold">
                  {{ props.hoveredElementValidation.message }}
                </AlertDescription>
              </Alert>
            </div>

            <Button
              @click="emit('addField')"
              :disabled="!canAddField"
              class="w-full h-12 text-base font-semibold shadow-md hover:shadow-lg transition-all"
              size="lg"
            >
              <span v-if="canAddField" class="flex items-center gap-2">
                <span class="text-lg">✓</span>
                <span>Add Field</span>
              </span>
              <span v-else class="text-gray-400">Select an element to continue</span>
            </Button>
          </TabsContent>

          <!-- Tab Content - Key-Value Pair Selector -->
          <TabsContent value="key-value" class="mt-4">
            <KeyValuePairSelector
              ref="kvSelectorRef"
              v-model:field-name="kvFieldName"
              @add="handleAddKeyValueField"
            />
          </TabsContent>
        </Tabs>

        <!-- Selected Fields List with improved design -->
        <div v-if="!props.detailedViewField" class="mt-6">
          <div class="flex items-center justify-between mb-4">
            <h3 class="text-base font-bold text-gray-900 flex items-center gap-2">
              <span>📦</span>
              <span>Selected Fields</span>
            </h3>
            <Badge variant="secondary" class="text-sm px-3 py-1 font-bold bg-blue-100 text-blue-700 border border-blue-300">
              {{ props.selectedFields.length }}
            </Badge>
          </div>
          
          <div v-if="props.selectedFields.length === 0" class="text-sm text-gray-500 text-center py-10 border-2 border-dashed border-gray-300 rounded-xl bg-gray-50">
            <div class="text-5xl mb-3 animate-bounce">📋</div>
            <div class="font-semibold text-gray-700 text-base">No fields selected yet</div>
            <div class="text-xs mt-2 text-gray-600">Click on elements in the page to start selecting</div>
          </div>

          <div v-else class="space-y-3">
            <Card
              v-for="field in props.selectedFields"
              :key="field.id"
              class="cursor-pointer hover:shadow-lg hover:scale-[1.02] transition-all duration-200 border-l-4 bg-gradient-to-r from-white to-gray-50"
              :class="getFieldBorderClass(field)"
              @click="emit('openDetailedView', field)"
            >
              <CardContent class="p-4">
                <div class="flex items-start justify-between gap-3">
                  <div class="flex-1 min-w-0">
                    <div class="flex items-center gap-2 mb-2">
                      <div class="font-bold text-gray-900 truncate text-base">{{ field.name }}</div>
                      <!-- Mode Badge with improved styling -->
                      <Badge
                        v-if="field.mode === 'key-value-pairs'"
                        variant="secondary"
                        class="bg-gradient-to-r from-purple-100 to-purple-200 text-purple-800 border-purple-300 text-xs font-bold"
                      >
                        🔗 K-V
                      </Badge>
                      <Badge
                        v-else-if="field.matchCount && field.matchCount > 1"
                        variant="secondary"
                        class="bg-gradient-to-r from-purple-100 to-indigo-100 text-purple-800 border-purple-300 text-xs font-bold"
                      >
                        📋 {{ field.matchCount }}
                      </Badge>
                      <Badge
                        v-else
                        variant="outline"
                        class="text-xs font-semibold"
                        :class="getFieldTypeBadgeClass(field)"
                      >
                        {{ field.type }}
                      </Badge>
                    </div>
                    
                    <!-- Selector Display with better contrast -->
                    <div v-if="field.mode === 'key-value-pairs' && field.attributes?.extractions?.[0]" class="text-xs font-mono mt-2 space-y-1 bg-gray-100 p-2 rounded border border-gray-200">
                      <div class="text-green-700 truncate font-semibold">🔑 {{ field.attributes.extractions[0].key_selector }}</div>
                      <div class="text-blue-700 truncate font-semibold">💎 {{ field.attributes.extractions[0].value_selector }}</div>
                    </div>
                    <div v-else class="text-xs text-gray-600 font-mono truncate mt-2 bg-gray-100 px-2 py-1.5 rounded border border-gray-200">
                      {{ field.selector }}
                    </div>
                    
                    <div v-if="field.matchCount && field.mode !== 'key-value-pairs'" class="flex items-center gap-2 mt-2">
                      <Badge variant="outline" class="text-xs font-semibold" :class="field.matchCount > 1 ? 'border-purple-400 text-purple-700 bg-purple-50' : 'border-blue-400 text-blue-700 bg-blue-50'">
                        {{ field.matchCount }} {{ field.matchCount === 1 ? 'match' : 'matches' }}
                      </Badge>
                    </div>
                    <div v-if="field.sampleValue && field.mode !== 'key-value-pairs'" class="text-xs text-gray-700 truncate mt-2 italic bg-blue-50 px-2 py-1 rounded border border-blue-200">
                      💬 "{{ field.sampleValue }}"
                    </div>
                  </div>
                  <Button
                    @click.stop="emit('removeField', field.id)"
                    variant="ghost"
                    size="sm"
                    class="h-8 w-8 p-0 ml-2 text-red-500 hover:text-red-700 hover:bg-red-100 rounded-lg transition-all hover:scale-110"
                    title="Remove field"
                  >
                    <span class="text-lg">✕</span>
                  </Button>
                </div>
              </CardContent>
            </Card>
          </div>
        </div>

        <!-- Detailed View Content (inside panel) -->
        <DetailedFieldContent
          v-if="props.detailedViewField"
          :field="props.detailedViewField"
          :tab="props.detailedViewTab"
          :edit-mode="props.editMode"
          :test-results="props.testResults"
          @switch-tab="emit('switchTab', $event)"
          @enable-edit="emit('enableEditMode')"
          @save-edit="emit('saveEdit', $event)"
          @cancel-edit="emit('cancelEdit')"
          @test-selector="emit('testSelector', $event)"
          @scroll-to-result="emit('scrollToResult', $event)"
        />
      </div>
    </ScrollArea>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { SelectedField, FieldType, ValidationResult, TestResult, SelectionMode } from '../types'
import DetailedFieldContent from './DetailedFieldContent.vue'
import KeyValuePairSelector from './KeyValuePairSelector.vue'
import { getElementColor } from '../utils/elementColors'

// Shadcn Components
import { Button } from './ui/button'
import { Input } from './ui/input'
import { Label } from './ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './ui/select'
import { Card, CardContent } from './ui/card'
import { Badge } from './ui/badge'
import { Alert, AlertDescription } from './ui/alert'
import { ScrollArea } from './ui/scroll-area'
import { Tabs, TabsContent, TabsList, TabsTrigger } from './ui/tabs'

interface Props {
  fieldName: string
  fieldType: FieldType
  fieldAttribute: string
  mode: SelectionMode
  selectedFields: SelectedField[]
  hoveredElementCount: number
  hoveredElementValidation: ValidationResult | null
  detailedViewField: SelectedField | null
  detailedViewTab: 'preview' | 'edit'
  editMode: boolean
  testResults: TestResult[]
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:fieldName': [name: string]
  'update:fieldType': [type: FieldType]
  'update:fieldAttribute': [attr: string]
  'update:mode': [mode: SelectionMode]
  'addField': []
  'addKeyValueField': [data: any]
  'removeField': [id: string]
  'openDetailedView': [field: SelectedField]
  'closeDetailedView': []
  'switchTab': [tab: 'preview' | 'edit']
  'enableEditMode': []
  'saveEdit': [field: Partial<SelectedField>]
  'cancelEdit': []
  'testSelector': [field: SelectedField]
  'scrollToResult': [result: TestResult]
}>()

const activeTab = ref<'regular' | 'key-value'>('regular')
const extractMultiple = ref(false)
const kvFieldName = ref('')
const kvSelectorRef = ref<InstanceType<typeof KeyValuePairSelector> | null>(null)

// Update mode based on active tab
watch(activeTab, (tab) => {
  const mode = tab === 'key-value' ? 'key-value-pairs' : extractMultiple.value ? 'list' : 'single'
  emit('update:mode', mode)
})

// Update mode when extractMultiple changes
watch(extractMultiple, (isMultiple) => {
  if (activeTab.value === 'regular') {
    const mode = isMultiple ? 'list' : 'single'
    emit('update:mode', mode)
  }
})

const canAddField = computed(() => {
  if (!props.fieldName.trim()) return false
  if (props.hoveredElementCount === 0) return false
  if (props.fieldType === 'attribute' && !props.fieldAttribute.trim()) return false
  return true
})

function handleAddKeyValueField(data: { fieldName: string; extractions: any[] }) {
  emit('addKeyValueField', data)
  kvFieldName.value = ''
}

const getFieldBorderClass = (field: SelectedField) => {
  if (field.type === 'text') return 'border-l-blue-500'
  if (field.type === 'attribute') return 'border-l-purple-500'
  if (field.type === 'html') return 'border-l-pink-500'
  return 'border-l-gray-500'
}

const getFieldTypeBadgeClass = (field: SelectedField) => {
  if (field.type === 'text') return 'border-blue-300 text-blue-700'
  if (field.type === 'attribute') return 'border-purple-300 text-purple-700'
  if (field.type === 'html') return 'border-pink-300 text-pink-700'
  return 'border-gray-300 text-gray-700'
}
</script>
