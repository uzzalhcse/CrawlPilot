<script setup lang="ts">
import { ref } from 'vue'
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
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Loader2, AlertTriangle } from 'lucide-vue-next'
import type { Workflow } from '@/types'

interface Props {
  open: boolean
  workflow: Workflow | null
}

const props = defineProps<Props>()
const emit = defineEmits<{
  'update:open': [value: boolean]
}>()

const workflowsStore = useWorkflowsStore()
const isDeleting = ref(false)

const handleDelete = async () => {
  if (!props.workflow) return
  
  isDeleting.value = true
  try {
    await workflowsStore.deleteWorkflow(props.workflow.id)
    emit('update:open', false)
  } catch (error) {
    console.error('Failed to delete workflow:', error)
  } finally {
    isDeleting.value = false
  }
}

const handleClose = () => {
  if (!isDeleting.value) {
    emit('update:open', false)
  }
}
</script>

<template>
  <Dialog :open="open" @update:open="handleClose">
    <DialogContent class="max-w-md">
      <DialogHeader>
        <DialogTitle>Delete Workflow</DialogTitle>
        <DialogDescription>
          Are you sure you want to delete this workflow?
        </DialogDescription>
      </DialogHeader>

      <div class="space-y-4">
        <Alert variant="destructive">
          <AlertTriangle class="h-4 w-4" />
          <AlertDescription>
            This action cannot be undone. This will permanently delete the workflow
            <strong v-if="workflow">"{{ workflow.name }}"</strong> and all its configurations.
          </AlertDescription>
        </Alert>

        <div v-if="workflow" class="rounded-lg border p-4 bg-muted/50">
          <div class="space-y-1">
            <p class="text-sm font-medium">Workflow Details:</p>
            <p class="text-sm text-muted-foreground">Name: {{ workflow.name }}</p>
            <p class="text-sm text-muted-foreground">Status: {{ workflow.status }}</p>
            <p class="text-sm text-muted-foreground">
              Created: {{ new Date(workflow.created_at).toLocaleDateString() }}
            </p>
          </div>
        </div>
      </div>

      <DialogFooter>
        <Button
          type="button"
          variant="outline"
          @click="handleClose"
          :disabled="isDeleting"
        >
          Cancel
        </Button>
        <Button
          type="button"
          variant="destructive"
          @click="handleDelete"
          :disabled="isDeleting"
        >
          <Loader2 v-if="isDeleting" class="mr-2 h-4 w-4 animate-spin" />
          Delete Workflow
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
