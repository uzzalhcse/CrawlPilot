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
import { 
  Plus, 
  Trash2, 
  ChevronDown, 
  ChevronUp,
  Clock,
  MousePointer,
  Eye,
  ArrowDown,
  Zap
} from 'lucide-vue-next'
import FieldInput from './FieldInput.vue'
import SelectInput from './SelectInput.vue'

interface ActionNode {
  id: string
  type: string
  name: string
  params: Record<string, any>
}

interface Props {
  modelValue: ActionNode[] | string | undefined
}

const props = defineProps<Props>()
const emit = defineEmits<{
  'update:modelValue': [value: ActionNode[]]
}>()

// Parse incoming value (could be JSON string or array)
const actions = ref<ActionNode[]>([])
const isDragOver = ref(false)

watch(() => props.modelValue, (newVal) => {
  if (!newVal) {
    actions.value = []
  } else if (typeof newVal === 'string') {
    try {
      actions.value = JSON.parse(newVal)
    } catch {
      actions.value = []
    }
  } else if (Array.isArray(newVal)) {
    actions.value = newVal
  }
}, { immediate: true })

// Track which actions are expanded
const expandedActions = ref<Set<string>>(new Set())

// Available action types
const actionTypes = [
  { 
    type: 'wait_for', 
    label: 'Wait For Condition', 
    icon: Eye,
    color: 'text-blue-500',
    bgColor: 'bg-blue-500/20',
    defaultParams: { condition: 'selector', selector: '', state: 'visible', timeout: 10000 }
  },
  { 
    type: 'wait', 
    label: 'Wait (Duration)', 
    icon: Clock,
    color: 'text-purple-500',
    bgColor: 'bg-purple-500/20',
    defaultParams: { duration: 1000 }
  },
  { 
    type: 'click', 
    label: 'Click Element', 
    icon: MousePointer,
    color: 'text-green-500',
    bgColor: 'bg-green-500/20',
    defaultParams: { selector: '' }
  },
  { 
    type: 'scroll', 
    label: 'Scroll', 
    icon: ArrowDown,
    color: 'text-orange-500',
    bgColor: 'bg-orange-500/20',
    defaultParams: { selector: '', to_bottom: false }
  },
  { 
    type: 'hover', 
    label: 'Hover', 
    icon: MousePointer,
    color: 'text-pink-500',
    bgColor: 'bg-pink-500/20',
    defaultParams: { selector: '' }
  }
]

