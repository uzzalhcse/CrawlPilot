<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch, shallowRef } from 'vue'
import * as monaco from 'monaco-editor'
import editorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker'
import jsonWorker from 'monaco-editor/esm/vs/language/json/json.worker?worker'

self.MonacoEnvironment = {
  getWorker(_, label) {
    if (label === 'json') {
      return new jsonWorker()
    }
    return new editorWorker()
  }
}

const props = defineProps<{
  modelValue: string
  language?: string
  theme?: string
  readOnly?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
}>()

const editorContainer = ref<HTMLElement | null>(null)
const editor = shallowRef<monaco.editor.IStandaloneCodeEditor | null>(null)

// Initialize editor
onMounted(() => {
  if (!editorContainer.value) return

  // Configure workers for JSON (basic setup for Vite without plugin)
  // Note: For full worker support in Vite, we usually need a worker script.
  // This simple setup might rely on the main thread or need explicit worker config.
  // For now, we'll try default import.
  
  editor.value = monaco.editor.create(editorContainer.value, {
    value: props.modelValue,
    language: props.language || 'json',
    theme: props.theme || 'vs-dark',
    automaticLayout: true,
    readOnly: props.readOnly || false,
    minimap: { enabled: true }, // Enable minimap for better navigation
    scrollBeyondLastLine: false,
    fontSize: 14,
    tabSize: 2,
    wordWrap: 'on',
    folding: true, // Explicitly enable folding
    foldingStrategy: 'indentation', // Use indentation based folding if language service fails
    showFoldingControls: 'always',
    formatOnPaste: true,
    formatOnType: true
  })

  // Listen for changes
  editor.value.onDidChangeModelContent(() => {
    const value = editor.value?.getValue() || ''
    if (value !== props.modelValue) {
      emit('update:modelValue', value)
    }
  })
})

// Watch for external value changes
watch(
  () => props.modelValue,
  (newValue) => {
    if (editor.value && newValue !== editor.value.getValue()) {
      editor.value.setValue(newValue)
    }
  }
)

// Watch for theme changes
watch(
  () => props.theme,
  (newTheme) => {
    if (editor.value && newTheme) {
      monaco.editor.setTheme(newTheme)
    }
  }
)

// Cleanup
onBeforeUnmount(() => {
  if (editor.value) {
    editor.value.dispose()
  }
})
</script>

<template>
  <div ref="editorContainer" class="w-full h-full min-h-[300px] border rounded-md overflow-hidden"></div>
</template>

<style scoped>
/* Ensure container has size */
</style>
