<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Field } from '../types'
import {
  Drawer,
  DrawerClose,
  DrawerContent,
  DrawerDescription,
  DrawerFooter,
  DrawerHeader,
  DrawerTitle,
} from '@/components/ui/drawer'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card } from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { 
  Code, 
  Eye, 
  Copy, 
  Check, 
  ChevronDown, 
  ChevronRight, 
  List, 
  Type
} from 'lucide-vue-next'
import { toast } from 'vue-sonner'

const props = defineProps<{
  fields: Field[]
}>()

const isOpen = defineModel<boolean>('open', { default: false })
const activeTab = ref<'visual' | 'json'>('visual')
const expandedArrays = ref<Record<string, boolean>>({})
const copied = ref(false)

// Toggle array expansion
function toggleArray(fieldName: string) {
  expandedArrays.value[fieldName] = !expandedArrays.value[fieldName]
}

import { watch } from 'vue'
watch(isOpen, (val) => {
  if (val) {
    console.log('[DataPreview] Opened with fields:', props.fields)
    console.log('[DataPreview] Extracted data:', extractedData.value)
  }
})

// Extract data based on field configurations
const extractedData = computed(() => {
  if (props.fields.length === 0) return {}
  
  const row: Record<string, any> = {}
  props.fields.forEach(field => {
    row[field.name] = extractFieldValue(field)
  })
  
  return row
})

function extractFieldValue(field: Field): any {
  try {
    if (field.fieldType === 'single') {
      const elements = document.querySelectorAll(field.config.selector)
      if (elements.length === 0) return field.config.multiple ? [] : ''
      
      const extractValue = (el: Element) => {
        let val = ''
        if (field.config.type === 'text') {
          val = el.textContent?.trim() || ''
        } else if (field.config.type === 'attr' && field.config.attribute) {
          val = el.getAttribute(field.config.attribute) || ''
        } else if (field.config.type === 'html') {
          val = el.innerHTML
        }
        
        // Apply transform
        if (field.config.transform === 'trim') val = val.trim()
        if (field.config.transform === 'lowercase') val = val.toLowerCase()
        if (field.config.transform === 'uppercase') val = val.toUpperCase()
        
        return val
      }
      
      if (field.config.multiple) {
        return Array.from(elements).map(extractValue)
      } else {
        return extractValue(elements[0])
      }
    } else if (field.fieldType === 'keyvalue') {
      const results = field.pairs.map(pair => {
        const keyEls = document.querySelectorAll(pair.config.key_selector)
        const valEls = document.querySelectorAll(pair.config.value_selector)
        
        const key = keyEls.length > 0 ? keyEls[0].textContent?.trim() : 'Missing Key'
        const value = valEls.length > 0 ? valEls[0].textContent?.trim() : 'Missing Value'
        
        return { key, value }
      })
      
      if (field.config.output_format === 'object') {
        return results.reduce((acc, curr) => {
          if (curr.key) acc[curr.key] = curr.value
          return acc
        }, {} as Record<string, any>)
      }
      
      return results
    }
  } catch (e) {
    return 'Error'
  }
}

function copyJSON() {
  navigator.clipboard.writeText(JSON.stringify(extractedData.value, null, 2))
  copied.value = true
  toast.success('JSON copied to clipboard')
  setTimeout(() => copied.value = false, 2000)
}
</script>

