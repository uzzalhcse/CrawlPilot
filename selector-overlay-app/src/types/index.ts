export type SelectionMode = 'single' | 'list' | 'key-value-pairs'

export type FieldType = 'text' | 'attribute' | 'html'

export interface ExtractionPair {
  key_selector: string
  value_selector: string
  key_type: FieldType
  value_type: FieldType
  key_attribute?: string
  value_attribute?: string
  transform?: string
}

export interface KeyValueAttributes {
  extractions: ExtractionPair[]
}

export interface SelectedField {
  id: string
  name: string
  selector: string
  type: FieldType
  attribute?: string
  timestamp: number
  sampleValue?: string
  matchCount?: number
  mode?: SelectionMode
  attributes?: KeyValueAttributes
}

export interface KeyValuePair {
  key: string
  value: string
  index: number
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
