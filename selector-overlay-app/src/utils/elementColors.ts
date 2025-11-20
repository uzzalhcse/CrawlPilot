/**
 * Color schemes for different element types
 * Each type has border, background, and badge colors
 */

export interface ColorScheme {
  border: string
  bg: string
  badge: string
  description: string
}

const colorSchemes: Record<string, ColorScheme> = {
  // Field types
  text: {
    border: 'blue-500',
    bg: 'blue-500/15',
    badge: 'blue-600',
    description: 'Text content'
  },
  attribute: {
    border: 'purple-500',
    bg: 'purple-500/15',
    badge: 'purple-600',
    description: 'Attribute value'
  },
  html: {
    border: 'pink-500',
    bg: 'pink-500/15',
    badge: 'pink-600',
    description: 'HTML content'
  },
  
  // Element types
  button: {
    border: 'green-500',
    bg: 'green-500/15',
    badge: 'green-600',
    description: 'Button'
  },
  input: {
    border: 'yellow-500',
    bg: 'yellow-500/15',
    badge: 'yellow-600',
    description: 'Input field'
  },
  link: {
    border: 'cyan-500',
    bg: 'cyan-500/15',
    badge: 'cyan-600',
    description: 'Link'
  },
  image: {
    border: 'orange-500',
    bg: 'orange-500/15',
    badge: 'orange-600',
    description: 'Image'
  },
  heading: {
    border: 'indigo-500',
    bg: 'indigo-500/15',
    badge: 'indigo-600',
    description: 'Heading'
  },
  default: {
    border: 'gray-500',
    bg: 'gray-500/15',
    badge: 'gray-600',
    description: 'Element'
  }
}

/**
 * Get color scheme for an element or field type
 */
export function getElementColor(type: string): ColorScheme {
  return colorSchemes[type] || colorSchemes.default
}

/**
 * Get all available color schemes (useful for legend)
 */
export function getAllColorSchemes(): Record<string, ColorScheme> {
  return colorSchemes
}

/**
 * Get color legend for field types
 */
export function getFieldTypeLegend(): Array<{ type: string; scheme: ColorScheme }> {
  return [
    { type: 'text', scheme: colorSchemes.text },
    { type: 'attribute', scheme: colorSchemes.attribute },
    { type: 'html', scheme: colorSchemes.html }
  ]
}

/**
 * Get color legend for element types
 */
export function getElementTypeLegend(): Array<{ type: string; scheme: ColorScheme }> {
  return [
    { type: 'button', scheme: colorSchemes.button },
    { type: 'input', scheme: colorSchemes.input },
    { type: 'link', scheme: colorSchemes.link },
    { type: 'image', scheme: colorSchemes.image },
    { type: 'heading', scheme: colorSchemes.heading },
    { type: 'text', scheme: colorSchemes.text }
  ]
}
