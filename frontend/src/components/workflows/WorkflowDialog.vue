<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useWorkflowsStore } from '@/stores/workflows'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import { Loader2 } from 'lucide-vue-next'
import type { Workflow } from '@/types'

interface Props {
  open: boolean
  mode: 'create' | 'edit'
  workflow?: Workflow | null
}

const props = defineProps<Props>()
const emit = defineEmits<{
  'update:open': [value: boolean]
}>()

const workflowsStore = useWorkflowsStore()

const formData = ref({
  name: '',
  description: '',
  status: 'draft' as 'draft' | 'active' | 'inactive',
  config: {
    start_urls: [''],
    max_depth: 3,
    rate_limit_delay: 2000,
    storage: {
      type: 'database' as const
    },
    url_discovery: [],
    data_extraction: []
  }
})

const isSubmitting = ref(false)

const dialogTitle = computed(() => 
  props.mode === 'create' ? 'Create New Workflow' : 'Edit Workflow'
)

const dialogDescription = computed(() => 
  props.mode === 'create' 
    ? 'Create a new web crawling workflow' 
    : 'Update workflow configuration'
)

watch(() => props.open, (newValue) => {
  if (newValue) {
    if (props.mode === 'edit' && props.workflow) {
      formData.value = {
        name: props.workflow.name,
        description: props.workflow.description,
        status: props.workflow.status,
        config: JSON.parse(JSON.stringify(props.workflow.config))
      }
    } else {
      resetForm()
    }
  }
})

const resetForm = () => {
  formData.value = {
    name: '',
    description: '',
    status: 'draft',
    config: {
      start_urls: [''],
      max_depth: 3,
      rate_limit_delay: 2000,
      storage: {
        type: 'database'
      },
      url_discovery: [],
      data_extraction: []
    }
  }
}

const addUrl = () => {
  formData.value.config.start_urls.push('')
}

const removeUrl = (index: number) => {
  formData.value.config.start_urls.splice(index, 1)
}

const handleSubmit = async () => {
  isSubmitting.value = true
  try {
    // Filter out empty URLs
    const cleanedConfig = {
      ...formData.value.config,
      start_urls: formData.value.config.start_urls.filter(url => url.trim() !== '')
    }

    if (props.mode === 'create') {
      await workflowsStore.createWorkflow({
        name: formData.value.name,
        description: formData.value.description,
        config: cleanedConfig
      })
    } else if (props.workflow) {
      await workflowsStore.updateWorkflow(props.workflow.id, {
        name: formData.value.name,
        description: formData.value.description,
        status: formData.value.status,
        config: cleanedConfig
      })
    }
    
    emit('update:open', false)
    resetForm()
  } catch (error) {
    console.error('Failed to save workflow:', error)
  } finally {
    isSubmitting.value = false
  }
}

const handleClose = () => {
  if (!isSubmitting.value) {
    emit('update:open', false)
    resetForm()
  }
}
</script>

<template>
  <Dialog :open="open" @update:open="handleClose">
    <DialogContent class="max-w-2xl max-h-[90vh] overflow-y-auto">
      <DialogHeader>
        <DialogTitle>{{ dialogTitle }}</DialogTitle>
        <DialogDescription>{{ dialogDescription }}</DialogDescription>
      </DialogHeader>

      <form @submit.prevent="handleSubmit" class="space-y-6">
        <!-- Basic Info -->
        <div class="space-y-4">
          <div class="space-y-2">
            <Label for="name">Workflow Name *</Label>
            <Input
              id="name"
              v-model="formData.name"
              placeholder="My Awesome Crawler"
              required
            />
          </div>

          <div class="space-y-2">
            <Label for="description">Description</Label>
            <Textarea
              id="description"
              v-model="formData.description"
              placeholder="Describe what this workflow does..."
              rows="3"
            />
          </div>

          <div v-if="mode === 'edit'" class="space-y-2">
            <Label for="status">Status</Label>
            <Select v-model="formData.status">
              <SelectTrigger id="status">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="draft">Draft</SelectItem>
                <SelectItem value="active">Active</SelectItem>
                <SelectItem value="inactive">Inactive</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        <!-- Configuration -->
        <div class="space-y-4 border-t pt-4">
          <h3 class="text-lg font-semibold">Configuration</h3>
          
          <!-- Start URLs -->
          <div class="space-y-2">
            <Label>Start URLs *</Label>
            <div 
              v-for="(url, index) in formData.config.start_urls" 
              :key="index"
              class="flex gap-2"
            >
              <Input
                v-model="formData.config.start_urls[index]"
                placeholder="https://example.com"
                required
                class="flex-1"
              />
              <Button
                v-if="formData.config.start_urls.length > 1"
                type="button"
                variant="outline"
                size="icon"
                @click="removeUrl(index)"
              >
                ×
              </Button>
            </div>
            <Button
              type="button"
              variant="outline"
              size="sm"
              @click="addUrl"
            >
              + Add URL
            </Button>
          </div>

          <!-- Max Depth -->
          <div class="space-y-2">
            <Label for="max_depth">Max Depth</Label>
            <Input
              id="max_depth"
              v-model.number="formData.config.max_depth"
              type="number"
              min="1"
              max="10"
            />
            <p class="text-xs text-muted-foreground">
              Maximum depth for URL discovery (1-10)
            </p>
          </div>

          <!-- Rate Limit Delay -->
          <div class="space-y-2">
            <Label for="rate_limit">Rate Limit Delay (ms)</Label>
            <Input
              id="rate_limit"
              v-model.number="formData.config.rate_limit_delay"
              type="number"
              min="0"
              step="100"
            />
            <p class="text-xs text-muted-foreground">
              Delay between requests in milliseconds
            </p>
          </div>

          <!-- Storage Type -->
          <div class="space-y-2">
            <Label for="storage">Storage Type</Label>
            <Select v-model="formData.config.storage.type">
              <SelectTrigger id="storage">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="database">Database</SelectItem>
                <SelectItem value="file">File</SelectItem>
                <SelectItem value="webhook">Webhook</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            @click="handleClose"
            :disabled="isSubmitting"
          >
            Cancel
          </Button>
          <Button type="submit" :disabled="isSubmitting">
            <Loader2 v-if="isSubmitting" class="mr-2 h-4 w-4 animate-spin" />
            {{ mode === 'create' ? 'Create' : 'Update' }}
          </Button>
        </DialogFooter>
      </form>
    </DialogContent>
  </Dialog>
</template>
