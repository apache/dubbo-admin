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

package gorm

import (
	"time"
)

// LockRecord represents a distributed lock record in the database
type LockRecord struct {
	ID        uint      `gorm:"primarykey"`
	LockKey   string    `gorm:"uniqueIndex;size:255;not null"` // Unique lock identifier
	Owner     string    `gorm:"size:255;not null"`             // UUID of the lock holder
	ExpireAt  time.Time `gorm:"index;not null"`                // Lock expiration time
	CreatedAt time.Time `gorm:"autoCreateTime"`                // Lock creation time
	UpdatedAt time.Time `gorm:"autoUpdateTime"`                // Last renewal time
}

// TableName returns the table name for LockRecord
func (LockRecord) TableName() string {
	return "distributed_locks"
}
