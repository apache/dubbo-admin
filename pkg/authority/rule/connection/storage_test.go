// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package connection_test

import (
	"github.com/apache/dubbo-admin/pkg/authority/rule"
	"github.com/apache/dubbo-admin/pkg/authority/rule/connection"
	"github.com/stretchr/testify/assert"
	"io"
	"testing"
	"time"
)

type fakeConnection struct {
	sends        []*connection.ObserveResponse
	recvChan     chan recvResult
	disconnected bool
}

type recvResult struct {
	request *connection.ObserveRequest
	err     error
}

func (f *fakeConnection) Send(response *connection.ObserveResponse) error {
	f.sends = append(f.sends, response)
	return nil
}

func (f *fakeConnection) Recv() (*connection.ObserveRequest, error) {
	request := <-f.recvChan
	return request.request, request.err
}

func (f *fakeConnection) Disconnect() {
	f.disconnected = true
}

func TestStorage_Connected(t *testing.T) {
	t.Parallel()

	storage := connection.NewStorage()

	fake := &fakeConnection{
		recvChan: make(chan recvResult, 1),
	}

	storage.Connected(&rule.Endpoint{
		ID: "test",
	}, fake)

	fake.recvChan <- recvResult{
		request: &connection.ObserveRequest{
			Nonce: "test",
			Type:  "test",
		},
		err: nil,
	}

	fake.recvChan <- recvResult{
		request: nil,
		err:     io.EOF,
	}

	assert.Eventually(t, func() bool {
		return fake.disconnected
	}, 10*time.Second, time.Millisecond)
}
