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

package net_test

import (
	. "github.com/onsi/ginkgo/v2"

	"github.com/apache/dubbo-admin/pkg/common/util/net"
	. "github.com/onsi/gomega"
)

var _ = DescribeTable("ToV6",
	func(given string, expected string) {
		Expect(net.ToV6(given)).To(Equal(expected))
	},
	Entry("v6 already", "2001:db8::ff00:42:8329", "2001:db8::ff00:42:8329"),
	Entry("v6 not compacted", "2001:0db8:0000:0000:0000:ff00:0042:8329", "2001:0db8:0000:0000:0000:ff00:0042:8329"),
	Entry("v4 adds prefix", "240.0.0.0", "::ffff:f000:0"),
	Entry("v4 adds prefix", "240.0.255.0", "::ffff:f000:ff00"),
)

var _ = DescribeTable("IsIPv6",
	func(given string, expected bool) {
		Expect(net.IsAddressIPv6(given)).To(Equal(expected))
	},
	Entry("127.0.0.1 should not be IPv6 ", "127.0.0.1", false),
	Entry("should be IPv6", "2001:0db8:0000:0000:0000:ff00:0042:8329", true),
	Entry("::6", "::6", true),
)
