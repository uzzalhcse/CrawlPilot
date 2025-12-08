import { ref } from 'vue'
import type { Field, FieldConfig, SingleField, KeyValueGroup, KeyValuePair, KeyValuePairConfig } from '../types'

export function useFieldManager() {
    const fields = ref<Field[]>([])
    const keyValueGroups = ref<KeyValueGroup[]>([])

    function addField(name: string, element: HTMLElement, config: FieldConfig) {
        const field: SingleField = {
            id: crypto.randomUUID(),
            name,
            element,
            config,
            fieldType: 'single'
        }
        fields.value.push(field)
    }

    function updateField(id: string, updates: Partial<Omit<SingleField, 'id' | 'fieldType'>>) {
        const index = fields.value.findIndex(f => f.id === id)
        if (index !== -1 && fields.value[index].fieldType === 'single') {
            fields.value[index] = { ...fields.value[index] as SingleField, ...updates }
        }
    }

    function removeField(id: string) {
        fields.value = fields.value.filter(f => f.id !== id)
    }

    function addKeyValuePair(
        groupName: string,
        labelElement: HTMLElement,
        valueElement: HTMLElement,
        config: KeyValuePairConfig,
        outputFormat: 'object' | 'array' = 'array'
    ) {
        let group = keyValueGroups.value.find(g => g.name === groupName)

        if (!group) {
            group = {
                id: crypto.randomUUID(),
                name: groupName,
                pairs: [],
                output_format: outputFormat
            }
            keyValueGroups.value.push(group)
        }

        const pair: KeyValuePair = {
            id: crypto.randomUUID(),
            keyElement: labelElement,
            valueElement: valueElement,
            config
        }

        group.pairs.push(pair)

        return pair.id
    }

    function removeKeyValuePair(groupId: string, pairId: string) {
        const group = keyValueGroups.value.find(g => g.id === groupId)
        if (group) {
            group.pairs = group.pairs.filter(p => p.id !== pairId)

            // Remove group if empty
            if (group.pairs.length === 0) {
                keyValueGroups.value = keyValueGroups.value.filter(g => g.id !== groupId)
            }
        }
    }

    function removeKeyValueGroup(groupId: string) {
        keyValueGroups.value = keyValueGroups.value.filter(g => g.id !== groupId)
    }

    function undoLastPair(groupName: string) {
        const group = keyValueGroups.value.find(g => g.name === groupName)
        if (group && group.pairs.length > 0) {
            group.pairs.pop()

            // Remove group if empty
            if (group.pairs.length === 0) {
                keyValueGroups.value = keyValueGroups.value.filter(g => g.id !== group.id)
            }
            return true
        }
        return false
    }

    function exportToJSON() {
        const output: Record<string, any> = {}

        // Export single fields
        fields.value.forEach(field => {
            if (field.fieldType === 'single') {
                output[field.name] = field.config
            }
        })

        // Export key-value groups
        keyValueGroups.value.forEach(group => {
            output[group.name] = {
                extractions: group.pairs.map(pair => ({
                    key_selector: pair.config.key_selector,
                    value_selector: pair.config.value_selector,
                    key_type: pair.config.key_type,
                    value_type: pair.config.value_type,
                    ...(pair.config.key_attribute && { key_attribute: pair.config.key_attribute }),
                    ...(pair.config.value_attribute && { value_attribute: pair.config.value_attribute }),
                    ...(pair.config.transform && { transform: pair.config.transform })
                })),
                output_format: group.output_format
            }
        })

        return JSON.stringify(output, null, 2)
    }

    return {
        fields,
        keyValueGroups,
        addField,
        updateField,
        removeField,
        addKeyValuePair,
        removeKeyValuePair,
        removeKeyValueGroup,
        undoLastPair,
        exportToJSON
    }
}
