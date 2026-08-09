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

package react

import (
	"context"
	"errors"
	"testing"

	"dubbo-admin-ai/schema"
)

func newState() *state {
	return &state{Input: &schema.UserInput{Content: "hi"}, Session: "s"}
}

func TestRunLoop_TerminatesWhenStepReportsDone(t *testing.T) {
	var think, act, observe int
	steps := []step{
		func(ctx context.Context, s *state) (bool, error) { think++; return false, nil },
		func(ctx context.Context, s *state) (bool, error) { act++; return false, nil },
		func(ctx context.Context, s *state) (bool, error) {
			observe++
			return observe >= 2, nil // done on the second round
		},
	}

	if err := runLoop(context.Background(), newState(), 10, steps...); err != nil {
		t.Fatalf("runLoop error: %v", err)
	}
	if think != 2 || act != 2 || observe != 2 {
		t.Fatalf("expected 2 rounds, got think=%d act=%d observe=%d", think, act, observe)
	}
}

func TestRunLoop_StopsAtMaxIterations(t *testing.T) {
	var calls int
	stepFn := func(ctx context.Context, s *state) (bool, error) { calls++; return false, nil }

	if err := runLoop(context.Background(), newState(), 3, stepFn); err != nil {
		t.Fatalf("runLoop error: %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected step to run maxIter=3 times, got %d", calls)
	}
}

func TestRunLoop_PropagatesStepError(t *testing.T) {
	sentinel := errors.New("boom")
	var second int
	steps := []step{
		func(ctx context.Context, s *state) (bool, error) { return false, sentinel },
		func(ctx context.Context, s *state) (bool, error) { second++; return false, nil },
	}

	err := runLoop(context.Background(), newState(), 5, steps...)
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", err)
	}
	if second != 0 {
		t.Fatalf("later step should not run after an error, ran %d times", second)
	}
}

func TestRunLoop_RejectsNilInput(t *testing.T) {
	if err := runLoop(context.Background(), nil, 1, func(context.Context, *state) (bool, error) { return true, nil }); err == nil {
		t.Fatal("expected error for nil state")
	}
	if err := runLoop(context.Background(), &state{}, 1, func(context.Context, *state) (bool, error) { return true, nil }); err == nil {
		t.Fatal("expected error for nil input")
	}
}
