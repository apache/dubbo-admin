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
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/apache/dubbo-admin/pkg/authority/rule/authentication"

	"github.com/apache/dubbo-admin/pkg/authority/rule/authorization"

	"github.com/apache/dubbo-admin/pkg/authority/rule"
	"github.com/apache/dubbo-admin/pkg/authority/rule/connection"
	"github.com/stretchr/testify/assert"
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

func TestStorage_CloseEOF(t *testing.T) {
	t.Parallel()

	storage := connection.NewStorage()
	fake := &fakeConnection{
		recvChan: make(chan recvResult, 1),
	}

	storage.Connected(&rule.Endpoint{
		ID: "test",
	}, fake)

	fake.recvChan <- recvResult{
		request: nil,
		err:     io.EOF,
	}

	assert.Eventually(t, func() bool {
		return fake.disconnected
	}, 10*time.Second, time.Millisecond)

	if len(storage.Connection) != 0 {
		t.Error("expected connection to be removed")
	}
}

func TestStorage_CloseErr(t *testing.T) {
	t.Parallel()

	storage := connection.NewStorage()
	fake := &fakeConnection{
		recvChan: make(chan recvResult, 1),
	}

	storage.Connected(&rule.Endpoint{
		ID: "test",
	}, fake)

	fake.recvChan <- recvResult{
		request: nil,
		err:     fmt.Errorf("test"),
	}

	assert.Eventually(t, func() bool {
		return fake.disconnected
	}, 10*time.Second, time.Millisecond)

	if len(storage.Connection) != 0 {
		t.Error("expected connection to be removed")
	}
}

func TestStorage_UnknownType(t *testing.T) {
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
			Nonce: "",
			Type:  "test",
		},
		err: nil,
	}

	fake.recvChan <- recvResult{
		request: &connection.ObserveRequest{
			Nonce: "",
			Type:  "",
		},
		err: nil,
	}

	conn := storage.Connection[0]

	fake.recvChan <- recvResult{
		request: nil,
		err:     io.EOF,
	}

	assert.Eventually(t, func() bool {
		return fake.disconnected
	}, 10*time.Second, time.Millisecond)

	if len(conn.TypeListened) != 0 {
		t.Error("expected no type listened")
	}
}

func TestStorage_StartNonEmptyNonce(t *testing.T) {
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
			Type:  authorization.RuleType,
		},
		err: nil,
	}

	conn := storage.Connection[0]

	fake.recvChan <- recvResult{
		request: nil,
		err:     io.EOF,
	}

	assert.Eventually(t, func() bool {
		return fake.disconnected
	}, 10*time.Second, time.Millisecond)

	if len(conn.TypeListened) != 0 {
		t.Error("expected no type listened")
	}
}

func TestStorage_Listen(t *testing.T) {
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
			Nonce: "",
			Type:  authorization.RuleType,
		},
		err: nil,
	}

	conn := storage.Connection[0]

	fake.recvChan <- recvResult{
		request: nil,
		err:     io.EOF,
	}

	assert.Eventually(t, func() bool {
		return fake.disconnected
	}, 10*time.Second, time.Millisecond)

	if len(conn.TypeListened) == 0 {
		t.Error("expected type listened")
	}

	if !conn.TypeListened[authorization.RuleType] {
		t.Error("expected type listened")
	}
}

