<template>
  <Dialog v-model:open="isOpen">
    <DialogContent class="sm:max-w-[600px]">
      <DialogHeader>
        <DialogTitle>Create New Plugin</DialogTitle>
        <DialogDescription>
          Scaffold a new plugin from a template
        </DialogDescription>
      </DialogHeader>

      <form @submit.prevent="handleSubmit" class="space-y-4">
        <!-- Plugin Name -->
        <div class="space-y-2">
          <Label for="name">Plugin Name *</Label>
          <Input
            id="name"
            v-model="form.name"
            placeholder="My Awesome Plugin"
            required
            @input="autoGenerateSlug"
          />
        </div>

        <!-- Slug -->
        <div class="space-y-2">
          <Label for="slug">Slug *</Label>
          <Input
            id="slug"
            v-model="form.slug"
            placeholder="my-awesome-plugin"
            required
            pattern="[a-z0-9-]+"
          />
          <p class="text-xs text-muted-foreground">Lowercase letters, numbers, and hyphens only</p>
        </div>

        <!-- Description -->
        <div class="space-y-2">
          <Label for="description">Description *</Label>
          <Textarea
            id="description"
            v-model="form.description"
            placeholder="Describe what your plugin does..."
            rows="3"
            required
          />
        </div>

        <!-- Phase Type -->
        <div class="space-y-2">
          <Label for="phase_type">Phase Type *</Label>
          <Select v-model="form.phase_type" required>
            <SelectTrigger>
              <SelectValue placeholder="Select phase type" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="discovery">
                🔍 Discovery - Find and collect URLs
              </SelectItem>
              <SelectItem value="extraction">
                📦 Extraction - Extract data from pages
              </SelectItem>
              <SelectItem value="processing">
                ⚙️ Processing - Transform and process data
              </SelectItem>
            </SelectContent>
          </Select>
        </div>

        <!-- Author Info -->
        <div class="grid grid-cols-2 gap-4">
          <div class="space-y-2">
            <Label for="author_name">Author Name *</Label>
            <Input
              id="author_name"
              v-model="form.author_name"
              placeholder="John Doe"
              required
            />
          </div>
          <div class="space-y-2">
            <Label for="author_email">Author Email *</Label>
            <Input
              id="author_email"
              v-model="form.author_email"
              type="email"
              placeholder="john@example.com"
              required
            />
          </div>
        </div>

        <!-- Category -->
        <div class="space-y-2">
          <Label for="category">Category</Label>
          <Input
            id="category"
            v-model="form.category"
            placeholder="ecommerce, news, social-media, etc."
          />
        </div>

        <!-- Tags -->
        <div class="space-y-2">
          <Label for="tags">Tags</Label>
          <Input
            id="tags"
            v-model="tagsInput"
            placeholder="Comma-separated tags"
          />
          <p class="text-xs text-muted-foreground">e.g., ecommerce, products, amazon</p>
        </div>

        <!-- Actions -->
        <DialogFooter>
          <Button type="button" variant="outline" @click="isOpen = false">
            Cancel
          </Button>
          <Button type="submit" :disabled="creating">
            <Loader2 v-if="creating" class="h-4 w-4 mr-2 animate-spin" />
            Create Plugin
          </Button>
        </DialogFooter>
      </form>
    </DialogContent>
  </Dialog>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Loader2 } from 'lucide-vue-next'
import pluginAPI from '@/lib/plugin-api'

interface Props {
  open?: boolean
}

interface Emits {
  (e: 'update:open', value: boolean): void
  (e: 'created', pluginId: string): void
}

const props = withDefaults(defineProps<Props>(), {
  open: false
})

const emit = defineEmits<Emits>()
const router = useRouter()

const isOpen = computed({
  get: () => props.open,
  set: (value) => emit('update:open', value)
})

const form = ref({
  name: '',
  slug: '',
  description: '',
  phase_type: '',
  author_name: '',
  author_email: '',
  category: '',
})

const tagsInput = ref('')
const creating = ref(false)
let slugAutoGenerated = true

function autoGenerateSlug() {
  if (slugAutoGenerated) {
    form.value.slug = form.value.name
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, '-')
      .replace(/^-|-$/g, '')
  }
}

async function handleSubmit() {
  creating.value = true
  try {
    const tags = tagsInput.value
      .split(',')
      .map(t => t.trim())
      .filter(t => t.length > 0)

    const plugin = await pluginAPI.scaffoldPlugin({
      ...form.value,
      tags: tags.length > 0 ? tags : undefined,
      category: form.value.category || undefined,
    })

    // Reset form
    form.value = {
      name: '',
      slug: '',
      description: '',
      phase_type: '',
      author_name: '',
      author_email: '',
      category: '',
    }
    tagsInput.value = ''
    
    emit('created', plugin.id)
    emit('update:open', false)
    
    // Navigate to plugin detail page
    router.push(`/plugins/${plugin.id}`)
  } catch (error) {
    console.error('Failed to create plugin:', error)
    alert('Failed to create plugin. Please try again.')
  } finally {
    creating.value = false
  }
}
</script>
