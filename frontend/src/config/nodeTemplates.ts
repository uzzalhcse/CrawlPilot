import type { NodeTemplate, NodeCategory } from '@/types'

export const nodeTemplates: NodeTemplate[] = [
  // URL Discovery Nodes
  {
    type: 'extract_links',
    label: 'Extract Links',
    description: 'Extract all links from the page',
    category: 'URL Discovery',
    defaultParams: {
      selector: 'a',
      limit: 0,
      marker: ''
    },
    paramSchema: [
      {
        key: 'selector',
        label: 'CSS Selector',
        type: 'text',
        required: true,
        defaultValue: 'a',
        placeholder: 'a, .link-class'
      },
      {
        key: 'marker',
        label: 'URL Marker/Tag',
        type: 'text',
        placeholder: 'category',
        description: 'Tag/marker to assign to discovered URLs (e.g., "product", "category")'
      },
      {
        key: 'limit',
        label: 'Limit',
        type: 'number',
        defaultValue: 0,
        description: 'Maximum links to extract (0 = unlimited)'
      }
    ]
  },

  {
    type: 'paginate',
    label: 'Paginate',
    description: 'Navigate through paginated content',
    category: 'URL Discovery',
    defaultParams: {
      selector: '',
      max_pages: 10,
      type: 'auto',
      wait_after: 0,
      link_selector: '',
      marker: '',
      limit: 0
    },
    paramSchema: [
      {
        key: 'selector',
        label: 'Next Button Selector',
        type: 'text',
        required: true,
        placeholder: '.next-page, a[rel="next"]',
        description: 'Selector for the "Next" button to click for pagination'
      },
      {
        key: 'link_selector',
        label: 'Link Selector',
        type: 'text',
        placeholder: 'ul.product-list a',
        description: 'Selector for product/item links to extract from each page'
      },
      {
        key: 'marker',
        label: 'URL Marker/Tag',
        type: 'text',
        placeholder: 'product',
        description: 'Tag/marker to assign to discovered URLs (e.g., "product", "category")'
      },
      {
        key: 'limit',
        label: 'Links Limit Per Page',
        type: 'number',
        defaultValue: 0,
        description: 'Max links to extract per page (0 = unlimited)'
      },
      {
        key: 'type',
        label: 'Pagination Type',
        type: 'select',
        defaultValue: 'auto',
        options: [
          { label: 'Auto (Next Button)', value: 'auto' },
          { label: 'Manual (URL Pattern)', value: 'manual' }
        ]
      },
      {
        key: 'max_pages',
        label: 'Max Pages',
        type: 'number',
        defaultValue: 10
      },
      {
        key: 'wait_after',
        label: 'Wait After Navigation (ms)',
        type: 'number',
        defaultValue: 0,
        description: 'Time to wait after navigating to next page'
      }
    ]
  },

  // Extraction Nodes
  {
    type: 'extract',
    label: 'Extract Data',
    description: 'Extract structured data from the page',
    category: 'Extraction',
    defaultParams: {
      schema: '',
      fields: {},
      timeout: 10000,
      limit: 0
    },
    paramSchema: [
      {
        key: 'schema',
        label: 'Schema Name',
        type: 'text',
        required: true,
        placeholder: 'product_data, article_data',
        description: 'Name of the data schema being extracted'
      },
      {
        key: 'fields',
        label: 'Field Definitions',
        type: 'field_array',
        required: true,
        description: 'Define fields to extract from the page. Supports single values, simple arrays, and nested object arrays.',
        arrayItemSchema: [
          {
            key: 'name',
            label: 'Field Name',
            type: 'text',
            required: true,
            placeholder: 'title, images, attributes'
          },
          {
            key: 'selector',
            label: 'CSS Selector',
            type: 'text',
            required: true,
            placeholder: '.product-title, .gallery img, table.specs tr'
          },
          {
            key: 'type',
            label: 'Extraction Type',
            type: 'select',
            defaultValue: 'text',
            options: [
              { label: 'Text', value: 'text' },
              { label: 'Attribute', value: 'attr' },
              { label: 'HTML', value: 'html' },
              { label: 'Href', value: 'href' },
              { label: 'Src', value: 'src' }
            ]
          },
          {
            key: 'attribute',
            label: 'Attribute Name',
            type: 'text',
            placeholder: 'src, href, data-id, alt',
            description: 'Required for "attr" type - specify which HTML attribute to extract'
          },
          {
            key: 'multiple',
            label: 'Extract Multiple Values',
            type: 'boolean',
            defaultValue: false,
            description: 'Extract array of values instead of single value'
          },
          {
            key: 'required',
            label: 'Required Field',
            type: 'boolean',
            defaultValue: true,
            description: 'Mark this field as required. Missing required fields will cause health check failure, while missing optional fields only trigger warnings.'
          },
          {
            key: 'limit',
            label: 'Array Limit',
            type: 'number',
            defaultValue: 0,
            description: 'Max items in array (0 = unlimited). Only applies when "multiple" is enabled'
          },
          {
            key: 'transform',
            label: 'Transform',
            type: 'select',
            defaultValue: 'none',
            options: [
              { label: 'None', value: 'none' },
              { label: 'Trim', value: 'trim' },
              { label: 'Clean HTML', value: 'clean_html' },
              { label: 'Lowercase', value: 'lowercase' },
              { label: 'Uppercase', value: 'uppercase' },
              { label: 'Extract Price', value: 'extract_price' }
            ]
          },
          {
            key: 'default_value',
            label: 'Default Value',
            type: 'text',
            placeholder: 'N/A, 0, []',
            description: 'Value to use if extraction fails'
          },
          {
            key: 'fields',
            label: 'Nested Fields (for object arrays)',
            type: 'textarea',
            placeholder: '{\n  "key": {"selector": "th", "type": "text"},\n  "value": {"selector": "td", "type": "text"}\n}',
            description: 'JSON object defining nested fields. Only used when "multiple" is true for extracting array of objects'
          },
          {
            key: 'extractions',
            label: 'Independent Array Extractions',
            type: 'textarea',
            placeholder: '[\n  {\n    "key_selector": ".spec-label",\n    "value_selector": ".spec-value",\n    "key_type": "text",\n    "value_type": "text",\n    "transform": "trim"\n  }\n]',
            description: 'JSON array for extracting key-value pairs from independent selectors (not nested). Use this when keys and values are in separate lists that need to be paired by index.'
          }
        ]
      },
      {
        key: 'timeout',
        label: 'Timeout (ms)',
        type: 'number',
        defaultValue: 10000,
        description: 'Maximum time to wait for extraction'
      },
      {
        key: 'limit',
        label: 'Item Limit',
        type: 'number',
        defaultValue: 0,
        description: 'Maximum number of items to extract (0 = unlimited)'
      }
    ]
  },

  {
    type: 'extract_json',
    label: 'Extract JSON',
    description: 'Extract JSON data from script tags or API responses',
    category: 'Extraction',
    defaultParams: {
      selector: 'script[type="application/ld+json"]',
      path: ''
    },
    paramSchema: [
      {
        key: 'selector',
        label: 'CSS Selector',
        type: 'text',
        required: true,
        placeholder: 'script[type="application/ld+json"]',
        description: 'Selector for element containing JSON data'
      },
      {
        key: 'path',
        label: 'JSON Path',
        type: 'text',
        placeholder: '$.data.items',
        description: 'Optional JSON path to extract specific data'
      }
    ]
  },

  // Interaction Nodes
  {
    type: 'navigate',
    label: 'Navigate',
    description: 'Navigate to a URL',
    category: 'URL Discovery',
    defaultParams: {
      url: '',
      wait_until: 'load'
    },
    paramSchema: [
      {
        key: 'url',
        label: 'URL',
        type: 'text',
        required: true,
        placeholder: 'https://example.com'
      },
      {
        key: 'wait_until',
        label: 'Wait Until',
        type: 'select',
        defaultValue: 'load',
        options: [
          { label: 'Load', value: 'load' },
          { label: 'DOM Content Loaded', value: 'domcontentloaded' },
          { label: 'Network Idle', value: 'networkidle' }
        ],
        description: 'When to consider navigation succeeded'
      }
    ]
  },
  {
    type: 'click',
    label: 'Click Element',
    description: 'Click on an element',
    category: 'Interaction',
    defaultParams: {
      selector: ''
    },
    paramSchema: [
      {
        key: 'selector',
        label: 'CSS Selector',
        type: 'text',
        required: true,
        placeholder: 'button.load-more'
      }
    ]
  },
  {
    type: 'scroll',
    label: 'Scroll',
    description: 'Scroll the page by coordinates',
    category: 'Interaction',
    defaultParams: {
      x: 0,
      y: 1000
    },
    paramSchema: [
      {
        key: 'x',
        label: 'Horizontal Scroll (px)',
        type: 'number',
        defaultValue: 0,
        description: 'Pixels to scroll horizontally (positive = right, negative = left)'
      },
      {
        key: 'y',
        label: 'Vertical Scroll (px)',
        type: 'number',
        defaultValue: 1000,
        description: 'Pixels to scroll vertically (positive = down, negative = up)'
      }
    ]
  },
  {
    type: 'type',
    label: 'Type Text',
    description: 'Type text into an input field',
    category: 'Interaction',
    defaultParams: {
      selector: '',
      text: '',
      delay: 0
    },
    paramSchema: [
      {
        key: 'selector',
        label: 'CSS Selector',
        type: 'text',
        required: true,
        placeholder: 'input[name="search"]'
      },
      {
        key: 'text',
        label: 'Text to Type',
        type: 'text',
        required: true,
        placeholder: 'Search query'
      },
      {
        key: 'delay',
        label: 'Delay Between Keys (ms)',
        type: 'number',
        defaultValue: 0,
        description: 'Delay between keystrokes to simulate human typing'
      }
    ]
  },
  {
    type: 'hover',
    label: 'Hover Element',
    description: 'Hover over an element',
    category: 'Interaction',
    defaultParams: {
      selector: ''
    },
    paramSchema: [
      {
        key: 'selector',
        label: 'CSS Selector',
        type: 'text',
        required: true,
        placeholder: '.menu-item'
      }
    ]
  },
  {
    type: 'wait',
    label: 'Wait',
    description: 'Wait for specified duration or element',
    category: 'Interaction',
    defaultParams: {
      duration: 1000,
      selector: '',
      state: 'visible',
      timeout: 5000
    },
    paramSchema: [
      {
        key: 'duration',
        label: 'Duration (ms)',
        type: 'number',
        defaultValue: 1000,
        description: 'Time to wait (leave empty if waiting for selector)'
      },
      {
        key: 'selector',
        label: 'CSS Selector (Optional)',
        type: 'text',
        placeholder: '.dynamic-content',
        description: 'Wait for this element to appear (overrides duration)'
      },
      {
        key: 'state',
        label: 'Element State',
        type: 'select',
        defaultValue: 'visible',
        options: [
          { label: 'Visible', value: 'visible' },
          { label: 'Hidden', value: 'hidden' },
          { label: 'Attached', value: 'attached' }
        ],
        description: 'Only applies when waiting for selector'
      },
      {
        key: 'timeout',
        label: 'Timeout (ms)',
        type: 'number',
        defaultValue: 5000,
        description: 'Max time to wait for selector (only applies when selector is set)'
      }
    ]
  },

  {
    type: 'screenshot',
    label: 'Screenshot',
    description: 'Capture screenshot of page or element',
    category: 'Interaction',
    defaultParams: {
      selector: '',
      filename: '',
      path: 'screenshots',
      full_page: true
    },
    paramSchema: [
      {
        key: 'selector',
        label: 'CSS Selector (Optional)',
        type: 'text',
        placeholder: '.product-image, #main-content',
        description: 'Capture specific element (leave empty for full page)'
      },
      {
        key: 'filename',
        label: 'Filename',
        type: 'text',
        placeholder: 'product_screenshot.png',
        description: 'Screenshot filename (auto-generated if empty)'
      },
      {
        key: 'path',
        label: 'Directory Path',
        type: 'text',
        defaultValue: 'screenshots',
        placeholder: 'screenshots, evidence/errors',
        description: 'Directory to save screenshot'
      },
      {
        key: 'full_page',
        label: 'Full Page Screenshot',
        type: 'boolean',
        defaultValue: true,
        description: 'Capture entire scrollable page (only for full page, not element)'
      }
    ]
  },

  {
    type: 'sequence',
    label: 'Sequence',
    description: 'Execute multiple nodes in order (for complex interactions)',
    category: 'Control Flow',
    defaultParams: {
      steps: []
    },
    paramSchema: [
      {
        key: 'steps',
        label: 'Execution Steps',
        type: 'sequence_steps',
        required: true,
        description: 'Define a sequence of actions to execute in order',
        arrayItemSchema: [
          {
            key: 'type',
            label: 'Node Type',
            type: 'select',
            required: true,
            options: [
              { label: 'Click', value: 'click' },
              { label: 'Hover', value: 'hover' },
              { label: 'Scroll', value: 'scroll' },
              { label: 'Type', value: 'type' },
              { label: 'Wait', value: 'wait' },
              { label: 'Extract', value: 'extract' },
              { label: 'Navigate', value: 'navigate' }
            ]
          },
          {
            key: 'params',
            label: 'Parameters (JSON)',
            type: 'textarea',
            required: true,
            placeholder: '{\n  "selector": ".button",\n  "timeout": 5000\n}',
            description: 'JSON object with parameters for this step'
          },
          {
            key: 'optional',
            label: 'Optional Step',
            type: 'boolean',
            defaultValue: false,
            description: 'Continue execution even if this step fails'
          }
        ]
      }
    ]
  },
  {
    type: 'conditional',
    label: 'Conditional',
    description: 'Execute based on condition',
    category: 'Control Flow',
    defaultParams: {
      condition: ''
    },
    paramSchema: [
      {
        key: 'condition',
        label: 'Condition',
        type: 'text',
        required: true,
        placeholder: 'data.price > 100'
      }
    ]
  },

]

export const nodeCategories: NodeCategory[] = [
  {
    name: 'URL Discovery',
    icon: 'Globe',
    nodes: nodeTemplates.filter(n => n.category === 'URL Discovery')
  },
  {
    name: 'Extraction',
    icon: 'Database',
    nodes: nodeTemplates.filter(n => n.category === 'Extraction')
  },
  {
    name: 'Interaction',
    icon: 'MousePointer',
    nodes: nodeTemplates.filter(n => n.category === 'Interaction')
  },

  {
    name: 'Control Flow',
    icon: 'GitBranch',
    nodes: nodeTemplates.filter(n => n.category === 'Control Flow')
  }
]

export function getNodeTemplate(type: string): NodeTemplate | undefined {
  return nodeTemplates.find(n => n.type === type)
}
