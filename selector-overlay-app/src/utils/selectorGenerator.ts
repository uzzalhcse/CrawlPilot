/**
 * Generate an optimal CSS selector for an element
 * Priority: ID > unique data attributes > unique classes > classes with parent > tag with parent > nth-child
 */
export function generateSelector(element: Element): string {
  // Try ID first (most specific and stable)
  if (element.id && !element.id.match(/^[0-9]/) && !element.id.includes(' ')) {
    const id = CSS.escape(element.id)
    const selector = `#${id}`
    if (isUnique(selector)) {
      return selector
    }
  }
  
  // For list items or repeated elements, try to generate a selector that matches all similar items
  const repeatedSelector = tryRepeatedElementSelector(element)
  if (repeatedSelector && !isUnique(repeatedSelector)) {
    // This is likely a list item, return the selector that matches all similar items
    return repeatedSelector
  }

  // Try data attributes (good for semantic selection)
  const dataAttrs = Array.from(element.attributes).filter(attr => 
    attr.name.startsWith('data-') && attr.value
  )
  if (dataAttrs.length > 0) {
    const tagName = element.tagName.toLowerCase()
    
    // Try each data attribute
    for (const dataAttr of dataAttrs) {
      const attrValue = CSS.escape(dataAttr.value)
      const selector = `${tagName}[${dataAttr.name}="${attrValue}"]`
      if (isUnique(selector)) {
        return selector
      }
    }
  }

  // Try combining tag + all classes (most specific class combination)
  if (element.classList.length > 0) {
    const tagName = element.tagName.toLowerCase()
    const classes = Array.from(element.classList)
      .filter(cls => !cls.startsWith('crawlify-') && !cls.match(/^[0-9]/))
      .map(cls => `.${CSS.escape(cls)}`)
      .join('')
    
    if (classes) {
      // ALWAYS try with parent context first for better specificity
      if (element.parentElement) {
        const parentSelector = getSimpleParentSelector(element.parentElement)
        const contextSelector = `${parentSelector} > ${tagName}${classes}`
        
        // Return parent context selector even if not unique (better clarity)
        // unless it's too generic (matches >1000 elements)
        try {
          const count = document.querySelectorAll(contextSelector).length
          if (count > 0 && count <= 1000) {
            return contextSelector
          }
        } catch {
          // If selector fails, continue to other strategies
        }
        
        // Try with ancestor context (descendant combinator)
        const ancestorSelector = `${parentSelector} ${tagName}${classes}`
        try {
          const count = document.querySelectorAll(ancestorSelector).length
          if (count > 0 && count <= 1000) {
            return ancestorSelector
          }
        } catch {
          // Continue
        }
      }
      
      // Fallback to selector without parent if parent context is too broad
      const selector = `${tagName}${classes}`
      if (isUnique(selector)) {
        return selector
      }
      
      // Try with fewer classes (most meaningful ones first)
      if (element.classList.length > 1) {
        const firstClass = `.${CSS.escape(element.classList[0])}`
        const simpleSelector = `${tagName}${firstClass}`
        if (isUnique(simpleSelector)) {
          return simpleSelector
        }
        
        // Try with parent
        if (element.parentElement) {
          const parentSelector = getSimpleParentSelector(element.parentElement)
          const contextSelector = `${parentSelector} > ${simpleSelector}`
          if (isUnique(contextSelector)) {
            return contextSelector
          }
        }
      }
    }
  }

  // Try tag name with attributes (like type, name, href, etc.)
  const tagName = element.tagName.toLowerCase()
  const meaningfulAttrs = ['name', 'type', 'rel', 'role', 'aria-label']
  for (const attrName of meaningfulAttrs) {
    const attrValue = element.getAttribute(attrName)
    if (attrValue) {
      const selector = `${tagName}[${attrName}="${CSS.escape(attrValue)}"]`
      if (isUnique(selector)) {
        return selector
      }
    }
  }

  // Try tag with parent
  if (element.parentElement) {
    const parentSelector = getSimpleParentSelector(element.parentElement)
    const selector = `${parentSelector} > ${tagName}`
    if (isUnique(selector)) {
      return selector
    }
  }

  // Fall back to nth-child (least stable but always works)
  return getNthChildSelector(element)
}

function isUnique(selector: string): boolean {
  try {
    return document.querySelectorAll(selector).length === 1
  } catch {
    return false
  }
}

