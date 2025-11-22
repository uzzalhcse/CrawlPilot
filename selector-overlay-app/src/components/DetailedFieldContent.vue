<template>
  <div class="space-y-4 mt-4">
    <!-- Field Header -->
    <div class="space-y-3">
      <div class="flex items-start justify-between">
        <div class="flex-1 min-w-0">
          <h3 class="text-lg font-bold text-gray-900 truncate">{{ field.name }}</h3>
          <div class="flex items-center gap-2 mt-1">
            <Badge variant="outline" :class="getFieldTypeBadgeClass(field)">
              {{ field.type }}
            </Badge>
            <Badge v-if="field.mode === 'key-value-pairs'" variant="secondary" class="bg-purple-100 text-purple-700 border-purple-300">
              🔗 Key-Value
            </Badge>
            <Badge v-else-if="field.matchCount && field.matchCount > 1" variant="secondary" class="bg-purple-100 text-purple-700 border-purple-300">
              📋 {{ field.matchCount }} items
            </Badge>
          </div>
        </div>
      </div>
      
      <!-- Selector Display -->
      <Card class="bg-gray-50 border-gray-200">
        <CardContent class="p-3">
          <div class="space-y-2">
            <Label class="text-xs font-medium text-gray-600">CSS Selector</Label>
            <div class="text-xs font-mono text-gray-800 break-all bg-white p-2 rounded border border-gray-200">
              {{ field.selector }}
            </div>
          </div>
        </CardContent>
      </Card>
    </div>

    <!-- Tabs for organizing content -->
    <Tabs v-model="currentTab" class="w-full">
      <TabsList class="grid w-full grid-cols-3">
        <TabsTrigger value="preview">
          <span class="text-sm">👁️ Preview</span>
        </TabsTrigger>
        <TabsTrigger value="config">
          <span class="text-sm">⚙️ Config</span>
        </TabsTrigger>
        <TabsTrigger value="transform">
          <span class="text-sm">✨ Transform</span>
        </TabsTrigger>
      </TabsList>

      <!-- Preview Tab -->
      <TabsContent value="preview" class="space-y-3 mt-4">
        <!-- Test Results Section -->
        <Card>
          <CardHeader class="pb-3">
            <div class="flex items-center justify-between">
              <CardTitle class="text-sm font-semibold">Test Results</CardTitle>
              <Button
                @click="emit('testSelector', field)"
                size="sm"
                variant="outline"
              >
                🔄 Re-test
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            <div v-if="testResults.length === 0" class="text-sm text-gray-500 text-center py-6 border-2 border-dashed border-gray-200 rounded-lg">
              <div class="text-3xl mb-2">🧪</div>
              <div>No test results yet</div>
              <div class="text-xs mt-1">Click "Re-test" to validate selector</div>
            </div>
            
            <div v-else class="space-y-2">
              <div class="flex items-center gap-2 mb-3">
                <Badge variant="default" class="bg-green-100 text-green-800 border-green-300">
                  ✓ {{ testResults.length }} {{ testResults.length === 1 ? 'match' : 'matches' }} found
                </Badge>
              </div>
              
              <ScrollArea class="h-[200px] w-full rounded-md border">
                <div class="p-3 space-y-2">
                  <Card
                    v-for="(result, index) in testResults"
                    :key="index"
                    class="cursor-pointer hover:bg-blue-50 transition-colors"
                    @click="emit('scrollToResult', result)"
                  >
                    <CardContent class="p-3">
                      <div class="flex items-start gap-2">
                        <Badge variant="outline" class="text-xs shrink-0">{{ index + 1 }}</Badge>
                        <div class="flex-1 min-w-0">
                          <div class="text-sm text-gray-800 break-words">
                            {{ result.value || '(empty)' }}
                          </div>
                          <div v-if="result.element" class="text-xs text-gray-500 mt-1 font-mono">
                            {{ result.element }}
                          </div>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                </div>
              </ScrollArea>
            </div>
          </CardContent>
        </Card>

        <!-- Sample Value -->
        <Card v-if="field.sampleValue">
          <CardHeader class="pb-3">
            <CardTitle class="text-sm font-semibold">Sample Value</CardTitle>
          </CardHeader>
          <CardContent>
            <div class="text-sm text-gray-800 p-3 bg-gray-50 rounded border border-gray-200 break-words">
              {{ field.sampleValue }}
            </div>
          </CardContent>
        </Card>
      </TabsContent>

      <!-- Configuration Tab -->
      <TabsContent value="config" class="space-y-3 mt-4">
        <!-- Key-Value Field Configuration -->
        <div v-if="field.mode === 'key-value-pairs'">
          <!-- Field Name -->
          <Card class="mb-3">
            <CardContent class="p-3">
              <div>
                <Label for="kv-field-name" class="text-sm font-medium">Field Name</Label>
                <Input
                  id="kv-field-name"
                  v-model="editableField.name"
                  :disabled="!editMode && editingPairIndex === null"
                  placeholder="Field name"
                  class="mt-1"
                />
              </div>
            </CardContent>
          </Card>

          <!-- Editing Specific Pair -->
          <div v-if="editingPairIndex !== null && editingPairIndex >= 0">
            <Alert class="mb-3 bg-purple-50 border-purple-300">
              <AlertDescription>
                ✏️ Editing extraction pair {{ editingPairIndex + 1 }} - Modify selectors below
              </AlertDescription>
            </Alert>

            <KeyValuePairSelector
              ref="kvEditSelectorRef"
              v-model:field-name="tempFieldName"
              :edit-mode="true"
              @add="handleSavePairEdit"
            />

            <div class="flex gap-2 mt-3">
              <Button @click="cancelPairEdit" variant="outline" class="flex-1">
                ✕ Cancel
              </Button>
            </div>
          </div>

          <!-- Adding New Pair -->
          <div v-else-if="editingPairIndex === -1">
            <Alert class="mb-3 bg-green-50 border-green-300">
              <AlertDescription>
                ➕ Adding new extraction pair
              </AlertDescription>
            </Alert>

            <KeyValuePairSelector
              ref="kvEditSelectorRef"
              v-model:field-name="tempFieldName"
              :edit-mode="false"
              @add="handleSavePairEdit"
            />

            <div class="flex gap-2 mt-3">
              <Button @click="cancelPairEdit" variant="outline" class="flex-1">
                ✕ Cancel
              </Button>
            </div>
          </div>

          <!-- Extraction Pairs List -->
          <div v-else>
            <div class="flex items-center justify-between mb-3">
              <h4 class="text-sm font-semibold text-gray-900">Extraction Pairs</h4>
              <Badge variant="secondary">
                {{ editableField.attributes?.extractions?.length || 0 }}
              </Badge>
            </div>

            <div v-if="!editableField.attributes?.extractions?.length" class="text-sm text-gray-500 text-center py-8 border-2 border-dashed border-gray-200 rounded-lg">
              <div class="text-3xl mb-2">🔗</div>
              <div>No extraction pairs</div>
            </div>

            <div v-else class="space-y-2">
              <Card
                v-for="(extraction, index) in editableField.attributes.extractions"
                :key="index"
                class="border-l-4 border-l-purple-500 hover:shadow-md transition-all"
              >
                <CardContent class="p-3">
                  <div class="flex items-start justify-between gap-2">
                    <div class="flex-1 min-w-0 space-y-2">
                      <!-- Key Info -->
                      <div class="bg-green-50 border border-green-200 rounded p-2">
                        <div class="flex items-center gap-2 mb-1">
                          <span class="text-xs font-semibold text-green-800">🔑 Key</span>
                          <Badge variant="outline" class="text-xs border-green-300 text-green-700">
                            {{ extraction.key_type }}
                          </Badge>
                          <Badge v-if="extraction.key_attribute" variant="outline" class="text-xs border-green-300 text-green-700">
                            {{ extraction.key_attribute }}
                          </Badge>
                        </div>
                        <div class="text-xs font-mono text-gray-800 break-all">
                          {{ extraction.key_selector }}
                        </div>
                      </div>

                      <!-- Value Info -->
                      <div class="bg-blue-50 border border-blue-200 rounded p-2">
                        <div class="flex items-center gap-2 mb-1">
                          <span class="text-xs font-semibold text-blue-800">💎 Value</span>
                          <Badge variant="outline" class="text-xs border-blue-300 text-blue-700">
                            {{ extraction.value_type }}
                          </Badge>
                          <Badge v-if="extraction.value_attribute" variant="outline" class="text-xs border-blue-300 text-blue-700">
                            {{ extraction.value_attribute }}
                          </Badge>
                        </div>
                        <div class="text-xs font-mono text-gray-800 break-all">
                          {{ extraction.value_selector }}
                        </div>
                      </div>
                    </div>

                    <!-- Action Buttons -->
                    <div class="flex flex-col gap-1 shrink-0">
                      <Button
                        @click="startEditPair(index)"
                        variant="ghost"
                        size="sm"
                        class="h-8 w-8 p-0 text-blue-600 hover:text-blue-700 hover:bg-blue-50"
                        title="Edit pair"
                      >
                        ✏️
                      </Button>
                      <Button
                        @click="deletePair(index)"
                        variant="ghost"
                        size="sm"
                        class="h-8 w-8 p-0 text-red-500 hover:text-red-700 hover:bg-red-50"
                        title="Delete pair"
                      >
                        🗑️
                      </Button>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </div>

            <Separator class="my-4" />

            <!-- Add New Pair Button -->
            <Button @click="startAddPair" variant="outline" class="w-full">
              ➕ Add Extraction Pair
            </Button>

            <!-- Save/Cancel for field name changes -->
            <div v-if="editMode" class="flex gap-2 mt-3">
              <Button @click="saveFieldNameChange" class="flex-1">
                ✓ Save Field Name
              </Button>
              <Button @click="emit('cancelEdit')" variant="outline" class="flex-1">
                ✕ Cancel
              </Button>
            </div>
            <div v-else class="mt-3">
              <Button @click="emit('enableEdit')" variant="outline" class="w-full">
                ✏️ Edit Field Name
              </Button>
            </div>
          </div>
        </div>

        <!-- Regular Field Configuration -->
        <Card v-else>
          <CardHeader class="pb-3">
            <CardTitle class="text-sm font-semibold">Basic Configuration</CardTitle>
          </CardHeader>
          <CardContent class="space-y-4">
            <div>
              <Label for="field-name-edit" class="text-sm font-medium">Field Name</Label>
              <Input
                id="field-name-edit"
                :model-value="editableField.name"
                @update:model-value="updateEditableField('name', $event)"
                :disabled="!editMode"
                placeholder="Field name"
                class="mt-1"
              />
            </div>

            <div>
              <Label for="field-type-edit" class="text-sm font-medium">Extract Type</Label>
              <Select
                :model-value="editableField.type"
                @update:model-value="updateEditableField('type', $event)"
                :disabled="!editMode"
              >
                <SelectTrigger id="field-type-edit" class="mt-1">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="text">📝 Text Content</SelectItem>
                  <SelectItem value="attribute">🏷️ Attribute</SelectItem>
                  <SelectItem value="html">📄 HTML</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div v-if="editableField.type === 'attribute'">
              <Label for="field-attribute-edit" class="text-sm font-medium">Attribute Name</Label>
              <Input
                id="field-attribute-edit"
                :model-value="editableField.attribute"
                @update:model-value="updateEditableField('attribute', $event)"
                :disabled="!editMode"
                placeholder="e.g., href, src, data-id"
                class="mt-1"
              />
            </div>

            <div>
              <Label for="field-selector-edit" class="text-sm font-medium">CSS Selector</Label>
              <Textarea
                id="field-selector-edit"
                :model-value="editableField.selector"
                @update:model-value="updateEditableField('selector', $event)"
                :disabled="!editMode"
                placeholder="CSS selector"
                rows="3"
                class="mt-1 font-mono text-xs"
              />
            </div>

            <Separator />

            <div v-if="!editMode">
              <Button @click="emit('enableEdit')" variant="outline" class="w-full">
                ✏️ Edit Configuration
              </Button>
            </div>
            <div v-else class="flex gap-2">
              <Button @click="saveChanges" class="flex-1">
                ✓ Save Changes
              </Button>
              <Button @click="emit('cancelEdit')" variant="outline" class="flex-1">
                ✕ Cancel
              </Button>
            </div>
          </CardContent>
        </Card>
      </TabsContent>

      <!-- Transformations Tab -->
      <TabsContent value="transform" class="space-y-3 mt-4">
        <Card>
          <CardHeader class="pb-3">
            <div class="flex items-center justify-between">
              <CardTitle class="text-sm font-semibold">Transformations</CardTitle>
              <Badge variant="secondary" v-if="activeTransformationsCount > 0">
                {{ activeTransformationsCount }} active
              </Badge>
            </div>
            <p class="text-xs text-gray-600 mt-1">Apply transformations to extracted values</p>
          </CardHeader>
          <CardContent>
            <Accordion type="multiple" class="w-full">
              <!-- Text Transformations -->
              <AccordionItem value="text">
                <AccordionTrigger class="text-sm">
                  <div class="flex items-center gap-2">
                    <span>📝 Text Transformations</span>
                    <Badge v-if="textTransformCount > 0" variant="secondary" class="text-xs">
                      {{ textTransformCount }}
                    </Badge>
                  </div>
                </AccordionTrigger>
                <AccordionContent>
                  <div class="space-y-3 pt-2">
                    <label class="flex items-center gap-2 cursor-pointer">
                      <input
                        type="checkbox"
                        v-model="editableField.transformations.trim"
                        :disabled="!editMode"
                        class="w-4 h-4 text-blue-600 rounded border-gray-300 focus:ring-2 focus:ring-blue-500"
                      />
                      <div class="flex-1">
                        <div class="text-sm font-medium">Trim Whitespace</div>
                        <div class="text-xs text-gray-600">Remove leading and trailing spaces</div>
                      </div>
                    </label>

                    <label class="flex items-center gap-2 cursor-pointer">
                      <input
                        type="checkbox"
                        v-model="editableField.transformations.lowercase"
                        :disabled="!editMode"
                        class="w-4 h-4 text-blue-600 rounded border-gray-300 focus:ring-2 focus:ring-blue-500"
                      />
                      <div class="flex-1">
                        <div class="text-sm font-medium">Lowercase</div>
                        <div class="text-xs text-gray-600">Convert all text to lowercase</div>
                      </div>
                    </label>

                    <label class="flex items-center gap-2 cursor-pointer">
                      <input
                        type="checkbox"
                        v-model="editableField.transformations.uppercase"
                        :disabled="!editMode"
                        class="w-4 h-4 text-blue-600 rounded border-gray-300 focus:ring-2 focus:ring-blue-500"
                      />
                      <div class="flex-1">
                        <div class="text-sm font-medium">Uppercase</div>
                        <div class="text-xs text-gray-600">Convert all text to UPPERCASE</div>
                      </div>
                    </label>
                  </div>
                </AccordionContent>
              </AccordionItem>

              <!-- String Operations -->
              <AccordionItem value="string">
                <AccordionTrigger class="text-sm">
                  <div class="flex items-center gap-2">
                    <span>✂️ String Operations</span>
                    <Badge v-if="stringOpCount > 0" variant="secondary" class="text-xs">
                      {{ stringOpCount }}
                    </Badge>
                  </div>
                </AccordionTrigger>
                <AccordionContent>
                  <div class="space-y-3 pt-2">
                    <div>
                      <Label class="text-sm font-medium mb-2 flex items-center gap-2">
                        <input
                          type="checkbox"
                          v-model="editableField.transformations.regex_enabled"
                          :disabled="!editMode"
                          class="w-4 h-4 text-blue-600 rounded border-gray-300 focus:ring-2 focus:ring-blue-500"
                        />
                        <span>Regex Extract</span>
                      </Label>
                      <Input
                        v-model="editableField.transformations.regex"
                        :disabled="!editMode || !editableField.transformations.regex_enabled"
                        placeholder="e.g., \d+\.?\d* for numbers"
                        class="mt-1 font-mono text-xs"
                      />
                      <p class="text-xs text-gray-600 mt-1">Extract pattern from text</p>
                    </div>

                    <div>
                      <Label class="text-sm font-medium mb-2 flex items-center gap-2">
                        <input
                          type="checkbox"
                          v-model="editableField.transformations.replace_enabled"
                          :disabled="!editMode"
                          class="w-4 h-4 text-blue-600 rounded border-gray-300 focus:ring-2 focus:ring-blue-500"
                        />
                        <span>Find & Replace</span>
                      </Label>
                      <div class="space-y-2">
                        <Input
                          v-model="editableField.transformations.replace_find"
                          :disabled="!editMode || !editableField.transformations.replace_enabled"
                          placeholder="Find text"
                          class="font-mono text-xs"
                        />
                        <Input
                          v-model="editableField.transformations.replace_with"
                          :disabled="!editMode || !editableField.transformations.replace_enabled"
                          placeholder="Replace with"
                          class="font-mono text-xs"
                        />
                      </div>
                    </div>

                    <div>
                      <Label class="text-sm font-medium mb-2 flex items-center gap-2">
                        <input
                          type="checkbox"
                          v-model="editableField.transformations.prefix_enabled"
                          :disabled="!editMode"
                          class="w-4 h-4 text-blue-600 rounded border-gray-300 focus:ring-2 focus:ring-blue-500"
                        />
                        <span>Add Prefix</span>
                      </Label>
                      <Input
                        v-model="editableField.transformations.prefix"
                        :disabled="!editMode || !editableField.transformations.prefix_enabled"
                        placeholder="e.g., $"
                        class="mt-1"
                      />
                    </div>

                    <div>
                      <Label class="text-sm font-medium mb-2 flex items-center gap-2">
                        <input
                          type="checkbox"
                          v-model="editableField.transformations.suffix_enabled"
                          :disabled="!editMode"
                          class="w-4 h-4 text-blue-600 rounded border-gray-300 focus:ring-2 focus:ring-blue-500"
                        />
                        <span>Add Suffix</span>
                      </Label>
                      <Input
                        v-model="editableField.transformations.suffix"
                        :disabled="!editMode || !editableField.transformations.suffix_enabled"
                        placeholder="e.g., USD"
                        class="mt-1"
                      />
                    </div>
                  </div>
                </AccordionContent>
              </AccordionItem>

              <!-- Type Conversions -->
              <AccordionItem value="type">
                <AccordionTrigger class="text-sm">
                  <div class="flex items-center gap-2">
                    <span>🔢 Type Conversions</span>
                    <Badge v-if="editableField.transformations.parse_number || editableField.transformations.parse_date" variant="secondary" class="text-xs">
                      {{ (editableField.transformations.parse_number ? 1 : 0) + (editableField.transformations.parse_date ? 1 : 0) }}
                    </Badge>
                  </div>
                </AccordionTrigger>
                <AccordionContent>
                  <div class="space-y-3 pt-2">
                    <label class="flex items-center gap-2 cursor-pointer">
                      <input
                        type="checkbox"
                        v-model="editableField.transformations.parse_number"
                        :disabled="!editMode"
                        class="w-4 h-4 text-blue-600 rounded border-gray-300 focus:ring-2 focus:ring-blue-500"
                      />
                      <div class="flex-1">
                        <div class="text-sm font-medium">Parse as Number</div>
                        <div class="text-xs text-gray-600">Convert text to numeric value</div>
                      </div>
                    </label>

                    <label class="flex items-center gap-2 cursor-pointer">
                      <input
                        type="checkbox"
                        v-model="editableField.transformations.parse_date"
                        :disabled="!editMode"
                        class="w-4 h-4 text-blue-600 rounded border-gray-300 focus:ring-2 focus:ring-blue-500"
                      />
                      <div class="flex-1">
                        <div class="text-sm font-medium">Parse as Date</div>
                        <div class="text-xs text-gray-600">Convert text to ISO date format</div>
                      </div>
                    </label>
                  </div>
                </AccordionContent>
              </AccordionItem>

              <!-- Advanced -->
              <AccordionItem value="advanced">
                <AccordionTrigger class="text-sm">
                  <div class="flex items-center gap-2">
                    <span>⚡ Advanced</span>
                    <Badge v-if="editableField.transformations.js_code" variant="secondary" class="text-xs">
                      Custom JS
                    </Badge>
                  </div>
                </AccordionTrigger>
                <AccordionContent>
                  <div class="space-y-3 pt-2">
                    <div>
                      <Label class="text-sm font-medium mb-2 flex items-center gap-2">
                        <input
                          type="checkbox"
                          v-model="editableField.transformations.js_enabled"
                          :disabled="!editMode"
                          class="w-4 h-4 text-blue-600 rounded border-gray-300 focus:ring-2 focus:ring-blue-500"
                        />
                        <span>Custom JavaScript</span>
                      </Label>
                      <Textarea
                        v-model="editableField.transformations.js_code"
                        :disabled="!editMode || !editableField.transformations.js_enabled"
                        placeholder="// Transform the value
