<template>
  <div class="code-editor-container h-full flex flex-col">
    <!-- File Tabs -->
    <div class="flex items-center gap-1 px-2 py-1 bg-muted/30 border-b overflow-x-auto">
      <button
        v-for="file in fileNames"
        :key="file"
        :class="[
          'px-3 py-1.5 text-sm font-mono rounded transition-colors flex items-center gap-2 whitespace-nowrap',
          activeFile === file
            ? 'bg-background text-foreground shadow-sm'
            : 'text-muted-foreground hover:text-foreground hover:bg-background/50'
        ]"
        @click="activeFile = file"
      >
        <FileCode class="h-3.5 w-3.5" />
        {{ file }}
      </button>
    </div>

    <!-- Code Editor -->
    <div class="flex-1 relative">
      <div ref="editorContainer" class="h-full w-full"></div>
    </div>

    <!-- Bottom Actions -->
    <div class="flex items-center justify-between px-4 py-2 bg-muted/30 border-t">
      <div class="text-xs text-muted-foreground">
        {{ fileNames.length }} file{{ fileNames.length !== 1 ? 's' : '' }}
      </div>
      <div class="flex items-center gap-2">
        <Button
          variant="outline"
          size="sm"
          @click="$emit('cancel')"
        >
          Cancel
        </Button>
        <Button
          size="sm"
          @click="handleSave"
          :disabled="saving"
        >
          <Save class="h-4 w-4 mr-2" v-if="!saving" />
          <Loader2 class="h-4 w-4 mr-2 animate-spin" v-else />
          {{ saving ? 'Saving...' : 'Save Changes' }}
        </Button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { useColorMode } from '@vueuse/core'
import * as monaco from 'monaco-editor'
import { Button } from '@/components/ui/button'
import { FileCode, Save, Loader2 } from 'lucide-vue-next'

interface Props {
  modelValue: Record<string, string>
  readOnly?: boolean
}

interface Emits {
  (e: 'update:modelValue', value: Record<string, string>): void
  (e: 'save', files: Record<string, string>): void
  (e: 'cancel'): void
}

const props = withDefaults(defineProps<Props>(), {
  readOnly: false
})

const emit = defineEmits<Emits>()

const mode = useColorMode()
const isDark = computed(() => mode.value === 'dark')

const files = ref<Record<string, string>>({ ...props.modelValue })
const activeFile = ref<string>(Object.keys(files.value)[0] || '')
const saving = ref(false)
const editorContainer = ref<HTMLElement | null>(null)
let editor: monaco.editor.IStandaloneCodeEditor | null = null

const fileNames = computed(() => Object.keys(files.value).sort())

// Watch for active file changes
watch(activeFile, async (newFile) => {
  if (newFile && editor && files.value[newFile] !== undefined) {
    const language = getLanguage(newFile)
    const model = monaco.editor.createModel(files.value[newFile], language)
    editor.setModel(model)
  }
})

// Watch for theme changes
watch(isDark, (dark) => {
  if (editor) {
    monaco.editor.setTheme(dark ? 'vs-dark' : 'vs')
  }
})

// Sync with parent
watch(() => props.modelValue, (newValue) => {
  files.value = { ...newValue }
  if (!activeFile.value && fileNames.value.length > 0) {
    activeFile.value = fileNames.value[0]
  }
}, { deep: true })

// Watch editor changes and update files
watch(files, (newFiles) => {
  emit('update:modelValue', { ...newFiles })
}, { deep: true })

function getLanguage(filename: string): string {
  if (filename.endsWith('.go')) return 'go'
  if (filename.endsWith('.mod') || filename.endsWith('.sum')) return 'plaintext'
  if (filename.endsWith('.md')) return 'markdown'
  if (filename.endsWith('.json')) return 'json'
  return 'plaintext'
}

async function handleSave() {
  saving.value = true
  try {
    // Update current file content from editor
    if (editor && activeFile.value) {
      const model = editor.getModel()
      if (model) {
        files.value[activeFile.value] = model.getValue()
      }
    }
    emit('save', { ...files.value })
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  await nextTick()
  if (editorContainer.value) {
    editor = monaco.editor.create(editorContainer.value, {
      value: activeFile.value ? files.value[activeFile.value] : '',
      language: activeFile.value ? getLanguage(activeFile.value) : 'plaintext',
      theme: isDark.value ? 'vs-dark' : 'vs',
      minimap: { enabled: false },
      fontSize: 13,
      lineNumbers: 'on',
      roundedSelection: true,
      scrollBeyondLastLine: false,
      readOnly: props.readOnly,
      automaticLayout: true,
      tabSize: 4,
      insertSpaces: false,
      wordWrap: 'on'
    })

    // Listen to content changes
    editor.onDidChangeModelContent(() => {
      if (editor && activeFile.value) {
        const model = editor.getModel()
        if (model) {
          files.value[activeFile.value] = model.getValue()
        }
      }
    })
  }
})

onBeforeUnmount(() => {
  if (editor) {
    editor.dispose()
  }
})
</script>

<style scoped>
.code-editor-container {
  min-height: 600px;
  height: 600px;
}
</style>
