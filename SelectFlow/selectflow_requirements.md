# Build Prompt: visual element selector (Inline Badge UI)

## Project Goal
Create a visual element selector overlay that allows users to click HTML elements and configure extraction rules. Selected fields appear as **inline badges** directly on the DOM elements (no floating panel). Export generates JSON configuration matching Crawlify workflow format.

---

## Tech Stack (MUST USE)
- **Framework**: Vue 3 with Composition API + TypeScript
- **Build Tool**: Vite
- **Styling**: Tailwind CSS v3
- **Selector Generation**: `@medv/finder` (npm package)
- **Element Highlighting**: `driver.js` (npm package)
- **Icons**: `lucide-vue-next`
- **UI**: Custom components with Tailwind (keep simple, NO complex libraries)

---

## Core Concept: Inline Badges (No Floating Panel!)

### Visual Design:
```
┌─────────────────────────────────┐
│  Product Title Element          │  ← Actual DOM element
│  "Wireless Headphones"          │
│  ┌───────────────────────────┐  │
│  │ 🏷️ product_title  [✏️] [🗑️]│  │  ← Always-visible badge
│  │ text • 1 match            │  │
│  └───────────────────────────┘  │
└─────────────────────────────────┘
```

**Key Points:**
- Badges appear directly on selected elements
- Always visible (not just on hover)
- Positioned inline absolute smartly (adjusts to avoid overflow)
- Each badge shows: field name, type, match count, edit/delete buttons

---

## Field Types & Naming Patterns

### 1. **Simple Field**
```
Name: product_name
JSON: {
  "product_name": {
    "selector": ".product-title",
    "type": "text",
    "transform": "trim"
  }
}
```

### 2. **Array Field** (Multiple)
```
Name: images
Multiple: ✓ (checkbox enabled)
JSON: {
  "images": {
    "selector": ".gallery img",
    "type": "attr",
    "attribute": "src",
    "multiple": true
  }
}
```

### 3. **Key-Value Pairs** (Smart Naming)
```
Names: 
  - attributes.key_selector1
  - attributes.value_selector1
  - attributes.key_selectorA
  - attributes.value_selectorA

JSON: {
  "attributes": {
    "extractions": [
      {
        "key_selector": "...",
        "value_selector": "...",
        "key_type": "text",
        "value_type": "text",
        "transform": "trim"
      },
      {
        "key_selector": "...",
        "value_selector": "...",
        "key_type": "text",
        "value_type": "text",
        "transform": "trim"
      }
    ],
    "output_format": "array"
  }
}
```

**Pattern Rules:**
- Format: `{group}.{type}_{identifier}`
- `{group}`: attributes, specs, details, etc.
- `{type}`: "key" or "value"
- `{identifier}`: selector1, selectorA, color, etc.
- System auto-pairs matching identifiers

---

## User Workflows

### **Workflow 1: Add Simple Field**
1. User clicks element → Quick-add tooltip appears
2. Tooltip shows suggested name and selector
3. User clicks "Add" → Badge appears on element
4. Or clicks "Configure" → Opens edit popover for full options

### **Workflow 2: Add Array Field (Multiple)**
1. Same as simple field
2. In edit popover, check "Extract multiple" checkbox
3. Badge shows purple color to indicate array type

### **Workflow 3: Add Key-Value Pairs** (Sequential Flow)

#### Step 1: Add Key
```
User clicks spec label element
Quick-add tooltip:
┌──────────────────────────────┐
│ Quick Add                    │
│ Name: attributes.key_1       │
│ Type: [text ▼]               │
│ [Add Key & Continue →]       │
│ [Add Key Only]               │
└──────────────────────────────┘
```

#### Step 2: System Enters "Value Selection Mode"
```
Page dims slightly
Key badge appears with pulsing border:
┌──────────────────────┐
│ 🔑 attributes        │
│ key_1                │
│ ⏳ Waiting for pair  │
└──────────────────────┘

Overlay message:
"Select the value element for attributes.key_1
Press ESC to cancel"
```

