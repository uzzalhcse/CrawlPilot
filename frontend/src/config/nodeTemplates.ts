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
      limit: 0
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
      max_pages: 10
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
        key: 'max_pages',
        label: 'Max Pages',
        type: 'number',
        defaultValue: 10
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
      schema: {}
    },
    paramSchema: [
      {
        key: 'schema',
        label: 'Data Schema (JSON)',
        type: 'textarea',
        required: true,
        placeholder: '{\n  "title": ".product-title",\n  "price": ".product-price"\n}',
        description: 'JSON object mapping field names to CSS selectors'
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

  // Interaction Nodes
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
