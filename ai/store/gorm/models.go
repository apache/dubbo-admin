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

package gormstore

import "time"

// SessionModel is the database representation of a conversation session.
// Associations are intentionally represented by ordinary indexed columns;
// cleanup is performed explicitly by the Store without database foreign keys.
type SessionModel struct {
	ID        string    `gorm:"primaryKey;size:64"`
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null;index"`
	Status    string    `gorm:"size:16;not null;index"`
}

func (SessionModel) TableName() string { return "ai_sessions" }

// TurnModel is one persisted conversation turn. A nil CompletedAt identifies
// the current active turn for a session.
type TurnModel struct {
	ID          uint64     `gorm:"primaryKey;autoIncrement"`
	SessionID   string     `gorm:"size:64;not null;index"`
	CreatedAt   time.Time  `gorm:"not null;index"`
	CompletedAt *time.Time `gorm:"index"`
}

func (TurnModel) TableName() string { return "ai_turns" }

// MessageModel stores the complete Genkit message as JSON. Sequence is
// scoped to a turn and provides deterministic message ordering.
type MessageModel struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement"`
	TurnID    uint64    `gorm:"not null;uniqueIndex:uidx_ai_message_turn_sequence"`
	Sequence  uint64    `gorm:"not null;uniqueIndex:uidx_ai_message_turn_sequence"`
	Payload   []byte    `gorm:"not null"`
	CreatedAt time.Time `gorm:"not null"`
}

func (MessageModel) TableName() string { return "ai_messages" }