#### Step 3: Add Value
```
User clicks value element
Quick-add tooltip (auto-filled!):
┌──────────────────────────────┐
│ Pair Value                   │
│ Name: attributes.value_1     │ ← Auto-generated!
│ Type: [text ▼]               │
│ [Add & Complete ✓]           │
└──────────────────────────────┘
```

#### Step 4: Pair Complete
```
[Label Element]           [Value Element]
┌──────────────┐         ┌──────────────┐
│ 🔑 attributes│─────────│🔓 attributes │
│ key_1        │         │ value_1      │
│ [✏️] [🗑️]    │         │ [✏️] [🗑️]    │
└──────────────┘         └──────────────┘
     ↑ Visual connection line ↑
```

### **Workflow 4: Edit Field**
1. Click ✏️ icon on badge → Edit popover opens
2. Modify settings → Click Save
3. Badge updates

### **Workflow 5: Delete Field**
1. Click 🗑️ icon on badge → Confirmation dialog
2. Confirm → Badge disappears
3. For K-V pairs: Ask "Also delete paired value?" if deleting key

### **Workflow 6: Export**
1. Click "Copy JSON" in minimal toolbar (top-right corner)
2. Or press keyboard shortcut: `Ctrl+Shift+E`
3. JSON copied to clipboard in exact Crawlify format

---

## UI Components

### 1. **Field Badge** (Main Component)

**Always visible, positioned on each selected element**

```vue
<template>
  <div class="field-badge" :class="badgeClass" :style="position">
    <!-- Icon based on type -->
    <span class="icon">
      {{ isPairKey ? '🔑' : isPairValue ? '🔓' : '🏷️' }}
    </span>
    
    <!-- Field info -->
    <div class="info">
      <span class="name">{{ displayName }}</span>
      <span class="meta">{{ field.type }} • {{ matchCount }} match(es)</span>
      <span v-if="isPaired" class="paired">✓ Paired</span>
      <span v-if="!isPaired && isPairType" class="unpaired">⚠️ No pair</span>
    </div>
    
    <!-- Actions -->
    <div class="actions">
      <button @click="edit" title="Edit">✏️</button>
      <button @click="remove" title="Delete">🗑️</button>
      <button v-if="!isPaired && isPairKey" @click="addValue">
        + Value
      </button>
    </div>
  </div>
</template>
```

**Badge Colors:**
- 🔵 Blue: Simple field
- 🟣 Purple: Array field (multiple: true)
- 🟡 Gold: K-V key
- 🟢 Green: K-V value (paired)
- 🟠 Orange: K-V value (unpaired warning)

**Positioning:** position badge at bottom-right of element, but adjust if:
- Element near screen edge
- Element too small
- Multiple badges on same element (stack them)

### 2. **Quick-Add Tooltip**

**Appears when user clicks an element**

```vue
<template>
  <div class="quick-add-tooltip" :style="position">
    <h4>Quick Add Field</h4>
    
    <label>Field Name</label>
    <input v-model="fieldName" placeholder="e.g., product_name" />
    
    <label>Type</label>
    <select v-model="fieldType">
      <option value="text">Text Content</option>
      <option value="attr">Attribute</option>
      <option value="html">HTML</option>
    </select>
    
    <input v-if="fieldType === 'attr'" 
           v-model="attribute" 
           placeholder="Attribute name (e.g., src)" />
    
    <div class="preview">
      <strong>Selector:</strong>
      <code>{{ selector }}</code>
      <span class="match-count">{{ matchCount }} matches</span>
    </div>
    
    <div class="actions">
      <!-- If name suggests K-V pair -->
      <button v-if="isKeyField" @click="addKeyAndContinue" class="primary">
        Add Key & Continue →
      </button>
      <button v-if="isKeyField" @click="addOnly" class="secondary">
        Add Key Only
      </button>
      
      <!-- Regular field -->
      <button v-else @click="add" class="primary">Add</button>
      <button @click="configure" class="secondary">Configure</button>
      <button @click="cancel" class="ghost">Cancel</button>
    </div>
  </div>
</template>
```

### 3. **Edit Popover** (Full Configuration)

