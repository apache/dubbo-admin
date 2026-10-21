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

package versioning

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

// DeleteSpecJSON is the canonical snapshot stored for a rule delete marker.
const DeleteSpecJSON = "{}"

const routerRuleSnapshotFormat = "router-rule/v1"

// routerRuleSnapshot keeps the scalar protobuf spec separate from the
// field-presence information that proto3 cannot represent for optional bools.
// Both members are included in the canonical JSON and therefore in the
// content hash used by diff and rollback checks.
type routerRuleSnapshot struct {
	Format        string          `json:"format"`
	Spec          json.RawMessage `json:"spec"`
	OmittedFields []string        `json:"omittedFields,omitempty"`
}

// scriptRouteSnapshotSpec deliberately does not use protojson because
// proto3 omits false scalars. Script enabled must retain an explicit false
// across history and rollback.
type scriptRouteSnapshotSpec struct {
	ConfigVersion string `json:"configVersion"`
	Scope         string `json:"scope"`
	Key           string `json:"key"`
	Enabled       bool   `json:"enabled"`
	Type          string `json:"type"`
	Script        string `json:"script"`
}

// NormalizeSpec returns the spec hash followed by canonical JSON for stable comparisons.
// It does not validate whether the spec is acceptable to the registry.
func NormalizeSpec(spec coremodel.ResourceSpec) (specHash string, specJSON string, err error) {
	if spec == nil {
		return HashSpecJSON(DeleteSpecJSON), DeleteSpecJSON, nil
	}
	var raw []byte
	if msg, ok := spec.(proto.Message); ok {
		var err error
		raw, err = protojson.MarshalOptions{
			UseProtoNames:   false,
			EmitUnpopulated: false,
		}.Marshal(msg)
		if err != nil {
			return "", "", err
		}
	} else {
		var err error
		raw, err = json.Marshal(spec)
		if err != nil {
			return "", "", err
		}
	}
	specJSON, err = canonicalJSON(raw)
	if err != nil {
		return "", "", err
	}
	return HashSpecJSON(specJSON), specJSON, nil
}

// HashSpecJSON hashes canonical spec JSON for comparison and dedup filters.
// It is not sufficient on its own to prove operation source.
func HashSpecJSON(specJSON string) string {
	sum := sha256.Sum256([]byte(specJSON))
	return hex.EncodeToString(sum[:])
}

func NormalizeResource(res coremodel.Resource) (string, string, error) {
	if res == nil {
		return "", "", fmt.Errorf("resource is nil")
	}
	if snapshot, ok, err := normalizeRouterRuleSnapshot(res); err != nil {
		return "", "", err
	} else if ok {
		raw, err := json.Marshal(snapshot)
		if err != nil {
			return "", "", err
		}
		specJSON, err := canonicalJSON(raw)
		if err != nil {
			return "", "", err
		}
		return HashSpecJSON(specJSON), specJSON, nil
	}
	return NormalizeSpec(res.ResourceSpec())
}

// ResourceFromSpecJSON rebuilds a typed rule Resource from a stored version's
// spec JSON. Used by rollback to re-publish a historical snapshot through the
// normal ResourceManager mutation path. The special Router rule snapshots also
// restore the field-presence metadata required by their codecs. protojson is
// tried first (matching how specs are
// normalized), falling back to plain JSON for resilience.
func ResourceFromSpecJSON(kind coremodel.ResourceKind, mesh, ruleName, specJSON string) (coremodel.Resource, error) {
	switch kind {
	case meshresource.ConditionRouteKind:
		res := meshresource.NewConditionRouteResourceWithAttributes(ruleName, mesh)
		var spec meshproto.ConditionRoute
		if err := unmarshalSpec(specJSON, &spec); err != nil {
			return nil, err
		}
		res.Spec = &spec
		return res, nil
	case meshresource.TagRouteKind:
		res := meshresource.NewTagRouteResourceWithAttributes(ruleName, mesh)
		var spec meshproto.TagRoute
		if err := unmarshalSpec(specJSON, &spec); err != nil {
			return nil, err
		}
		res.Spec = &spec
		return res, nil
	case meshresource.DynamicConfigKind:
		res := meshresource.NewDynamicConfigResourceWithAttributes(ruleName, mesh)
		var spec meshproto.DynamicConfig
		if err := unmarshalSpec(specJSON, &spec); err != nil {
			return nil, err
		}
		res.Spec = &spec
		return res, nil
	case meshresource.AffinityRouteKind:
		if res, handled, err := affinityResourceFromSnapshot(mesh, ruleName, specJSON); handled || err != nil {
			return res, err
		}
		res := meshresource.NewAffinityRouteResourceWithAttributes(ruleName, mesh)
		var spec meshproto.AffinityRoute
		if err := unmarshalSpec(specJSON, &spec); err != nil {
			return nil, err
		}
		res.Spec = &spec
		return res, nil
	case meshresource.ScriptRouteKind:
		if res, handled, err := scriptResourceFromSnapshot(mesh, ruleName, specJSON); handled || err != nil {
			return res, err
		}
		res := meshresource.NewScriptRouteResourceWithAttributes(ruleName, mesh)
		var spec meshproto.ScriptRoute
		if err := unmarshalSpec(specJSON, &spec); err != nil {
			return nil, err
		}
		res.Spec = &spec
		return res, nil
	default:
		return nil, bizerror.New(bizerror.InvalidArgument, "unsupported rule kind")
	}
}

