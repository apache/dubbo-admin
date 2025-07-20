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

package sdk

import (
	"context"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/sdk/dubbo"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/util"
	"path"
)

type template struct {
	name       string
	runtime    string
	repository string
	fs         util.Filesystem
}

type Template interface {
	Name() string
	Fullname() string
	Runtime() string
	Write(ctx context.Context, f *dubbo.DubboConfig) error
}

func (t template) Name() string {
	return t.name
}

func (t template) Fullname() string {
	return t.repository + "/" + t.name
}

func (t template) Runtime() string {
	return t.runtime
}

func (t template) Write(ctx context.Context, dc *dubbo.DubboConfig) error {
	mask := func(p string) bool {
		_, f := path.Split(p)
		return f == "template.yaml"
	}

	return util.CopyFromFS(".", dc.Root, util.NewMaskingFS(mask, t.fs))
}
