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

package agent

import (
	"context"
	"sync"

	"dubbo-admin-ai/schema"
)

type Agent interface {
	Interact(context.Context, *schema.UserInput, string) *Channels
}

// Channels carries streaming output back to the caller for one interaction.
// UserRespChan streams user-facing text/progress and the final answer;
// ErrorChan surfaces failures. A fresh Channels is created per interaction.
type Channels struct {
	closeOnce sync.Once
	done      chan struct{}
	nextIndex int

	UserRespChan chan *schema.StreamFeedback
	ErrorChan    chan error
}

func NewChannels(bufferSize int) *Channels {
	return &Channels{
		done:         make(chan struct{}),
		UserRespChan: make(chan *schema.StreamFeedback, bufferSize),
		ErrorChan:    make(chan error, bufferSize),
	}
}

// Close marks the Channels as finished without tearing down the underlying
// channels, so the consumer can drain any buffered messages.
func (chans *Channels) Close() {
	if chans == nil {
		return
	}
	chans.closeOnce.Do(func() { close(chans.done) })
}

func (chans *Channels) Closed() bool {
	if chans == nil {
		return true
	}
	select {
	case <-chans.done:
		return true
	default:
		return false
	}
}

// Done returns a channel that is closed when the interaction has finished.
// The output channels remain open so consumers can drain buffered events.
func (chans *Channels) Done() <-chan struct{} {
	if chans == nil {
		return nil
	}
	return chans.done
}

// Send assigns the next content-block index to the feedback and forwards it to
// the consumer. Sends for one interaction run sequentially (the strategy drives
// its steps on a single goroutine), so the counter needs no locking and never
// bleeds across sessions.
func (chans *Channels) Send(sf *schema.StreamFeedback) {
	sf.SetIndex(chans.nextIndex)
	chans.nextIndex++
	chans.UserRespChan <- sf
}

// EmitProgress streams a pre-rendered progress line to the consumer. A nil
// Channels is a no-op, so steps can run without a consumer (e.g. tests). The
// wording is owned by each strategy; this only handles delivery.
func EmitProgress(chans *Channels, text string) {
	if chans == nil {
		return
	}
	chans.Send(schema.NewStreamFeedback(text))
}