return value.toUpperCase();"
                        rows="4"
                        class="mt-1 font-mono text-xs"
                      />
                      <p class="text-xs text-gray-600 mt-1">Access extracted value via <code class="bg-gray-200 px-1 rounded">value</code> variable</p>
                    </div>
                  </div>
                </AccordionContent>
              </AccordionItem>
            </Accordion>

            <Separator class="my-4" />

            <div v-if="!editMode">
              <Button @click="emit('enableEdit')" variant="outline" class="w-full">
                ✏️ Edit Transformations
              </Button>
            </div>
            <div v-else class="flex gap-2">
              <Button @click="saveChanges" class="flex-1">
                ✓ Save Changes
              </Button>
              <Button @click="emit('cancelEdit')" variant="outline" class="flex-1">
                ✕ Cancel
              </Button>
            </div>
          </CardContent>
        </Card>
      </TabsContent>
    </Tabs>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'
import type { SelectedField, TestResult } from '../types'
import KeyValuePairSelector from './KeyValuePairSelector.vue'

// Shadcn Components
import { Button } from './ui/button'
import { Input } from './ui/input'
import { Label } from './ui/label'
import { Textarea } from './ui/textarea'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './ui/select'
import { Card, CardContent, CardHeader, CardTitle } from './ui/card'
import { Badge } from './ui/badge'
import { ScrollArea } from './ui/scroll-area'
import { Tabs, TabsContent, TabsList, TabsTrigger } from './ui/tabs'
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from './ui/accordion'
import { Separator } from './ui/separator'
import { Alert, AlertDescription } from './ui/alert'

