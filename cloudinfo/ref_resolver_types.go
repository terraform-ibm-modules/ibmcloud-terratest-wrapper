// Package cloudinfo contains functions and methods for searching and detailing various resources located in the IBM Cloud
package cloudinfo

import "fmt"

// Reference represents a reference to resolve
type Reference struct {
	Reference string `json:"reference"`
	Context   string `json:"context,omitempty"`
}

// ResolveRequest represents the request to the ref-resolver API
type ResolveRequest struct {
	References []Reference `json:"references"`
}

// ValueObjectResolvedItem represents the resolved value of a reference when it is application/json
type ValueObjectResolvedItem struct {
	// Fields can be added here based on the actual response format
}

// BatchReferenceResolvedItem represents a single reference resolution result
type BatchReferenceResolvedItem struct {
	Message     string                   `json:"message,omitempty"`
	Value       interface{}              `json:"value,omitempty"`
	ValueObject *ValueObjectResolvedItem `json:"value_object,omitempty"`
	ContentType string                   `json:"content_type"`
	TypeID      string                   `json:"type_id,omitempty"`
	Reference   string                   `json:"reference"`
	Context     string                   `json:"context,omitempty"`
	CRN         string                   `json:"crn,omitempty"`
	State       string                   `json:"state"`
	StateCode   string                   `json:"state_code,omitempty"`
	Code        int                      `json:"code"`
	RequestID   string                   `json:"request_id,omitempty"`
}

// GetValueAsString safely converts the Value field to a string representation
// Returns the string value for string types, converts bool/numbers to strings,
// and returns empty string for null/nil values
func (item *BatchReferenceResolvedItem) GetValueAsString() string {
	if item.Value == nil {
		return ""
	}

	switch v := item.Value.(type) {
	case string:
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	case float64:
		return fmt.Sprintf("%.0f", v)
	case int:
		return fmt.Sprintf("%d", v)
	default:
		// For any other type, convert to string representation
		return fmt.Sprintf("%v", v)
	}
}

// ResolveResponse represents the response from the ref-resolver API
type ResolveResponse struct {
	CorrelationID string                       `json:"correlation_id,omitempty"`
	RequestID     string                       `json:"request_id,omitempty"`
	References    []BatchReferenceResolvedItem `json:"references"`
}

// ProjectInfo stores project metadata for context resolution
type ProjectInfo struct {
	ID      string
	Name    string
	Region  string
	Configs map[string]string // Map of config IDs to config names
}
