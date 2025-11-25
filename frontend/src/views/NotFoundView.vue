<template>
  <div class="h-full flex flex-col items-center justify-center text-center p-6">
    <div class="space-y-6 max-w-md">
      <!-- Icon -->
      <div class="w-24 h-24 rounded-full bg-secondary/30 flex items-center justify-center mx-auto mb-6">
        <FileQuestion class="w-12 h-12 text-muted-foreground" />
      </div>
      
      <!-- Text -->
      <div class="space-y-2">
        <h1 class="text-4xl font-bold tracking-tight">Page not found</h1>
        <p class="text-muted-foreground text-lg">
          Sorry, we couldn't find the page you're looking for. It might have been moved or deleted.
        </p>
      </div>

      <!-- Action -->
      <div class="pt-4">
        <Button size="lg" class="gap-2" @click="router.push('/')">
          <Home class="w-4 h-4" />
          Go back home
        </Button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { FileQuestion, Home } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'

const router = useRouter()
</script>
