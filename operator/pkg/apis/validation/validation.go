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
	"github.com/apache/dubbo-admin/pkg/kube"
	"sigs.k8s.io/yaml"
)

type Warnings = util.Errors

func ParseAndValidateDubboOperator(dopMap values.Map, _ kube.CLIClient) (Warnings, util.Errors) {
	iop := &apis.DubboOperator{}
	dec := json.NewDecoder(bytes.NewBufferString(dopMap.JSON()))
	dec.DisallowUnknownFields()
	if err := dec.Decode(iop); err != nil {
		return nil, util.NewErrs(fmt.Errorf("could not unmarshal: %v", err))
	}
	var warnings Warnings
	var errors util.Errors

	vw, ve := validateValues(iop)
	warnings = util.AppendErrs(warnings, vw)
	errors = util.AppendErrs(errors, ve)
	errors = util.AppendErr(errors, validateComponentNames(iop.Spec.Components))
	return warnings, errors
}

type FeatureValidator func(*apis.Values, apis.DubboOperatorSpec) (Warnings, util.Errors)

func validateFeatures(values *apis.Values, spec apis.DubboOperatorSpec) (Warnings, util.Errors) {
	validators := []FeatureValidator{}
	var warnings Warnings
	var errs util.Errors
	for _, validator := range validators {
		newWarnings, newErrs := validator(values, spec)
		errs = util.AppendErrs(errs, newErrs)
		warnings = append(warnings, newWarnings...)
	}
	return warnings, errs
}

func validateValues(raw *apis.DubboOperator) (Warnings, util.Errors) {
	vls := &apis.Values{}
	if err := yaml.Unmarshal(raw.Spec.Values, vls); err != nil {
		return nil, util.NewErrs(fmt.Errorf("could not unmarshal: %v", err))
	}
	warnings, errs := validateFeatures(vls, raw.Spec)

	return warnings, errs
}

func validateComponentNames(components *apis.DubboComponentSpec) error {
	if components == nil {
		return nil
	}
	return nil
}