func TestStorage_PreNotify(t *testing.T) {
	t.Parallel()

	storage := connection.NewStorage()

	handler := authorization.NewHandler(storage)
	handler.Add("test", &authorization.Policy{
		Name: "test",
		Spec: &authorization.PolicySpec{
			Action: "allow",
		},
	})

	fake := &fakeConnection{
		recvChan: make(chan recvResult, 1),
	}

	storage.Connected(&rule.Endpoint{
		ID: "test",
	}, fake)

	fake.recvChan <- recvResult{
		request: &connection.ObserveRequest{
			Nonce: "",
			Type:  authorization.RuleType,
		},
		err: nil,
	}

	assert.Eventually(t, func() bool {
		return len(fake.sends) == 1
	}, 10*time.Second, time.Millisecond)

	if fake.sends[0].Type != authorization.RuleType {
		t.Error("expected rule type")
	}

	if fake.sends[0].Nonce == "" {
		t.Error("expected non empty nonce")
	}

	if fake.sends[0].Data == nil {
		t.Error("expected data")
	}

	if fake.sends[0].Data.Type() != authorization.RuleType {
		t.Error("expected rule type")
	}

	if fake.sends[0].Data.Revision() != 1 {
		t.Error("expected revision 1")
	}

	if fake.sends[0].Data.Data() != "[{\"name\":\"test\",\"spec\":{\"action\":\"allow\"}}]" {
		t.Error("expected data [{\"name\":\"test\",\"spec\":{\"action\":\"allow\"}}], got: ", fake.sends[0].Data.Data())
	}

	fake.recvChan <- recvResult{
		request: &connection.ObserveRequest{
			Nonce: fake.sends[0].Nonce,
			Type:  authorization.RuleType,
		},
		err: nil,
	}

	conn := storage.Connection[0]

	assert.Eventually(t, func() bool {
		return conn.ClientRules[authorization.RuleType].PushingStatus == connection.Pushed
	}, 10*time.Second, time.Millisecond)

	fake.recvChan <- recvResult{
		request: nil,
		err:     io.EOF,
	}

	assert.Eventually(t, func() bool {
		return fake.disconnected
	}, 10*time.Second, time.Millisecond)

	if len(conn.TypeListened) == 0 {
		t.Error("expected type listened")
	}

	if !conn.TypeListened[authorization.RuleType] {
		t.Error("expected type listened")
	}
}

func TestStorage_AfterNotify(t *testing.T) {
	t.Parallel()

	storage := connection.NewStorage()

	handler := authorization.NewHandler(storage)

	fake := &fakeConnection{
		recvChan: make(chan recvResult, 1),
	}

	storage.Connected(&rule.Endpoint{
		ID: "test",
	}, fake)

	fake.recvChan <- recvResult{
		request: &connection.ObserveRequest{
			Nonce: "",
			Type:  authorization.RuleType,
		},
		err: nil,
	}

	conn := storage.Connection[0]

	assert.Eventually(t, func() bool {
		return conn.TypeListened[authorization.RuleType]
	}, 10*time.Second, time.Millisecond)

	handler.Add("test", &authorization.Policy{
		Name: "test",
		Spec: &authorization.PolicySpec{
			Action: "allow",
		},
	})

	assert.Eventually(t, func() bool {
		return len(fake.sends) == 1
	}, 10*time.Second, time.Millisecond)

	if fake.sends[0].Type != authorization.RuleType {
		t.Error("expected rule type")
	}

	if fake.sends[0].Nonce == "" {
		t.Error("expected non empty nonce")
	}

	if fake.sends[0].Data == nil {
		t.Error("expected data")
	}

	if fake.sends[0].Data.Type() != authorization.RuleType {
		t.Error("expected rule type")
	}

	if fake.sends[0].Data.Revision() != 1 {
		t.Error("expected revision 1")
	}

	if fake.sends[0].Data.Data() != "[{\"name\":\"test\",\"spec\":{\"action\":\"allow\"}}]" {
		t.Error("expected data [{\"name\":\"test\",\"spec\":{\"action\":\"allow\"}}], got: ", fake.sends[0].Data.Data())
	}

	fake.recvChan <- recvResult{
		request: &connection.ObserveRequest{
			Nonce: fake.sends[0].Nonce,
			Type:  authorization.RuleType,
		},
		err: nil,
	}

	assert.Eventually(t, func() bool {
		return conn.ClientRules[authorization.RuleType].PushingStatus == connection.Pushed
	}, 10*time.Second, time.Millisecond)

	fake.recvChan <- recvResult{
		request: nil,
		err:     io.EOF,
	}

	assert.Eventually(t, func() bool {
		return fake.disconnected
	}, 10*time.Second, time.Millisecond)

	if len(conn.TypeListened) == 0 {
		t.Error("expected type listened")
	}

	if !conn.TypeListened[authorization.RuleType] {
		t.Error("expected type listened")
	}
}

