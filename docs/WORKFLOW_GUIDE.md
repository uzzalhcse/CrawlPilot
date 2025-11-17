# Workflow Configuration Guide

This guide explains how to create and configure workflows for Crawlify.

## Workflow Structure

A workflow consists of:

1. **Metadata**: Start URLs, depth limits, rate limiting
2. **URL Discovery**: Nodes for finding and queueing new URLs
3. **Data Extraction**: Nodes for extracting data from pages
4. **Storage**: Configuration for where to store extracted data

## Basic Example

```yaml
start_urls:
  - "https://example.com"

max_depth: 2
max_pages: 100
rate_limit_delay: 1000

url_discovery:
  - id: "extract_links"
    type: "extract_links"
    params:
      selector: "a[href]"

data_extraction:
  - id: "extract_title"
    type: "extract"
    params:
      selector: "h1"
      type: "text"

storage:
  type: "database"
  table_name: "pages"
```

## Configuration Options

### Top-Level Configuration

```yaml
# Required: Starting URLs for the crawler
start_urls:
  - "https://example.com"
  - "https://example.com/page2"

# Optional: Maximum crawl depth (default: 3)
max_depth: 3

# Optional: Maximum pages to crawl (default: unlimited)
max_pages: 1000

# Optional: Delay between requests in milliseconds (default: 0)
rate_limit_delay: 1000

# Optional: Custom HTTP headers
headers:
  User-Agent: "Crawlify/1.0"
  Accept-Language: "en-US,en;q=0.9"

# Optional: Cookies to set
cookies:
  - name: "session_id"
    value: "abc123"
    domain: "example.com"
    path: "/"
    secure: true
    http_only: true

# Optional: Authentication configuration
authentication:
  type: "basic"  # or "bearer", "oauth2", "form"
  username: "user"
  password: "pass"

# Optional: Proxy configuration
proxy_config:
  enabled: true
  server: "http://proxy.example.com:8080"
  username: "proxy_user"
  password: "proxy_pass"
```

### Storage Configuration

```yaml
storage:
  # Store in database
  type: "database"
  table_name: "extracted_data"

  # OR store in file
  type: "file"
  file_path: "/data/output.json"

  # OR send to webhook
  type: "webhook"
  webhook_url: "https://api.example.com/webhook"
  params:
    auth_token: "secret"
```

## Node Types

### URL Discovery Nodes

#### extract_links

Extract links from the current page.

```yaml
- id: "get_links"
  type: "extract_links"
  name: "Extract all links"
  params:
    selector: "a[href]"  # CSS selector for links
  output_key: "links"
```

#### filter_urls

Filter extracted URLs based on patterns.

```yaml
- id: "filter"
  type: "filter_urls"
  name: "Filter URLs"
  params:
    same_domain: true
    include_patterns:
      - ".*\\/products\\/.*"
    exclude_patterns:
      - ".*\\.pdf$"
      - ".*logout.*"
  dependencies:
    - "get_links"
```

#### navigate

Navigate to a specific URL.

```yaml
- id: "go_to_page"
  type: "navigate"
  name: "Navigate to login page"
  params:
    url: "https://example.com/login"
```

### Interaction Nodes

#### click

Click on an element.

```yaml
- id: "click_button"
  type: "click"
  name: "Click submit button"
  params:
    selector: "button[type='submit']"
```

#### type

Type text into an input field.

```yaml
- id: "fill_search"
  type: "type"
  name: "Fill search field"
  params:
    selector: "input[name='q']"
    text: "search query"
    delay: 100  # milliseconds between keystrokes
```

#### scroll

Scroll the page.

```yaml
- id: "scroll_down"
  type: "scroll"
  name: "Scroll down"
  params:
    x: 0
    y: 1000
```

#### hover

Hover over an element.

```yaml
- id: "hover_menu"
  type: "hover"
  name: "Hover over menu"
  params:
    selector: ".dropdown-trigger"
```

#### wait

Wait for a fixed duration.

```yaml
- id: "wait_2s"
  type: "wait"
  name: "Wait 2 seconds"
  params:
    duration: 2000  # milliseconds
```

#### wait_for

Wait for an element to appear/disappear.

```yaml
- id: "wait_element"
  type: "wait_for"
  name: "Wait for results"
  params:
    selector: ".search-results"
    timeout: 10000  # milliseconds
    state: "visible"  # or "hidden", "attached"
```

#### screenshot

Take a screenshot.

```yaml
- id: "screenshot"
  type: "screenshot"
  name: "Take screenshot"
  params:
    path: "/screenshots/page.png"
    full_page: true
```

### Extraction Nodes

#### extract

Extract data using selectors.

**Simple extraction**:
```yaml
- id: "get_title"
  type: "extract"
  name: "Extract title"
  params:
    selector: "h1"
    type: "text"  # or "attr", "html", "href", "src"
    transform:
      - type: "trim"
      - type: "lowercase"
  output_key: "title"
```

**Extract attribute**:
```yaml
- id: "get_image"
  type: "extract"
  name: "Extract image URL"
  params:
    selector: "img.main-image"
    type: "attr"
    attribute: "src"
  output_key: "image_url"
```

**Extract multiple items**:
```yaml
- id: "get_items"
  type: "extract"
  name: "Extract all items"
  params:
    selector: ".item"
    type: "text"
    multiple: true
  output_key: "items"
```

