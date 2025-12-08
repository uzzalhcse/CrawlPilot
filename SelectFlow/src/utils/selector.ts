export function generateSelector(element: HTMLElement, preferSpecific: boolean = false): string {
    // Try ID first
    if (element.id) {
        // Ensure ID is valid and unique
        try {
            if (document.querySelectorAll(`#${CSS.escape(element.id)}`).length === 1) {
                return `#${CSS.escape(element.id)}`
            }
        } catch (e) {
            // Ignore invalid IDs
        }
    }

    // Try classes with combination approaches
    if (element.className) {
        const className = typeof element.className === 'string' ? element.className : ''
        const classes = className.split(' ')
            .filter(Boolean)
            .filter(c => !c.startsWith('selectflow-')) // Filter internal classes
            .filter(c => !c.includes(':')) // Filter Tailwind modifiers (hover:, etc)
            .filter(c => !c.includes('/')) // Filter Tailwind fractions (w-1/2)
            .filter(c => !c.startsWith('!')) // Filter important modifiers

        // 1. Try each class individually to see if it's unique
        for (const cls of classes) {
            const selector = `.${CSS.escape(cls)}`
            try {
                if (document.querySelectorAll(selector).length === 1) {
                    return selector
                }
            } catch (e) {
                continue
            }
        }

        // 2. Try combination of all valid classes
        if (classes.length > 0) {
            const selector = classes.map(c => `.${CSS.escape(c)}`).join('')
            try {
                if (document.querySelectorAll(selector).length === 1) {
                    return selector
                }
            } catch (e) {
                // Ignore
            }
        }

        // 3. For single mode (non-specific), return class-based selector
        // This will match all similar elements, which is what users expect
        if (!preferSpecific && classes.length > 0) {
            return classes.map(c => `.${CSS.escape(c)}`).join('')
        }
    }

    // Build path-based selector with nth-child for specificity
    // Use this for pair mode or when class-based selectors aren't available
    const path: string[] = []
    let current: HTMLElement | null = element
    let depth = 0
    const maxDepth = preferSpecific ? 5 : 3 // Limit depth to avoid overly long selectors

    while (current && current !== document.body && depth < maxDepth) {
        let selector = current.tagName.toLowerCase()

        // Add class if available
        if (current.className && typeof current.className === 'string') {
            const classes = current.className.trim().split(/\s+/).filter(cls =>
                !cls.startsWith('selectflow-')
            )
            if (classes.length > 0) {
                selector += '.' + classes[0] // Use first class only for path
            }
        }

        // Add nth-child for uniqueness (primarily for pair mode)
        if (preferSpecific && current.parentElement) {
            const siblings = Array.from(current.parentElement.children)
            const index = siblings.indexOf(current) + 1
            selector += `:nth-child(${index})`
        }

        path.unshift(selector)
        current = current.parentElement
        depth++
    }

    return path.join(' > ')
}

// Generate selector specifically for pair mode (more specific)
export function generatePairSelector(element: HTMLElement): string {
    return generateSelector(element, true)
}

// Generate selector for single mode (more general)
export function generateSingleSelector(element: HTMLElement): string {
    return generateSelector(element, false)
}

export function validateSelector(selector: string): number {
    try {
        return document.querySelectorAll(selector).length
    } catch {
        return 0
    }
}

export function getMatchingElements(selector: string): Element[] {
    try {
        return Array.from(document.querySelectorAll(selector))
    } catch {
        return []
    }
}

export function highlightElement(element: HTMLElement) {
    // Add hover class
    element.classList.add('selectflow-highlight-hover')
}

export function removeHighlight(element: HTMLElement) {
    // Remove hover class
    element.classList.remove('selectflow-highlight-hover')
}
