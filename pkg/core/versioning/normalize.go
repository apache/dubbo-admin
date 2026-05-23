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

	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

const DeleteSpecJSON = "{}"

func NormalizeSpec(spec coremodel.ResourceSpec) (string, string, error) {
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
	specJSON := string(canonical)
	return HashSpecJSON(specJSON), specJSON, nil
}

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
