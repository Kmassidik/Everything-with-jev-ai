// Package jev is a thin client for TypeSafe's Jev "System One" API.
// The whole API is one POST, so there is no SDK dependency.
package jev

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const Endpoint = "https://api.typesafe.ai/v1/systemone"

// Client calls the Jev API with the platform key (never a per-user key).
type Client struct {
	APIKey string
	Model  string
	HTTP   *http.Client
}

func New(apiKey, model string) *Client {
	if model == "" {
		model = "jev-latest"
	}
	return &Client{APIKey: apiKey, Model: model, HTTP: &http.Client{Timeout: 30 * time.Second}}
}

// Live reports whether a real key is configured. Implements jev.Judge.
func (c *Client) Live() bool { return c.APIKey != "" }

// Question is one typed question. Type is "choice" | "score" | "noul".
type Question struct {
	Type         string `json:"type"`
	Instructions string `json:"instructions"`
	Criteria     any    `json:"criteria,omitempty"` // map for choice, []string for score
}

type request struct {
	State     any                 `json:"state"`
	Model     string              `json:"model"`
	Questions map[string]Question `json:"questions"`
}

// Response mirrors the Jev reply. Answers are left raw so each caller decodes the
// shape it asked for; Usage is what the ledger meters.
type Response struct {
	Model   string                     `json:"model"`
	Answers map[string]json.RawMessage `json:"answers"`
	Usage   struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// SystemOne runs one request: a state plus a map of typed questions, answered in parallel.
func (c *Client) SystemOne(ctx context.Context, state any, questions map[string]Question) (*Response, error) {
	body, err := json.Marshal(request{State: state, Model: c.Model, Questions: questions})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, Endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("authorization", "Bearer "+c.APIKey)
	req.Header.Set("content-type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jev: unexpected status %d", resp.StatusCode)
	}
	var out Response
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}