func TestStorage_MissNotify(t *testing.T) {
	t.Parallel()

	storage := connection.NewStorage()

	handler1 := authorization.NewHandler(storage)
	handler2 := authentication.NewHandler(storage)

	fake := &fakeConnection{
		recvChan: make(chan recvResult, 1),
	}

	storage.Connected(&rule.Endpoint{
		ID: "test",
	}, fake)

	fake.recvChan <- recvResult{
		request: &connection.ObserveRequest{
			Nonce: "",
			Type:  authorization.RuleType,
		},
		err: nil,
	}

	conn := storage.Connection[0]

	assert.Eventually(t, func() bool {
		return conn.TypeListened[authorization.RuleType]
	}, 10*time.Second, time.Millisecond)

	handler1.Add("test", &authorization.Policy{
		Name: "test",
		Spec: &authorization.PolicySpec{
			Action: "allow",
		},
	})

	handler2.Add("test", &authentication.Policy{
		Name: "test",
		Spec: &authentication.PolicySpec{
			Action: "allow",
		},
	})

	assert.Eventually(t, func() bool {
		return len(fake.sends) == 1
	}, 10*time.Second, time.Millisecond)

	if fake.sends[0].Type != authorization.RuleType {
		t.Error("expected rule type")
	}

	if fake.sends[0].Nonce == "" {
		t.Error("expected non empty nonce")
	}

	if fake.sends[0].Data == nil {
		t.Error("expected data")
	}

	if fake.sends[0].Data.Type() != authorization.RuleType {
		t.Error("expected rule type")
	}

	if fake.sends[0].Data.Revision() != 1 {
		t.Error("expected revision 1")
	}

	if fake.sends[0].Data.Data() != "[{\"name\":\"test\",\"spec\":{\"action\":\"allow\"}}]" {
		t.Error("expected data [{\"name\":\"test\",\"spec\":{\"action\":\"allow\"}}], got: ", fake.sends[0].Data.Data())
	}

	fake.recvChan <- recvResult{
		request: &connection.ObserveRequest{
			Nonce: fake.sends[0].Nonce,
			Type:  authorization.RuleType,
		},
		err: nil,
	}

	assert.Eventually(t, func() bool {
		return conn.ClientRules[authorization.RuleType].PushingStatus == connection.Pushed
	}, 10*time.Second, time.Millisecond)

	fake.recvChan <- recvResult{
		request: nil,
		err:     io.EOF,
	}

	assert.Eventually(t, func() bool {
		return fake.disconnected
	}, 10*time.Second, time.Millisecond)

	if len(conn.TypeListened) == 0 {
		t.Error("expected type listened")
	}

	if !conn.TypeListened[authorization.RuleType] {
		t.Error("expected type listened")
	}

	if len(fake.sends) != 1 {
		t.Error("expected 1 send")
	}
}

type fakeOrigin struct {
	hash int
}

type errOrigin struct{}

func (e errOrigin) Type() string {
	return authorization.RuleType
}

func (e errOrigin) Revision() int64 {
	return 1
}

func (e errOrigin) Exact(endpoint *rule.Endpoint) (rule.ToClient, error) {
	return nil, fmt.Errorf("test")
}

func (f *fakeOrigin) Type() string {
	return authorization.RuleType
}

func (f *fakeOrigin) Revision() int64 {
	return 1
}

func (f *fakeOrigin) Exact(endpoint *rule.Endpoint) (rule.ToClient, error) {
	return &fakeToClient{}, nil
}

type fakeToClient struct{}

func (f *fakeToClient) Type() string {
	return authorization.RuleType
}

func (f *fakeToClient) Revision() int64 {
	return 1
}

func (f *fakeToClient) Data() string {
	return "data"
}

