package models

import (
	"encoding/json"
	"time"
)

// Workflow represents a complete crawling workflow configuration
type Workflow struct {
	ID          string         `json:"id" db:"id"`
	Name        string         `json:"name" db:"name"`
	Description string         `json:"description" db:"description"`
	Config      WorkflowConfig `json:"config" db:"config"`
	Status      WorkflowStatus `json:"status" db:"status"`
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at" db:"updated_at"`
}

// WorkflowConfig contains the workflow execution configuration
type WorkflowConfig struct {
	StartURLs      []string          `json:"start_urls" yaml:"start_urls"`
	Phases         []WorkflowPhase   `json:"phases" yaml:"phases"` // NEW: Phase-based workflow
	MaxDepth       int               `json:"max_depth" yaml:"max_depth"`
	RateLimitDelay int               `json:"rate_limit_delay" yaml:"rate_limit_delay"`
	Headers        map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	Cookies        []Cookie          `json:"cookies,omitempty" yaml:"cookies,omitempty"`
	Authentication *AuthConfig       `json:"authentication,omitempty" yaml:"authentication,omitempty"`
	ProxyConfig    *ProxyConfig      `json:"proxy_config,omitempty" yaml:"proxy_config,omitempty"`
	Storage        StorageConfig     `json:"storage" yaml:"storage"`
}

// Node represents a single workflow node (atomic task)
type Node struct {
	ID           string                 `json:"id" yaml:"id"`
	Type         NodeType               `json:"type" yaml:"type"`
	Name         string                 `json:"name" yaml:"name"`
	Params       map[string]interface{} `json:"params" yaml:"params"`
	Dependencies []string               `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
	Optional     bool                   `json:"optional,omitempty" yaml:"optional,omitempty"`
	Retry        RetryConfig            `json:"retry,omitempty" yaml:"retry,omitempty"`
}

// NodeType defines the type of workflow node
type NodeType string

const (
	// URL Discovery nodes
	NodeTypeFetch        NodeType = "fetch"
	NodeTypeExtractLinks NodeType = "extract_links"
	NodeTypeFilterURLs   NodeType = "filter_urls"
	NodeTypeNavigate     NodeType = "navigate"
	NodeTypePaginate     NodeType = "paginate"

	// Interaction nodes
	NodeTypeClick      NodeType = "click"
	NodeTypeScroll     NodeType = "scroll"
	NodeTypeType       NodeType = "type"
	NodeTypeHover      NodeType = "hover"
	NodeTypeWait       NodeType = "wait"
	NodeTypeWaitFor    NodeType = "wait_for"
	NodeTypeScreenshot NodeType = "screenshot"

	// Extraction nodes
	NodeTypeExtract     NodeType = "extract"
	NodeTypeExtractText NodeType = "extract_text"
	NodeTypeExtractAttr NodeType = "extract_attr"
	NodeTypeExtractJSON NodeType = "extract_json"

	// Transformation nodes
	NodeTypeTransform NodeType = "transform"
	NodeTypeFilter    NodeType = "filter"
	NodeTypeMap       NodeType = "map"
	NodeTypeValidate  NodeType = "validate"

	// Control flow nodes
	NodeTypeSequence    NodeType = "sequence" // NEW: Execute nodes in sequence
	NodeTypeConditional NodeType = "conditional"
	NodeTypeLoop        NodeType = "loop"
	NodeTypeParallel    NodeType = "parallel"
)

// WorkflowStatus represents the current status of a workflow
type WorkflowStatus string

const (
	WorkflowStatusDraft    WorkflowStatus = "draft"
	WorkflowStatusActive   WorkflowStatus = "active"
	WorkflowStatusPaused   WorkflowStatus = "paused"
	WorkflowStatusArchived WorkflowStatus = "archived"
)

// Cookie represents a browser cookie
type Cookie struct {
	Name     string `json:"name" yaml:"name"`
	Value    string `json:"value" yaml:"value"`
	Domain   string `json:"domain" yaml:"domain"`
	Path     string `json:"path" yaml:"path"`
	Secure   bool   `json:"secure,omitempty" yaml:"secure,omitempty"`
	HttpOnly bool   `json:"http_only,omitempty" yaml:"http_only,omitempty"`
}

// AuthConfig contains authentication configuration
type AuthConfig struct {
	Type     string            `json:"type" yaml:"type"` // basic, bearer, oauth2, form
	Username string            `json:"username,omitempty" yaml:"username,omitempty"`
	Password string            `json:"password,omitempty" yaml:"password,omitempty"`
	Token    string            `json:"token,omitempty" yaml:"token,omitempty"`
	Params   map[string]string `json:"params,omitempty" yaml:"params,omitempty"`
}

// ProxyConfig contains proxy configuration
type ProxyConfig struct {
	Enabled  bool   `json:"enabled" yaml:"enabled"`
	Server   string `json:"server" yaml:"server"`
	Username string `json:"username,omitempty" yaml:"username,omitempty"`
	Password string `json:"password,omitempty" yaml:"password,omitempty"`
}

// StorageConfig defines where and how to store extracted data
type StorageConfig struct {
	Type       string                 `json:"type" yaml:"type"` // database, file, webhook
	TableName  string                 `json:"table_name,omitempty" yaml:"table_name,omitempty"`
	FilePath   string                 `json:"file_path,omitempty" yaml:"file_path,omitempty"`
	WebhookURL string                 `json:"webhook_url,omitempty" yaml:"webhook_url,omitempty"`
	Params     map[string]interface{} `json:"params,omitempty" yaml:"params,omitempty"`
}

// RetryConfig defines retry behavior for a node
type RetryConfig struct {
	MaxRetries int `json:"max_retries" yaml:"max_retries"`
	Delay      int `json:"delay" yaml:"delay"` // milliseconds
}

// Scan implements sql.Scanner for WorkflowConfig
func (wc *WorkflowConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		// Try to marshal it if it's already a map/struct
		var err error
		bytes, err = json.Marshal(v)
		if err != nil {
			return err
		}
	}

	err := json.Unmarshal(bytes, wc)
	if err != nil {
		return err
	}
	return nil
}

// Value implements driver.Valuer for WorkflowConfig
func (wc WorkflowConfig) Value() (interface{}, error) {
	return json.Marshal(wc)
}
