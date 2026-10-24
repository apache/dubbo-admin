/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package modelstest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"dubbo-admin-ai/component/models"
	"dubbo-admin-ai/config"
)

// loadModelsSpec runs the real shipped models.yaml through the real config
// loader, so schema validation, ${VAR} expansion and strict field checking all
// apply. The provider and model_info schema definitions set
// additionalProperties:false, which means a compat key added to the Go struct
// but not to the JSON schema is rejected here rather than at startup.
func loadModelsSpec(t *testing.T) *models.ModelsSpec {
	t.Helper()

	modelsYAML := filepath.Join("..", "models.yaml")
	schemaDir, err := filepath.Abs(filepath.Join("..", "..", "..", "schema", "json"))
	if err != nil {
		t.Fatalf("resolve schema dir: %v", err)
	}
	t.Setenv("SCHEMA_DIR", schemaDir)

	loaded, err := config.NewLoader(modelsYAML).LoadComponent("models.yaml")
	if err != nil {
		t.Fatalf("load %s: %v", modelsYAML, err)
	}

	var spec models.ModelsSpec
	if err := loaded.Spec.Decode(&spec); err != nil {
		t.Fatalf("decode models spec: %v", err)
	}
	return &spec
}

func TestShippedModelsYAML_Loads(t *testing.T) {
	spec := loadModelsSpec(t)
	if spec.DefaultModel == "" {
		t.Error("default_model should be set")
	}
	if len(spec.Providers) == 0 {
		t.Fatal("expected at least one provider")
	}
}

func TestShippedDashScopeCompat(t *testing.T) {
	spec := loadModelsSpec(t)

	dashscope, ok := spec.Providers["dashscope"]
	if !ok {
		t.Fatal("dashscope provider missing from models.yaml")
	}
	if dashscope.Compat == nil {
		t.Fatal("dashscope should carry a compat block")
	}

	// The block has to survive cross-field validation, or the service refuses
	// to start.
	if err := dashscope.Compat.Validate("dashscope"); err != nil {
		t.Fatalf("shipped dashscope compat is invalid: %v", err)
	}

	resolved := models.ResolveCompat(dashscope.Compat, nil)
	if resolved.ThinkingFormat != models.ThinkingFormatQwen {
		t.Errorf("thinking_format = %q, want qwen", resolved.ThinkingFormat)
	}
	if resolved.MaxTokensField != "max_tokens" {
		t.Errorf("max_tokens_field = %q, want max_tokens", resolved.MaxTokensField)
	}
	if resolved.ThinkingBudgetField != "thinking_budget" {
		t.Errorf("thinking_budget_field = %q, want thinking_budget", resolved.ThinkingBudgetField)
	}
	if resolved.SupportsDeveloperRole {
		t.Error("dashscope should not be marked as accepting the developer role")
	}
	if resolved.SupportsStrictMode {
		t.Error("the Qwen family rejects strict tool definitions")
	}
	if resolved.SupportsForcedToolChoice {
		t.Error("dashscope should not be marked as accepting a forced tool choice")
	}
}

// A model that is not configured individually must inherit the provider block
// rather than silently reverting to defaults.
func TestShippedCompat_InheritedByModels(t *testing.T) {
	spec := loadModelsSpec(t)
	dashscope := spec.Providers["dashscope"]

	for _, m := range dashscope.Models {
		if m.Compat != nil {
			t.Errorf("model %q overrides compat; the inheritance case is not covered", m.Name)
		}
		resolved := models.ResolveCompat(dashscope.Compat, m.Compat)
		if resolved.ThinkingFormat != models.ThinkingFormatQwen {
			t.Errorf("model %q: thinking_format = %q, want qwen inherited from the provider",
				m.Name, resolved.ThinkingFormat)
		}
	}
}

// Models with no compat block anywhere must still resolve, using the
// conservative defaults rather than failing.
func TestShippedSiliconFlowFallsBackToDefaults(t *testing.T) {
	spec := loadModelsSpec(t)
	siliconflow, ok := spec.Providers["siliconflow"]
	if !ok {
		t.Fatal("siliconflow provider missing from models.yaml")
	}

	resolved := models.ResolveCompat(siliconflow.Compat, nil)
	if resolved.ThinkingFormat != models.ThinkingFormatOpenAI {
		t.Errorf("thinking_format = %q, want the openai default", resolved.ThinkingFormat)
	}
	if resolved.MaxTokensField != "max_tokens" {
		t.Errorf("max_tokens_field = %q, want the max_tokens default", resolved.MaxTokensField)
	}
	if !resolved.SupportsForcedToolChoice {
		t.Error("forced tool choice should default to allowed")
	}
}

// A compat block carrying a value the Go side does not know must be rejected by
// the JSON schema, not silently ignored.
func TestCompatSchemaRejectsUnknownField(t *testing.T) {
	schemaDir, err := filepath.Abs(filepath.Join("..", "..", "..", "schema", "json"))
	if err != nil {
		t.Fatalf("resolve schema dir: %v", err)
	}
	t.Setenv("SCHEMA_DIR", schemaDir)

	_, err = config.NewLoader("config.yaml").LoadComponent(
		"testdata/unknown_compat_field.yaml")
	if err == nil {
		t.Fatal("expected the loader to reject an unknown compat key")
	}
}

// Guard the shipped schema itself: the compat block must be reachable from both
// provider and model_info, or operators can only set it at one level.
func TestCompatSchemaReachableFromBothLevels(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "schema", "json", "models.schema.json"))
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse schema: %v", err)
	}

	defsKey := "$" + "defs"
	defs, _ := doc[defsKey].(map[string]any)
	if defs == nil {
		t.Fatalf("schema has no %s", defsKey)
	}
	if _, ok := defs["compat"]; !ok {
		t.Fatal("schema is missing the compat definition")
	}
	for _, level := range []string{"provider", "model_info"} {
		def, _ := defs[level].(map[string]any)
		props, _ := def["properties"].(map[string]any)
		if props == nil {
			t.Fatalf("schema %s definition has no properties", level)
		}
		if _, ok := props["compat"]; !ok {
			t.Errorf("schema %s does not accept compat", level)
		}
	}
}
