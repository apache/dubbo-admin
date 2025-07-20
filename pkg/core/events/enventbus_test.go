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

package events_test

import (
	. "github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"

	"github.com/apache/dubbo-admin/pkg/core/events"
)

var _ = Describe("EventBus", func() {
	chHadEvent := func(ch <-chan events.Event) bool {
		select {
		case <-ch:
			return true
		default:
			return false
		}
	}

	It("should not block on Send", func() {
		// given
		eventBus, err := events.NewEventBus(1)
		Expect(err).ToNot(HaveOccurred())
		listener := eventBus.Subscribe()
		event1 := events.ResourceChangedEvent{TenantID: "1"}
		event2 := events.ResourceChangedEvent{TenantID: "2"}

		// when
		eventBus.Send(event1)
		eventBus.Send(event2)

		// then
		event := <-listener.Recv()
		Expect(event).To(Equal(event1))

		// and second event was ignored because buffer was full
		Expect(chHadEvent(listener.Recv())).To(BeFalse())
	})

	It("should only send events matched predicate", func() {
		// given
		eventBus, err := events.NewEventBus(10)
		Expect(err).ToNot(HaveOccurred())
		listener := eventBus.Subscribe(func(event events.Event) bool {
			return event.(events.ResourceChangedEvent).TenantID == "1"
		})
		event1 := events.ResourceChangedEvent{TenantID: "1"}
		event2 := events.ResourceChangedEvent{TenantID: "2"}

		// when
		eventBus.Send(event1)
		eventBus.Send(event2)

		// then
		event := <-listener.Recv()
		Expect(event).To(Equal(event1))

		// and second event was ignored, because it did not match predicate
		Expect(chHadEvent(listener.Recv())).To(BeFalse())
	})
})