func normalizeRouterRuleSnapshot(res coremodel.Resource) (routerRuleSnapshot, bool, error) {
	var (
		specJSON      string
		omittedFields []string
		err           error
	)
	switch typed := res.(type) {
	case *meshresource.AffinityRouteResource:
		if typed == nil || typed.Spec == nil {
			return routerRuleSnapshot{}, false, nil
		}
		_, specJSON, err = NormalizeSpec(typed.Spec)
		if err != nil {
			return routerRuleSnapshot{}, true, err
		}
		runtime, enabled := meshresource.AffinityRuleOptionalFields(typed)
		if runtime == nil {
			omittedFields = append(omittedFields, "runtime")
		}
		if enabled == nil {
			omittedFields = append(omittedFields, "enabled")
		}
	case *meshresource.ScriptRouteResource:
		if typed == nil || typed.Spec == nil {
			return routerRuleSnapshot{}, false, nil
		}
		raw, marshalErr := json.Marshal(scriptRouteSnapshotSpec{
			ConfigVersion: typed.Spec.ConfigVersion,
			Scope:         typed.Spec.Scope,
			Key:           typed.Spec.Key,
			Enabled:       typed.Spec.Enabled,
			Type:          typed.Spec.Type,
			Script:        typed.Spec.Script,
		})
		if marshalErr != nil {
			return routerRuleSnapshot{}, true, marshalErr
		}
		specJSON, err = canonicalJSON(raw)
		if err != nil {
			return routerRuleSnapshot{}, true, err
		}
	default:
		return routerRuleSnapshot{}, false, nil
	}
	return routerRuleSnapshot{
		Format:        routerRuleSnapshotFormat,
		Spec:          json.RawMessage(specJSON),
		OmittedFields: omittedFields,
	}, true, nil
}

func affinityResourceFromSnapshot(mesh, ruleName, specJSON string) (coremodel.Resource, bool, error) {
	snapshot, handled, err := parseRouterRuleSnapshot(specJSON)
	if !handled || err != nil {
		return nil, handled, err
	}
	if err := validateOmittedFields(snapshot.OmittedFields, "runtime", "enabled"); err != nil {
		return nil, true, err
	}
	res := meshresource.NewAffinityRouteResourceWithAttributes(ruleName, mesh)
	var spec meshproto.AffinityRoute
	if err := unmarshalSpec(string(snapshot.Spec), &spec); err != nil {
		return nil, true, err
	}
	res.Spec = &spec
	runtime, enabled := &res.Spec.Runtime, &res.Spec.Enabled
	if contains(snapshot.OmittedFields, "runtime") {
		runtime = nil
	}
	if contains(snapshot.OmittedFields, "enabled") {
		enabled = nil
	}
	meshresource.SetAffinityRuleOptionalFields(res, runtime, enabled)
	return res, true, nil
}

func scriptResourceFromSnapshot(mesh, ruleName, specJSON string) (coremodel.Resource, bool, error) {
	snapshot, handled, err := parseRouterRuleSnapshot(specJSON)
	if !handled || err != nil {
		return nil, handled, err
	}
	if err := validateOmittedFields(snapshot.OmittedFields, "enabled"); err != nil {
		return nil, true, err
	}
	res := meshresource.NewScriptRouteResourceWithAttributes(ruleName, mesh)
	var spec meshproto.ScriptRoute
	if err := unmarshalSpec(string(snapshot.Spec), &spec); err != nil {
		return nil, true, err
	}
	res.Spec = &spec
	enabled := &res.Spec.Enabled
	if contains(snapshot.OmittedFields, "enabled") {
		enabled = nil
	}
	meshresource.SetScriptRuleOptionalEnabled(res, enabled)
	return res, true, nil
}

func parseRouterRuleSnapshot(specJSON string) (routerRuleSnapshot, bool, error) {
	var snapshot routerRuleSnapshot
	if err := json.Unmarshal([]byte(specJSON), &snapshot); err != nil {
		return routerRuleSnapshot{}, false, err
	}
	if snapshot.Format == "" {
		return routerRuleSnapshot{}, false, nil
	}
	if snapshot.Format != routerRuleSnapshotFormat {
		return routerRuleSnapshot{}, true, fmt.Errorf("unsupported router rule snapshot format %q", snapshot.Format)
	}
	if len(snapshot.Spec) == 0 || string(snapshot.Spec) == "null" {
		return routerRuleSnapshot{}, true, fmt.Errorf("router rule snapshot spec is required")
	}
	return snapshot, true, nil
}

func validateOmittedFields(fields []string, allowed ...string) error {
	for _, field := range fields {
		if !contains(allowed, field) {
			return fmt.Errorf("unsupported omitted router rule field %q", field)
		}
	}
	return nil
}

func contains(fields []string, candidate string) bool {
	for _, field := range fields {
		if field == candidate {
			return true
		}
	}
	return false
}

func canonicalJSON(raw []byte) (string, error) {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", err
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(canonical), nil
}

func unmarshalSpec(specJSON string, spec proto.Message) error {
	if err := protojson.Unmarshal([]byte(specJSON), spec); err != nil {
		if jsonErr := json.Unmarshal([]byte(specJSON), spec); jsonErr != nil {
			return err
		}
	}
	return nil
}
