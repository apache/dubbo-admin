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

package controller

import (
	"k8s.io/client-go/tools/cache"

	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

type ResourceListerWatcher interface {
	cache.ListerWatcher
	// ResourceKind returns the kind of resource this listerwatcher is for
	ResourceKind() coremodel.ResourceKind
	// TransformFunc transform the raw resource into your need before the raw resource pushing into the delta fifo,
	// return nil if there is no need to transform, see cache.SharedInformer for detail
	TransformFunc() cache.TransformFunc
}
