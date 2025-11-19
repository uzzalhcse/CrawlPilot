import { ref } from 'vue'
import { defineStore } from 'pinia'

export const useThemeStore = defineStore('theme', () => {
  const isDark = ref(false)

  function toggleTheme() {
    isDark.value = !isDark.value
    updateTheme()
  }

  function setTheme(dark: boolean) {
    isDark.value = dark
    updateTheme()
  }

  function updateTheme() {
    if (isDark.value) {
      document.documentElement.classList.add('dark')
      localStorage.setItem('theme', 'dark')
    } else {
      document.documentElement.classList.remove('dark')
      localStorage.setItem('theme', 'light')
    }
  }

  function initTheme() {
    const savedTheme = localStorage.getItem('theme')
    const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
    
    isDark.value = savedTheme === 'dark' || (!savedTheme && prefersDark)
    updateTheme()
  }

  return {
    isDark,
    toggleTheme,
    setTheme,
    initTheme
  }
})
