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
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return "", "", err
	}
	canonical, err := json.Marshal(v)
	if err != nil {
		return "", "", err
	}
	specJSON = string(canonical)
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
	return NormalizeSpec(res.ResourceSpec())
}

// ResourceFromSpecJSON rebuilds a typed rule Resource from a stored version's
// spec JSON. Used by rollback to re-publish a historical snapshot through the
// normal ResourceManager mutation path. Only the three governor-managed rule
// kinds are supported. protojson is tried first (matching how specs are
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
		res := meshresource.NewAffinityRouteResourceWithAttributes(ruleName, mesh)
		var spec meshproto.AffinityRoute
		if err := unmarshalSpec(specJSON, &spec); err != nil {
			return nil, err
		}
		res.Spec = &spec
		return res, nil
	case meshresource.ScriptRouteKind:
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

func unmarshalSpec(specJSON string, spec proto.Message) error {
	if err := protojson.Unmarshal([]byte(specJSON), spec); err != nil {
		if jsonErr := json.Unmarshal([]byte(specJSON), spec); jsonErr != nil {
			return err
		}
	}
	return nil
}
