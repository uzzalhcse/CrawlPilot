<script setup lang="ts">
import { Button } from '@/components/ui/button'

defineProps<{
  step: 'idle' | 'awaiting_label' | 'awaiting_value'
  labelText?: string
  pairCount?: number
}>()

const emit = defineEmits<{
  (e: 'cancel'): void
  (e: 'done'): void
  (e: 'undo'): void
}>()
</script>

<template>
  <div v-if="step !== 'idle'" class="pair-status" data-selectflow-ui="status">
    <div class="status-content">
      <div v-if="step === 'awaiting_label'" class="status-step">
        <span class="icon">🔑</span>
        <span class="text">Select Label Element...</span>
        <span class="step-indicator">Step 1/2</span>
      </div>
      
      <div v-else-if="step === 'awaiting_value'" class="status-step">
        <div class="flex items-center gap-2">
          <span class="icon">🔑</span>
          <span class="label-preview">{{ labelText }}</span>
          <span class="arrow">→</span>
          <span class="icon">🔓</span>
          <span class="text">Select Value...</span>
        </div>
        <span class="step-indicator">Step 2/2</span>
      </div>
      
      <div class="status-actions">
        <Button 
          v-if="pairCount && pairCount > 0" 
          size="sm" 
          variant="ghost" 
          @click="emit('undo')"
          class="h-7 text-xs"
        >
          Undo Last
        </Button>
        <Button 
          size="sm" 
          variant="ghost" 
          @click="emit('cancel')"
          class="h-7 text-xs"
        >
          Cancel (Esc)
        </Button>
        <Button 
          v-if="pairCount && pairCount > 0" 
          size="sm" 
          @click="emit('done')"
          class="h-7 text-xs"
        >
          Done
        </Button>
      </div>
    </div>
    
    <div v-if="pairCount && pairCount > 0" class="pair-count">
      ✓ {{ pairCount }} {{ pairCount === 1 ? 'pair' : 'pairs' }} added
    </div>
  </div>
</template>

<style scoped>
.pair-status {
  position: fixed;
  top: 80px;
  left: 50%;
  transform: translateX(-50%);
  background: white;
  border-radius: 12px;
  padding: 12px 16px;
  box-shadow: 0 4px 16px rgba(0,0,0,0.2);
  z-index: 450;
  min-width: 400px;
  border: 2px solid #3b82f6;
  pointer-events: auto;
}

.status-content {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.status-step {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 14px;
}

.status-step .flex {
  display: flex;
  align-items: center;
}

.icon {
  font-size: 20px;
}

.text {
  font-weight: 500;
  color: #374151;
}

.arrow {
  color: #9ca3af;
  font-size: 16px;
  font-weight: bold;
}

.label-preview {
  background: #dbeafe;
  color: #1e40af;
  padding: 4px 10px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.step-indicator {
  margin-left: auto;
  font-size: 11px;
  color: #6b7280;
  background: #f3f4f6;
  padding: 2px 8px;
  border-radius: 4px;
  font-weight: 500;
}

.status-actions {
  display: flex;
  gap: 6px;
  justify-content: flex-end;
  padding-top: 8px;
  border-top: 1px solid #e5e7eb;
}

.pair-count {
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px solid #e5e7eb;
  font-size: 13px;
  color: #10b981;
  font-weight: 600;
  text-align: center;
}
</style>
