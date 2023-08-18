# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

#! /bin/bash

echo "Running Resource-gen collections"
go run ./resource-gen/cmd/cmd.go collections ./resource-gen/metadata.yaml ../pkg/core/schema/collections/collections.gen.go

echo "Sleep for a short time to wait collections.gen.go update"
sleep 5

echo "Running Resource-gen gvk"
go run ./resource-gen/cmd/cmd.go gvk ./resource-gen/metadata.yaml ../pkg/core/schema/gvk/gvk.gen.go

echo "Running Types-gen"
go run ./types-gen/main.go --template ./types-gen/types.go.tmpl --output ../pkg/dds/kube/crdclient/types.gen.go

echo "Running Resource Deepcopy"
go run ./deepcopy-gen/generate.go --output ../api/resource/v1alpha1/resource_deepcopy.go --template ./deepcopy-gen/template.go.tmpl

echo "Running Code-generator-gen"
go run ./code-generator-gen/main.go --ot ../pkg/core/gen/apis/dubbo.apache.org/v1alpha1/types.go --or ../pkg/core/gen/apis/dubbo.apache.org/v1alpha1/register.go --tt ./code-generator-gen/typesgen.go.tmpl --tr ./code-generator-gen/register.go.tmpl

echo "Running code-generator to gen deepcopy and generated"
