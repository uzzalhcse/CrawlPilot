/**
 * Generate an optimal CSS selector for an element
 * Priority: ID > data attributes > classes > nth-child
 */
export function generateSelector(element: Element): string {
  // Try ID first
  if (element.id) {
    const id = CSS.escape(element.id)
    return `#${id}`
  }

  // Try data attributes
  const dataAttrs = Array.from(element.attributes).filter(attr => 
    attr.name.startsWith('data-')
  )
  if (dataAttrs.length > 0) {
    const tagName = element.tagName.toLowerCase()
    const dataAttr = dataAttrs[0]
    const attrValue = CSS.escape(dataAttr.value)
    const selector = `${tagName}[${dataAttr.name}="${attrValue}"]`
    if (isUnique(selector)) {
      return selector
    }
  }

  // Try combining tag + classes
  if (element.classList.length > 0) {
    const tagName = element.tagName.toLowerCase()
    const classes = Array.from(element.classList)
      .filter(cls => !cls.startsWith('crawlify-'))
      .map(cls => `.${CSS.escape(cls)}`)
      .join('')
    
    if (classes) {
      const selector = `${tagName}${classes}`
      if (isUnique(selector)) {
        return selector
      }

      // Try with parent context
      if (element.parentElement) {
        const parentSelector = getSimpleParentSelector(element.parentElement)
        const contextSelector = `${parentSelector} > ${selector}`
        if (isUnique(contextSelector)) {
          return contextSelector
        }
      }
    }
  }

  // Fall back to nth-child
  return getNthChildSelector(element)
}

function isUnique(selector: string): boolean {
  try {
    return document.querySelectorAll(selector).length === 1
  } catch {
    return false
  }
}

function getSimpleParentSelector(element: Element): string {
  if (element.id) {
    return `#${CSS.escape(element.id)}`
  }
  
  const tagName = element.tagName.toLowerCase()
  if (element.classList.length > 0) {
    const firstClass = element.classList[0]
    return `${tagName}.${CSS.escape(firstClass)}`
  }
  
  return tagName
}

function getNthChildSelector(element: Element): string {
  const parent = element.parentElement
  if (!parent) {
    return element.tagName.toLowerCase()
  }

  const siblings = Array.from(parent.children)
  const index = siblings.indexOf(element) + 1
  const tagName = element.tagName.toLowerCase()
  
  const parentSelector = getSimpleParentSelector(parent)
  return `${parentSelector} > ${tagName}:nth-child(${index})`
}

/**
 * Validate a CSS selector
 */
export function validateSelector(selector: string): { valid: boolean; count: number } {
  try {
    const count = document.querySelectorAll(selector).length
    return { valid: true, count }
  } catch {
    return { valid: false, count: 0 }
  }
}

/**
 * Get all elements matching a selector
 */
export function getElementsForSelector(selector: string): Element[] {
  try {
    return Array.from(document.querySelectorAll(selector))
  } catch {
    return []
  }
}