interface Props {
  field: SelectedField
  tab: 'preview' | 'edit'
  editMode: boolean
  testResults: TestResult[]
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'switchTab': [tab: 'preview' | 'edit']
  'enableEdit': []
  'saveEdit': [field: Partial<SelectedField>]
  'cancelEdit': []
  'testSelector': [field: SelectedField]
  'scrollToResult': [result: TestResult]
}>()

const currentTab = ref<'preview' | 'config' | 'transform'>('preview')
const editableField = ref<SelectedField>(JSON.parse(JSON.stringify(props.field)))
const kvEditSelectorRef = ref<InstanceType<typeof KeyValuePairSelector> | null>(null)
const editingPairIndex = ref<number | null>(null)
const tempFieldName = ref('')

const currentEditingPair = computed(() => {
  if (editingPairIndex.value !== null && editingPairIndex.value >= 0 && editableField.value.attributes?.extractions) {
    return editableField.value.attributes.extractions[editingPairIndex.value]
  }
  return null
})

// Watch for field changes
watch(() => props.field, (newField) => {
  editableField.value = JSON.parse(JSON.stringify(newField))
}, { deep: true })

// Watch for edit mode changes
watch(() => props.editMode, (isEdit) => {
  if (!isEdit) {
    // Reset editable field when edit mode is cancelled
    editableField.value = JSON.parse(JSON.stringify(props.field))
  }
})

