package modelstest

import (
	"dubbo-admin-ai/component/models"
	"strings"
	"testing"
)

func TestModelsComponent_Validate(t *testing.T) {
	comp, err := models.NewModelsComponent("dashscope/qwen-max", "dashscope/text-embedding-v4", map[string]models.ProviderConfig{})
	if err != nil {
		t.Fatalf("NewModelsComponent() error: %v", err)
	}
	if err := comp.Validate(); err == nil || !strings.Contains(err.Error(), "at least one provider") {
		t.Fatalf("expected providers validation error, got %v", err)
	}

	comp2, err := models.NewModelsComponent("dashscope/qwen-max", "dashscope/text-embedding-v4", map[string]models.ProviderConfig{
		"dashscope": {APIKey: "x", BaseURL: ""},
	})
	if err != nil {
		t.Fatalf("NewModelsComponent() error: %v", err)
	}
	if err := comp2.Validate(); err == nil || !strings.Contains(err.Error(), "base_url") {
		t.Fatalf("expected base_url validation error, got %v", err)
	}
}
