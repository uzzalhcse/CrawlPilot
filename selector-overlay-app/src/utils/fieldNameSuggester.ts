/**
 * Auto-suggest field names from element content and attributes
 */

interface FieldNameSuggestion {
    name: string
    confidence: 'high' | 'medium' | 'low'
}

/**
 * Convert camelCase or PascalCase to snake_case
 */
function camelToSnake(str: string): string {
    return str
        .replace(/([A-Z])/g, '_$1')
        .toLowerCase()
        .replace(/^_/, '')
        .replace(/_+/g, '_')
}

/**
 * Extract semantic meaning from class names
 */
function findSemanticClass(classes: string[]): string | null {
    const semanticPatterns = [
        /product[-_]?name/i,
        /product[-_]?title/i,
        /item[-_]?name/i,
        /price/i,
        /description/i,
        /image/i,
        /category/i,
        /brand/i,
        /rating/i,
        /review/i,
    ]

    for (const className of classes) {
        for (const pattern of semanticPatterns) {
            if (pattern.test(className)) {
                return className
                    .replace(/[-\s]+/g, '_')
                    .replace(/^[^a-zA-Z]+/, '')
                    .toLowerCase()
            }
        }
    }

    return null
}

/**
 * Detect if text looks like a price
 */
function isPrice(text: string | null): boolean {
    if (!text) return false
    return /[\$€£¥₹]\s*[\d,]+\.?\d*|\d+\.\d{2}/.test(text)
}

/**
 * Detect if text looks like a date
 */
function isDate(text: string | null): boolean {
    if (!text) return false
    // Match common date patterns
    const datePatterns = [
        /\d{1,2}[\/\-]\d{1,2}[\/\-]\d{2,4}/,
        /\d{4}[\/\-]\d{1,2}[\/\-]\d{1,2}/,
        /(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+\d{1,2},?\s+\d{4}/i,
    ]
    return datePatterns.some(pattern => pattern.test(text))
}

/**
 * Extract keywords from text for field naming
 */
function extractKeywords(text: string): string {
    // Remove special characters and extra whitespace
    const cleaned = text
        .toLowerCase()
        .replace(/[^\w\s]/g, ' ')
        .replace(/\s+/g, '_')
        .trim()

    // Truncate if too long
    if (cleaned.length > 30) {
        return cleaned.substring(0, 30).replace(/_$/, '')
    }

    return cleaned || 'field'
}

/**
 * Main function to suggest field name from element
 */
export function suggestFieldName(element: Element): FieldNameSuggestion {
    // Priority 1: Check for semantic data attributes
    if (element.hasAttribute('data-field-name')) {
        return {
            name: element.getAttribute('data-field-name')!,
            confidence: 'high'
        }
    }

    if (element.hasAttribute('name')) {
        const name = element.getAttribute('name')!
        return {
            name: camelToSnake(name),
            confidence: 'high'
        }
    }

    if (element.hasAttribute('id')) {
        const id = element.getAttribute('id')!
        // Skip if ID looks auto-generated
        if (!/^\d+$/.test(id) && !/^[a-f0-9-]{20,}$/i.test(id)) {
            return {
                name: camelToSnake(id),
                confidence: 'medium'
            }
        }
    }

    // Priority 2: Analyze class names for semantic meaning
    const classes = element.className?.toString().split(/\s+/) || []
    const semanticClass = findSemanticClass(classes)
    if (semanticClass) {
        return {
            name: semanticClass,
            confidence: 'high'
        }
    }

    // Priority 3: Check text content for patterns
    const text = element.textContent?.trim() || ''

    if (text && text.length < 100) {
        // Check for specific patterns
        if (isPrice(text)) {
            return { name: 'price', confidence: 'high' }
        }

        if (isDate(text)) {
            return { name: 'date', confidence: 'medium' }
        }

        // Extract keywords from short text
        if (text.length < 50) {
            const keywords = extractKeywords(text)
            if (keywords && keywords !== 'field') {
                return { name: keywords, confidence: 'low' }
            }
        }
    }

    // Priority 4: Check element attributes for semantic meaning
    if (element.hasAttribute('aria-label')) {
        const label = element.getAttribute('aria-label')!
        return {
            name: extractKeywords(label),
            confidence: 'medium'
        }
    }

    if (element.hasAttribute('title')) {
        const title = element.getAttribute('title')!
        return {
            name: extractKeywords(title),
            confidence: 'low'
        }
    }

    // Priority 5: Check for common element type patterns
    const tagName = element.tagName.toLowerCase()

    if (tagName === 'img') {
        const alt = element.getAttribute('alt')
        if (alt) {
            return { name: extractKeywords(alt), confidence: 'medium' }
        }
        return { name: 'image', confidence: 'medium' }
    }

    if (tagName === 'a') {
        const href = element.getAttribute('href')
        if (href && href.startsWith('http')) {
            return { name: 'link', confidence: 'medium' }
        }
    }

    // Fallback: Use element tag name
    return {
        name: tagName === 'div' || tagName === 'span' ? 'field' : tagName,
        confidence: 'low'
    }
}

/**
 * Generate unique field name by avoiding conflicts
 */
export function makeUniqueFieldName(baseName: string, existingNames: string[]): string {
    const nameSet = new Set(existingNames)

    if (!nameSet.has(baseName)) {
        return baseName
    }

    // Try appending numbers
    let counter = 1
    let uniqueName = `${baseName}_${counter}`

    while (nameSet.has(uniqueName)) {
        counter++
        uniqueName = `${baseName}_${counter}`
    }

    return uniqueName
}