**Opens when user clicks ✏️ edit button**

```vue
<template>
  <div class="edit-popover" :style="position">
    <h4>Edit Field: {{ field.name }}</h4>
    
    <label>Field Name</label>
    <input v-model="field.name" />
    
    <label>Extract Type</label>
    <select v-model="field.type">
      <option value="text">Text Content</option>
      <option value="attr">Attribute</option>
      <option value="html">HTML</option>
    </select>
    
    <input v-if="field.type === 'attr'" 
           v-model="field.attribute" 
           placeholder="Attribute name" />
    
    <label>
      <input type="checkbox" v-model="field.multiple" />
      Extract multiple values (array)
    </label>
    
    <label>Transform</label>
    <select v-model="field.transform">
      <option value="">None</option>
      <option value="trim">Trim whitespace</option>
      <option value="clean_html">Clean HTML</option>
      <option value="lowercase">Lowercase</option>
      <option value="uppercase">Uppercase</option>
      <option value="extract_number">Extract number</option>
      <option value="extract_price">Extract price</option>
    </select>
    
    <label>Default Value (optional)</label>
    <input v-model="field.default" placeholder="Fallback if not found" />
    
    <div class="preview">
      <strong>Live Preview:</strong>
      <code>{{ previewValue }}</code>
    </div>
    
    <div class="actions">
      <button @click="save" class="primary">Save ✓</button>
      <button @click="cancel" class="secondary">Cancel</button>
    </div>
  </div>
</template>
```

### 4. **Minimal Toolbar** (Top-Right)

**Only for global actions, small and unobtrusive**

```vue
<template>
  <div class="toolbar">
    <span class="field-count">{{ fieldCount }} fields</span>
    <button @click="copyJSON" title="Copy JSON (Ctrl+Shift+E)">
      📋 Copy JSON
    </button>
    <button @click="downloadJSON" title="Download">
      ⬇️ Download
    </button>
    <button @click="clearAll" title="Clear all fields" class="danger">
      🗑️ Clear All
    </button>
  </div>
</template>
```

**Style:**
- Position: `fixed top-4 right-4`
- Size: Compact, ~250px width
- Auto-hide after 5 seconds of inactivity
- Reappears on mouse move to top-right corner

---

## Implementation Details

### **Type Definitions** (src/types/field.ts)

```typescript
export interface FieldConfig {
  selector: string
  type: 'text' | 'attr' | 'html'
  attribute?: string
  multiple?: boolean
  transform?: Transform
  default?: string
}

export interface FieldWithElement {
  id: string
  name: string
  element: Element
  config: FieldConfig
  matchCount: number
}

export interface KeyValuePair {
  group: string
  identifier: string
  keyField: FieldWithElement
  valueField: FieldWithElement
}

export type Transform = 
  | 'trim' 
  | 'clean_html' 
  | 'lowercase' 
  | 'uppercase' 
  | 'extract_number' 
  | 'extract_price'
  | ''
```

### **Composables**

#### useFieldManager.ts
```typescript
- fields: Ref<FieldWithElement[]>
- addField(field): Add new field
- updateField(id, updates): Update field
- removeField(id): Delete field (handles K-V pair cleanup)
- exportToJSON(): Convert to Crawlify format with K-V grouping
- getKeyValuePairs(): Get all paired K-V fields grouped
- findPair(fieldName): Find matching key or value
```

#### useElementSelection.ts
```typescript
- selectedElement: Ref<Element | null>
- isSelectingValue: Ref<boolean> // For K-V pair value mode
- pendingKeyField: Ref<string | null> // Key waiting for value
- selectElement(element): Handle element click
- startValueSelection(keyFieldName): Enter value selection mode
- cancelValueSelection(): Exit value selection mode
```

#### useSelectorGenerator.ts
```typescript
- generateSelector(element): Use @medv/finder
- validateSelector(selector): Get match count
- getMatches(selector): Get all matching elements
```

### **Auto-Pairing Logic** (K-V Pairs)

