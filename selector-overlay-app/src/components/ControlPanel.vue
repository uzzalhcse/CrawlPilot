<template>
  <div class="fixed top-5 right-5 bg-white rounded-xl shadow-2xl w-[420px] lg:w-[460px] max-h-[90vh] flex flex-col pointer-events-auto z-[1000000] border border-gray-200 overflow-hidden" @click.stop>
    <!-- Header (Fixed) -->
    <div class="flex-shrink-0 px-5 pt-5 pb-3 border-b border-gray-200 bg-gradient-to-r from-blue-50 to-indigo-50">
      <div class="flex items-start justify-between">
        <div class="flex-1">
          <div class="flex items-center gap-2">
            <h2 class="text-xl font-bold text-gray-900">Element Selector</h2>
            <Button
              v-if="props.detailedViewField"
              @click="emit('closeDetailedView')"
              variant="ghost"
              size="sm"
              class="h-7 w-7 p-0"
              title="Back to list (ESC)"
            >
              <span class="text-lg">←</span>
            </Button>
          </div>
          <p class="text-sm text-gray-600 mt-1">
            {{ props.detailedViewField ? 'Configure field details' : 'Click elements to select them' }}
          </p>
        </div>
      </div>
      
      <!-- Keyboard hints - Compact -->
      <div v-if="!props.detailedViewField" class="mt-3 flex items-center gap-3 text-xs text-gray-600">
        <div class="flex items-center gap-1">
          <kbd class="px-1.5 py-0.5 bg-white border border-gray-300 rounded text-gray-700 font-mono">ESC</kbd>
          <span>Clear</span>
        </div>
        <div class="flex items-center gap-1">
          <kbd class="px-1.5 py-0.5 bg-white border border-gray-300 rounded text-gray-700 font-mono">Enter</kbd>
          <span>Add</span>
        </div>
      </div>
    </div>

    <!-- Scrollable Content Area -->
    <ScrollArea class="flex-1">
      <div class="px-5 pb-5">
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

          <!-- Tab Content - Regular Mode -->
          <TabsContent value="regular" class="space-y-3 mt-4">
            <div>
              <Label for="field-name" class="text-sm font-medium mb-1.5">Field Name</Label>
              <Input
                id="field-name"
                :model-value="props.fieldName"
                @update:model-value="emit('update:fieldName', $event)"
                @keydown.enter="canAddField && emit('addField')"
                type="text"
                placeholder="e.g., title, price, description"
                class="mt-1"
                autofocus
              />
            </div>

            <!-- Multiple Value Option -->
            <Card class="bg-blue-50 border-blue-200">
              <CardContent class="p-3">
                <label class="flex items-start gap-2.5 cursor-pointer">
                  <input
                    type="checkbox"
                    v-model="extractMultiple"
                    class="mt-0.5 w-4 h-4 text-blue-600 rounded border-gray-300 focus:ring-2 focus:ring-blue-500"
                  />
                  <div class="flex-1">
                    <span class="text-sm font-medium text-gray-900">Extract Multiple Values</span>
                    <p class="text-xs text-gray-600 mt-0.5">Extract an array of values from all matching elements</p>
                  </div>
                </label>
              </CardContent>
            </Card>

            <div>
              <Label for="extract-type" class="text-sm font-medium mb-1.5">Extract Type</Label>
              <Select
                :model-value="props.fieldType"
                @update:model-value="emit('update:fieldType', $event)"
              >
                <SelectTrigger id="extract-type" class="mt-1">
                  <SelectValue placeholder="Select extraction type" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="text">📝 Text Content</SelectItem>
                  <SelectItem value="attribute">🏷️ Attribute</SelectItem>
                  <SelectItem value="html">📄 HTML</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div v-if="props.fieldType === 'attribute'">
              <Label for="attribute-name" class="text-sm font-medium mb-1.5">Attribute Name</Label>
              <Input
                id="attribute-name"
                :model-value="props.fieldAttribute"
                @update:model-value="emit('update:fieldAttribute', $event)"
                type="text"
                placeholder="e.g., href, src, data-id"
                class="mt-1"
              />
            </div>

            <!-- Validation Message -->
            <div v-if="props.hoveredElementValidation" class="text-sm">
              <Alert
                :variant="props.hoveredElementValidation.isValid ? 'default' : 'destructive'"
                class="py-2"
              >
                <span class="text-base mr-2">{{ props.hoveredElementValidation.isValid ? '✓' : '✗' }}</span>
                <AlertDescription class="font-medium">
                  {{ props.hoveredElementValidation.message }}
                </AlertDescription>
              </Alert>
            </div>

            <Button
              @click="emit('addField')"
              :disabled="!canAddField"
              class="w-full"
              size="lg"
            >
              <span v-if="canAddField">✓ Add Field</span>
              <span v-else>Add Field</span>
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

        <!-- Selected Fields List -->
        <div v-if="!props.detailedViewField" class="mt-6">
          <div class="flex items-center justify-between mb-3">
            <h3 class="text-sm font-semibold text-gray-900">
              Selected Fields
            </h3>
            <Badge variant="secondary" class="text-xs">
              {{ props.selectedFields.length }}
            </Badge>
          </div>
          
          <div v-if="props.selectedFields.length === 0" class="text-sm text-gray-500 text-center py-8 border-2 border-dashed border-gray-200 rounded-lg">
            <div class="text-3xl mb-2">📋</div>
            <div>No fields selected yet</div>
            <div class="text-xs mt-1">Click on page elements to start</div>
          </div>

          <div v-else class="space-y-2">
            <Card
              v-for="field in props.selectedFields"
              :key="field.id"
              class="cursor-pointer hover:shadow-md transition-all border-l-4"
              :class="getFieldBorderClass(field)"
              @click="emit('openDetailedView', field)"
            >
              <CardContent class="p-3">
                <div class="flex items-start justify-between">
                  <div class="flex-1 min-w-0">
                    <div class="flex items-center gap-2 mb-1">
                      <div class="font-medium text-gray-900 truncate">{{ field.name }}</div>
                      <!-- Mode Badge -->
                      <Badge
                        v-if="field.mode === 'key-value-pairs'"
                        variant="secondary"
                        class="bg-purple-100 text-purple-700 border-purple-300 text-xs"
                      >
                        🔗 K-V
                      </Badge>
                      <Badge
                        v-else-if="field.matchCount && field.matchCount > 1"
                        variant="secondary"
                        class="bg-purple-100 text-purple-700 border-purple-300 text-xs"
                      >
                        📋 Array
                      </Badge>
                      <Badge
                        v-else
                        variant="outline"
                        class="text-xs"
                        :class="getFieldTypeBadgeClass(field)"
                      >
                        {{ field.type }}
                      </Badge>
                    </div>
                    
                    <!-- Selector Display -->
                    <div v-if="field.mode === 'key-value-pairs' && field.attributes?.extractions?.[0]" class="text-xs text-gray-500 font-mono mt-1 space-y-0.5">
                      <div class="text-green-700 truncate">🔑 {{ field.attributes.extractions[0].key_selector }}</div>
                      <div class="text-blue-700 truncate">💎 {{ field.attributes.extractions[0].value_selector }}</div>
                    </div>
                    <div v-else class="text-xs text-gray-500 font-mono truncate mt-1">
                      {{ field.selector }}
                    </div>
                    
                    <div v-if="field.matchCount && field.mode !== 'key-value-pairs'" class="flex items-center gap-1 mt-1">
                      <Badge variant="outline" class="text-xs" :class="field.matchCount > 1 ? 'border-purple-400 text-purple-700' : 'border-blue-400 text-blue-700'">
                        {{ field.matchCount }} {{ field.matchCount === 1 ? 'match' : 'matches' }}
                      </Badge>
                    </div>
                    <div v-if="field.sampleValue && field.mode !== 'key-value-pairs'" class="text-xs text-gray-600 truncate mt-1 italic">
                      "{{ field.sampleValue }}"
                    </div>
                  </div>
                  <Button
                    @click.stop="emit('removeField', field.id)"
                    variant="ghost"
                    size="sm"
                    class="h-7 w-7 p-0 ml-2 text-red-500 hover:text-red-700 hover:bg-red-50"
                    title="Remove field"
                  >
                    ✕
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
