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

package proto_test

import (
	"time"

	envoy_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/protobuf/types/known/durationpb"

	util_proto "github.com/apache/dubbo-admin/pkg/common/util/proto"
)

var _ = Describe("MergeDubbo", func() {
	It("should merge durations by replacing them", func() {
		dest := &envoy_cluster.Cluster{
			Name:           "old",
			ConnectTimeout: durationpb.New(time.Second * 10),
			EdsClusterConfig: &envoy_cluster.Cluster_EdsClusterConfig{
				ServiceName: "srv",
				EdsConfig: &envoy_config_core_v3.ConfigSource{
					InitialFetchTimeout: durationpb.New(time.Millisecond * 100),
				},
			},
		}
		src := &envoy_cluster.Cluster{
			Name:           "new",
			ConnectTimeout: durationpb.New(time.Millisecond * 500),
			EdsClusterConfig: &envoy_cluster.Cluster_EdsClusterConfig{
				EdsConfig: &envoy_config_core_v3.ConfigSource{
					InitialFetchTimeout: durationpb.New(time.Second),
					ResourceApiVersion:  envoy_config_core_v3.ApiVersion_V3,
				},
			},
		}
		util_proto.Merge(dest, src)
		Expect(dest.ConnectTimeout.AsDuration()).To(Equal(time.Millisecond * 500))
		Expect(dest.Name).To(Equal("new"))
		Expect(dest.EdsClusterConfig.ServiceName).To(Equal("srv"))
		Expect(dest.EdsClusterConfig.EdsConfig.InitialFetchTimeout.AsDuration()).To(Equal(time.Second))
		Expect(dest.EdsClusterConfig.EdsConfig.InitialFetchTimeout.AsDuration()).To(Equal(time.Second))
		Expect(dest.EdsClusterConfig.EdsConfig.ResourceApiVersion).To(Equal(envoy_config_core_v3.ApiVersion_V3))
	})
})
