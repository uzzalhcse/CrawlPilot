<script setup lang="ts">
import { ref, watch } from 'vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import { Plus, Trash2 } from 'lucide-vue-next'
import type { ExtractionPair } from '@/types'

interface Props {
  modelValue: ExtractionPair[]
}

interface Emits {
  (e: 'update:modelValue', value: ExtractionPair[]): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const extractions = ref<ExtractionPair[]>(props.modelValue || [])

// Watch for prop changes and update local state
watch(() => props.modelValue, (newValue) => {
  console.log('🔄 [IndependentArrayManager] Props changed, updating local state')
  console.log('  - Old extractions count:', extractions.value.length)
  console.log('  - New extractions count:', newValue?.length || 0)
  console.log('  - New value:', newValue)
  
  if (newValue) {
    extractions.value = [...newValue] // Create new array for reactivity
    console.log('  - ✅ Local state updated')
  }
}, { deep: true, immediate: true })

function addExtraction() {
  const newExtraction: ExtractionPair = {
    key_selector: '',
    value_selector: '',
    key_type: 'text',
    value_type: 'text',
    transform: 'trim',
    limit: 0
  }
  extractions.value.push(newExtraction)
  emitUpdate()
}

function removeExtraction(index: number) {
  extractions.value.splice(index, 1)
  emitUpdate()
}

function updateExtraction(index: number, field: keyof ExtractionPair, value: any) {
  if (extractions.value[index]) {
    extractions.value[index][field] = value as never
    emitUpdate()
  }
}

function emitUpdate() {
  emit('update:modelValue', extractions.value)
}

const extractionTypes = [
  { label: 'Text', value: 'text' },
  { label: 'Attribute', value: 'attr' },
  { label: 'HTML', value: 'html' },
  { label: 'Href', value: 'href' },
  { label: 'Src', value: 'src' }
]

const transforms = [
  { label: 'None', value: 'none' },
  { label: 'Trim', value: 'trim' },
  { label: 'Lowercase', value: 'lowercase' },
  { label: 'Uppercase', value: 'uppercase' },
  { label: 'Extract Price', value: 'extract_price' },
  { label: 'Clean HTML', value: 'clean_html' }
]
</script>

<template>
  <div class="space-y-4">
    <!-- Help Banner -->
    <div class="p-3 bg-orange-50 border-2 border-orange-200 rounded-lg">
      <div class="text-sm font-semibold text-orange-900 mb-2">⚡ Independent Array Extraction</div>
      <div class="text-xs text-orange-700">
        <div class="mb-1">Extract key-value pairs from <strong>separate independent lists</strong> that are paired by index position.</div>
        <div class="mt-2 p-2 bg-white rounded border border-orange-300">
          <strong>Example:</strong> Keys in <code class="bg-orange-50 px-1 py-0.5 rounded">.spec-label</code> + 
          Values in <code class="bg-orange-50 px-1 py-0.5 rounded">.spec-value</code> → 
          <code class="bg-orange-50 px-1 py-0.5 rounded">[{"key": "color", "value": "black"}]</code>
        </div>
      </div>
    </div>

    <!-- Extraction Pairs -->
    <div v-if="extractions.length > 0" class="space-y-3">
      <div
        v-for="(extraction, index) in extractions"
        :key="index"
        class="border-2 border-orange-200 rounded-lg p-4 bg-white hover:shadow-md transition-shadow"
      >
        <div class="flex items-center justify-between mb-3">
          <div class="flex items-center gap-2">
            <div class="flex items-center justify-center w-6 h-6 rounded-full bg-orange-100 text-orange-700 text-xs font-semibold">
              {{ index + 1 }}
            </div>
            <span class="font-semibold text-sm">Extraction Pair {{ index + 1 }}</span>
          </div>
          <Button
            type="button"
            size="sm"
            variant="ghost"
            class="h-8 w-8 p-0 hover:bg-destructive/10 hover:text-destructive"
            @click="removeExtraction(index)"
          >
            <Trash2 class="h-4 w-4" />
          </Button>
        </div>

        <div class="grid gap-4">
          <!-- Key Selector -->
          <div class="space-y-2">
            <Label :for="`key-selector-${index}`" class="text-xs font-medium text-orange-900">
              Key Selector <span class="text-red-500">*</span>
            </Label>
            <Input
              :id="`key-selector-${index}`"
              :model-value="extraction.key_selector"
              @update:model-value="(val) => updateExtraction(index, 'key_selector', String(val))"
              placeholder=".spec-label, .product-attr-name, th"
              class="font-mono text-sm"
            />
            <p class="text-xs text-muted-foreground">CSS selector for the key/label elements</p>
          </div>

