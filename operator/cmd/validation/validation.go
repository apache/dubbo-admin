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

package validation

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/apache/dubbo-admin/operator/pkg/apis"
	"github.com/apache/dubbo-admin/operator/pkg/util"
	"github.com/apache/dubbo-admin/operator/pkg/values"
	"sigs.k8s.io/yaml"
)

type Warnings = util.Errors

func ParseAndValidateDubboOperator(dopMap values.Map) (Warnings, util.Errors) {
	dop := &apis.DubboOperator{}
	dec := json.NewDecoder(bytes.NewBufferString(dopMap.JSON()))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dop); err != nil {
		return nil, util.NewErrs(fmt.Errorf("could not unmarshal: %v", err))
	}
	var warnings Warnings
	var errors util.Errors
	vw, ve := validateValues(dop)
	warnings = util.AppendErrs(warnings, vw)
	errors = util.AppendErrs(errors, ve)
	errors = util.AppendErr(errors, validateComponentNames(dop.Spec.Components))
	return warnings, errors
}

func validateValues(raw *apis.DubboOperator) (Warnings, util.Errors) {
	v := &apis.Values{}
	if err := yaml.Unmarshal(raw.Spec.Values, v); err != nil {
		return nil, util.NewErrs(fmt.Errorf("could not unmarshal: %v", err))
	}
	return nil, nil
}

func validateComponentNames(components *apis.DubboComponentSpec) error {
	if components == nil {
		return fmt.Errorf("components not found")
	}
	return nil
}