const getFieldTypeBadgeClass = (field: SelectedField) => {
  if (field.type === 'text') return 'border-blue-300 text-blue-700'
  if (field.type === 'attribute') return 'border-purple-300 text-purple-700'
  if (field.type === 'html') return 'border-pink-300 text-pink-700'
  return 'border-gray-300 text-gray-700'
}

const activeTransformationsCount = computed(() => {
  const t = editableField.value.transformations || {}
  let count = 0
  if (t.trim) count++
  if (t.lowercase) count++
  if (t.uppercase) count++
  if (t.regex_enabled && t.regex) count++
  if (t.replace_enabled && t.replace_find) count++
  if (t.prefix_enabled && t.prefix) count++
  if (t.suffix_enabled && t.suffix) count++
  if (t.parse_number) count++
  if (t.parse_date) count++
  if (t.js_enabled && t.js_code) count++
  return count
})

const textTransformCount = computed(() => {
  const t = editableField.value.transformations || {}
  return (t.trim ? 1 : 0) + (t.lowercase ? 1 : 0) + (t.uppercase ? 1 : 0)
})

const stringOpCount = computed(() => {
  const t = editableField.value.transformations || {}
  let count = 0
  if (t.regex_enabled && t.regex) count++
  if (t.replace_enabled && t.replace_find) count++
  if (t.prefix_enabled && t.prefix) count++
  if (t.suffix_enabled && t.suffix) count++
  return count
})

