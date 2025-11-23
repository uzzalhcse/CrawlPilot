# Plugin-Based Node System

## Overview

Crawlify now uses a plugin-based architecture for node execution, making it easy to add custom nodes without modifying core code.

## How It Works

### 1. Node Registry

All node types are registered in a central registry during executor initialization:

```go
registry := NewNodeRegistry()
registry.RegisterDefaultNodes() // Registers all built-in nodes
```

### 2. Node Execution Flow

When a workflow executes a node:
1. **Registry Check**: Executor checks if node type is registered
2. **Validation**: Node parameters are validated
3. **Execution**: Plugin executor runs the node logic
4. **Fallback**: If not registered, falls back to legacy switch-based execution

### 3. Backward Compatibility

Existing workflows continue to work without changes. The system seamlessly supports both:
- New plugin-based nodes
- Legacy switch-based execution

## Available Nodes

### Interaction Nodes
- **click** - Click on elements
- **scroll** - Scroll the page
- **type** - Type text into inputs
- **hover** - Hover over elements
- **wait** - Wait for a duration
- **wait_for** - Wait for a selector

### Discovery Nodes
- **extract_links** - Extract links from page
- **navigate** - Navigate to a URL
- **paginate** - Handle pagination

### Extraction Nodes
- **extract** - Extract data with field-based config
- **extract_json** - Extract JSON from script tags

## Creating Custom Nodes

### 1. Implement NodeExecutor Interface

```go
package custom

import (
    "context"
    "github.com/uzzalhcse/crawlify/internal/workflow/nodes"
    "github.com/uzzalhcse/crawlify/pkg/models"
)

type MyCustomExecutor struct {
    nodes.BaseNodeExecutor
}

func NewMyCustomExecutor() *MyCustomExecutor {
    return &MyCustomExecutor{}
}

func (e *MyCustomExecutor) Type() models.NodeType {
    return models.NodeType("my_custom_node")
}

func (e *MyCustomExecutor) Validate(params map[string]interface{}) error {
    // Validate required parameters
    if nodes.GetStringParam(params, "required_param") == "" {
        return fmt.Errorf("required_param is required")
    }
    return nil
}

func (e *MyCustomExecutor) Execute(ctx context.Context, input *nodes.ExecutionInput) (*nodes.ExecutionOutput, error) {
    // Your custom logic here
    param := nodes.GetStringParam(input.Params, "required_param")
    
    // Access browser context
    page := input.BrowserContext.Page
    
    // Do something with the page...
    
    return &nodes.ExecutionOutput{
        Result: map[string]interface{}{
            "success": true,
        },
    }, nil
}
```

### 2. Register Your Node

```go
// During executor initialization
registry.Register(custom.NewMyCustomExecutor())
```

### 3. Use in Workflow

```json
{
  "id": "my_custom_step",
  "type": "my_custom_node",
  "params": {
    "required_param": "value"
  }
}
```

## Key-Value Extraction Enhancements

The extract node now supports multiple output formats for key-value pairs:

### Array of Objects (Default)
```json
{
  "specifications": {
    "extractions": [{
      "key_selector": ".label",
      "value_selector": ".value"
    }]
  }
}
```
**Output**: `[{"key": "Color", "value": "Black"}, ...]`

### Flat Object
```json
{
  "specifications": {
    "extractions": [{...}],
    "output_format": "object"
  }
}
```
**Output**: `{"Color": "Black", "Size": "Large"}`

### Array of Arrays
```json
{
  "specifications": {
    "extractions": [{...}],
    "output_format": "array_of_arrays"
  }
}
```
**Output**: `[["Color", "Black"], ["Size", "Large"]]`

## Benefits

### For Users
- ✅ No code changes needed for existing workflows
- ✅ Enhanced validation catches configuration errors early
- ✅ Better error messages

### For Developers
- ✅ Add new node types without modifying executor
- ✅ Each node is self-contained and testable
- ✅ Clear extension points for customization
- ✅ Community can create and share custom nodes

## Example Workflow

See [`plugin_system_example.json`](./plugin_system_example.json) for a complete working example demonstrating:
- URL discovery with `extract_links`
- Page interaction with `wait_for`
- Data extraction with nested fields
- Key-value extraction with object output format

## Migration Notes

No migration required! All existing workflows work as-is. New features are opt-in.
