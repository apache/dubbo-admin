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

import "github.com/apache/dubbo-admin/pkg/core/model"

type Warning error

// ValidateFunc defines a validation func for an API proto.
type ValidateFunc func(config model.Config) (Warning, error)

var (
	// EmptyValidate is a Validate that does nothing and returns no error.
	EmptyValidate = registerValidateFunc("EmptyValidate",
		func(model.Config) (Warning, error) {
			return nil, nil
		})

	validateFuncs = make(map[string]ValidateFunc)
)

func registerValidateFunc(name string, f ValidateFunc) ValidateFunc {
	validateFuncs[name] = f
	return f
}

// IsValidateFunc indicates whether there is a validation function with the given name.
func IsValidateFunc(name string) bool {
	return GetValidateFunc(name) != nil
}

// GetValidateFunc returns the validation function with the given name, or null if it does not exist.
func GetValidateFunc(name string) ValidateFunc {
	return validateFuncs[name]
}
