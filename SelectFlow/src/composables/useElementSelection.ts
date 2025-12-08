import { ref } from 'vue'

export function useElementSelection() {
    const selectedElement = ref<HTMLElement | null>(null)

    function selectElement(element: HTMLElement) {
        selectedElement.value = element
    }

    function clearSelection() {
        selectedElement.value = null
    }

    return {
        selectedElement,
        selectElement,
        clearSelection
    }
}
