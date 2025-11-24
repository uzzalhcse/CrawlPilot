<script setup lang="ts">
import { Plus, Trash2 } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'

interface ExtractionItem {
  key_selector: string
  value_selector: string
  key_type: string
  value_type: string
  transform: string
}

interface Props {
  modelValue: ExtractionItem[]
}

const props = defineProps<Props>()
const emit = defineEmits<{
  (e: 'update:modelValue', value: ExtractionItem[]): void
}>()

function addItem() {
  const newItem: ExtractionItem = {
    key_selector: '',
    value_selector: '',
    key_type: 'text',
    value_type: 'text',
    transform: ''
  }
  emit('update:modelValue', [...props.modelValue, newItem])
}

function removeItem(index: number) {
  const newValue = [...props.modelValue]
  newValue.splice(index, 1)
  emit('update:modelValue', newValue)
}

function updateItem(index: number, field: keyof ExtractionItem, value: string) {
  const newValue = [...props.modelValue]
  newValue[index] = { ...newValue[index], [field]: value }
  emit('update:modelValue', newValue)
}
</script>

<template>
  <div class="space-y-4">
    <div v-if="modelValue.length === 0" class="text-center p-8 border-2 border-dashed rounded-lg border-muted-foreground/25">
      <p class="text-sm text-muted-foreground mb-2">No extractions defined</p>
      <Button variant="outline" size="sm" @click="addItem">
        <Plus class="w-4 h-4 mr-2" />
        Add Extraction Pair
      </Button>
    </div>

    <div v-else class="space-y-3">
      <Card v-for="(item, index) in modelValue" :key="index" class="relative group">
        <CardContent class="p-3 space-y-3">
          <div class="absolute right-2 top-2 opacity-0 group-hover:opacity-100 transition-opacity">
            <Button variant="ghost" size="icon" class="h-6 w-6 text-destructive hover:bg-destructive/10" @click="removeItem(index)">
              <Trash2 class="w-3 h-3" />
            </Button>
          </div>

          <!-- Key Config -->
          <div class="grid grid-cols-2 gap-3">
            <div class="space-y-1.5">
              <Label class="text-xs text-muted-foreground">Key Selector</Label>
              <Input 
                :model-value="item.key_selector"
                @update:model-value="(val) => updateItem(index, 'key_selector', String(val))"
                placeholder=".key-class"
                class="h-8 text-xs"
              />
            </div>
            <div class="space-y-1.5">
              <Label class="text-xs text-muted-foreground">Key Type</Label>
              <Select 
                :model-value="item.key_type"
                @update:model-value="(val) => updateItem(index, 'key_type', String(val))"
              >
                <SelectTrigger class="h-8 text-xs">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="text">Text</SelectItem>
                  <SelectItem value="html">HTML</SelectItem>
                  <SelectItem value="attribute">Attribute</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          <!-- Value Config -->
          <div class="grid grid-cols-2 gap-3">
            <div class="space-y-1.5">
              <Label class="text-xs text-muted-foreground">Value Selector</Label>
              <Input 
                :model-value="item.value_selector"
                @update:model-value="(val) => updateItem(index, 'value_selector', String(val))"
                placeholder=".value-class"
                class="h-8 text-xs"
              />
            </div>
            <div class="space-y-1.5">
              <Label class="text-xs text-muted-foreground">Value Type</Label>
              <Select 
                :model-value="item.value_type"
                @update:model-value="(val) => updateItem(index, 'value_type', String(val))"
              >
                <SelectTrigger class="h-8 text-xs">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="text">Text</SelectItem>
                  <SelectItem value="html">HTML</SelectItem>
                  <SelectItem value="attribute">Attribute</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          <!-- Transform -->
          <div class="space-y-1.5">
            <Label class="text-xs text-muted-foreground">Transform</Label>
            <Select 
              :model-value="item.transform"
              @update:model-value="(val) => updateItem(index, 'transform', String(val))"
            >
              <SelectTrigger class="h-8 text-xs">
                <SelectValue placeholder="None" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="none">None</SelectItem>
                <SelectItem value="trim">Trim Whitespace</SelectItem>
                <SelectItem value="lowercase">Lowercase</SelectItem>
                <SelectItem value="uppercase">Uppercase</SelectItem>
                <SelectItem value="number">To Number</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </CardContent>
      </Card>

      <Button variant="outline" size="sm" class="w-full" @click="addItem">
        <Plus class="w-4 h-4 mr-2" />
        Add Another Pair
      </Button>
    </div>
  </div>
</template>