function tryRepeatedElementSelector(element: Element): string | null {
  const tagName = element.tagName.toLowerCase()
  
  // Check if element has classes (common for list items)
  if (element.classList.length > 0) {
    const classes = Array.from(element.classList)
      .filter(cls => !cls.startsWith('crawlify-') && !cls.match(/^[0-9]/))
    
    if (classes.length > 0) {
      // Try with all classes
      const allClasses = classes.map(cls => `.${CSS.escape(cls)}`).join('')
      const selector = `${tagName}${allClasses}`
      
      // If this matches multiple elements (2+), it's likely a repeated pattern
      try {
        const count = document.querySelectorAll(selector).length
        if (count >= 2 && count <= 1000) { // reasonable upper limit
          return selector
        }
      } catch {
        return null
      }
      
      // Try with first class only
      const firstClass = `.${CSS.escape(classes[0])}`
      const simpleSelector = `${tagName}${firstClass}`
      try {
        const count = document.querySelectorAll(simpleSelector).length
        if (count >= 2 && count <= 1000) {
          return simpleSelector
        }
      } catch {
        return null
      }
    }
  }
  
  return null
}

function getSimpleParentSelector(element: Element): string {
  // Prefer ID if available and valid
  if (element.id && !element.id.match(/^[0-9]/) && !element.id.includes(' ')) {
    return `#${CSS.escape(element.id)}`
  }
  
  const tagName = element.tagName.toLowerCase()
  
  // Try with multiple classes for better specificity
  if (element.classList.length > 0) {
    const classes = Array.from(element.classList)
      .filter(cls => !cls.startsWith('crawlify-') && !cls.match(/^[0-9]/))
      .slice(0, 2) // Use up to 2 classes for balance between specificity and brevity
      .map(cls => `.${CSS.escape(cls)}`)
      .join('')
    
    if (classes) {
      return `${tagName}${classes}`
    }
  }
  
  // Try data attributes
  const dataAttrs = Array.from(element.attributes).filter(attr => 
    attr.name.startsWith('data-') && attr.value
  )
  if (dataAttrs.length > 0) {
    const dataAttr = dataAttrs[0]
    return `${tagName}[${dataAttr.name}="${CSS.escape(dataAttr.value)}"]`
  }
  
  return tagName
}

