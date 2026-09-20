package jev

import (
	"context"
	"encoding/json"
)

// Judge runs one System One request. Both the live *Client and the offline Mock
// implement it, so callers (and tests) depend on the interface, not the transport.
type Judge interface {
	SystemOne(ctx context.Context, state any, questions map[string]Question) (*Response, error)
	// Live reports whether this judge is backed by a real API key.
	Live() bool
}

// Noul extracts a boolean-probability answer (0..1) for a question key.
// Returns 0 if the key is absent or the shape is unexpected.
func (r *Response) Noul(key string) float64 {
	raw, ok := r.Answers[key]
	if !ok {
		return 0
	}
	var a struct {
		Noul float64 `json:"noul"`
	}
	_ = json.Unmarshal(raw, &a)
	return a.Noul
}
