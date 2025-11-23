# Advanced Extraction Patterns

This guide shows how to handle complex extraction scenarios using Crawlify's built-in nodes.

## Available Interaction Nodes

### Basic Interactions
- **`click`** - Click on elements
- **`hover`** - Hover over elements (shows tooltips, dropdowns)
- **`scroll`** - Scroll the page (loads lazy content)
- **`type`** - Type text into inputs
- **`wait`** - Simple delay or wait for selector

### Control Flow
- **`sequence`** - Execute multiple nodes in order
- **`conditional`** - If/else logic based on conditions

## Common Patterns

### 1. Extract from Dialog/Modal

**Use Case:** Click button → Dialog opens → Extract data → Close dialog

```json
{
  "type": "sequence",
  "params": {
    "steps": [
      {
        "type": "click",
        "params": { "selector": "button.view-details" }
      },
      {
        "type": "wait",
        "params": {
          "selector": "div.modal",
          "state": "visible",
          "timeout": 5000
        }
      },
      {
        "type": "extract",
        "params": {
          "fields": {
            "modal_data": {
              "selector": "div.modal .content",
              "type": "text"
            }
          }
        }
      },
      {
        "type": "click",
        "params": { "selector": "button.close" }
      }
    ]
  }
}
```

### 2. Scroll to Load Lazy Content

**Use Case:** Page loads more content when you scroll down

```json
{
  "type": "sequence",
  "params": {
    "steps": [
      {
        "type": "scroll",
        "params": { "x": 0, "y": 1000 }
      },
      {
        "type": "wait",
        "params": { "duration": 2000 }
      },
      {
        "type": "wait",
        "params": {
          "selector": ".lazy-loaded-content",
          "state": "visible",
          "timeout": 10000
        }
      },
      {
        "type": "extract",
        "params": {
          "fields": {
            "lazy_data": {
              "selector": ".lazy-loaded-content",
              "type": "text"
            }
          }
        }
      }
    ]
  }
}
```

### 3. Extract from Multiple Tabs

**Use Case:** Product has multiple tabs (Description, Specs, Reviews)

```json
{
  "type": "sequence",
  "params": {
    "steps": [
      {
        "type": "extract",
        "params": {
          "fields": {
            "description": {
              "selector": "#tab-1-content",
              "type": "text"
            }
          }
        }
      },
      {
        "type": "click",
        "params": { "selector": "a[data-tab='specs']" }
      },
      {
        "type": "wait",
        "params": {
          "selector": "#tab-2-content",
          "state": "visible",
          "timeout": 3000
        }
      },
      {
        "type": "extract",
        "params": {
          "fields": {
            "specifications": {
              "selector": "#tab-2-content",
              "type": "text"
            }
          }
        }
      }
    ]
  }
}
```

### 4. Hover to Show Tooltip

**Use Case:** Data is shown in a tooltip on hover

```json
{
  "type": "sequence",
  "params": {
    "steps": [
      {
        "type": "hover",
        "params": { "selector": ".info-icon" }
      },
      {
        "type": "wait",
        "params": {
          "selector": ".tooltip",
          "state": "visible",
          "timeout": 2000
        }
      },
      {
        "type": "extract",
        "params": {
          "fields": {
            "tooltip_text": {
              "selector": ".tooltip",
              "type": "text"
            }
          }
        }
      }
    ]
  }
}
```

### 5. Conditional Extraction

**Use Case:** Extract different data based on whether an element exists

```json
{
  "type": "conditional",
  "params": {
    "condition": {
      "type": "element_exists",
      "selector": ".sale-badge"
    },
    "if_true": [
      {
        "type": "extract",
        "params": {
          "fields": {
            "sale_price": {
              "selector": ".sale-price",
              "type": "text"
            },
            "original_price": {
              "selector": ".original-price",
              "type": "text"
            }
          }
        }
      }
    ],
    "if_false": [
      {
        "type": "extract",
        "params": {
          "fields": {
            "regular_price": {
              "selector": ".price",
              "type": "text"
            }
          }
        }
      }
    ]
  }
}
```

### 6. Type and Submit Form