<template>
  <Drawer v-model:open="isOpen">
    <DrawerContent class="max-h-[90vh] flex flex-col">
      <div class="mx-auto w-full max-w-4xl flex flex-col h-full">
        <DrawerHeader class="flex-none">
          <div class="flex items-center justify-between">
            <div>
              <DrawerTitle>Data Preview</DrawerTitle>
              <DrawerDescription>
                Review your extracted data structure
              </DrawerDescription>
            </div>
            
            <!-- Tab Switcher -->
            <div class="flex bg-muted p-1 rounded-lg">
              <button
                @click="activeTab = 'visual'"
                class="px-3 py-1.5 text-sm font-medium rounded-md transition-all flex items-center gap-2"
                :class="activeTab === 'visual' ? 'bg-background shadow-sm text-foreground' : 'text-muted-foreground hover:text-foreground'"
              >
                <Eye class="w-4 h-4" />
                Visual
              </button>
              <button
                @click="activeTab = 'json'"
                class="px-3 py-1.5 text-sm font-medium rounded-md transition-all flex items-center gap-2"
                :class="activeTab === 'json' ? 'bg-background shadow-sm text-foreground' : 'text-muted-foreground hover:text-foreground'"
              >
                <Code class="w-4 h-4" />
                JSON
              </button>
            </div>
          </div>
        </DrawerHeader>
        
        <div class="flex-1 overflow-y-auto px-4 pb-4">
          <div v-if="fields.length === 0" class="flex flex-col items-center justify-center h-64 text-muted-foreground">
            <div class="p-4 rounded-full bg-muted mb-4">
              <List class="w-8 h-8 opacity-50" />
            </div>
            <p>No fields configured yet.</p>
            <p class="text-sm">Add fields to see extracted data.</p>
          </div>
          
          <!-- Visual View -->
          <div v-else-if="activeTab === 'visual'" class="space-y-6">
            <div v-for="field in fields" :key="field.id" class="space-y-2">
              
              <!-- Field Header -->
              <div class="flex items-center gap-2">
                <Badge variant="outline" class="font-mono text-[10px] uppercase tracking-wider opacity-70">
                  {{ field.fieldType === 'keyvalue' ? 'Group' : (field.config.multiple ? 'List' : 'Text') }}
                </Badge>
                <h3 class="font-medium text-lg">{{ field.name }}</h3>
              </div>

              <!-- Content based on type -->
              <Card class="overflow-hidden border-muted">
                <!-- Case 1: Single Value -->
                <div v-if="field.fieldType === 'single' && !field.config.multiple" class="p-4 bg-card">
                  <div class="flex items-start gap-3">
                    <Type class="w-5 h-5 text-muted-foreground mt-0.5" />
                    <span class="text-foreground/90 whitespace-pre-wrap">{{ extractedData[field.name] }}</span>
                  </div>
                </div>

                <!-- Case 2: Array (List) -->
                <div v-else-if="field.fieldType === 'single' && field.config.multiple">
                  <div 
                    class="flex items-center justify-between p-3 bg-muted/30 cursor-pointer hover:bg-muted/50 transition-colors"
                    @click="toggleArray(field.name)"
                  >
                    <div class="flex items-center gap-2">
                      <List class="w-4 h-4 text-muted-foreground" />
                      <span class="text-sm font-medium">{{ (extractedData[field.name] as any[]).length }} items</span>
                    </div>
                    <component :is="expandedArrays[field.name] ? ChevronDown : ChevronRight" class="w-4 h-4 text-muted-foreground" />
                  </div>
                  
                  <div v-if="expandedArrays[field.name]" class="divide-y divide-border border-t">
                    <div 
                      v-for="(item, idx) in extractedData[field.name]" 
                      :key="idx"
                      class="p-3 text-sm flex gap-3 hover:bg-muted/10"
                    >
                      <span class="text-muted-foreground font-mono text-xs w-6 pt-0.5">{{ idx + 1 }}.</span>
                      <span class="break-words break-all">{{ item }}</span>
                    </div>
                  </div>
                </div>

                <!-- Case 3: Key-Value Group -->
                <div v-else-if="field.fieldType === 'keyvalue'">
                  <div v-if="field.config.output_format === 'array'">
                    <Table>
                      <TableHeader>
                        <TableRow class="hover:bg-transparent">
                          <TableHead class="w-1/3">Key</TableHead>
                          <TableHead>Value</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        <TableRow v-for="(pair, idx) in extractedData[field.name]" :key="idx">
                          <TableCell class="font-medium text-blue-700 dark:text-blue-400 align-top break-words">
                            {{ pair.key }}
                          </TableCell>
                          <TableCell class="align-top break-words break-all">
                            {{ pair.value }}
                          </TableCell>
                        </TableRow>
                      </TableBody>
                    </Table>
                  </div>
                  
                  <!-- Object Format -->
                  <div v-else>
                    <Table>
                      <TableHeader>
                        <TableRow class="hover:bg-transparent">
                          <TableHead class="w-1/3">Key</TableHead>
                          <TableHead>Value</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        <TableRow v-for="(value, key) in extractedData[field.name]" :key="key">
                          <TableCell class="font-medium text-blue-700 dark:text-blue-400 align-top break-words">
                            {{ key }}
                          </TableCell>
                          <TableCell class="align-top break-words break-all">
                            {{ value }}
                          </TableCell>
                        </TableRow>
                      </TableBody>
                    </Table>
                  </div>
                </div>
              </Card>
            </div>
          </div>

          <!-- JSON View -->
          <div v-else-if="activeTab === 'json'" class="h-full relative">
            <Card class="h-full bg-slate-950 text-slate-50 overflow-hidden flex flex-col">
              <div class="flex justify-between items-center p-2 border-b border-slate-800 bg-slate-900">
                <span class="text-xs text-slate-400 font-mono px-2">output.json</span>
                <Button variant="ghost" size="sm" class="h-7 text-xs hover:bg-slate-800 text-slate-300" @click="copyJSON">
                  <component :is="copied ? Check : Copy" class="w-3.5 h-3.5 mr-2" />
                  {{ copied ? 'Copied!' : 'Copy JSON' }}
                </Button>
              </div>
              <div class="flex-1 overflow-auto p-4">
                <pre class="font-mono text-sm leading-relaxed">{{ JSON.stringify(extractedData, null, 2) }}</pre>
              </div>
            </Card>
          </div>
        </div>
        
        <DrawerFooter class="flex-none border-t pt-4">
          <DrawerClose as-child>
            <Button variant="outline" class="w-full sm:w-auto self-end">Close Preview</Button>
          </DrawerClose>
        </DrawerFooter>
      </div>
    </DrawerContent>
  </Drawer>
</template>