function generateId() {
  return `action_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
}

function handleAddAction(typeValue: any) {
  if (typeof typeValue !== 'string' || !typeValue) return
  addActionByType(typeValue)
}

function addActionByType(typeValue: string) {
  const actionType = actionTypes.find(a => a.type === typeValue)
  if (!actionType) return
  
  const newAction: ActionNode = {
    id: generateId(),
    type: actionType.type,
    name: actionType.label,
    params: { ...actionType.defaultParams }
  }
  actions.value = [...actions.value, newAction]
  expandedActions.value.add(newAction.id)
  emitUpdate()
}

// Drag and Drop handlers
function handleDragOver(event: DragEvent) {
  event.preventDefault()
  isDragOver.value = true
}

function handleDragLeave() {
  isDragOver.value = false
}

function handleDrop(event: DragEvent) {
  event.preventDefault()
  isDragOver.value = false
  
  const data = event.dataTransfer?.getData('application/action-type')
  if (data) {
    addActionByType(data)
  }
}

function removeAction(id: string) {
  actions.value = actions.value.filter(a => a.id !== id)
  expandedActions.value.delete(id)
  emitUpdate()
}

function updateActionParam(id: string, key: string, value: any) {
  const action = actions.value.find(a => a.id === id)
  if (action) {
    action.params[key] = value
    emitUpdate()
  }
}

function updateActionName(id: string, name: string) {
  const action = actions.value.find(a => a.id === id)
  if (action) {
    action.name = name
    emitUpdate()
  }
}

function toggleExpanded(id: string) {
  if (expandedActions.value.has(id)) {
    expandedActions.value.delete(id)
  } else {
    expandedActions.value.add(id)
  }
}

function moveAction(index: number, direction: 'up' | 'down') {
  const newIndex = direction === 'up' ? index - 1 : index + 1
  if (newIndex < 0 || newIndex >= actions.value.length) return
  
  const newActions = [...actions.value]
  const [removed] = newActions.splice(index, 1)
  newActions.splice(newIndex, 0, removed)
  actions.value = newActions
  emitUpdate()
}

function emitUpdate() {
  emit('update:modelValue', [...actions.value])
}

function getActionIcon(type: string) {
  return actionTypes.find(a => a.type === type)?.icon || Zap
}

function getActionColor(type: string) {
  return actionTypes.find(a => a.type === type)?.color || 'text-gray-500'
}

// Parameter schemas for each action type
const paramSchemas: Record<string, Array<{ key: string; label: string; type: string; placeholder?: string; options?: Array<{label: string; value: string}> }>> = {
  wait_for: [
    { 
      key: 'condition', 
      label: 'Condition', 
      type: 'select',
      options: [
        { label: 'Selector Visible', value: 'selector' },
        { label: 'Network Idle', value: 'network_idle' },
        { label: 'Text on Page', value: 'text' }
      ]
    },
    { key: 'selector', label: 'CSS Selector', type: 'text', placeholder: '.price, #content' },
    { 
      key: 'state', 
      label: 'State', 
      type: 'select',
      options: [
        { label: 'Visible', value: 'visible' },
        { label: 'Hidden', value: 'hidden' },
        { label: 'Attached', value: 'attached' }
      ]
    },
    { key: 'timeout', label: 'Timeout (ms)', type: 'number', placeholder: '10000' }
  ],
  wait: [
    { key: 'duration', label: 'Duration (ms)', type: 'number', placeholder: '1000' }
  ],
  click: [
    { key: 'selector', label: 'CSS Selector', type: 'text', placeholder: '.button, #submit' }
  ],
  scroll: [
    { key: 'selector', label: 'Scroll to Element', type: 'text', placeholder: '.target-section' },
    { key: 'to_bottom', label: 'Scroll to Bottom', type: 'boolean' }
  ],
  hover: [
    { key: 'selector', label: 'CSS Selector', type: 'text', placeholder: '.tooltip-trigger' }
  ]
}
</script>

<template>
  <div class="space-y-3">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-2">
        <Zap class="w-4 h-4 text-amber-500" />
        <span class="text-sm font-medium">Pre-Extraction Actions</span>
        <span v-if="actions.length > 0" class="text-xs px-1.5 py-0.5 rounded-full bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400">
          {{ actions.length }}
        </span>
      </div>
      
      <!-- Add Action Select -->
      <Select @update:modelValue="handleAddAction">
        <SelectTrigger class="h-7 w-[140px] text-xs">
          <Plus class="w-3 h-3 mr-1" />
          <SelectValue placeholder="Add Action" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem v-for="actionType in actionTypes" :key="actionType.type" :value="actionType.type">
            <div class="flex items-center gap-2">
              <component :is="actionType.icon" :class="['w-3 h-3', actionType.color]" />
              {{ actionType.label }}
            </div>
          </SelectItem>
        </SelectContent>
      </Select>
    </div>
    
    <!-- Draggable Action Chips -->
    <div class="flex flex-wrap gap-1.5 p-2 bg-muted/30 rounded-lg border border-dashed">
      <div class="text-[10px] text-muted-foreground w-full mb-1">Drag actions below or click to add:</div>
      <div 
        v-for="actionType in actionTypes" 
        :key="actionType.type"
        draggable="true"
        @dragstart="(e) => { e.dataTransfer?.setData('application/action-type', actionType.type) }"
        class="flex items-center gap-1 px-2 py-1 rounded-md cursor-grab active:cursor-grabbing transition-all hover:scale-105"
        :class="actionType.bgColor"
        @click="addActionByType(actionType.type)"
      >
        <component :is="actionType.icon" :class="['w-3 h-3', actionType.color]" />
        <span class="text-[10px] font-medium">{{ actionType.label }}</span>
      </div>
    </div>
    
    <!-- Drop Zone / Empty State -->
    <div 
      v-if="actions.length === 0" 
      class="border-2 border-dashed rounded-lg p-4 text-center transition-colors"
      :class="isDragOver ? 'border-amber-500 bg-amber-500/5' : 'border-border'"
      @dragover="handleDragOver"
      @dragleave="handleDragLeave"
      @drop="handleDrop"
    >
      <Zap v-if="isDragOver" class="w-6 h-6 mx-auto mb-2 text-amber-500 animate-bounce" />
      <p class="text-sm text-muted-foreground">
        {{ isDragOver ? 'Drop to add action!' : 'Drag actions here or click chips above' }}
      </p>
    </div>
    
    <!-- Action Cards with Drop Zone -->
    <div 
      v-else 
      class="space-y-2 p-2 rounded-lg transition-colors"
      :class="isDragOver ? 'bg-amber-500/5 border-2 border-dashed border-amber-500' : ''"
      @dragover="handleDragOver"
      @dragleave="handleDragLeave"
      @drop="handleDrop"
    >
      <!-- Connection Line Start -->
      <div class="flex items-center gap-2 text-xs text-muted-foreground px-2">
        <div class="w-2 h-2 rounded-full bg-amber-500"></div>
        <span>Start</span>
      </div>
      
      <div 
        v-for="(action, index) in actions" 
        :key="action.id"
        class="relative"
      >
        <!-- Connection Line -->
        <div class="absolute left-3 -top-2 w-0.5 h-2 bg-border"></div>
        
        <!-- Action Card -->
        <div class="border rounded-lg bg-background overflow-hidden ml-1 border-l-4"
          :class="{
            'border-l-blue-500': action.type === 'wait_for',
            'border-l-purple-500': action.type === 'wait',
            'border-l-green-500': action.type === 'click',
            'border-l-orange-500': action.type === 'scroll',
            'border-l-pink-500': action.type === 'hover'
          }"
        >
          <!-- Action Header -->
          <div 
            class="flex items-center gap-2 p-2 bg-muted/30 cursor-pointer hover:bg-muted/50 transition-colors"
            @click="toggleExpanded(action.id)"
          >
            <!-- Reorder Buttons -->
            <div class="flex flex-col -my-1">
              <Button 
                v-if="index > 0"
                size="sm" 
                variant="ghost" 
                class="h-4 w-4 p-0 hover:bg-background"
                @click.stop="moveAction(index, 'up')"
              >
                <ChevronUp class="w-3 h-3" />
              </Button>
              <Button 
                v-if="index < actions.length - 1"
                size="sm" 
                variant="ghost" 
                class="h-4 w-4 p-0 hover:bg-background"
                @click.stop="moveAction(index, 'down')"
              >
                <ChevronDown class="w-3 h-3" />
              </Button>
            </div>
            
            <!-- Step Number -->
            <div class="flex items-center justify-center w-5 h-5 rounded-full bg-primary/10 text-primary text-xs font-bold shrink-0">
              {{ index + 1 }}
            </div>
            
            <!-- Icon -->
            <component :is="getActionIcon(action.type)" :class="['w-4 h-4 shrink-0', getActionColor(action.type)]" />
            
            <!-- Name -->
            <Input 
              :model-value="action.name"
              @update:model-value="updateActionName(action.id, String($event))"
              @click.stop
              class="h-6 text-xs font-medium flex-1 bg-transparent border-none shadow-none focus-visible:ring-0 px-1"
              :placeholder="action.type"
            />
            
            <!-- Type Badge -->
            <span class="text-[10px] font-mono px-1.5 py-0.5 rounded bg-muted text-muted-foreground shrink-0">
              {{ action.type }}
            </span>
            
            <!-- Delete -->
            <Button 
              size="sm" 
              variant="ghost" 
              class="h-6 w-6 p-0 hover:bg-destructive/10 hover:text-destructive shrink-0"
              @click.stop="removeAction(action.id)"
            >
              <Trash2 class="w-3 h-3" />
            </Button>
            
            <!-- Expand/Collapse -->
            <ChevronDown 
              :class="['w-4 h-4 transition-transform shrink-0', expandedActions.has(action.id) ? 'rotate-180' : '']" 
            />
          </div>
          
          <!-- Action Parameters -->
          <div v-if="expandedActions.has(action.id)" class="p-3 border-t space-y-3">
            <div 
              v-for="param in paramSchemas[action.type] || []" 
              :key="param.key"
              class="space-y-1"
            >
              <Label class="text-xs">{{ param.label }}</Label>
              
              <SelectInput
                v-if="param.type === 'select' && param.options"
                :model-value="action.params[param.key]"
                :options="param.options"
                @update:model-value="updateActionParam(action.id, param.key, $event)"
              />
              
              <FieldInput
                v-else-if="param.type === 'boolean'"
                type="boolean"
                :model-value="action.params[param.key]"
                @update:model-value="updateActionParam(action.id, param.key, $event)"
              />
              
              <FieldInput
                v-else
                :type="param.type as any"
                :model-value="action.params[param.key]"
                :placeholder="param.placeholder"
                @update:model-value="updateActionParam(action.id, param.key, $event)"
              />
            </div>
          </div>
        </div>
        
        <!-- Connection Line After -->
        <div class="absolute left-3 -bottom-2 w-0.5 h-2 bg-border"></div>
      </div>
      
      <!-- Connection to Extract -->
      <div class="flex items-center gap-2 text-xs text-muted-foreground px-2 pt-1">
        <div class="w-2 h-2 rounded-full bg-emerald-500"></div>
        <span>Extract Field</span>
      </div>
    </div>
  </div>
</template>