          <!-- Value Selector -->
          <div class="space-y-2">
            <Label :for="`value-selector-${index}`" class="text-xs font-medium text-orange-900">
              Value Selector <span class="text-red-500">*</span>
            </Label>
            <Input
              :id="`value-selector-${index}`"
              :model-value="extraction.value_selector"
              @update:model-value="(val) => updateExtraction(index, 'value_selector', String(val))"
              placeholder=".spec-value, .product-attr-value, td"
              class="font-mono text-sm"
            />
            <p class="text-xs text-muted-foreground">CSS selector for the value/data elements</p>
          </div>

          <!-- Key Type & Value Type (Side by Side) -->
          <div class="grid grid-cols-2 gap-3">
            <div class="space-y-2">
              <Label :for="`key-type-${index}`" class="text-xs font-medium">Key Type</Label>
              <Select
                :model-value="extraction.key_type"
                @update:model-value="(val) => updateExtraction(index, 'key_type', val as string)"
              >
                <SelectTrigger :id="`key-type-${index}`" class="text-sm">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="type in extractionTypes" :key="type.value" :value="type.value">
                    {{ type.label }}
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div class="space-y-2">
              <Label :for="`value-type-${index}`" class="text-xs font-medium">Value Type</Label>
              <Select
                :model-value="extraction.value_type"
                @update:model-value="(val) => updateExtraction(index, 'value_type', val as string)"
              >
                <SelectTrigger :id="`value-type-${index}`" class="text-sm">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="type in extractionTypes" :key="type.value" :value="type.value">
                    {{ type.label }}
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          <!-- Key Attribute (conditional) -->
          <div v-if="extraction.key_type === 'attr'" class="space-y-2">
            <Label :for="`key-attribute-${index}`" class="text-xs font-medium">Key Attribute Name</Label>
            <Input
              :id="`key-attribute-${index}`"
              :model-value="extraction.key_attribute || ''"
              @update:model-value="(val) => updateExtraction(index, 'key_attribute', String(val))"
              placeholder="property, data-key, name"
              class="text-sm"
            />
            <p class="text-xs text-muted-foreground">HTML attribute to extract from key elements</p>
          </div>

          <!-- Value Attribute (conditional) -->
          <div v-if="extraction.value_type === 'attr'" class="space-y-2">
            <Label :for="`value-attribute-${index}`" class="text-xs font-medium">Value Attribute Name</Label>
            <Input
              :id="`value-attribute-${index}`"
              :model-value="extraction.value_attribute || ''"
              @update:model-value="(val) => updateExtraction(index, 'value_attribute', String(val))"
              placeholder="content, data-value, href"
              class="text-sm"
            />
            <p class="text-xs text-muted-foreground">HTML attribute to extract from value elements</p>
          </div>

          <!-- Transform & Limit (Side by Side) -->
          <div class="grid grid-cols-2 gap-3">
            <div class="space-y-2">
              <Label :for="`transform-${index}`" class="text-xs font-medium">Transform</Label>
              <Select
                :model-value="extraction.transform || 'none'"
                @update:model-value="(val) => updateExtraction(index, 'transform', val === 'none' ? undefined : val as string)"
              >
                <SelectTrigger :id="`transform-${index}`" class="text-sm">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="transform in transforms" :key="transform.value" :value="transform.value">
                    {{ transform.label }}
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div class="space-y-2">
              <Label :for="`limit-${index}`" class="text-xs font-medium">Limit</Label>
              <Input
                :id="`limit-${index}`"
                type="number"
                :model-value="extraction.limit || 0"
                @update:model-value="(val) => updateExtraction(index, 'limit', Number(val))"
                placeholder="0 = unlimited"
                min="0"
                class="text-sm"
              />
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Add Button -->
    <Button
      type="button"
      variant="outline"
      size="sm"
      @click="addExtraction"
      class="w-full border-2 border-dashed border-orange-300 hover:bg-orange-50 hover:border-orange-400 text-orange-700"
    >
      <Plus class="h-4 w-4 mr-2" />
      Add Extraction Pair
    </Button>

    <!-- Empty State -->
    <div
      v-if="extractions.length === 0"
      class="text-center py-8 border-2 border-dashed border-orange-200 rounded-lg bg-orange-50/30"
    >
      <div class="text-orange-600 text-4xl mb-2">⚡</div>
      <p class="text-sm font-medium text-orange-900">No extraction pairs defined</p>
      <p class="text-xs text-orange-700 mt-1">Click "Add Extraction Pair" to start configuring independent array extraction</p>
    </div>
  </div>
</template>
