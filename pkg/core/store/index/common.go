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

import (
	"reflect"

	"github.com/duke-git/lancet/v2/slice"
	"k8s.io/client-go/tools/cache"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

const ByMeshIndex = "idx_mesh"

func init() {
	rks := coremodel.ResourceSchemaRegistry().AllResourceKinds()
	slice.ForEach(rks, func(_ int, rk coremodel.ResourceKind) {
		RegisterIndexers(rk, map[string]cache.IndexFunc{
			ByMeshIndex: ByMesh,
		})
	})
}

func ByMesh(obj interface{}) ([]string, error) {
	r, ok := obj.(coremodel.Resource)
	if !ok {
		return nil, bizerror.NewAssertionError("Resource", reflect.TypeOf(obj).Name())
	}
	return []string{r.MeshName()}, nil
}
