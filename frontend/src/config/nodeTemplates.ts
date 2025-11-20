import type { NodeTemplate, NodeCategory } from '@/types'

export const nodeTemplates: NodeTemplate[] = [
  // URL Discovery Nodes
  {
    type: 'fetch',
    label: 'Fetch URL',
    description: 'Fetch HTML content from a URL',
    category: 'URL Discovery',
    defaultParams: {
      url: '',
      method: 'GET'
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
        key: 'method',
        label: 'Method',
        type: 'select',
        defaultValue: 'GET',
        options: [
          { label: 'GET', value: 'GET' },
          { label: 'POST', value: 'POST' }
        ]
      }
    ]
  },
  {
    type: 'extract_links',
    label: 'Extract Links',
    description: 'Extract all links from the page',
    category: 'URL Discovery',
    defaultParams: {
      selector: 'a',
      limit: 0,
      url_type: ''
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
        key: 'url_type',
        label: 'URL Type',
        type: 'text',
        placeholder: 'category, product, listing',
        description: 'Type/category of URLs being extracted (for organization)'
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
    type: 'filter_urls',
    label: 'Filter URLs',
    description: 'Filter URLs by pattern or condition',
    category: 'URL Discovery',
    defaultParams: {
      pattern: '',
      type: 'include'
    },
    paramSchema: [
      {
        key: 'pattern',
        label: 'Pattern',
        type: 'text',
        required: true,
        placeholder: '/products/.*'
      },
      {
        key: 'type',
        label: 'Filter Type',
        type: 'select',
        defaultValue: 'include',
        options: [
          { label: 'Include', value: 'include' },
          { label: 'Exclude', value: 'exclude' }
        ]
      }
    ]
  },
  {
    type: 'paginate',
    label: 'Paginate',
    description: 'Navigate through paginated content',
    category: 'URL Discovery',
    defaultParams: {
      next_selector: '',
      max_pages: 10,
      type: 'auto',
      wait_after: 0,
      link_selector: '',
      item_selector: '',
      url_type: ''
    },
    paramSchema: [
      {
        key: 'next_selector',
        label: 'Next Button Selector',
        type: 'text',
        required: true,
        placeholder: '.next-page, a[rel="next"]'
      },
      {
        key: 'link_selector',
        label: 'Link Selector (for pagination links)',
        type: 'text',
        placeholder: '.pagination a',
        description: 'Selector for pagination number links'
      },
      {
        key: 'item_selector',
        label: 'Item Selector',
        type: 'text',
        placeholder: '.product-item a',
        description: 'Selector for items to extract URLs from on each page'
      },
      {
        key: 'url_type',
        label: 'URL Type',
        type: 'text',
        placeholder: 'product, article',
        description: 'Type of URLs being extracted from pagination'
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
        description: 'Define fields to extract from the page',
        arrayItemSchema: [
          {
            key: 'name',
            label: 'Field Name',
            type: 'text',
            required: true,
            placeholder: 'title, price, description'
          },
          {
            key: 'selector',
            label: 'CSS Selector',
            type: 'text',
            required: true,
            placeholder: '.product-title, #price'
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
            required: true,
            placeholder: 'src, href, data-id',
            description: 'Specify which HTML attribute to extract (e.g., src for images, href for links)'
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
            placeholder: 'N/A, 0',
            description: 'Value to use if extraction fails'
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
    type: 'extract_text',
    label: 'Extract Text',
    description: 'Extract text content from elements',
    category: 'Extraction',
    defaultParams: {
      selector: '',
      all: false
    },
    paramSchema: [
      {
        key: 'selector',
        label: 'CSS Selector',
        type: 'text',
        required: true,
        placeholder: '.content, h1'
      },
      {
        key: 'all',
        label: 'Extract All Matches',
        type: 'boolean',
        defaultValue: false,
        description: 'Extract all matching elements or just first'
      }
    ]
  },
  {
    type: 'extract_attr',
    label: 'Extract Attribute',
    description: 'Extract attribute values from elements',
    category: 'Extraction',
    defaultParams: {
      selector: '',
      attribute: '',
      all: false
    },
    paramSchema: [
      {
        key: 'selector',
        label: 'CSS Selector',
        type: 'text',
        required: true,
        placeholder: 'img, a'
      },
      {
        key: 'attribute',
        label: 'Attribute Name',
        type: 'text',
        required: true,
        placeholder: 'src, href, data-id'
      },
      {
        key: 'all',
        label: 'Extract All Matches',
        type: 'boolean',
        defaultValue: false
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
    category: 'Interaction',
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
    description: 'Scroll the page',
    category: 'Interaction',
    defaultParams: {
      direction: 'down',
      amount: 1000
    },
    paramSchema: [
      {
        key: 'direction',
        label: 'Direction',
        type: 'select',
        defaultValue: 'down',
        options: [
          { label: 'Down', value: 'down' },
          { label: 'Up', value: 'up' }
        ]
      },
      {
        key: 'amount',
        label: 'Amount (pixels)',
        type: 'number',
        defaultValue: 1000
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
    description: 'Wait for specified duration',
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
        defaultValue: 1000
      }
    ]
  },
  {
    type: 'wait_for',
    label: 'Wait For Element',
    description: 'Wait for element to appear',
    category: 'Interaction',
    defaultParams: {
      selector: '',
      timeout: 5000
    },
    paramSchema: [
      {
        key: 'selector',
        label: 'CSS Selector',
        type: 'text',
        required: true,
        placeholder: '.dynamic-content'
      },
      {
        key: 'timeout',
        label: 'Timeout (ms)',
        type: 'number',
        defaultValue: 5000
      }
    ]
  },
  {
    type: 'screenshot',
    label: 'Take Screenshot',
    description: 'Capture a screenshot of the page or element',
    category: 'Interaction',
    defaultParams: {
      selector: '',
      full_page: false,
      path: ''
    },
    paramSchema: [
      {
        key: 'selector',
        label: 'CSS Selector (Optional)',
        type: 'text',
        placeholder: '.target-element',
        description: 'Leave empty for full page screenshot'
      },
      {
        key: 'full_page',
        label: 'Full Page',
        type: 'boolean',
        defaultValue: false,
        description: 'Capture entire scrollable page'
      },
      {
        key: 'path',
        label: 'Save Path',
        type: 'text',
        placeholder: 'screenshots/page.png',
        description: 'Path to save screenshot'
      }
    ]
  },

  // Transformation Nodes
  {
    type: 'transform',
    label: 'Transform Data',
    description: 'Transform data with custom logic',
    category: 'Transformation',
    defaultParams: {
      operation: 'trim'
    },
    paramSchema: [
      {
        key: 'operation',
        label: 'Operation',
        type: 'select',
        defaultValue: 'trim',
        options: [
          { label: 'Trim', value: 'trim' },
          { label: 'Lowercase', value: 'lowercase' },
          { label: 'Uppercase', value: 'uppercase' }
        ]
      }
    ]
  },
  {
    type: 'filter',
    label: 'Filter Data',
    description: 'Filter data based on conditions',
    category: 'Transformation',
    defaultParams: {
      condition: ''
    },
    paramSchema: [
      {
        key: 'condition',
        label: 'Condition',
        type: 'text',
        required: true,
        placeholder: 'field > 100'
      }
    ]
  },
  {
    type: 'map',
    label: 'Map Data',
    description: 'Transform data by applying a mapping function',
    category: 'Transformation',
    defaultParams: {
      mapping: {}
    },
    paramSchema: [
      {
        key: 'mapping',
        label: 'Field Mapping',
        type: 'textarea',
        required: true,
        placeholder: '{\n  "new_field": "old_field",\n  "full_name": "first_name + last_name"\n}',
        description: 'JSON object mapping new field names to transformations or expressions'
      }
    ]
  },
  {
    type: 'validate',
    label: 'Validate Data',
    description: 'Validate data against rules',
    category: 'Transformation',
    defaultParams: {
      rules: {}
    },
    paramSchema: [
      {
        key: 'rules',
        label: 'Validation Rules (JSON)',
        type: 'textarea',
        required: true,
        placeholder: '{\n  "email": "required|email",\n  "age": "required|numeric"\n}'
      }
    ]
  },

  // Control Flow Nodes
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
  {
    type: 'loop',
    label: 'Loop',
    description: 'Repeat actions in a loop',
    category: 'Control Flow',
    defaultParams: {
      iterations: 10
    },
    paramSchema: [
      {
        key: 'iterations',
        label: 'Max Iterations',
        type: 'number',
        defaultValue: 10
      }
    ]
  },
  {
    type: 'parallel',
    label: 'Parallel',
    description: 'Execute nodes in parallel',
    category: 'Control Flow',
    defaultParams: {
      max_concurrent: 5
    },
    paramSchema: [
      {
        key: 'max_concurrent',
        label: 'Max Concurrent',
        type: 'number',
        defaultValue: 5
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
    name: 'Transformation',
    icon: 'RefreshCw',
    nodes: nodeTemplates.filter(n => n.category === 'Transformation')
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
