# Visual Element Selector - ParseHub-like Feature

## Overview

The Visual Element Selector is a powerful feature that allows users to select elements on a webpage visually, similar to ParseHub. Instead of manually writing complex CSS selectors, users can simply click on elements in a browser window and the system automatically generates optimal selectors.

## Architecture

### How It Works

1. **Backend Browser Launch**: When a user clicks "Visual Selector", the backend launches a headed (visible) Playwright browser
2. **Vue.js Overlay Injection**: A complete Vue.js-based UI is injected directly into the target webpage
3. **Interactive Selection**: Users hover over and click elements to select them
4. **Automatic Selector Generation**: The system generates optimal CSS selectors using multiple strategies (ID, class, path-based)
5. **Real-time Sync**: Selected fields are automatically synced back to the workflow builder
6. **Session Management**: The browser session is managed by the backend and can be closed when done

### Key Components

#### Backend Components

1. **`internal/browser/element_selector.go`**
   - Manages selector sessions
   - Handles browser lifecycle (create, monitor, close)
   - Polls for selected fields from the injected UI
   - Automatic cleanup of inactive sessions

2. **`internal/browser/selector_overlay_template.go`**
   - Contains the complete Vue.js overlay application
   - Provides interactive element highlighting
   - Smart selector generation algorithms
   - User-friendly UI for managing selected fields

3. **`internal/browser/pool.go`** (Updated)
   - Added support for headed browser mode
   - New `Acquire(ctx, headed bool)` method
   - Automatic cleanup for headed sessions

4. **`api/handlers/selector.go`**
   - REST API endpoints for selector sessions
   - Session creation, status, and field retrieval

#### Frontend Components

1. **`frontend/src/api/selector.ts`**
   - TypeScript API client for selector endpoints
   - Polling helper for real-time field updates

2. **`frontend/src/components/workflow-builder/NodeConfigPanel.vue`** (Updated)
   - Visual Selector button in Extract Data node configuration
   - Real-time status indicators
   - Automatic field import from browser selections

## API Endpoints

### Create Selector Session
```http
POST /api/v1/selector/sessions
Content-Type: application/json

{
  "url": "https://example.com"
}
```

**Response:**
```json
{
  "session_id": "uuid",
  "url": "https://example.com",
  "message": "Browser window opened. Select elements in the browser window, then close it when done."
}
```

### Get Session Status
```http
GET /api/v1/selector/sessions/:session_id
```

**Response:**
```json
{
  "session_id": "uuid",
  "url": "https://example.com",
  "created_at": "2024-01-01T00:00:00Z",
  "last_activity": "2024-01-01T00:05:00Z",
  "active": true,
  "fields_count": 3
}
```

### Get Selected Fields
```http
GET /api/v1/selector/sessions/:session_id/fields
```

**Response:**
```json
{
  "session_id": "uuid",
  "fields": [
    {
      "name": "title",
      "selector": ".product-title",
      "type": "text",
      "multiple": false,
      "preview": "Sample Product Title"
    },
    {
      "name": "price",
      "selector": ".price",
      "type": "text",
      "multiple": false,
      "preview": "$19.99"
    }
  ],
  "count": 2
}
```

### Close Session
```http
DELETE /api/v1/selector/sessions/:session_id
```

## User Guide

### Using the Visual Selector

1. **Open Workflow Builder**: Create or edit a workflow
2. **Add Extract Data Node**: Add an "Extract Data" node to your workflow
3. **Click Visual Selector**: In the node configuration panel, click the "Visual Selector" button
4. **Enter URL**: Provide the URL of the page you want to scrape
5. **Wait for Browser**: A browser window will open with the target page
6. **Select Elements**:
   - Hover over elements to highlight them
   - Enter a field name (e.g., "title", "price")
   - Choose the extraction type (text, attribute, or HTML)
   - Click "Add Field" to save the selector
   - Repeat for all fields you want to extract
7. **Close Browser**: Click "Done" in the overlay when finished
8. **Review Fields**: Selected fields automatically appear in your workflow configuration

### Selector Generation Strategies

The Visual Selector uses multiple strategies to generate optimal CSS selectors:

1. **ID-based**: Prefers unique element IDs (e.g., `#product-123`)
2. **Class-based**: Uses unique class combinations (e.g., `.product.featured`)
3. **Path-based**: Builds a DOM path when IDs and classes aren't unique (e.g., `body > div.container > h1:nth-child(2)`)

