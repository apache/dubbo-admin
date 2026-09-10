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
	"time"

	"github.com/firebase/genkit/go/ai"
)

// SessionStore persists Session metadata.
type SessionStore interface {
	Create(ctx context.Context, session *Session) error
	Get(ctx context.Context, sessionID string) (*Session, error)
	List(ctx context.Context) ([]*Session, error)
	Touch(ctx context.Context, sessionID string, updatedAt time.Time) error
	Delete(ctx context.Context, sessionID string) error
	DeleteExpired(ctx context.Context, now time.Time) (int, error)
}

// MessageStore persists conversation Turns and Genkit messages.
type MessageStore interface {
	BeginTurn(ctx context.Context, sessionID string) (uint64, error)
	AddHistoryToTurn(ctx context.Context, sessionID string, turnID uint64, messages ...*ai.Message) error
	IsTurnEmpty(ctx context.Context, sessionID string, turnID uint64) (bool, error)
	WindowMemoryForTurn(ctx context.Context, sessionID string, turnID uint64) ([]*ai.Message, error)
	AllMemory(ctx context.Context, sessionID string) ([]*ai.Message, error)
	NextTurnForTurn(ctx context.Context, sessionID string, turnID uint64) error
	AbortTurnForTurn(ctx context.Context, sessionID string, turnID uint64) error
}

// Store combines the Session and conversation history contracts.
type Store interface {
	SessionStore
	MessageStore
}
