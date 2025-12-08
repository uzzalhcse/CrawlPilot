// Single field configuration (existing)
export interface FieldConfig {
    selector: string
    type: 'text' | 'attr' | 'html'
    attribute?: string
    multiple?: boolean
    transform?: '' | 'trim' | 'lowercase' | 'uppercase'
}

// Key-Value pair configuration (new)
export interface KeyValuePairConfig {
    key_selector: string
    value_selector: string
    key_type: 'text' | 'attr' | 'html'
    value_type: 'text' | 'attr' | 'html'
    key_attribute?: string
    value_attribute?: string
    transform?: '' | 'trim' | 'lowercase' | 'uppercase'
}

// Individual key-value pair
export interface KeyValuePair {
    id: string
    keyElement: HTMLElement
    valueElement: HTMLElement
    config: KeyValuePairConfig
}

// Group of key-value pairs
export interface KeyValueGroup {
    id: string
    name: string  // e.g., "attributes", "specifications"
    pairs: KeyValuePair[]
    output_format: 'object' | 'array'
}

// Single field type
export interface SingleField {
    id: string
    name: string
    element: HTMLElement
    config: FieldConfig
    fieldType: 'single'
}

// Key-value group field type
export interface KeyValueGroupField {
    id: string
    name: string
    pairs: KeyValuePair[]
    config: {
        output_format: 'object' | 'array'
    }
    fieldType: 'keyvalue'
}

// Discriminated union for all field types
export type Field = SingleField | KeyValueGroupField
