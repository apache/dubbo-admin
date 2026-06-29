package fallback

import (
	"encoding/json"
	"fmt"
	"strings"

	"dubbo-admin-ai/runtime"
	"dubbo-admin-ai/schema"

	"github.com/firebase/genkit/go/ai"
)

// FallbackConfig defines fallback behavior for schema parsing failures
type FallbackConfig struct {
	EnableForThink        bool // Enable fallback for ThinkOutput parsing
	EnableForAct          bool // Enable fallback for ToolOutputs parsing
	EnableForObserve      bool // Enable fallback for Observation parsing
	EnableForExecuteError bool // Enable fallback when model Execute fails
	LogRawOutput          bool // Log raw model output for debugging
	MaxRawOutputLength    int  // Max length of raw output to log
}

// DefaultFallbackConfig returns default fallback configuration
func DefaultFallbackConfig() *FallbackConfig {
	return &FallbackConfig{
		EnableForThink:        true,
		EnableForAct:          true,
		EnableForObserve:      true,
		EnableForExecuteError: true,
		LogRawOutput:          true,
		MaxRawOutputLength:    500,
	}
}

// Handler handles fallback logic for agent stages
type Handler struct {
	config *FallbackConfig
}

// NewHandler creates a new fallback handler
func NewHandler(config *FallbackConfig) *Handler {
	if config == nil {
		config = DefaultFallbackConfig()
	}
	return &Handler{config: config}
}

// ParseResponse defines the interface for responses that can be parsed
type ParseResponse interface {
	Output(dst any) error
	Text() string
}

// DefaultThinkOutput returns a default ThinkOutput for fallback scenarios
func (h *Handler) DefaultThinkOutput(reason string) *schema.ThinkOutput {
	return &schema.ThinkOutput{
		Thought:        fmt.Sprintf("Model execution failed: %s. Continuing with general inquiry mode.", reason),
		Intent:         schema.GeneralInquiry,
		SuggestedTools: []string{},
		UsageInfo:      &ai.GenerationUsage{},
	}
}

// DefaultObservation returns a default Observation for fallback scenarios
func (h *Handler) DefaultObservation(reason string, userQuery string) *schema.Observation {
	return &schema.Observation{
		Summary:     fmt.Sprintf("Model execution failed: %s", reason),
		Heartbeat:   false,
		FinalAnswer: "I apologize, but I encountered an issue processing your request. Please try again.",
		Focus:       "",
		Evidence:    reason,
		UsageInfo:   &ai.GenerationUsage{},
	}
}

// ParseThinkOutput parses ThinkOutput with fallback
func (h *Handler) ParseThinkOutput(resp ParseResponse) (*schema.ThinkOutput, error) {
	var thinkOut schema.ThinkOutput
	thinkOut.UsageInfo = &ai.GenerationUsage{}

	if err := resp.Output(&thinkOut); err != nil {
		if !h.config.EnableForThink {
			return nil, fmt.Errorf("failed to parse ThinkOutput (fallback disabled): %w", err)
		}

		runtime.GetLogger().Warn("ThinkOutput schema parsing failed, using fallback", "error", err)
		return h.fallbackThinkOutput(resp)
	}

	return &thinkOut, nil
}

// ParseObservation parses Observation with fallback
func (h *Handler) ParseObservation(resp ParseResponse) (*schema.Observation, error) {
	var observation schema.Observation
	observation.UsageInfo = &ai.GenerationUsage{}

	if err := resp.Output(&observation); err != nil {
		if !h.config.EnableForObserve {
			return nil, fmt.Errorf("failed to parse Observation (fallback disabled): %w", err)
		}

		runtime.GetLogger().Warn("Observation schema parsing failed, using fallback", "error", err)
		return h.fallbackObservation(resp)
	}

	return &observation, nil
}

// fallbackThinkOutput creates a ThinkOutput from raw text when schema parsing fails
func (h *Handler) fallbackThinkOutput(resp ParseResponse) (*schema.ThinkOutput, error) {
	rawText := resp.Text()
	h.logRawOutput("ThinkOutput", rawText)

	// Try to extract JSON from the text
	if parsed := h.extractJSON(rawText); parsed != nil {
		if thought, ok := parsed["thought"].(string); ok {
			return &schema.ThinkOutput{
				Thought:        thought,
				Intent:         schema.GeneralInquiry,
				SuggestedTools: []string{},
				UsageInfo:      &ai.GenerationUsage{},
			}, nil
		}
	}

	// Fallback to using raw text as thought
	return &schema.ThinkOutput{
		Thought:        h.truncateText(rawText, 1000),
		Intent:         schema.GeneralInquiry,
		SuggestedTools: []string{},
		UsageInfo:      &ai.GenerationUsage{},
	}, nil
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
	if h.config.LogRawOutput {
		logOutput := output
		if len(logOutput) > h.config.MaxRawOutputLength {
			logOutput = logOutput[:h.config.MaxRawOutputLength] + "..."
		}
		runtime.GetLogger().Debug("Raw model output", "stage", stage, "output", logOutput)
	}
}

// ============================================
// JSON Marshal Fallback for Message Creation
// ============================================

