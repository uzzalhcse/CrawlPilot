<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { useBrowserProfilesStore } from '@/stores/browserProfiles'
import FieldInput from './FieldInput.vue'
import SelectInput from './SelectInput.vue'

interface ParamField {
  key: string
  label: string
  type: string
  required?: boolean
  placeholder?: string
  description?: string
  options?: Array<{label: string; value: string}>
  defaultValue?: any
  showWhen?: { field: string; value: string | string[] } // Conditional visibility
}

interface Props {
  params: Record<string, any>
  schema: ParamField[]
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:params': [key: string, value: any]
}>()

const profilesStore = useBrowserProfilesStore()
onMounted(() => {
  if (profilesStore.profiles.length === 0) {
    profilesStore.fetchProfiles()
  }
})

// Filter schema based on conditional visibility
const visibleFields = computed(() => {
  return props.schema.filter(field => {
    if (!field.showWhen) return true
    const currentValue = props.params?.[field.showWhen.field]
    if (Array.isArray(field.showWhen.value)) {
      return field.showWhen.value.includes(currentValue)
    }
    return currentValue === field.showWhen.value
  })
})

// Get profiles filtered by the current driver selection
const getFilteredProfiles = (driverType: string) => {
  if (!driverType || driverType === 'default') {
    return profilesStore.profiles
  }
  return profilesStore.profiles.filter(p => p.driver_type === driverType)
}
</script>

<template>
  <div class="space-y-4">
    <div 
      v-for="field in visibleFields" 
      :key="field.key"
      class="space-y-2"
    >
      <Label :for="`param-${field.key}`">
        {{ field.label }}
        <span v-if="field.required" class="text-red-500">*</span>
      </Label>
      
      <!-- Profile Select (for browser_profile_id) -->
      <template v-if="field.type === 'profile_select'">
        <Select
          :model-value="params?.[field.key] || 'workflow_default'"
          @update:model-value="emit('update:params', field.key, $event === 'workflow_default' ? null : $event)"
        >
          <SelectTrigger>
            <SelectValue placeholder="Use workflow default">
              {{ 
                params?.[field.key] && params[field.key] !== 'workflow_default'
                  ? (profilesStore.profiles.find(p => p.id === params[field.key])?.name || 'Unknown Profile')
                  : 'Use workflow default' 
              }}
            </SelectValue>
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="workflow_default">Use workflow default</SelectItem>
            <template v-for="profile in getFilteredProfiles(params?.driver)" :key="profile.id">
              <SelectItem :value="profile.id">
                <div class="flex items-center gap-2">
                  <span class="w-2 h-2 rounded-full" :class="profile.status === 'active' ? 'bg-green-500' : 'bg-gray-300'"></span>
                  {{ profile.name }}
                  <span class="text-xs text-muted-foreground">({{ profile.browser_type }})</span>
                </div>
              </SelectItem>
            </template>
          </SelectContent>
        </Select>
      </template>
      
      <!-- Select Input -->
      <SelectInput
        v-else-if="field.type === 'select' && field.options"
        :id="`param-${field.key}`"
        :model-value="params?.[field.key] || field.defaultValue"
        :options="field.options"
        :placeholder="field.placeholder"
        @update:model-value="emit('update:params', field.key, $event)"
      />
      
      <!-- Boolean as separate row for better UX -->
      <div v-else-if="field.type === 'boolean'" class="flex items-center space-x-2">
        <FieldInput
          type="boolean"
          :model-value="params[field.key] ?? field.defaultValue ?? false"
          @update:model-value="emit('update:params', field.key, $event)"
        />
        <Label class="font-normal text-sm text-muted-foreground">
          {{ field.description }}
        </Label>
      </div>
      
      <!-- Other Inputs -->
      <FieldInput
        v-else
        :id="`param-${field.key}`"
        :type="field.type as any"
        :model-value="params?.[field.key] || field.defaultValue || ''"
        :placeholder="field.placeholder"
        @update:model-value="emit('update:params', field.key, $event)"
      />
      
      <!-- Description (not for boolean as it's already inline) -->
      <p v-if="field.description && field.type !== 'boolean'" class="text-xs text-muted-foreground">
        {{field.description }}
      </p>
    </div>
  </div>
</template>