**Extract structured data**:
```yaml
- id: "extract_products"
  type: "extract"
  name: "Extract product list"
  params:
    selector: ".product-card"
    multiple: true
    fields:
      title:
        selector: ".product-title"
        type: "text"
        transform:
          - type: "trim"

      price:
        selector: ".price"
        type: "text"
        transform:
          - type: "regex"
            params:
              pattern: "[^0-9.]"
              replacement: ""
          - type: "parse_float"

      url:
        selector: "a"
        type: "href"

      image:
        selector: "img"
        type: "src"
  output_key: "products"
```

#### extract_json

Extract JSON from a script tag or data attribute.

```yaml
- id: "get_json_data"
  type: "extract_json"
  name: "Extract JSON-LD data"
  params:
    selector: "script[type='application/ld+json']"
  output_key: "structured_data"
```

## Data Transformations

### Available Transformations

#### trim

Remove leading and trailing whitespace.

```yaml
transform:
  - type: "trim"
```

#### lowercase / uppercase

Convert text case.

```yaml
transform:
  - type: "lowercase"
  # or
  - type: "uppercase"
```

#### regex

Replace using regular expressions.

```yaml
transform:
  - type: "regex"
    params:
      pattern: "[^0-9]"
      replacement: ""
```

#### replace

Simple string replacement.

```yaml
transform:
  - type: "replace"
    params:
      old: "$"
      new: ""
```

#### split

Split string into array.

```yaml
transform:
  - type: "split"
    params:
      delimiter: ","
```

#### join

Join array into string.

```yaml
transform:
  - type: "join"
    params:
      delimiter: ", "
```

#### parse_int / parse_float

Convert string to number.

```yaml
transform:
  - type: "parse_int"
  # or
  - type: "parse_float"
```

### Chaining Transformations

Transformations are applied in order:

```yaml
transform:
  - type: "trim"
  - type: "replace"
    params:
      old: "$"
      new: ""
  - type: "regex"
    params:
      pattern: "[^0-9.]"
      replacement: ""
  - type: "parse_float"
```

## Node Options

### Dependencies

Specify which nodes must complete before this node executes:

```yaml
- id: "extract_data"
  type: "extract"
  dependencies:
    - "wait_for_page"
    - "scroll_to_content"
  params:
    selector: ".data"
```

### Optional Nodes

Mark a node as optional (failure won't stop execution):

```yaml
- id: "optional_data"
  type: "extract"
  optional: true
  params:
    selector: ".optional-element"
```

### Retry Configuration

Configure retry behavior:

```yaml
- id: "extract_with_retry"
  type: "extract"
  retry:
    max_retries: 3
    delay: 1000  # milliseconds
  params:
    selector: ".data"
```

### Output Key

Store node results in execution context:

```yaml
- id: "get_links"
  type: "extract_links"
  output_key: "discovered_links"  # Accessible by later nodes
  params:
    selector: "a"
```

## Best Practices

### 1. Use Dependencies

Structure your workflow as a DAG:

```yaml
url_discovery:
  - id: "wait_page"
    type: "wait_for"
    params:
      selector: ".content"

  - id: "scroll"
    type: "scroll"
    dependencies: ["wait_page"]
    params:
      y: 1000

  - id: "extract"
    type: "extract_links"
    dependencies: ["scroll"]
    params:
      selector: "a"
```

### 2. Handle Dynamic Content

Wait for elements before extracting:

```yaml
- id: "wait_for_results"
  type: "wait_for"
  params:
    selector: ".search-results"
    timeout: 10000

- id: "extract_results"
  type: "extract"
  dependencies: ["wait_for_results"]
  params:
    selector: ".result-item"
```

### 3. Use Transformations

Clean data during extraction:

```yaml
- id: "extract_price"
  type: "extract"
  params:
    selector: ".price"
    type: "text"
    transform:
      - type: "trim"
      - type: "replace"
        params:
          old: "$"
          new: ""
      - type: "parse_float"
```

### 4. Rate Limiting

Be respectful of target servers:

```yaml
rate_limit_delay: 1000  # 1 second between requests
max_depth: 3
max_pages: 1000
```

### 5. Error Handling

Use optional nodes and retries:

```yaml
- id: "optional_image"
  type: "extract"
  optional: true
  retry:
    max_retries: 2
    delay: 500
  params:
    selector: ".product-image"
```

## Common Patterns

### Pagination

```yaml
url_discovery:
  - id: "extract_next"
    type: "extract_links"
    params:
      selector: ".pagination .next"
```

### Infinite Scroll

```yaml
url_discovery:
  - id: "scroll_down"
    type: "scroll"
    params:
      y: 1000

  - id: "wait_for_load"
    type: "wait"
    dependencies: ["scroll_down"]
    params:
      duration: 2000

  - id: "extract_items"
    type: "extract_links"
    dependencies: ["wait_for_load"]
    params:
      selector: ".item a"
```

### Login Forms

```yaml
url_discovery:
  - id: "fill_username"
    type: "type"
    params:
      selector: "input[name='username']"
      text: "myuser"

  - id: "fill_password"
    type: "type"
    params:
      selector: "input[name='password']"
      text: "mypass"

  - id: "submit"
    type: "click"
    dependencies: ["fill_username", "fill_password"]
    params:
      selector: "button[type='submit']"

  - id: "wait_for_dashboard"
    type: "wait_for"
    dependencies: ["submit"]
    params:
      selector: ".dashboard"
```
