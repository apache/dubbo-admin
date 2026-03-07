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

package index

// IndexOperator defines the comparison operator for index queries
type IndexOperator string

const (
	// Equals performs exact match on the index value
	Equals IndexOperator = "Equals"
	// HasPrefix performs prefix match on the index value
	HasPrefix IndexOperator = "HasPrefix"
)

// IndexCondition represents a single index query condition
type IndexCondition struct {
	// IndexName is the name of the index to query
	IndexName string
	// Value is the value to match against the index
	Value string
	// Operator is the comparison operator to use (Equals, HasPrefix, etc.)
	Operator IndexOperator
}