function getNthChildSelector(element: Element): string {
  const parent = element.parentElement
  if (!parent) {
    return element.tagName.toLowerCase()
  }

  const tagName = element.tagName.toLowerCase()
  const siblings = Array.from(parent.children)
  const index = siblings.indexOf(element) + 1
  
  // Check if nth-of-type would be better (when there are mixed element types)
  const sameTagSiblings = siblings.filter(el => el.tagName === element.tagName)
  const ofTypeIndex = sameTagSiblings.indexOf(element) + 1
  
  const parentSelector = getSimpleParentSelector(parent)
  
  // Use nth-of-type if it's different from nth-child (indicates mixed element types)
  if (ofTypeIndex !== index && sameTagSiblings.length > 1) {
    return `${parentSelector} > ${tagName}:nth-of-type(${ofTypeIndex})`
  }
  
  // Use nth-child for standard cases
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

/**
 * Selector quality rating (1-5 stars)
 */
export interface SelectorQuality {
  score: number // 1-5
  rating: 'excellent' | 'good' | 'fair' | 'poor' | 'fragile'
  reasons: string[]
  issues: string[]
}

/**
 * Alternative selector suggestion
 */
export interface AlternativeSelector {
  selector: string
  quality: SelectorQuality
  matchCount: number
  description: string
}

/**
 * Analyze selector quality and provide alternatives
 */
export function analyzeSelectorQuality(element: Element, currentSelector: string): {
  current: SelectorQuality & { matchCount: number }
  alternatives: AlternativeSelector[]
} {
  const currentMatchCount = validateSelector(currentSelector).count
  const currentQuality = rateSelectorQuality(currentSelector, element)
  
  const alternatives = generateAlternativeSelectors(element, currentSelector)
  
  return {
    current: {
      ...currentQuality,
      matchCount: currentMatchCount
    },
    alternatives
  }
}

/**
 * Rate the quality of a selector
 */
function rateSelectorQuality(selector: string, element: Element): SelectorQuality {
  const reasons: string[] = []
  const issues: string[] = []
  let score = 5
  
  // Check for ID (excellent - most stable)
  if (selector.startsWith('#') && !selector.includes(' ')) {
    reasons.push('Uses unique ID')
    return { score: 5, rating: 'excellent', reasons, issues }
  }
  
  // Check for data attributes (excellent - semantic)
  if (selector.includes('[data-')) {
    reasons.push('Uses semantic data attribute')
    score = 5
  }
  
  // Check for nth-child/nth-of-type (fragile)
  if (selector.includes(':nth-child') || selector.includes(':nth-of-type')) {
    issues.push('Uses positional selector - breaks if order changes')
    score = Math.min(score, 2)
  }
  
  // Check for dynamic/generated classes
  const dynamicClassPattern = /\.([\w-]*\d{3,}[\w-]*|\w*[A-Z]{2,}\w*)/
  if (dynamicClassPattern.test(selector)) {
    issues.push('Contains generated/dynamic class names')
    score = Math.min(score, 2)
  }
  
  // Check for overly specific selectors (too many combinators)
  const combinatorCount = (selector.match(/>/g) || []).length
  if (combinatorCount > 2) {
    issues.push('Overly specific with many nested selectors')
    score = Math.min(score, 3)
  }
  
  // Check for class-based selectors (good)
  if (selector.includes('.') && !selector.includes(':nth-')) {
    reasons.push('Uses class-based selector')
    if (score === 5) score = 4
  }
  
  // Check for attribute selectors (good)
  if (selector.includes('[') && !selector.includes('[data-')) {
    reasons.push('Uses attribute selector')
    if (score === 5) score = 4
  }
  
  // Check for parent context
  if (selector.includes('>') && combinatorCount <= 2) {
    reasons.push('Has parent context for specificity')
  }
  
  // Determine rating based on score
  let rating: SelectorQuality['rating']
  if (score === 5) rating = 'excellent'
  else if (score === 4) rating = 'good'
  else if (score === 3) rating = 'fair'
  else if (score === 2) rating = 'poor'
  else rating = 'fragile'
  
  return { score, rating, reasons, issues }
}

/**
 * Generate alternative selectors for an element
 */
function generateAlternativeSelectors(element: Element, currentSelector: string): AlternativeSelector[] {
  const alternatives: AlternativeSelector[] = []
  const tagName = element.tagName.toLowerCase()
  
  // Try ID-based selector
  if (element.id && !element.id.match(/^[0-9]/) && !element.id.includes(' ')) {
    const selector = `#${CSS.escape(element.id)}`
    if (selector !== currentSelector) {
      const validation = validateSelector(selector)
      if (validation.valid && validation.count > 0) {
        alternatives.push({
          selector,
          quality: rateSelectorQuality(selector, element),
          matchCount: validation.count,
          description: 'ID-based selector (most reliable)'
        })
      }
    }
  }
  
  // Try data attribute selectors
  const dataAttrs = Array.from(element.attributes).filter(attr => 
    attr.name.startsWith('data-') && attr.value
  )
  for (const dataAttr of dataAttrs) {
    const selector = `${tagName}[${dataAttr.name}="${CSS.escape(dataAttr.value)}"]`
    if (selector !== currentSelector) {
      const validation = validateSelector(selector)
      if (validation.valid && validation.count > 0) {
        alternatives.push({
          selector,
          quality: rateSelectorQuality(selector, element),
          matchCount: validation.count,
          description: `Uses ${dataAttr.name} attribute`
        })
      }
    }
  }
  
  // Try class-based selectors
  if (element.classList.length > 0) {
    const classes = Array.from(element.classList)
      .filter(cls => !cls.startsWith('crawlify-') && !cls.match(/^[0-9]/))
    
    // Try with all classes
    if (classes.length > 0) {
      const allClasses = classes.map(cls => `.${CSS.escape(cls)}`).join('')
      const selector = `${tagName}${allClasses}`
      if (selector !== currentSelector) {
        const validation = validateSelector(selector)
        if (validation.valid && validation.count > 0) {
          alternatives.push({
            selector,
            quality: rateSelectorQuality(selector, element),
            matchCount: validation.count,
            description: validation.count === 1 ? 'Class-based unique selector' : `Matches ${validation.count} elements (list pattern)`
          })
        }
      }
      
      // Try with first class only
      if (classes.length > 1) {
        const firstClass = `.${CSS.escape(classes[0])}`
        const selector = `${tagName}${firstClass}`
        if (selector !== currentSelector && !alternatives.some(alt => alt.selector === selector)) {
          const validation = validateSelector(selector)
          if (validation.valid && validation.count > 0) {
            alternatives.push({
              selector,
              quality: rateSelectorQuality(selector, element),
              matchCount: validation.count,
              description: `Simpler class selector (${validation.count} matches)`
            })
          }
        }
      }
    }
  }
  
  // Try meaningful attribute selectors
  const meaningfulAttrs = ['name', 'type', 'rel', 'role', 'aria-label']
  for (const attrName of meaningfulAttrs) {
    const attrValue = element.getAttribute(attrName)
    if (attrValue) {
      const selector = `${tagName}[${attrName}="${CSS.escape(attrValue)}"]`
      if (selector !== currentSelector && !alternatives.some(alt => alt.selector === selector)) {
        const validation = validateSelector(selector)
        if (validation.valid && validation.count > 0) {
          alternatives.push({
            selector,
            quality: rateSelectorQuality(selector, element),
            matchCount: validation.count,
            description: `Uses ${attrName} attribute`
          })
        }
      }
    }
  }
  
  // Sort by quality score (descending), then by match count (ascending for uniqueness)
  alternatives.sort((a, b) => {
    if (b.quality.score !== a.quality.score) {
      return b.quality.score - a.quality.score
    }
    return a.matchCount - b.matchCount
  })
  
  // Return top 3 alternatives
  return alternatives.slice(0, 3)
}