function updateEditableField(key: string, value: any) {
  (editableField.value as any)[key] = value
}

function saveChanges() {
  emit('saveEdit', editableField.value)
}

function saveFieldNameChange() {
  emit('saveEdit', editableField.value)
  emit('cancelEdit')
}

async function startEditPair(index: number) {
  editingPairIndex.value = index
  tempFieldName.value = ''
  
  // Wait for next tick to ensure component is rendered
  await nextTick()
  
  // Initialize the KeyValuePairSelector with existing data
  const pair = editableField.value.attributes?.extractions?.[index]
  
  if (pair) {
    // Use a longer timeout to ensure component is fully mounted
    setTimeout(() => {
      if (kvEditSelectorRef.value) {
        kvEditSelectorRef.value.initializeWithData({
          key_selector: pair.key_selector,
          value_selector: pair.value_selector,
          key_type: pair.key_type,
          value_type: pair.value_type,
          key_attribute: pair.key_attribute,
          value_attribute: pair.value_attribute
        })
      }
    }, 200)
  }
}

function startAddPair() {
  editingPairIndex.value = -1 // -1 means adding new
  tempFieldName.value = ''
}

function cancelPairEdit() {
  editingPairIndex.value = null
  tempFieldName.value = ''
}

function handleSavePairEdit(data: { fieldName: string; extractions: any[] }) {
  if (!editableField.value.attributes) {
    editableField.value.attributes = {}
  }
  if (!editableField.value.attributes.extractions) {
    editableField.value.attributes.extractions = []
  }

  if (editingPairIndex.value === -1) {
    // Adding new pair
    editableField.value.attributes.extractions.push(...data.extractions)
  } else if (editingPairIndex.value !== null) {
    // Editing existing pair
    editableField.value.attributes.extractions[editingPairIndex.value] = data.extractions[0]
  }

  // Save changes and reset editing state
  emit('saveEdit', editableField.value)
  editingPairIndex.value = null
  tempFieldName.value = ''
}

function deletePair(index: number) {
  if (!editableField.value.attributes?.extractions) return
  
  if (confirm('Are you sure you want to delete this extraction pair?')) {
    editableField.value.attributes.extractions.splice(index, 1)
    emit('saveEdit', editableField.value)
  }
}
</script>
