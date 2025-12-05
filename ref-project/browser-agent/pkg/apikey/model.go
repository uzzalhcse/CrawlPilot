package apikey

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// APIKey represents an API key document in MongoDB
type APIKey struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Provider         string             `bson:"provider" json:"provider"`                                         // openai, gemini, claude, qwen
	Key              string             `bson:"key" json:"key"`                                                   // the actual API key
	IsActive         bool               `bson:"is_active" json:"is_active"`                                       // whether this key is active
	UsageCount       int64              `bson:"usage_count" json:"usage_count"`                                   // total number of times used
	LastUsedAt       *time.Time         `bson:"last_used_at" json:"last_used_at"`                                 // last time this key was used
	FailureCount     int                `bson:"failure_count" json:"failure_count"`                               // consecutive failures
	RateLimitedUntil *time.Time         `bson:"rate_limited_until,omitempty" json:"rate_limited_until,omitempty"` // when rate limit expires
	CreatedAt        time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt        time.Time          `bson:"updated_at" json:"updated_at"`
	Metadata         map[string]string  `bson:"metadata,omitempty" json:"metadata,omitempty"` // additional info
}

// IsAvailable checks if the key is available for use
func (k *APIKey) IsAvailable() bool {
	if !k.IsActive {
		return false
	}

	// Check if rate limited
	if k.RateLimitedUntil != nil && time.Now().Before(*k.RateLimitedUntil) {
		return false
	}

	return true
}

// APIKeyUsage represents usage statistics for an API key
type APIKeyUsage struct {
	KeyID        primitive.ObjectID `bson:"key_id" json:"key_id"`
	Provider     string             `bson:"provider" json:"provider"`
	Timestamp    time.Time          `bson:"timestamp" json:"timestamp"`
	RequestType  string             `bson:"request_type" json:"request_type"` // generation, chat, etc.
	Success      bool               `bson:"success" json:"success"`
	ErrorMessage string             `bson:"error_message,omitempty" json:"error_message,omitempty"`
	ResponseTime int64              `bson:"response_time_ms" json:"response_time_ms"` // in milliseconds
}
