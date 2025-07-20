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

package grpc

import (
	"sync"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type ReverseUnaryMessage interface {
	proto.Message
	GetRequestId() string
}

// ReverseUnaryRPCs helps to implement reverse unary rpcs where server sends requests to a client and receives responses from the client.
type ReverseUnaryRPCs interface {
	Send(client string, req ReverseUnaryMessage) error
	WatchResponse(client string, reqID string, resp chan ReverseUnaryMessage) error
	DeleteWatch(client string, reqID string)

	ClientConnected(client string, stream grpc.ServerStream)
	ClientDisconnected(client string)
	ResponseReceived(client string, resp ReverseUnaryMessage) error
}

type clientStreams struct {
	streamForClient map[string]*clientStream
	sync.Mutex      // protects streamForClient
}

func (x *clientStreams) ResponseReceived(client string, resp ReverseUnaryMessage) error {
	stream, err := x.clientStream(client)
	if err != nil {
		return err
	}
	stream.Lock()
	ch, ok := stream.watchForRequestId[resp.GetRequestId()]
	stream.Unlock()
	if !ok {
		return errors.Errorf("callback for request Id %s not found", resp.GetRequestId())
	}
	ch <- resp
	return nil
}

func NewReverseUnaryRPCs() ReverseUnaryRPCs {
	return &clientStreams{
		streamForClient: map[string]*clientStream{},
	}
}

func (x *clientStreams) ClientConnected(client string, stream grpc.ServerStream) {
	x.Lock()
	defer x.Unlock()
	x.streamForClient[client] = &clientStream{
		stream:            stream,
		watchForRequestId: map[string]chan ReverseUnaryMessage{},
	}
}

func (x *clientStreams) clientStream(client string) (*clientStream, error) {
	x.Lock()
	defer x.Unlock()
	stream, ok := x.streamForClient[client]
	if !ok {
		return nil, errors.Errorf("client %s is not connected", client)
	}
	return stream, nil
}

func (x *clientStreams) ClientDisconnected(client string) {
	x.Lock()
	defer x.Unlock()
	delete(x.streamForClient, client)
}

type clientStream struct {
	stream            grpc.ServerStream
	watchForRequestId map[string]chan ReverseUnaryMessage
	sync.Mutex        // protects watchForRequestId
}

func (x *clientStreams) Send(client string, req ReverseUnaryMessage) error {
	stream, err := x.clientStream(client)
	if err != nil {
		return err
	}
	return stream.stream.SendMsg(req)
}

func (x *clientStreams) WatchResponse(client string, reqID string, resp chan ReverseUnaryMessage) error {
	stream, err := x.clientStream(client)
	if err != nil {
		return err
	}
	stream.Lock()
	defer stream.Unlock()
	stream.watchForRequestId[reqID] = resp
	return nil
}

func (x *clientStreams) DeleteWatch(client string, reqID string) {
	stream, err := x.clientStream(client)
	if err != nil {
		return // client was already deleted
	}
	stream.Lock()
	defer stream.Unlock()
	delete(stream.watchForRequestId, reqID)
}
