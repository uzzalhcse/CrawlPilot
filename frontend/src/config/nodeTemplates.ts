import type { NodeTemplate, NodeCategory } from '@/types'

export const nodeTemplates: NodeTemplate[] = [
  // URL Discovery Nodes
  {
    type: 'navigate',
    label: 'Navigate',
    description: 'Navigate to a URL',
    category: 'URL Discovery',
    defaultParams: {
      url: '',
      timeout: 60000,
      wait_until: 'load'
    },
    paramSchema: [
      {
        key: 'url',
        label: 'URL',
        type: 'text',
        required: true,
        placeholder: 'https://example.com or {{current_url}}'
      },
      {
        key: 'timeout',
        label: 'Timeout (ms)',
        type: 'number',
        defaultValue: 60000,
        description: 'Maximum time to wait for navigation'
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
    type: 'extract_links',
    label: 'Extract Links',
    description: 'Extract all links from the page with marker support',
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
        placeholder: 'a, .link-class, a.product-link'
      },
      {
        key: 'marker',
        label: 'URL Marker/Tag',
        type: 'text',
        placeholder: 'product, category',
        description: 'Tag/marker to assign to discovered URLs for phase filtering'
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
    description: 'Navigate through paginated content and collect links',
    category: 'URL Discovery',
    defaultParams: {
      selector: '',
      link_selector: '',
      marker: '',
      max_pages: 10,
      wait_between_pages: 1500,
      timeout: 30000
    },
    paramSchema: [
      {
        key: 'selector',
        label: 'Next Button Selector',
        type: 'text',
        required: true,
        placeholder: 'a[aria-label="Next page"], .pagination .next',
        description: 'Selector for the "Next" button to click for pagination'
      },
      {
        key: 'link_selector',
        label: 'Item Link Selector',
        type: 'text',
        required: true,
        placeholder: 'a.card-header, .product-list a',
        description: 'Selector for product/item links to extract from each page'
      },
      {
        key: 'marker',
        label: 'URL Marker/Tag',
        type: 'text',
        placeholder: 'product',
        description: 'Tag/marker to assign to discovered URLs for phase filtering'
      },
      {
        key: 'max_pages',
        label: 'Max Pages',
        type: 'number',
        defaultValue: 10,
        description: 'Maximum number of pages to paginate through'
      },
      {
        key: 'wait_between_pages',
        label: 'Wait Between Pages (ms)',
        type: 'number',
        defaultValue: 1500,
        description: 'Time to wait after clicking next before extracting links'
      },
      {
        key: 'timeout',
        label: 'Timeout (ms)',
        type: 'number',
        defaultValue: 30000,
        description: 'Maximum time to wait for page load'
      }
    ]
  },

  // Extraction Nodes
  {
    type: 'extract',
    label: 'Extract Data',
    description: 'Extract structured data with field-level actions support',
    category: 'Extraction',
    defaultParams: {
      schema: '',
      fields: {},
      timeout: 10000
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
        description: 'Define fields to extract. Each field can have pre-extraction actions.',
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
            placeholder: '.product-title, .gallery img'
          },
          {
            key: 'type',
            label: 'Extraction Type',
            type: 'select',
            defaultValue: 'text',
            options: [
              { label: 'Text', value: 'text' },
              { label: 'Attribute', value: 'attr' },
              { label: 'HTML', value: 'html' }
            ]
          },
          {
            key: 'attribute',
            label: 'Attribute Name',
            type: 'text',
            placeholder: 'src, href, data-id, alt',
            description: 'Required for "attr" type'
          },
          {
            key: 'multiple',
            label: 'Extract Multiple Values',
            type: 'boolean',
            defaultValue: false,
            description: 'Extract array of values instead of single value'
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
              { label: 'Uppercase', value: 'uppercase' }
            ]
          },
          {
            key: 'default',
            label: 'Default Value',
            type: 'text',
            placeholder: 'N/A, 0, []',
            description: 'Value to use if extraction fails'
          },
          {
            key: 'actions',
            label: 'Pre-Extraction Actions',
            type: 'textarea',
            placeholder: '[\n  {\n    "id": "wait_for_price",\n    "type": "wait_for",\n    "name": "Wait for Price",\n    "params": {\n      "condition": "selector",\n      "selector": ".price",\n      "timeout": 5000\n    }\n  }\n]',
            description: 'JSON array of action nodes to execute before extracting this field'
          }
        ]
      },
      {
        key: 'timeout',
        label: 'Timeout (ms)',
        type: 'number',
        defaultValue: 10000,
        description: 'Maximum time to wait for extraction'
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

  {
    type: 'extractField',
    label: 'Extraction Field',
    description: 'Single field extraction (Virtual Node)',
    category: 'Extraction',
    defaultParams: {
      selector: '',
      type: 'text',
      transform: 'none',
      multiple: false,
      limit: 0,
      default_value: ''
    },
    paramSchema: []
  },

  // Interaction Nodes
  {
    type: 'click',
    label: 'Click Element',
    description: 'Click on an element',
    category: 'Interaction',
    defaultParams: {
      selector: '',
      wait_after: 0
    },
    paramSchema: [
      {
        key: 'selector',
        label: 'CSS Selector',
        type: 'text',
        required: true,
        placeholder: 'button.load-more, .show-details'
      },
      {
        key: 'wait_after',
        label: 'Wait After Click (ms)',
        type: 'number',
        defaultValue: 0,
        description: 'Time to wait after clicking'
      }
    ]
  },

  {
    type: 'scroll',
    label: 'Scroll',
    description: 'Scroll the page or to a specific element',
    category: 'Interaction',
    defaultParams: {
      selector: '',
      x: 0,
      y: 0,
      to_bottom: false,
      wait_after: 500
    },
    paramSchema: [
      {
        key: 'selector',
        label: 'Scroll to Element (Optional)',
        type: 'text',
        placeholder: '.specs-section, #footer',
        description: 'Scroll to a specific element (overrides x/y)'
      },
      {
        key: 'to_bottom',
        label: 'Scroll to Bottom',
        type: 'boolean',
        defaultValue: false,
        description: 'Scroll to the bottom of the page'
      },
      {
        key: 'x',
        label: 'Horizontal Scroll (px)',
        type: 'number',
        defaultValue: 0,
        description: 'Pixels to scroll horizontally'
      },
      {
        key: 'y',
        label: 'Vertical Scroll (px)',
        type: 'number',
        defaultValue: 0,
        description: 'Pixels to scroll vertically'
      },
      {
        key: 'wait_after',
        label: 'Wait After Scroll (ms)',
        type: 'number',
        defaultValue: 500,
        description: 'Time to wait after scrolling'
      }
    ]
  },

  {
    type: 'type',
    label: 'Type Text',
    description: 'Type text into an input field (simulates keystrokes)',
    category: 'Interaction',
    defaultParams: {
      selector: '',
      text: '',
      clear: false
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
        key: 'clear',
        label: 'Clear Existing Text',
        type: 'boolean',
        defaultValue: false,
        description: 'Clear field before typing'
      }
    ]
  },

  {
    type: 'input',
    label: 'Fill Input',
    description: 'Fill input directly (faster than type)',
    category: 'Interaction',
    defaultParams: {
      selector: '',
      value: ''
    },
    paramSchema: [
      {
        key: 'selector',
        label: 'CSS Selector',
        type: 'text',
        required: true,
        placeholder: 'input[name="email"]'
      },
      {
        key: 'value',
        label: 'Value',
        type: 'text',
        required: true,
        placeholder: 'user@email.com'
      }
    ]
  },

  {
    type: 'hover',
    label: 'Hover Element',
    description: 'Hover over an element (for tooltips, dropdowns)',
    category: 'Interaction',
    defaultParams: {
      selector: '',
      wait_after: 300
    },
    paramSchema: [
      {
        key: 'selector',
        label: 'CSS Selector',
        type: 'text',
        required: true,
        placeholder: '.menu-item, .info-icon'
      },
      {
        key: 'wait_after',
        label: 'Wait After Hover (ms)',
        type: 'number',
        defaultValue: 300,
        description: 'Time to wait after hovering'
      }
    ]
  },

  {
    type: 'wait',
    label: 'Wait (Duration)',
    description: 'Wait for a specified duration',
    category: 'Interaction',
    defaultParams: {
      duration: 1000
    },
    paramSchema: [
      {
        key: 'duration',
        label: 'Duration (ms)',
        type: 'number',
        required: true,
        defaultValue: 1000,
        description: 'Time to wait in milliseconds'
      }
    ]
  },

  {
    type: 'wait_for',
    label: 'Wait For Condition',
    description: 'Wait for selector, text, or network idle',
    category: 'Interaction',
    defaultParams: {
      condition: 'selector',
      selector: '',
      state: 'visible',
      timeout: 30000
    },
    paramSchema: [
      {
        key: 'condition',
        label: 'Condition Type',
        type: 'select',
        required: true,
        defaultValue: 'selector',
        options: [
          { label: 'Selector', value: 'selector' },
          { label: 'Text on Page', value: 'text' },
          { label: 'Network Idle', value: 'network_idle' },
          { label: 'Page Load', value: 'load' },
          { label: 'DOM Content Loaded', value: 'domcontentloaded' },
          { label: 'URL Pattern', value: 'url' }
        ]
      },
      {
        key: 'selector',
        label: 'CSS Selector',
        type: 'text',
        placeholder: '.product-price, #content',
        description: 'Required for "selector" condition'
      },
      {
        key: 'state',
        label: 'Element State',
        type: 'select',
        defaultValue: 'visible',
        options: [
          { label: 'Visible', value: 'visible' },
          { label: 'Hidden', value: 'hidden' },
          { label: 'Attached (in DOM)', value: 'attached' },
          { label: 'Detached', value: 'detached' }
        ],
        description: 'Only applies to "selector" condition'
      },
      {
        key: 'text',
        label: 'Text to Find',
        type: 'text',
        placeholder: 'Loading complete',
        description: 'Required for "text" condition'
      },
      {
        key: 'url',
        label: 'URL Pattern',
        type: 'text',
        placeholder: '**/success**',
        description: 'Required for "url" condition'
      },
      {
        key: 'timeout',
        label: 'Timeout (ms)',
        type: 'number',
        defaultValue: 30000,
        description: 'Maximum time to wait'
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
      full_page: false,
      save_as_item: true,
      save_to_disk: ''
    },
    paramSchema: [
      {
        key: 'selector',
        label: 'Element Selector (Optional)',
        type: 'text',
        placeholder: '.product-image, #main-content',
        description: 'Capture specific element (leave empty for page)'
      },
      {
        key: 'full_page',
        label: 'Full Page Screenshot',
        type: 'boolean',
        defaultValue: false,
        description: 'Capture entire scrollable page'
      },
      {
        key: 'save_as_item',
        label: 'Save as Extracted Item',
        type: 'boolean',
        defaultValue: true,
        description: 'Store screenshot data in extracted items (base64)'
      },
      {
        key: 'save_to_disk',
        label: 'Save to Disk Path',
        type: 'text',
        placeholder: './screenshots',
        description: 'Directory to save screenshot as PNG file'
      }
    ]
  },

  {
    type: 'infinite_scroll',
    label: 'Infinite Scroll',
    description: 'Scroll to load all lazy-loaded content',
    category: 'Interaction',
    defaultParams: {
      max_scrolls: 10,
      wait_between: 1000,
      end_selector: ''
    },
    paramSchema: [
      {
        key: 'max_scrolls',
        label: 'Max Scrolls',
        type: 'number',
        defaultValue: 10,
        description: 'Maximum number of scroll iterations'
      },
      {
        key: 'wait_between',
        label: 'Wait Between Scrolls (ms)',
        type: 'number',
        defaultValue: 1000,
        description: 'Time to wait between scrolls for content to load'
      },
      {
        key: 'end_selector',
        label: 'End Marker Selector',
        type: 'text',
        placeholder: '.no-more-items, #end-of-list',
        description: 'Stop scrolling when this element appears'
      }
    ]
  },

  // Control Flow Nodes
  {
    type: 'conditional',
    label: 'Conditional',
    description: 'Execute different nodes based on condition',
    category: 'Control Flow',
    defaultParams: {
      condition: 'exists',
      selector: '',
      value: ''
    },
    paramSchema: [
      {
        key: 'condition',
        label: 'Condition Type',
        type: 'select',
        required: true,
        defaultValue: 'exists',
        options: [
          { label: 'Element Exists', value: 'exists' },
          { label: 'Element Not Exists', value: 'not_exists' },
          { label: 'Element Visible', value: 'visible' },
          { label: 'Text Contains', value: 'contains' },
          { label: 'Text Equals', value: 'equals' },
          { label: 'Text Matches (Regex)', value: 'matches' },
          { label: 'Count Greater Than', value: 'count_gt' },
          { label: 'Count Less Than', value: 'count_lt' }
        ]
      },
      {
        key: 'selector',
        label: 'CSS Selector',
        type: 'text',
        required: true,
        placeholder: '.out-of-stock, .error-message'
      },
      {
        key: 'value',
        label: 'Value (for text/count conditions)',
        type: 'text',
        placeholder: 'Out of Stock, 5',
        description: 'Text to compare or count threshold'
      },
      {
        key: 'then',
        label: 'Then (if true)',
        type: 'textarea',
        placeholder: '[\n  {"type": "click", "params": {"selector": ".retry"}}\n]',
        description: 'JSON array of nodes to execute if condition is true'
      },
      {
        key: 'else',
        label: 'Else (if false)',
        type: 'textarea',
        placeholder: '[\n  {"type": "wait", "params": {"duration": 1000}}\n]',
        description: 'JSON array of nodes to execute if condition is false'
      }
    ]
  },

  {
    type: 'loop',
    label: 'Loop',
    description: 'Iterate over elements and execute child nodes',
    category: 'Control Flow',
    defaultParams: {
      selector: '',
      max_iterations: 100
    },
    paramSchema: [
      {
        key: 'selector',
        label: 'Element Selector',
        type: 'text',
        required: true,
        placeholder: '.product-card, tr.item',
        description: 'Selector for elements to iterate over'
      },
      {
        key: 'max_iterations',
        label: 'Max Iterations',
        type: 'number',
        defaultValue: 100,
        description: 'Maximum number of iterations'
      },
      {
        key: 'nodes',
        label: 'Child Nodes',
        type: 'textarea',
        placeholder: '[\n  {"type": "click", "params": {"selector": ".expand"}}\n]',
        description: 'JSON array of nodes to execute for each element'
      }
    ]
  },

  {
    type: 'script',
    label: 'Custom Script',
    description: 'Execute custom JavaScript on the page',
    category: 'Control Flow',
    defaultParams: {
      code: '',
      store_as: ''
    },
    paramSchema: [
      {
        key: 'code',
        label: 'JavaScript Code',
        type: 'textarea',
        required: true,
        placeholder: 'return document.title;',
        description: 'JavaScript code to execute in the browser context'
      },
      {
        key: 'store_as',
        label: 'Store Result As',
        type: 'text',
        placeholder: 'page_title',
        description: 'Variable name to store the result'
      }
    ]
  }
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
