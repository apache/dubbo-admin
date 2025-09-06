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

const (
	HasToBeGreaterThan                = "must be greater than"
	HasToBeLessThan                   = "must be less than"
	HasToBeGreaterOrEqualThen         = "must be greater or equal then"
	HasToBeGreaterThanZero            = "must be greater than 0"
	MustNotBeEmpty                    = "must not be empty"
	MustBeDefined                     = "must be defined"
	MustBeSet                         = "must be set"
	MustNotBeSet                      = "must not be set"
	MustNotBeDefined                  = "must not be defined"
	MustBeDefinedAndGreaterThanZero   = "must be defined and greater than zero"
	WhenDefinedHasToBeNonNegative     = "must not be negative when defined"
	WhenDefinedHasToBeGreaterThanZero = "must be greater than zero when defined"
	HasToBeInRangeFormat              = "must be in inclusive range [%v, %v]"
	WhenDefinedHasToBeValidPath       = "must be a valid path when defined"
	StringHasToBeValidNumber          = "string must be a valid number"
	MustHaveBPSUnit                   = "must be in kbps/Mbps/Gbps units"
)

var (
	HasToBeInPercentageRange     = fmt.Sprintf(HasToBeInRangeFormat, "0.0", "100.0")
	HasToBeInUintPercentageRange = fmt.Sprintf(HasToBeInRangeFormat, 0, 100)
)

func MustHaveOnlyOne(entity string, allowedValues ...string) string {
	return fmt.Sprintf(`%s must have only one type defined: %s`, entity, strings.Join(allowedValues, ", "))
}

func MustHaveExactlyOneOf(entity string, allowedValues ...string) string {
	return fmt.Sprintf(`%s must have exactly one defined: %s`, entity, strings.Join(allowedValues, ", "))
}

func MustHaveAtLeastOne(allowedValues ...string) string {
	return fmt.Sprintf(`must have at least one defined: %s`, strings.Join(allowedValues, ", "))
}
