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

package validators

import (
	"fmt"
	"strings"
)

type ValidationError struct {
	Violations []Violation `json:"violations"`
}

type Violation struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// OK returns and empty validation error (i.e. success).
func OK() ValidationError {
	return ValidationError{}
}

func (v *ValidationError) Error() string {
	msg := ""
	for _, violation := range v.Violations {
		if msg != "" {
			msg = fmt.Sprintf("%s; %s: %s", msg, violation.Field, violation.Message)
		} else {
			msg += fmt.Sprintf("%s: %s", violation.Field, violation.Message)
		}
	}
	return msg
}

func (v *ValidationError) HasViolations() bool {
	return len(v.Violations) > 0
}

func (v *ValidationError) OrNil() error {
	if v.HasViolations() {
		return v
	}
	return nil
}

func (v *ValidationError) AddViolationAt(path PathBuilder, message string) {
	v.AddViolation(path.String(), message)
}

func (v *ValidationError) AddViolation(field string, message string) {
	violation := Violation{
		Field:   field,
		Message: message,
	}
	v.Violations = append(v.Violations, violation)
}

func (v *ValidationError) AddErrorAt(path PathBuilder, validationErr ValidationError) {
	for _, violation := range validationErr.Violations {
		field := Root()
		if violation.Field != "" {
			field = RootedAt(violation.Field)
		}
		newViolation := Violation{
			Field:   path.concat(field).String(),
			Message: violation.Message,
		}
		v.Violations = append(v.Violations, newViolation)
	}
}

func (v *ValidationError) Add(err ValidationError) {
	v.AddErrorAt(Root(), err)
}

func (v *ValidationError) AddError(rootField string, validationErr ValidationError) {
	root := Root()
	if rootField != "" {
		root = RootedAt(rootField)
	}
	v.AddErrorAt(root, validationErr)
}

// Transform returns a new ValidationError with every violation
// transformed by a given transformFunc.
func (v *ValidationError) Transform(transformFunc func(Violation) Violation) *ValidationError {
	if v == nil {
		return nil
	}
	if transformFunc == nil || len(v.Violations) == 0 {
		rv := *v
		return &rv
	}
	result := ValidationError{
		Violations: make([]Violation, len(v.Violations)),
	}
	for i := range v.Violations {
		result.Violations[i] = transformFunc(v.Violations[i])
	}
	return &result
}

func MakeUnimplementedFieldErr(path PathBuilder) ValidationError {
	var err ValidationError
	err.AddViolationAt(path, "field is not implemented")
	return err
}

func MakeRequiredFieldErr(path PathBuilder) ValidationError {
	var err ValidationError
	err.AddViolationAt(path, "cannot be empty")
	return err
}

func MakeOneOfErr(fieldA, fieldB, msg string, oneOf []string) ValidationError {
	var err ValidationError
	var quoted []string

	for _, value := range oneOf {
		quoted = append(quoted, fmt.Sprintf("%q", value))
	}

	message := fmt.Sprintf(
		"%q %s one of [%s]",
		fieldA,
		msg,
		strings.Join(quoted, ", "),
	)

	if fieldB != "" {
		message = fmt.Sprintf(
			"%q %s when %q is one of [%s]",
			fieldA,
			msg,
			fieldB,
			strings.Join(quoted, ", "),
		)
	}

	err.AddViolationAt(Root(), message)

	return err
}

func MakeFieldMustBeOneOfErr(field string, allowed ...string) ValidationError {
	return MakeOneOfErr(field, "", "must be", allowed)
}

func IsValidationError(err error) bool {
	_, ok := err.(*ValidationError)
	return ok
}

type PathBuilder []string

func RootedAt(name string) PathBuilder {
	return PathBuilder{name}
}

func Root() PathBuilder {
	return PathBuilder{}
}

func (p PathBuilder) Field(name string) PathBuilder {
	element := name
	if len(p) > 0 {
		element = fmt.Sprintf(".%s", element)
	}
	return append(p, element)
}

func (p PathBuilder) Index(index int) PathBuilder {
	return append(p, fmt.Sprintf("[%d]", index))
}

func (p PathBuilder) Key(key string) PathBuilder {
	return append(p, fmt.Sprintf("[%q]", key))
}

func (p PathBuilder) String() string {
	return strings.Join(p, "")
}

func (p PathBuilder) concat(other PathBuilder) PathBuilder {
	if len(other) == 0 {
		return p
	}
	if len(p) == 0 {
		return other
	}

	firstOther := other[0]
	if !strings.HasPrefix(firstOther, "[") {
		firstOther = "." + firstOther
	}

	return append(append(p, firstOther), other[1:]...)
}
