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

package watchserver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/apache/dubbo-admin/pkg/logger"
	"github.com/apache/dubbo-admin/pkg/traffic/cache"
	"github.com/apache/dubbo-admin/pkg/traffic/client"
	pb2 "github.com/apache/dubbo-admin/pkg/traffic/internal/pb"
	"github.com/apache/dubbo-admin/pkg/traffic/internal/util"
)

// We send ctrl response inside the read loop. We do not want
// send to block read, but we still want ctrl response we sent to
// be serialized. Thus we use a buffered chan to solve the problem.
// A small buffer should be OK for most cases, since we expect the
// ctrl requests are infrequent.
const ctrlStreamBufLen = 16

var WatchSrv *WatchServer

func init() {
	WatchSrv = NewWatchServer()
}

type WatchServer struct {
	Watchable *WatchableStore
}

func NewWatchServer() *WatchServer {
	srv := &WatchServer{
		Watchable: NewWatchableStore(),
	}
	return srv
}

// serverWatchStream is an etcd server side stream. It receives requests
// from client side gRPC stream. It receives watch events from mvcc.WatchStream,
// and creates responses that forwarded to gRPC stream.
// It also forwards control message like watch created and canceled.
type serverWatchStream struct {
	watchable *WatchableStore

	gRPCStream  pb2.Watch_WatchServer
	watchStream WatchStream
	ctrlStream  chan *pb2.WatchResponse

	// mu protects progress, prevKV, fragment
	mu sync.RWMutex

	// closec indicates the stream is closed.
	closec chan struct{}

	// wg waits for the send loop to complete
	wg sync.WaitGroup
}

func (ws *WatchServer) GetRule(ctx context.Context, req *pb2.GetRuleRequest) (*pb2.GetRuleResponse, error) {
	key := req.Key
	config := cache.ConfigMap[key]
	return &pb2.GetRuleResponse{Value: config}, nil
}

func (ws *WatchServer) Watch(stream pb2.Watch_WatchServer) (err error) {
	sws := serverWatchStream{
		gRPCStream:  stream,
		watchStream: ws.Watchable.NewWatchStream(),
		// chan for sending control response like watcher created and canceled.
		ctrlStream: make(chan *pb2.WatchResponse, ctrlStreamBufLen),
		closec:     make(chan struct{}),
		watchable:  ws.Watchable,
	}

	sws.wg.Add(1)
	go func() {
		sws.sendLoop()
		sws.wg.Done()
	}()

	errc := make(chan error, 1)
	// Ideally recvLoop would also use sws.wg to signal its completion
	// but when stream.Context().Done() is closed, the stream's recv
	// may continue to block since it uses a different context, leading to
	// deadlock when calling sws.close().
	go func() {
		if rerr := sws.recvLoop(); rerr != nil {
			if util.IsClientCtxErr(stream.Context().Err(), rerr) {
				logger.Sugar().Debug("failed to receive watch request from gRPC stream", rerr)
			}
			errc <- rerr
		} else {
			logger.Sugar().Warn("failed to receive watch request from gRPC stream", err)
		}
	}()
	select {
	case err = <-errc:
		if err == context.Canceled {
			err = errors.New("grpc watch canceled")
		}
		close(sws.ctrlStream)
	case <-stream.Context().Done():
		err = stream.Context().Err()
		if err == context.Canceled {
			err = errors.New("grpc watch canceled")
		}
	}
	sws.close()
	return err
}

func (sws *serverWatchStream) close() {
	sws.watchStream.Close()
	close(sws.closec)
	sws.wg.Wait()
}

func (sws *serverWatchStream) sendLoop() {
	// watch ids that are currently active
	ids := make(map[WatchID]struct{})
	// watch responses pending on a watch id creation message
	pending := make(map[WatchID][]*pb2.WatchResponse)

	for {
		select {
		case wresp, ok := <-sws.watchStream.Chan():
			if !ok {
				return
			}

			evs := wresp.Events
			events := make([]*pb2.Event, len(evs))
			for i := range evs {
				events[i] = &evs[i]
			}
			wr := &pb2.WatchResponse{
				WatchId: int64(wresp.WatchID),
				Events:  events,
			}

			// Progress notifications can have WatchID -1
			// if they announce on behalf of multiple watchers
			if wresp.WatchID != client.InvalidWatchID {
				if _, okID := ids[wresp.WatchID]; !okID {
					// buffer if id not yet announced
					wrs := append(pending[wresp.WatchID], wr)
					pending[wresp.WatchID] = wrs
					continue
				}
			}

			var serr error

			serr = sws.gRPCStream.Send(wr)
			if serr != nil {
				if util.IsClientCtxErr(sws.gRPCStream.Context().Err(), serr) {
					logger.Sugar().Debug("failed to send watch response to gRPC stream", serr)
				} else {
					logger.Sugar().Warn("failed to send watch response to gRPC stream", serr)
				}
				return
			}
		case c, ok := <-sws.ctrlStream:
			if !ok {
				return
			}
			if err := sws.gRPCStream.Send(c); err != nil {
				if util.IsClientCtxErr(sws.gRPCStream.Context().Err(), err) {
					logger.Sugar().Debug("failed to send watch control response to gRPC stream", err)
				} else {
					logger.Sugar().Warn("failed to send watch control response to gRPC stream", err)
				}
				return
			}

			// track id creation
			wid := WatchID(c.WatchId)
			if !(!(c.Canceled && c.Created) || wid == client.InvalidWatchID) {
				panic(fmt.Sprintf("unexpected watchId: %d, wanted: %d, since both 'Canceled' and 'Created' are true", wid, client.InvalidWatchID))
			}

			if c.Canceled && wid != client.InvalidWatchID {
				delete(ids, wid)
				continue
			}

			if c.Created {
				// flush buffered events
				ids[wid] = struct{}{}
				for _, v := range pending[wid] {
					if err := sws.gRPCStream.Send(v); err != nil {
						if util.IsClientCtxErr(sws.gRPCStream.Context().Err(), err) {
							logger.Sugar().Debug("failed to send pending watch response to gRPC stream", err)
						} else {
							logger.Sugar().Warn("failed to send pending watch response to gRPC stream", err)
						}
						return
					}
				}
				delete(pending, wid)
			}
		case <-sws.closec:
			return
		}
	}
}

func (sws *serverWatchStream) recvLoop() error {
	for {
		req, err := sws.gRPCStream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		switch uv := req.RequestUnion.(type) {
		case *pb2.WatchRequest_CreateRequest:
			if uv.CreateRequest == nil {
				break
			}

			creq := uv.CreateRequest
			if len(creq.Key) == 0 {
				creq.Key = []byte{0}
			}

			id, err := sws.watchStream.Watch(WatchID(creq.WatchId), creq.Key)
			if err != nil {
				id = client.InvalidWatchID
			}
			wr := &pb2.WatchResponse{
				WatchId:  int64(id),
				Created:  true,
				Canceled: err != nil,
			}
			if err != nil {
				wr.CancelReason = err.Error()
			}
			select {
			case sws.ctrlStream <- wr:
			case <-sws.closec:
				return nil
			}
		case *pb2.WatchRequest_CancelRequest:
			if uv.CancelRequest != nil {
				id := uv.CancelRequest.WatchId
				err := sws.watchStream.Cancel(WatchID(id))
				if err == nil {
					sws.ctrlStream <- &pb2.WatchResponse{
						WatchId:  id,
						Canceled: true,
					}
				}
			}
		default:
			// we probably should not shutdown the entire stream when
			// receive an valid command.
			// so just do nothing instead.
			continue
		}
	}
}
