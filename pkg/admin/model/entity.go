// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package model

import (
	"reflect"
	"time"
)

type Entity struct {
	Id              int64     `json:"id"`
	Ids             []int64   `json:"ids"`
	Hash            string    `json:"hash"`
	Created         time.Time `json:"created"`
	Modified        time.Time `json:"modified"`
	Now             time.Time `json:"now"`
	Operator        string    `json:"operator"`
	OperatorAddress string    `json:"operatorAddress"`
	Miss            bool      `json:"miss"`
}

func NewEntity(id int64) Entity {
	return Entity{
		Id: id,
	}
}

func (e *Entity) SetOperator(operator string) {
	if len(operator) > 200 {
		operator = operator[:200]
	}
	e.Operator = operator
}

func (e *Entity) Equals(other *Entity) bool {
	return reflect.DeepEqual(e, other)
}
