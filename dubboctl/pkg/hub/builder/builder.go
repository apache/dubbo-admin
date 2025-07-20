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

package builder

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/sdk/dubbo"
)

const (
	Pack = "pack"
)

type ErrNoDefaultImage struct {
	Builder string
	Runtime string
}

type ErrRuntimeRequired struct {
	Builder string
}

func (e ErrNoDefaultImage) Error() string {
	return fmt.Sprintf("the '%v' runtime defines no default '%v' builder image", e.Runtime, e.Builder)
}

func (e ErrRuntimeRequired) Error() string {
	return fmt.Sprintf("runtime required to choose a default '%v' builder image", e.Builder)
}

func Image(dc *dubbo.DubboConfig, builder string, defaults map[string]string) (string, error) {
	v, ok := dc.Build.BuilderImages[builder]
	if ok {
		return v, nil
	}
	if dc.Runtime == "" {
		return "", ErrRuntimeRequired{Builder: builder}
	}
	v, ok = defaults[dc.Runtime]
	dc.Build.BuilderImages[builder] = v
	if ok {
		return v, nil
	}
	return "", ErrNoDefaultImage{
		Builder: builder,
		Runtime: dc.Runtime,
	}
}