### Selection Modes

- **Single Element Mode**: Select one element that matches the selector
- **Multiple Elements Mode**: Select all elements matching the selector (useful for lists, tables, etc.)

### Extraction Types

1. **Text Content**: Extracts the text content of the element
2. **Attribute**: Extracts a specific attribute value (e.g., `href`, `src`, `data-id`)
3. **HTML**: Extracts the inner HTML of the element

## Features

### 🎯 Visual Selection
- Hover to highlight elements
- Real-time preview of selected content
- Visual feedback for selected fields

### 🔍 Smart Selector Generation
- Automatically generates optimal CSS selectors
- Prioritizes unique identifiers
- Fallback to DOM path when needed

### 🔄 Real-time Sync
- Selected fields automatically appear in workflow builder
- No need to manually copy/paste selectors
- Live updates as you select elements

### 🎨 User-Friendly Interface
- Clean, intuitive overlay UI
- Field management (add, remove, duplicate)
- Search and filter fields
- Collapse/expand field details

### ⚡ Session Management
- Automatic session cleanup after 30 minutes of inactivity
- Manual session closing
- Multiple concurrent sessions supported

## Technical Details

### Browser Configuration

The headed browser is launched with:
- Chromium engine (Playwright)
- 1920x1080 viewport
- JavaScript enabled
- Ignore HTTPS errors enabled

### Vue.js Overlay

The overlay is a self-contained Vue 3 application that:
- Loads Vue.js from CDN (no build step required)
- Injects into any webpage without conflicts
- Uses scoped CSS to avoid style collisions
- Communicates with backend via `window` object

### Selector Algorithm

```javascript
function generateSelector(element) {
  // 1. Try ID first
  if (element.id) return '#' + element.id;
  
  // 2. Try unique class combination
  if (hasUniqueClasses(element)) return getClassSelector(element);
  
  // 3. Build DOM path with nth-child
  return buildDOMPath(element);
}
```

## Configuration

### Backend Configuration

No additional configuration needed. The system uses the existing browser pool configuration from `config.yaml`:

```yaml
browser:
  headless: true  # This is for normal workflow execution
  pool_size: 5
  timeout: 60000
```

Headed browsers for the visual selector are created on-demand and don't use the pool.

### Security Considerations

- Sessions automatically expire after 30 minutes
- Only one browser per session
- Sessions are user-isolated
- No persistent storage of browser data

## Troubleshooting

### Browser doesn't open
- Check that Playwright is properly installed: `playwright install chromium`
- Ensure the server has access to display (for headed mode)
- Check server logs for errors

### Elements not selectable
- Ensure JavaScript is enabled on the target page
- Check if the page uses shadow DOM (not currently supported)
- Try refreshing the session

### Selectors not working in workflow
- Test the selector in browser DevTools
- Some pages may have dynamic IDs that change
- Try using data attributes or stable classes

## Future Enhancements

- [ ] Support for XPath selectors
- [ ] Smart data type detection
- [ ] Selector validation and testing
- [ ] Screenshot capture of selected elements
- [ ] Support for shadow DOM
- [ ] Pagination detection and handling
- [ ] Template-based extraction for similar elements
- [ ] Export/import selector configurations

## Comparison with ParseHub

| Feature | Visual Selector | ParseHub |
|---------|----------------|----------|
| Visual element selection | ✅ | ✅ |
| Automatic selector generation | ✅ | ✅ |
| Multiple element selection | ✅ | ✅ |
| Real-time preview | ✅ | ✅ |
| Cloud-based | ❌ (Self-hosted) | ✅ |
| Pagination handling | ⏳ (Planned) | ✅ |
| Template detection | ⏳ (Planned) | ✅ |
| JavaScript interaction | ✅ (Full workflow support) | ✅ |

## Examples

### Example 1: Scraping Product Information

1. Open visual selector for `https://example-shop.com/products`
2. Select product title: `.product-title`
3. Select price: `.price-value`
4. Select image: `img.product-image` (attribute: `src`)
5. Switch to Multiple Elements mode
6. The selectors now extract all products on the page

### Example 2: Extracting Article Content

1. Open visual selector for `https://news-site.com/article/123`
2. Select article title: `h1.article-title`
3. Select author: `.author-name`
4. Select publish date: `time.publish-date` (attribute: `datetime`)
5. Select article body: `.article-content` (HTML type)

## License

Part of the Crawlify project. See main project LICENSE file.
