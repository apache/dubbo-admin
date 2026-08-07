// Package fallback recovers a usable schema.Observation when a model's output
// can't be parsed as structured JSON. It depends only on the generic
// schema.Observation, not on any single reasoning strategy, so it is a peer of
// (rather than nested under) the concrete agents — react today, and future
// strategies such as cot or plan-and-solve can share it unchanged.
package fallback

import (
	"encoding/json"
	"strings"

	"dubbo-admin-ai/runtime"
	"dubbo-admin-ai/schema"

	"github.com/firebase/genkit/go/ai"
)

// maxRawOutputLength caps how much raw model output is written to debug logs.
const maxRawOutputLength = 500

// Handler handles fallback logic for agent stages
type Handler struct{}

// NewHandler creates a new fallback handler
func NewHandler() *Handler {
	return &Handler{}
}

// ParseResponse defines the interface for responses that can be parsed
type ParseResponse interface {
	Output(dst any) error
	Text() string
}

// ParseObservation parses Observation with fallback
func (h *Handler) ParseObservation(resp ParseResponse) (*schema.Observation, error) {
	var observation schema.Observation
	observation.UsageInfo = &ai.GenerationUsage{}

	if err := resp.Output(&observation); err != nil {
		runtime.GetLogger().Warn("Observation schema parsing failed, using fallback", "error", err)
		return h.fallbackObservation(resp)
	}

	return &observation, nil
}

// fallbackObservation creates an Observation from raw text when schema parsing fails
func (h *Handler) fallbackObservation(resp ParseResponse) (*schema.Observation, error) {
	rawText := resp.Text()
	h.logRawOutput("Observation", rawText)

	// Try to extract structured data
	if parsed := h.extractJSON(rawText); parsed != nil {
		observation := &schema.Observation{
			Summary:     h.getStringField(parsed, "summary"),
			Heartbeat:   h.getBoolField(parsed, "heartbeat", true), // Default to true (continue) if uncertain
			FinalAnswer: h.getStringField(parsed, "final_answer"),
			Focus:       h.getStringField(parsed, "focus"),
			Evidence:    h.getStringField(parsed, "evidence"),
			UsageInfo:   &ai.GenerationUsage{},
		}

		// If we have a final_answer, use it; otherwise use raw text
		if observation.FinalAnswer == "" && observation.Heartbeat {
			observation.FinalAnswer = h.truncateText(rawText, 2000)
			observation.Heartbeat = false // Stop if we have some answer
		}

		return observation, nil
	}

	// Complete fallback: use raw text as final answer
	return &schema.Observation{
		Summary:     "Schema parsing failed, using raw response",
		Heartbeat:   false,
		FinalAnswer: h.truncateText(rawText, 2000),
		Focus:       "",
		Evidence:    "",
		UsageInfo:   &ai.GenerationUsage{},
	}, nil
}

// extractJSON attempts to extract and parse JSON from text
func (h *Handler) extractJSON(text string) map[string]interface{} {
	// Find JSON object in text
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")

	if start == -1 || end == -1 || start >= end {
		return nil
	}

	jsonStr := text[start : end+1]
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil
	}

	return result
}

// getStringField safely extracts a string field from parsed JSON
func (h *Handler) getStringField(parsed map[string]interface{}, field string) string {
	if val, ok := parsed[field]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// getBoolField safely extracts a bool field from parsed JSON
func (h *Handler) getBoolField(parsed map[string]interface{}, field string, defaultValue bool) bool {
	if val, ok := parsed[field]; ok {
		switch v := val.(type) {
		case bool:
			return v
		case string:
			return v == "true" || v == "1"
		case float64:
			return v > 0
		}
	}
	return defaultValue
}

// truncateText truncates text to max length
func (h *Handler) truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

// logRawOutput logs raw model output for debugging
func (h *Handler) logRawOutput(stage string, output string) {
	logOutput := output
	if len(logOutput) > maxRawOutputLength {
		logOutput = logOutput[:maxRawOutputLength] + "..."
	}
	runtime.GetLogger().Debug("Raw model output", "stage", stage, "output", logOutput)
}

// ============================================
// JSON Marshal Fallback for Message Creation
// ============================================

// MarshalObservation creates a Message from Observation with JSON fallback to text
func (h *Handler) MarshalObservation(observation *schema.Observation) *ai.Message {
	obsJson, err := json.Marshal(observation)
	if err != nil {
		runtime.GetLogger().Debug("JSON marshal failed for Observation, using text fallback", "error", err)
		return ai.NewMessage(ai.RoleModel, nil, ai.NewTextPart(observation.Summary))
	}
	return ai.NewMessage(ai.RoleModel, nil, ai.NewJSONPart(string(obsJson)))
}