```typescript
function parseFieldName(name: string) {
  const match = name.match(/^(.+)\.(key|value)_(.+)$/)
  if (!match) return null
  
  return {
    group: match[1],      // "attributes"
    type: match[2],       // "key" or "value"
    identifier: match[3]  // "selector1"
  }
}

function findPairFor(fieldName: string, fields: FieldWithElement[]) {
  const parsed = parseFieldName(fieldName)
  if (!parsed) return null
  
  const oppositeType = parsed.type === 'key' ? 'value' : 'key'
  const pairName = `${parsed.group}.${oppositeType}_${parsed.identifier}`
  
  return fields.find(f => f.name === pairName)
}

function exportToJSON(fields: FieldWithElement[]): Record<string, any> {
  const output: Record<string, any> = {}
  const processedPairs = new Set<string>()
  
  // Group K-V pairs
  const kvGroups: Record<string, any[]> = {}
  
  fields.forEach(field => {
    const parsed = parseFieldName(field.name)
    
    if (parsed && parsed.type === 'key' && !processedPairs.has(field.name)) {
      const valuePair = findPairFor(field.name, fields)
      
      if (valuePair) {
        if (!kvGroups[parsed.group]) {
          kvGroups[parsed.group] = []
        }
        
        kvGroups[parsed.group].push({
          key_selector: field.config.selector,
          value_selector: valuePair.config.selector,
          key_type: field.config.type,
          value_type: valuePair.config.type,
          transform: field.config.transform
        })
        
        processedPairs.add(field.name)
        processedPairs.add(valuePair.name)
      }
    } else if (!parsed || !processedPairs.has(field.name)) {
      // Simple or array field
      output[field.name] = field.config
    }
  })
  
  // Add K-V groups
  Object.entries(kvGroups).forEach(([group, extractions]) => {
    output[group] = {
      extractions,
      output_format: 'array'
    }
  })
  
  return output
}
```

---

## Expected JSON Output

```json
{
  "product_name": {
    "selector": ".product-title",
    "type": "text",
    "transform": "trim"
  },
  "images": {
    "selector": ".gallery img",
    "type": "attr",
    "attribute": "src",
    "multiple": true
  },
  "price": {
    "selector": ".price",
    "type": "text",
    "transform": "extract_price",
    "default": "0"
  },
  "attributes": {
    "extractions": [
      {
        "key_selector": ".spec-label:nth-child(1)",
        "value_selector": ".spec-value:nth-child(1)",
        "key_type": "text",
        "value_type": "text",
        "transform": "trim"
      },
      {
        "key_selector": ".spec-label:nth-child(2)",
        "value_selector": ".spec-value:nth-child(2)",
        "key_type": "text",
        "value_type": "text",
        "transform": "trim"
      }
    ],
    "output_format": "array"
  }
}
```

---

## Critical Requirements

1. ✅ **Inline Badges** - No floating panel, badges on elements
2. ✅ **Always Visible** - Badges persist, not just on hover
3. ✅ **Sequential K-V** - Add key, then immediately add value
4. ✅ **Smart Pairing** - Auto-detect and pair K-V fields by naming
5. ✅ **Visual Connections** - Show lines connecting paired K-V fields
6. ✅ **Full Config** - All options in edit popover (transform, default, etc.)
7. ✅ **Event Prevention** - Prevent default link/form clicks
8. ✅ **Smart Positioning** - Badges adjust to avoid screen edges
9. ✅ **Type Safety** - Full TypeScript typing
10. ✅ **@medv/finder** - For optimal selector generation

---

## Success Criteria

✅ Click element → Badge appears on it  
✅ Badge always visible, positioned smartly  
✅ Edit badge → Full configuration options  
✅ Delete badge → Field removed  
✅ Add K-V key → System prompts for value  
✅ Add K-V value → Auto-pairs with key  
✅ Visual line connects K-V pairs  
✅ Export → Perfect Crawlify JSON format  
✅ Multiple K-V groups supported  
✅ No console errors, TypeScript compiles  



**BUILD THIS EXACTLY. Focus on inline badges, sequential K-V workflow, and smart auto-pairing.**
