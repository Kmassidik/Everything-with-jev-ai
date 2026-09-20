package jev

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
)

// Mock is a deterministic, offline Judge for development and tests. It makes no
// network calls; each answer is a stable hash of (state, question key), so the UI
// and tests are reproducible. The UI labels any Mock-produced output as SAMPLE.
type Mock struct{}

// Live is always false — Mock is not backed by a real key. Implements jev.Judge.
func (Mock) Live() bool { return false }

// SystemOne returns one deterministic noul per question.
func (Mock) SystemOne(_ context.Context, state any, questions map[string]Question) (*Response, error) {
	sb, err := json.Marshal(state)
	if err != nil {
		return nil, err
	}
	r := &Response{Model: "mock", Answers: make(map[string]json.RawMessage, len(questions))}
	for key := range questions {
		p := stableProb(string(sb) + "|" + key)
		r.Answers[key] = json.RawMessage(fmt.Sprintf(`{"type":"noul","noul":%.3f}`, p))
	}
	r.Usage.InputTokens = len(sb)/4 + 1
	r.Usage.OutputTokens = len(questions)
	return r, nil
}

// stableProb maps a string to a stable value in [0,1).
func stableProb(s string) float64 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return float64(h.Sum32()%1000) / 1000.0
}