func TestStorage_MulitiNotify(t *testing.T) {
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
			Nonce: "",
			Type:  authorization.RuleType,
		},
		err: nil,
	}

	conn := storage.Connection[0]

	assert.Eventually(t, func() bool {
		return conn.TypeListened[authorization.RuleType]
	}, 10*time.Second, time.Millisecond)

	// should err
	conn.RawRuleQueue.Add(&fakeToClient{})
	conn.RawRuleQueue.Add(&errOrigin{})

	conn.RawRuleQueue.Add(&fakeOrigin{
		hash: 1,
	})
	conn.RawRuleQueue.Add(&fakeOrigin{
		hash: 2,
	})
	conn.RawRuleQueue.Add(&fakeOrigin{
		hash: 3,
	})

	assert.Eventually(t, func() bool {
		return len(fake.sends) == 1
	}, 10*time.Second, time.Millisecond)

	if fake.sends[0].Type != authorization.RuleType {
		t.Error("expected rule type")
	}

	if fake.sends[0].Nonce == "" {
		t.Error("expected non empty nonce")
	}

	if fake.sends[0].Data == nil {
		t.Error("expected data")
	}

	if fake.sends[0].Data.Type() != authorization.RuleType {
		t.Error("expected rule type")
	}

	if fake.sends[0].Data.Revision() != 1 {
		t.Error("expected revision 1")
	}

	if fake.sends[0].Data.Data() != "data" {
		t.Error("expected data, got: ", fake.sends[0].Data.Data())
	}

	assert.Eventually(t, func() bool {
		return conn.ClientRules[authorization.RuleType].PushQueued
	}, 10*time.Second, time.Millisecond)

	fake.recvChan <- recvResult{
		request: &connection.ObserveRequest{
			Nonce: fake.sends[0].Nonce,
			Type:  authorization.RuleType,
		},
		err: nil,
	}
	assert.Eventually(t, func() bool {
		return conn.ClientRules[authorization.RuleType].PushingStatus == connection.Pushed
	}, 10*time.Second, time.Millisecond)

	assert.Eventually(t, func() bool {
		return conn.RawRuleQueue.Len() == 0
	}, 10*time.Second, time.Millisecond)

	fake.recvChan <- recvResult{
		request: nil,
		err:     io.EOF,
	}

	assert.Eventually(t, func() bool {
		return fake.disconnected
	}, 10*time.Second, time.Millisecond)

	if len(conn.TypeListened) == 0 {
		t.Error("expected type listened")
	}

	if !conn.TypeListened[authorization.RuleType] {
		t.Error("expected type listened")
	}

	if len(fake.sends) != 1 {
		t.Error("expected 1 send")
	}
}

func TestStorage_ReturnMisNonce(t *testing.T) {
	t.Parallel()

	storage := connection.NewStorage()

	handler := authorization.NewHandler(storage)
	handler.Add("test", &authorization.Policy{
		Name: "test",
		Spec: &authorization.PolicySpec{
			Action: "allow",
		},
	})

	fake := &fakeConnection{
		recvChan: make(chan recvResult, 1),
	}

	storage.Connected(&rule.Endpoint{
		ID: "test",
	}, fake)

	fake.recvChan <- recvResult{
		request: &connection.ObserveRequest{
			Nonce: "",
			Type:  authorization.RuleType,
		},
		err: nil,
	}

	assert.Eventually(t, func() bool {
		return len(fake.sends) == 1
	}, 10*time.Second, time.Millisecond)

	if fake.sends[0].Type != authorization.RuleType {
		t.Error("expected rule type")
	}

	if fake.sends[0].Nonce == "" {
		t.Error("expected non empty nonce")
	}

	if fake.sends[0].Data == nil {
		t.Error("expected data")
	}

	if fake.sends[0].Data.Type() != authorization.RuleType {
		t.Error("expected rule type")
	}

	if fake.sends[0].Data.Revision() != 1 {
		t.Error("expected revision 1")
	}

	if fake.sends[0].Data.Data() != "[{\"name\":\"test\",\"spec\":{\"action\":\"allow\"}}]" {
		t.Error("expected data [{\"name\":\"test\",\"spec\":{\"action\":\"allow\"}}], got: ", fake.sends[0].Data.Data())
	}

	fake.recvChan <- recvResult{
		request: &connection.ObserveRequest{
			Nonce: "test",
			Type:  authorization.RuleType,
		},
		err: nil,
	}

	conn := storage.Connection[0]

	fake.recvChan <- recvResult{
		request: nil,
		err:     io.EOF,
	}

	assert.Eventually(t, func() bool {
		return fake.disconnected
	}, 10*time.Second, time.Millisecond)

	if len(conn.TypeListened) == 0 {
		t.Error("expected type listened")
	}

	if !conn.TypeListened[authorization.RuleType] {
		t.Error("expected type listened")
	}

	if conn.ClientRules[authorization.RuleType].PushingStatus == connection.Pushed {
		t.Error("expected not pushed")
	}
}
