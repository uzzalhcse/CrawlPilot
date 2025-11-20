export type SelectionMode = 'single' | 'list'

export type FieldType = 'text' | 'attribute' | 'html'

export interface SelectedField {
  id: string
  name: string
  selector: string
  type: FieldType
  attribute?: string
  timestamp: number
  sampleValue?: string
  matchCount?: number
}

export interface ElementInfo {
  element: Element
  selector: string
  count: number
  validation: ValidationResult
}

export interface ValidationResult {
  isValid: boolean
  isUnique: boolean
  message: string
}

export interface TestResult {
  element: Element
  index: number
  value: string
}

export interface Highlight {
  element: Element
  type: 'hover' | 'locked' | 'selected' | 'test'
  rect?: DOMRect
}

// Global window interface extensions
declare global {
  interface Window {
    __crawlifyGetSelections?: () => SelectedField[]
    __crawlifyApp?: any
  }
}
