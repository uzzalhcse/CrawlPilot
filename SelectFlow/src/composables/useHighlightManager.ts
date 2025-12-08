import { type Ref, watch } from 'vue'
import type { Field, KeyValueGroup } from '../types'

export function useHighlightManager(fields: Ref<Field[]>, keyValueGroups: Ref<KeyValueGroup[]>) {

    // Global Highlight Management
    function refreshSavedHighlights() {
        // Clear all saved highlights
        document.querySelectorAll('.selectflow-highlight-saved').forEach(el => {
            el.classList.remove('selectflow-highlight-saved')
        })

        // Apply highlights for all saved single fields
        fields.value.forEach(field => {
            if (field.fieldType === 'single') {
                try {
                    const elements = document.querySelectorAll(field.config.selector)
                    elements.forEach(el => {
                        el.classList.add('selectflow-highlight-saved')
                    })
                } catch (e) {
                    // Ignore invalid selectors in saved fields
                }
            }
        })

        // Apply highlights for all key-value pairs
        keyValueGroups.value.forEach(group => {
            group.pairs.forEach(pair => {
                try {
                    // Highlight the key element
                    const keyElements = document.querySelectorAll(pair.config.key_selector)
                    keyElements.forEach(el => {
                        el.classList.add('selectflow-highlight-saved')
                    })

                    // Highlight the value element
                    const valueElements = document.querySelectorAll(pair.config.value_selector)
                    valueElements.forEach(el => {
                        el.classList.add('selectflow-highlight-saved')
                    })
                } catch (e) {
                    // Ignore invalid selectors in saved pairs
                }
            })
        })
    }

    function cleanupPairHighlights() {
        // Remove all pair-related highlight classes from the entire document
        document.querySelectorAll('.selectflow-highlight-label, .selectflow-highlight-value, .selectflow-highlight-hover-label, .selectflow-highlight-hover-value').forEach(el => {
            el.classList.remove('selectflow-highlight-label', 'selectflow-highlight-value', 'selectflow-highlight-hover-label', 'selectflow-highlight-hover-value')
        })
    }

    // Watch for changes in fields to update highlights
    watch(fields, refreshSavedHighlights, { deep: true })
    watch(keyValueGroups, refreshSavedHighlights, { deep: true })

    return {
        refreshSavedHighlights,
        cleanupPairHighlights
    }
}
