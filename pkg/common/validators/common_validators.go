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
	"math"
	"regexp"
	"time"

	"github.com/asaskevich/govalidator"
	k8s "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ValidateDurationNotNegative(path PathBuilder, duration *k8s.Duration) ValidationError {
	var err ValidationError
	if duration == nil {
		err.AddViolationAt(path, MustBeDefined)
		return err
	}
	if duration.Duration < 0 {
		err.AddViolationAt(path, WhenDefinedHasToBeNonNegative)
	}
	return err
}

func ValidateDurationNotNegativeOrNil(path PathBuilder, duration *k8s.Duration) ValidationError {
	var err ValidationError
	if duration == nil {
		return err
	}

	if duration.Duration < 0 {
		err.AddViolationAt(path, WhenDefinedHasToBeNonNegative)
	}

	return err
}

func ValidateDurationGreaterThanZero(path PathBuilder, duration k8s.Duration) ValidationError {
	var err ValidationError
	if duration.Duration <= 0 {
		err.AddViolationAt(path, MustBeDefinedAndGreaterThanZero)
	}
	return err
}

func ValidateDurationGreaterThanZeroOrNil(path PathBuilder, duration *k8s.Duration) ValidationError {
	var err ValidationError
	if duration == nil {
		return err
	}

	if duration.Duration <= 0 {
		err.AddViolationAt(path, WhenDefinedHasToBeGreaterThanZero)
	}

	return err
}

func ValidateValueGreaterThanZero(path PathBuilder, value int32) ValidationError {
	var err ValidationError
	if value <= 0 {
		err.AddViolationAt(path, MustBeDefinedAndGreaterThanZero)
	}
	return err
}

func ValidateValueGreaterThanZeroOrNil(path PathBuilder, value *int32) ValidationError {
	var err ValidationError
	if value == nil {
		return err
	}
	if *value <= 0 {
		err.AddViolationAt(path, WhenDefinedHasToBeGreaterThanZero)
	}
	return err
}

func ValidateIntPercentageOrNil(path PathBuilder, percentage *int32) ValidationError {
	var err ValidationError
	if percentage == nil {
		return err
	}

	if *percentage < 0 || *percentage > 100 {
		err.AddViolationAt(path, HasToBeInUintPercentageRange)
	}

	return err
}

func ValidateUInt32PercentageOrNil(path PathBuilder, percentage *uint32) ValidationError {
	var err ValidationError
	if percentage == nil {
		return err
	}

	if *percentage > 100 {
		err.AddViolationAt(path, HasToBeInUintPercentageRange)
	}

	return err
}

func ValidateStringDefined(path PathBuilder, value string) ValidationError {
	var err ValidationError
	if value == "" {
		err.AddViolationAt(path, MustBeDefined)
	}

	return err
}

func ValidatePathOrNil(path PathBuilder, filePath *string) ValidationError {
	var err ValidationError
	if filePath == nil {
		return err
	}

	isFilePath, _ := govalidator.IsFilePath(*filePath)
	if !isFilePath {
		err.AddViolationAt(path, WhenDefinedHasToBeValidPath)
	}

	return err
}

func ValidateStatusCode(path PathBuilder, status int32) ValidationError {
	var err ValidationError
	if status < 100 || status >= 600 {
		err.AddViolationAt(path, fmt.Sprintf(HasToBeInRangeFormat, 100, 599))
	}

	return err
}

func ValidateDurationGreaterThan(path PathBuilder, duration *k8s.Duration, minDuration time.Duration) ValidationError {
	var err ValidationError
	if duration == nil {
		err.AddViolationAt(path, MustBeDefined)
		return err
	}

	if duration.Duration <= minDuration {
		err.AddViolationAt(path, fmt.Sprintf("%s: %s", HasToBeGreaterThan, minDuration))
	}

	return err
}

func ValidateIntegerGreaterThanZeroOrNil(path PathBuilder, value *uint32) ValidationError {
	var err ValidationError
	if value == nil {
		return err
	}

	return ValidateIntegerGreaterThan(path, *value, 0)
}

func ValidateIntegerGreaterThan(path PathBuilder, value uint32, minValue uint32) ValidationError {
	var err ValidationError
	if value <= minValue {
		err.AddViolationAt(path, fmt.Sprintf("%s %d", HasToBeGreaterThan, minValue))
	}

	return err
}

var BandwidthRegex = regexp.MustCompile(`(\d*)\s?([GMk]?bps)`)

func ValidateBandwidth(path PathBuilder, value string) ValidationError {
	var err ValidationError
	if value == "" {
		err.AddViolationAt(path, MustBeDefined)
		return err
	}
	if matched := BandwidthRegex.MatchString(value); !matched {
		err.AddViolationAt(path, MustHaveBPSUnit)
	}
	return err
}

func ValidateNil[T any](path PathBuilder, t *T, msg string) ValidationError {
	var err ValidationError
	if t != nil {
		err.AddViolationAt(path, msg)
	}
	return err
}

func ValidatePort(path PathBuilder, value uint32) ValidationError {
	var err ValidationError
	if value == 0 || value > math.MaxUint16 {
		err.AddViolationAt(path, "port must be a valid (1-65535)")
	}
	return err
}