**Use Case:** Fill in a form to reveal content

```json
{
  "type": "sequence",
  "params": {
    "steps": [
      {
        "type": "type",
        "params": {
          "selector": "input[name='zipcode']",
          "text": "12345",
          "delay": 100
        }
      },
      {
        "type": "click",
        "params": { "selector": "button[type='submit']" }
      },
      {
        "type": "wait",
        "params": {
          "selector": ".shipping-info",
          "state": "visible",
          "timeout": 5000
        }
      },
      {
        "type": "extract",
        "params": {
          "fields": {
            "shipping_cost": {
              "selector": ".shipping-info .cost",
              "type": "text"
            }
          }
        }
      }
    ]
  }
}
```

## Step Parameters

### Optional Steps
Mark any step as optional - if it fails, the sequence continues:

```json
{
  "type": "click",
  "params": { "selector": "button.optional" },
  "optional": true
}
```

### Wait Node - Two Modes

**1. Simple Wait (duration):**
```json
{
  "type": "wait",
  "params": { "duration": 3000 }
}
```

**2. Wait for Selector:**
```json
{
  "type": "wait",
  "params": {
    "selector": ".element",
    "state": "visible",  // or "hidden", "attached"
    "timeout": 10000
  }
}
```

## Best Practices

1. **Always wait after interactions:**
   - After `click` → `wait` for new content
   - After `scroll` → `wait` for lazy load
   - After `type` → `wait` for validation

2. **Use specific selectors:**
   - Use IDs and unique classes
   - Avoid generic selectors like `div` or `.btn`

3. **Set reasonable timeouts:**
   - Fast sites: 3-5 seconds
   - Slow sites: 10-15 seconds
   - Forms/AJAX: 5-10 seconds

4. **Mark optional steps:**
   - Closing modals (might auto-close)
   - Cookie banners (might not exist)
   - Optional fields

5. **Sequence ordering:**
   - Interactive nodes first (click, scroll, hover)
   - Wait for content to appear
   - Extract last

## Example: Columbia Products with Reviews

```json
{
  "id": "extract_columbia_with_reviews",
  "type": "sequence",
  "params": {
    "steps": [
      {
        "type": "extract",
        "params": {
          "fields": {
            "product_name": {
              "selector": "h1.block-goods-name--text span",
              "type": "text"
            },
            "price": {
              "selector": ".block-goods-price--price",
              "type": "text"
            }
          }
        }
      },
      {
        "type": "scroll",
        "params": { "x": 0, "y": 2000 }
      },
      {
        "type": "wait",
        "params": {
          "selector": ".reviews-section",
          "state": "visible",
          "timeout": 5000
        }
      },
      {
        "type": "extract",
        "params": {
          "fields": {
            "review_count": {
              "selector": ".reviews-count",
              "type": "text",
              "default": "0"
            },
            "average_rating": {
              "selector": ".average-rating",
              "type": "text",
              "default": "N/A"
            }
          }
        }
      }
    ]
  }
}
```

## Troubleshooting

### Problem: Element not found
**Solution:** Add a wait node before extraction:
```json
{
  "type": "wait",
  "params": {
    "selector": "your-selector",
    "state": "visible",
    "timeout": 10000
  }
}
```

### Problem: Modal/Dialog doesn't open
**Solution:** 
1. Check selector is correct
2. Add wait before click:
```json
{
  "type": "wait",
  "params": {
    "selector": "button.trigger",
    "state": "visible",
    "timeout": 5000
  }
},
{
  "type": "click",
  "params": { "selector": "button.trigger" }
}
```

### Problem: Data loads too slowly
**Solution:** Increase timeout:
```json
{
  "type": "wait",
  "params": {
    "selector": ".slow-content",
    "state": "visible",
    "timeout": 30000  // 30 seconds
  }
}
```

### Problem: Need to extract from multiple similar elements
**Solution:** Use list extraction (if supported):
```json
{
  "fields": {
    "items": {
      "selector": ".item",
      "type": "list",
      "item": {
        "name": { "selector": ".name", "type": "text" },
        "price": { "selector": ".price", "type": "text" }
      }
    }
  }
}
```
