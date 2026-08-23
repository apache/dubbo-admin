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
	"fmt"
	"time"

	"dubbo-admin-ai/component/hooks"
	"dubbo-admin-ai/schema"

	"github.com/firebase/genkit/go/ai"
)

// state is the shared work set for a single ReAct interaction. Steps read and
// write its concrete fields directly (Tools/Observe) instead of type-asserting
// an erased payload, keeping the reasonAct → observe handoff cheap and explicit.
type state struct {
	Input         *schema.UserInput
	Session       string
	InteractionID string
	Iteration     int
	Stage         string
	Model         string
	hookManager   *hooks.Manager

	Tools   *schema.ToolOutputs // reasonAct step writes
	Observe *schema.Observation // observe step writes

	// Usage is the running token accounting for the whole interaction; each
	// step accumulates its model call into it. The final observation reports it.
	Usage *ai.GenerationUsage

	FallbackUsed   bool
	FallbackReason string
	Degraded       bool // set when any tool call fails
}

// addUsage folds one or more model-call usages into the interaction total,
// lazily allocating the accumulator. The final observation reports s.Usage.
func (s *state) addUsage(src ...*ai.GenerationUsage) {
	if s.Usage == nil {
		s.Usage = &ai.GenerationUsage{}
	}
	schema.AccumulateUsage(s.Usage, src...)
}

// step advances state in place and reports whether the loop should terminate.
// Termination is decided by the step itself (the observe step stops once it has
// a final answer), so runLoop stays free of any Observation knowledge.
type step func(ctx context.Context, s *state) (done bool, err error)

// runLoop drives the reasonAct/observe steps up to maxIter rounds, stopping as
// soon as a step reports done or an error occurs.
func runLoop(ctx context.Context, s *state, maxIter int, steps ...step) error {
	if s == nil || s.Input == nil {
		return errors.New("nil input")
	}
	for i := 0; i < maxIter; i++ {
		s.Iteration = i + 1
		done, err := runIteration(ctx, s, steps)
		if err != nil {
			return err
		}
		if done {
			return nil
		}
	}
	return nil
}

func runIteration(ctx context.Context, s *state, steps []step) (done bool, err error) {
	startedAt := time.Now()
	ctx = emitHook(s.hookManager, ctx, hooks.State{
		Event:         hooks.EventIterationStart,
		InteractionID: s.InteractionID,
		SessionID:     s.Session,
		Iteration:     s.Iteration,
		StartedAt:     startedAt,
	})
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("iteration %d panicked: %v", s.Iteration, recovered)
			emitIterationEnd(s, ctx, startedAt, err)
			panic(recovered)
		}
		emitIterationEnd(s, ctx, startedAt, err)
	}()

	for _, st := range steps {
		done, err = st(ctx, s)
		if err != nil || done {
			return done, err
		}
	}
	return false, nil
}

func emitIterationEnd(s *state, ctx context.Context, startedAt time.Time, err error) {
	emitHook(s.hookManager, ctx, hooks.State{
		Event:         hooks.EventIterationEnd,
		InteractionID: s.InteractionID,
		SessionID:     s.Session,
		Iteration:     s.Iteration,
		Error:         hooks.ErrorMessage(err),
		ErrorType:     hooks.ErrorType(err),
		StartedAt:     startedAt,
		EndedAt:       time.Now(),
	})
}

func emitHook(manager *hooks.Manager, ctx context.Context, state hooks.State) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if manager == nil {
		return ctx
	}
	return manager.Emit(ctx, state)
}

func withHookInput(manager *hooks.Manager, event hooks.Event, toolName string, state hooks.State, provider func() any) hooks.State {
	if manager == nil || !manager.NeedsContent(event, toolName) {
		return state
	}
	return state.WithInputContent(provider)
}

func withHookOutput(manager *hooks.Manager, event hooks.Event, toolName string, state hooks.State, provider func() any) hooks.State {
	if manager == nil || !manager.NeedsContent(event, toolName) {
		return state
	}
	return state.WithOutputContent(provider)
}
