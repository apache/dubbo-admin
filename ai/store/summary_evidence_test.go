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

package store

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestSummaryInspectionPreservesCrossBoundarySecretDetection(t *testing.T) {
	for _, offset := range []int{8192 - 5, 8192 + 500} {
		text := strings.Repeat("plain ", 3000)
		text = text[:offset] + " password=must-never-escape " + text[offset:]
		read := func(start, size int) (string, int, error) {
			return text[start:min(start+size, len(text))], len(text), nil
		}
		check, err := InspectSummaryRecord(3, read)
		if err != nil || !check.Sensitive || !check.Valid(3) || check.Valid(4) {
			t.Fatalf("invalid source inspection: %+v %v", check, err)
		}
		// Even the first fragment is withheld when the secret occurs later.
		fragment, _, err := ReadInspectedSummaryFragment(check, 3, 0, 256, read)
		if err != nil || !strings.Contains(fragment, "Sensitive canonical record excluded") || strings.Contains(fragment, "plain") {
			t.Fatalf("sensitive record emitted an early fragment: %q %v", fragment, err)
		}
		check.Rule = "old-rule"
		if _, _, err := ReadInspectedSummaryFragment(check, 3, 0, 256, read); !errors.Is(err, ErrContextConflict) {
			t.Fatal("stale detector version accepted")
		}
	}
}

func TestInterruptedSummaryInspectionCannotBeReused(t *testing.T) {
	reads := 0
	check, err := InspectSummaryRecord(1, func(start, size int) (string, int, error) {
		reads++
		if reads == 3 {
			return "", 0, context.DeadlineExceeded
		}
		return strings.Repeat(" ", size), 1 << 20, nil
	})
	if !errors.Is(err, context.DeadlineExceeded) || check.Valid(1) {
		t.Fatalf("incomplete scan became reusable: %+v %v", check, err)
	}
}
