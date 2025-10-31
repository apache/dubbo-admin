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

package db

import (
	"encoding/json"
	"time"

	"github.com/apache/dubbo-admin/pkg/core/resource/model"
)

type ResourceModel struct {
	ID           uint      `gorm:"primarykey"`
	ResourceKey  string    `gorm:"uniqueIndex;not null"`
	ResourceKind string    `gorm:"index;not null"`
	Name         string    `gorm:"index;not null"`
	Mesh         string    `gorm:"index;not null"`
	Data         []byte    `gorm:"type:text;not null"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`
}

func (*ResourceModel) TableName() string {
	return "resources"
}

func (rm *ResourceModel) ToResource() (model.Resource, error) {
	newFunc, err := model.ResourceSchemaRegistry().NewResourceFunc(model.ResourceKind(rm.ResourceKind))
	if err != nil {
		return nil, err
	}
	resource := newFunc()
	if err := json.Unmarshal(rm.Data, resource); err != nil {
		return nil, err
	}
	return resource, nil
}

func FromResource(resource model.Resource) (*ResourceModel, error) {
	data, err := json.Marshal(resource)
	if err != nil {
		return nil, err
	}

	return &ResourceModel{
		ResourceKey:  resource.ResourceKey(),
		ResourceKind: resource.ResourceKind().ToString(),
		Name:         resource.ResourceMeta().Name,
		Mesh:         resource.MeshName(),
		Data:         data,
	}, nil
}
