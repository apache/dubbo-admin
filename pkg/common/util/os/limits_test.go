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

package os

import (
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/sys/unix"
)

var _ = Describe("File limits", func() {
	It("should query the open file limit", func() {
		Expect(CurrentFileLimit()).Should(BeNumerically(">", 0))
	})

	It("should raise the open file limit", func() {
		if runtime.GOOS == "darwin" {
			Skip("skipping on darwin because it requires priviledges")
		}
		initialLimits := unix.Rlimit{}
		Expect(unix.Getrlimit(unix.RLIMIT_NOFILE, &initialLimits)).Should(Succeed())

		Expect(CurrentFileLimit()).Should(BeNumerically("==", initialLimits.Cur))

		Expect(RaiseFileLimit()).Should(Succeed())

		Expect(CurrentFileLimit()).Should(BeNumerically("==", initialLimits.Max))

		// Restore the original limit.
		Expect(setFileLimit(initialLimits.Cur)).Should(Succeed())
		Expect(CurrentFileLimit()).Should(BeNumerically("==", initialLimits.Cur))
	})

	It("should fail to exceed the hard file limit", func() {
		Expect(setFileLimit(uint64(1) << 63)).Should(HaveOccurred())
	})
})
