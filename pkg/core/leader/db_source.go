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

package leader

import "gorm.io/gorm"

// DBSource is an interface for components that provide access to a database connection
// Used by leader election to access the shared database for leader lease management
type DBSource interface {
	// GetDB returns the shared database connection and a boolean indicating if a DB is available
	// Returns (db, true) if the component is backed by a database, (nil, false) otherwise
	GetDB() (*gorm.DB, bool)
}