// MarshalThinkOutput creates a Message from ThinkOutput with JSON fallback to text
func (h *Handler) MarshalThinkOutput(thinkOut *schema.ThinkOutput) *ai.Message {
	thinkJson, err := json.Marshal(thinkOut)
	if err != nil {
		runtime.GetLogger().Debug("JSON marshal failed for ThinkOutput, using text fallback", "error", err)
		return ai.NewMessage(ai.RoleModel, nil, ai.NewTextPart(thinkOut.Thought))
	}
	return ai.NewMessage(ai.RoleModel, nil, ai.NewJSONPart(string(thinkJson)))
}

// MarshalToolOutputs creates a Message from ToolOutputs with JSON fallback to text
func (h *Handler) MarshalToolOutputs(toolOut *schema.ToolOutputs) *ai.Message {
	actJson, err := json.Marshal(toolOut)
	if err != nil {
		runtime.GetLogger().Debug("JSON marshal failed for ToolOutputs, using text fallback", "error", err)
		return ai.NewMessage(ai.RoleModel, nil, ai.NewTextPart(toolOut.Thought))
	}
	return ai.NewMessage(ai.RoleModel, nil, ai.NewJSONPart(string(actJson)))
}

// MarshalObservation creates a Message from Observation with JSON fallback to text
func (h *Handler) MarshalObservation(observation *schema.Observation) *ai.Message {
	obsJson, err := json.Marshal(observation)
	if err != nil {
		runtime.GetLogger().Debug("JSON marshal failed for Observation, using text fallback", "error", err)
		return ai.NewMessage(ai.RoleModel, nil, ai.NewTextPart(observation.Summary))
	}
	return ai.NewMessage(ai.RoleModel, nil, ai.NewJSONPart(string(obsJson)))
}

// ============================================
// Loop Control Fallback
// ============================================

// LoopConfig defines loop control thresholds
type LoopConfig struct {
	MaxConsecutiveNoTools int // Max consecutive iterations without tools before fallback
	MaxIterations         int // Max total iterations before forced fallback
}

// DefaultLoopConfig returns default loop control configuration
func DefaultLoopConfig() *LoopConfig {
	return &LoopConfig{
		MaxConsecutiveNoTools: 2,
		MaxIterations:         10,
	}
}

// ShouldForceFallback checks if we should force a fallback based on loop state
// Returns (shouldFallback, reason)
func (h *Handler) ShouldForceFallback(consecutiveNoTools int, iteration int, maxIteration int) (bool, string) {
	if consecutiveNoTools >= DefaultLoopConfig().MaxConsecutiveNoTools {
		return true, fmt.Sprintf("too many iterations without tools (%d)", consecutiveNoTools)
	}
	if iteration >= maxIteration-1 {
		return true, fmt.Sprintf("reached maximum iterations (%d)", iteration+1)
	}
	return false, ""
}

// MaxIterationsFallback creates an Observation when max iterations is reached
func (h *Handler) MaxIterationsFallback(iteration int, userQuery string) *schema.Observation {
	return &schema.Observation{
		Summary:     fmt.Sprintf("Reached maximum iterations (%d) without completing the task", iteration),
		Heartbeat:   false,
		FinalAnswer: "I apologize, but I need more information or different tools to answer your question. Could you please rephrase or provide more context?",
		Focus:       "",
		Evidence:    fmt.Sprintf("Completed %d iterations without reaching a conclusion", iteration),
		UsageInfo:   &ai.GenerationUsage{},
	}
}

// NoToolsFallback creates an Observation when no tools are available
func (h *Handler) NoToolsFallback(intent string, userQuery string) *schema.Observation {
	return &schema.Observation{
		Summary:     fmt.Sprintf("No tools available for this query (intent=%s)", intent),
		Heartbeat:   false,
		FinalAnswer: h.getNoToolsAnswer(intent, userQuery),
		Focus:       "",
		Evidence:    fmt.Sprintf("Query classified as %s with no applicable tools", intent),
		UsageInfo:   &ai.GenerationUsage{},
	}
}

// getNoToolsAnswer provides a contextual answer when no tools are available
func (h *Handler) getNoToolsAnswer(intent string, userQuery string) string {
	// Provide a helpful response based on the intent type
	switch schema.PrimaryIntent(intent) {
	case schema.GeneralInquiry:
		return "I can help with general questions about Dubbo and Kubernetes. For specific service details or configurations, please provide more context so I can use the appropriate tools."
	default:
		return "I apologize, but I don't have the right tools available to answer this question. Please try asking in a different way or provide more specific details."
	}
}

// ExecuteErrorFallback creates a ThinkOutput when model Execute fails
func (h *Handler) ExecuteErrorFallback(stage string, err error, userQuery string) *schema.ThinkOutput {
	return &schema.ThinkOutput{
		Thought:        fmt.Sprintf("Model execution failed at %s stage: %v. Please check the logs and retry.", stage, err),
		Intent:         schema.GeneralInquiry,
		SuggestedTools: []string{},
		UsageInfo:      &ai.GenerationUsage{},
	}
}
