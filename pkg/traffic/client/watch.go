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

package client

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/apache/dubbo-admin/pkg/logger"
	pb2 "github.com/apache/dubbo-admin/pkg/traffic/internal/pb"
)

const (
	closeSendErrTimeout = 250 * time.Millisecond

	// AutoWatchID is the watcher ID passed in WatchStream.Watch when no
	// user-provided ID is available. If pass, an ID will automatically be assigned.
	AutoWatchID = 0

	// InvalidWatchID represents an invalid watch ID and prevents duplication with an existing watch.
	InvalidWatchID = -1
)

type Event pb2.Event

type WatchChan <-chan WatchResponse

type Watcher interface {
	Watch(ctx context.Context, key string) WatchChan

	GetRule(key string) string

	Close() error
}

type watchStreamRequest interface {
	toPB() *pb2.WatchRequest
}

// toPB converts an internal watch request structure to its protobuf WatchRequest structure.
func (wr *watchRequest) toPB() *pb2.WatchRequest {
	req := &pb2.WatchCreateRequest{
		Key: []byte(wr.key),
	}
	cr := &pb2.WatchRequest_CreateRequest{CreateRequest: req}
	return &pb2.WatchRequest{RequestUnion: cr}
}

func streamKeyFromCtx(ctx context.Context) string {
	if md, ok := metadata.FromOutgoingContext(ctx); ok {
		return fmt.Sprintf("%+v", md)
	}
	return ""
}

type watcher struct {
	remote pb2.WatchClient

	callOpts []grpc.CallOption

	mu sync.RWMutex

	streams map[string]*watchGrpcStream
}

type watchGrpcStream struct {
	owner    *watcher
	remote   pb2.WatchClient
	callOpts []grpc.CallOption

	ctx    context.Context
	ctxKey string
	cancel context.CancelFunc

	substreams map[int64]*watcherStream

	resuming []*watcherStream

	reqc     chan watchStreamRequest
	respc    chan *pb2.WatchResponse
	donec    chan struct{}
	errc     chan error
	closingc chan *watcherStream
	wg       sync.WaitGroup

	resumec  chan struct{}
	closeErr error
}

type watcherStream struct {
	initReq watchRequest

	outc    chan WatchResponse
	recvc   chan *WatchResponse
	donec   chan struct{}
	closing bool

	// watcher id
	id int64

	buf []*WatchResponse
}

type watchRequest struct {
	ctx context.Context
	key string

	retc chan chan WatchResponse
}

type WatchResponse struct {
	Events []*Event

	// Canceled is used to indicate watch failure.
	// If the watch failed and the stream was about to close, before the channel is closed,
	// the channel sends a final response that has Canceled set to true with a non-nil Err().
	Canceled bool

	// Created is used to indicate the creation of the watcher.
	Created bool

	closeErr error

	// cancelReason is a reason of canceling watch
	cancelReason string
}

func NewWatcher(c *Client) Watcher {
	return NewWatchFromWatchClient(pb2.NewWatchClient(c.conn), c)
}

func NewWatchFromWatchClient(wc pb2.WatchClient, c *Client) Watcher {
	w := &watcher{
		remote:  wc,
		streams: make(map[string]*watchGrpcStream),
	}
	return w
}

// never closes
var valCtxCh = make(chan struct{})
var zeroTime = time.Unix(0, 0)

// ctx with only the values; never Done
type valCtx struct{ context.Context }

func (vc *valCtx) Deadline() (time.Time, bool) { return zeroTime, false }
func (vc *valCtx) Done() <-chan struct{}       { return valCtxCh }
func (vc *valCtx) Err() error                  { return nil }

// unicastResponse sends a watch response to a specific watch substream.
func (w *watchGrpcStream) unicastResponse(wr *WatchResponse, watchId int64) bool {
	ws, ok := w.substreams[watchId]
	if !ok {
		return false
	}
	select {
	case ws.recvc <- wr:
	case <-ws.donec:
		return false
	}
	return true
}

// broadcastResponse send a watch response to all watch substreams.
func (w *watchGrpcStream) broadcastResponse(wr *WatchResponse) bool {
	for _, ws := range w.substreams {
		select {
		case ws.recvc <- wr:
		case <-ws.donec:
		}
	}
	return true
}

func (w *watchGrpcStream) dispatchEvent(pbresp *pb2.WatchResponse) bool {
	events := make([]*Event, len(pbresp.Events))
	for i, ev := range pbresp.Events {
		events[i] = (*Event)(ev)
	}
	wr := &WatchResponse{
		Events:       events,
		Created:      pbresp.Created,
		Canceled:     pbresp.Canceled,
		cancelReason: pbresp.CancelReason,
	}

	// watch IDs are zero indexed, so request notify watch responses are assigned a watch ID of InvalidWatchID to
	// indicate they should be broadcast.
	if pbresp.WatchId == InvalidWatchID {
		return w.broadcastResponse(wr)
	}

	return w.unicastResponse(wr, pbresp.WatchId)

}

func (w *watchGrpcStream) openWatchClient() (ws pb2.Watch_WatchClient, err error) {
	for {
		select {
		case <-w.ctx.Done():
			if err == nil {
				return nil, w.ctx.Err()
			}
			return nil, err
		default:
		}
		if ws, err = w.remote.Watch(w.ctx, w.callOpts...); ws != nil && err == nil {
			break
		}
	}
	return ws, nil
}

// joinSubstreams waits for all substream goroutines to complete.
func (w *watchGrpcStream) joinSubstreams() {
	for _, ws := range w.substreams {
		<-ws.donec
	}
	for _, ws := range w.resuming {
		if ws != nil {
			<-ws.donec
		}
	}
}

func (w *watchGrpcStream) waitCancelSubstreams(stopc <-chan struct{}) <-chan struct{} {
	var wg sync.WaitGroup
	wg.Add(len(w.resuming))
	donec := make(chan struct{})
	for i := range w.resuming {
		go func(ws *watcherStream) {
			defer wg.Done()
			if ws.closing {
				if ws.initReq.ctx.Err() != nil && ws.outc != nil {
					close(ws.outc)
					ws.outc = nil
				}
				return
			}
			select {
			case <-ws.initReq.ctx.Done():
				// closed ws will be removed from resuming
				ws.closing = true
				close(ws.outc)
				ws.outc = nil
				w.wg.Add(1)
				go func() {
					defer w.wg.Done()
					w.closingc <- ws
				}()
			case <-stopc:
			}
		}(w.resuming[i])
	}
	go func() {
		defer close(donec)
		wg.Wait()
	}()
	return donec
}

func (w *watchGrpcStream) newWatchClient() (pb2.Watch_WatchClient, error) {
	// mark all substreams as resuming
	close(w.resumec)
	w.resumec = make(chan struct{})
	w.joinSubstreams()
	for _, ws := range w.substreams {
		ws.id = InvalidWatchID
		w.resuming = append(w.resuming, ws)
	}
	// strip out nils, if any
	var resuming []*watcherStream
	for _, ws := range w.resuming {
		if ws != nil {
			resuming = append(resuming, ws)
		}
	}
	w.resuming = resuming
	w.substreams = make(map[int64]*watcherStream)

	// connect to grpc stream while accepting watcher cancelation
	stopc := make(chan struct{})
	donec := w.waitCancelSubstreams(stopc)
	wc, err := w.openWatchClient()
	close(stopc)
	<-donec

	// serve all non-closing streams, even if there's a client error
	// so that the teardown path can shutdown the streams as expected.
	for _, ws := range w.resuming {
		if ws.closing {
			continue
		}
		ws.donec = make(chan struct{})
		w.wg.Add(1)
		go w.serveSubstream(ws, w.resumec)
	}

	if err != nil {
		return nil, err
	}

	// receive data from new grpc stream
	go w.serveWatchClient(wc)
	return wc, nil
}

// serveWatchClient forwards messages from the grpc stream to run()
func (w *watchGrpcStream) serveWatchClient(wc pb2.Watch_WatchClient) {
	for {
		resp, err := wc.Recv()
		if err != nil {
			select {
			case w.errc <- err:
			case <-w.donec:
			}
			return
		}
		select {
		case w.respc <- resp:
		case <-w.donec:
			return
		}
	}
}

// serveSubstream forwards watch responses from run() to the subscriber
func (w *watchGrpcStream) serveSubstream(ws *watcherStream, resumec chan struct{}) {
	if ws.closing {
		panic("created substream goroutine but substream is closing")
	}

	resuming := false
	defer func() {
		if !resuming {
			ws.closing = true
		}
		close(ws.donec)
		if !resuming {
			w.closingc <- ws
		}
		w.wg.Done()
	}()

	emptyWr := &WatchResponse{}
	for {
		curWr := emptyWr
		outc := ws.outc

		if len(ws.buf) > 0 {
			curWr = ws.buf[0]
		} else {
			outc = nil
		}
		select {
		case outc <- *curWr:
			if ws.buf[0].Err() != nil {
				return
			}
			ws.buf[0] = nil
			ws.buf = ws.buf[1:]
		case wr, ok := <-ws.recvc:
			if !ok {
				// shutdown from closeSubstream
				return
			}

			if wr.Created {
				if ws.initReq.retc != nil {
					ws.initReq.retc <- ws.outc
					// to prevent next write from taking the slot in buffered channel
					// and posting duplicate create events
					ws.initReq.retc = nil

					// ws.outc <- *wr
				}
			}

			// created event is already sent above,
			// watcher should not post duplicate events
			if wr.Created {
				continue
			}

			ws.buf = append(ws.buf, wr)
		case <-w.ctx.Done():
			return
		case <-ws.initReq.ctx.Done():
			return
		case <-resumec:
			resuming = true
			return
		}
	}
	// lazily send cancel message if events on missing id
}

// Err is the error value if this WatchResponse holds an error.
func (wr *WatchResponse) Err() error {
	switch {
	case wr.closeErr != nil:
		return wr.closeErr
	case wr.Canceled:
		if len(wr.cancelReason) != 0 {
			return errors.New(wr.cancelReason)
		}
		return wr.Err()
	}
	return nil
}

func (w *watcher) Close() (err error) {
	w.mu.Lock()
	streams := w.streams
	w.streams = nil
	w.mu.Unlock()
	for _, wgs := range streams {
		if werr := wgs.close(); werr != nil {
			err = werr
		}
	}
	// Consider context.Canceled as a successful close
	if err == context.Canceled {
		err = nil
	}
	return err
}

func (w *watchGrpcStream) close() (err error) {
	w.cancel()
	<-w.donec
	select {
	case err = <-w.errc:
	default:
	}
	return toErr(w.ctx, err)
}

func (w *watchGrpcStream) addSubstream(resp *pb2.WatchResponse, ws *watcherStream) {
	// check watch ID for backward compatibility (<= v3.3)
	if resp.WatchId == InvalidWatchID || (resp.Canceled && resp.CancelReason != "") {
		w.closeErr = errors.New(resp.CancelReason)
		// failed; no channel
		close(ws.recvc)
		return
	}
	ws.id = resp.WatchId
	w.substreams[ws.id] = ws
}

func (w *watchGrpcStream) sendCloseSubstream(ws *watcherStream, resp *WatchResponse) {
	select {
	case ws.outc <- *resp:
	case <-ws.initReq.ctx.Done():
	case <-time.After(closeSendErrTimeout):
	}
	close(ws.outc)
}

func (w *watchGrpcStream) closeSubstream(ws *watcherStream) {
	// send channel response in case stream was never established
	select {
	case ws.initReq.retc <- ws.outc:
	default:
	}
	// close subscriber's channel
	if closeErr := w.closeErr; closeErr != nil && ws.initReq.ctx.Err() == nil {
		go w.sendCloseSubstream(ws, &WatchResponse{Canceled: true, closeErr: w.closeErr})
	} else if ws.outc != nil {
		close(ws.outc)
	}
	if ws.id != InvalidWatchID {
		delete(w.substreams, ws.id)
		return
	}
	for i := range w.resuming {
		if w.resuming[i] == ws {
			w.resuming[i] = nil
			return
		}
	}
}

func (w *watcher) Watch(ctx context.Context, key string) WatchChan {
	wr := &watchRequest{
		ctx:  ctx,
		key:  key,
		retc: make(chan chan WatchResponse, 1),
	}

	ok := false
	ctxKey := streamKeyFromCtx(ctx)
	log.Println(ctxKey)

	var closeCh chan WatchResponse
	for {
		// find or allocate appropriate grpc watch stream
		w.mu.Lock()
		if w.streams == nil {
			// closed
			w.mu.Unlock()
			ch := make(chan WatchResponse)
			close(ch)
			return ch
		}
		wgs := w.streams[ctxKey]
		if wgs == nil {
			wgs = w.newWatcherGrpcStream(ctx)
			w.streams[ctxKey] = wgs
		}
		donec := wgs.donec
		reqc := wgs.reqc
		w.mu.Unlock()

		// couldn't create channel; return closed channel
		if closeCh == nil {
			closeCh = make(chan WatchResponse, 1)
		}

		// submit request
		select {
		case reqc <- wr:
			ok = true
		case <-wr.ctx.Done():
			ok = false
		case <-donec:
			ok = false
			if wgs.closeErr != nil {
				closeCh <- WatchResponse{Canceled: true, closeErr: wgs.closeErr}
				break
			}
			// retry; may have dropped stream from no ctxs
			continue
		}

		// receive channel
		if ok {
			select {
			case ret := <-wr.retc:
				return ret
			case <-ctx.Done():
			case <-donec:
				if wgs.closeErr != nil {
					closeCh <- WatchResponse{Canceled: true, closeErr: wgs.closeErr}
					break
				}
				// retry; may have dropped stream from no ctxs
				continue
			}
		}
		break
	}

	close(closeCh)
	return closeCh
}

func (w *watcher) GetRule(key string) string {
	rule, err := w.remote.GetRule(context.Background(), &pb2.GetRuleRequest{Key: key})
	if err != nil {
		panic("get rule failed")
	}
	return rule.Value
}

func (w *watcher) newWatcherGrpcStream(inctx context.Context) *watchGrpcStream {
	ctx, cancel := context.WithCancel(&valCtx{inctx})
	wgs := &watchGrpcStream{
		owner:      w,
		remote:     w.remote,
		callOpts:   w.callOpts,
		ctx:        ctx,
		ctxKey:     streamKeyFromCtx(inctx),
		cancel:     cancel,
		substreams: make(map[int64]*watcherStream),
		respc:      make(chan *pb2.WatchResponse),
		reqc:       make(chan watchStreamRequest),
		donec:      make(chan struct{}),
		errc:       make(chan error, 1),
		closingc:   make(chan *watcherStream),
		resumec:    make(chan struct{}),
	}
	go wgs.run()
	return wgs
}

func (w *watcher) closeStream(wgs *watchGrpcStream) {
	w.mu.Lock()
	close(wgs.donec)
	wgs.cancel()
	if w.streams != nil {
		delete(w.streams, wgs.ctxKey)
	}
	w.mu.Unlock()
}

func (w *watchGrpcStream) run() {
	var wc pb2.Watch_WatchClient
	var closeErr error

	// substreams marked to close but goroutine still running; needed for
	// avoiding double-closing recvc on grpc stream teardown
	closing := make(map[*watcherStream]struct{})

	defer func() {
		w.closeErr = closeErr
		for _, ws := range w.substreams {
			if _, ok := closing[ws]; !ok {
				close(ws.recvc)
				closing[ws] = struct{}{}
			}
		}
		for _, ws := range w.resuming {
			if _, ok := closing[ws]; ws != nil && !ok {
				close(ws.recvc)
				closing[ws] = struct{}{}
			}
		}
		w.joinSubstreams()
		for range closing {
			w.closeSubstream(<-w.closingc)
		}
		w.wg.Wait()
		w.owner.closeStream(w)
	}()

	if wc, closeErr = w.newWatchClient(); closeErr != nil {
		return
	}

	cancelSet := make(map[int64]struct{})

	var cur *pb2.WatchResponse
	for {
		select {
		case req := <-w.reqc:
			switch wreq := req.(type) {
			case *watchRequest:
				outc := make(chan WatchResponse, 1)
				ws := &watcherStream{
					initReq: *wreq,
					id:      InvalidWatchID,
					outc:    outc,
					// unbuffered so resumes won't cause repeat events
					recvc: make(chan *WatchResponse),
				}

				ws.donec = make(chan struct{})
				w.wg.Add(1)
				go w.serveSubstream(ws, w.resumec)

				// queue up for watcher creation/resume
				w.resuming = append(w.resuming, ws)
				if len(w.resuming) == 1 {
					// head of resume queue, can register a new watcher
					if err := wc.Send(ws.initReq.toPB()); err != nil {
						logger.Sugar().Debug("error when sending request", err)
					}
				}
			}
		case pbresp := <-w.respc:
			if cur == nil || pbresp.Created || pbresp.Canceled {
				cur = pbresp
			} else if cur != nil && cur.WatchId == pbresp.WatchId {
				cur.Events = append(cur.Events, pbresp.Events...)
			}

			switch {
			case pbresp.Created:
				// 把事件分配给对应的watch subStream
				if ws := w.resuming[0]; ws != nil {
					w.addSubstream(pbresp, ws)
					w.dispatchEvent(pbresp)
					w.resuming[0] = nil
				}

				if ws := w.nextResume(); ws != nil {
					if err := wc.Send(ws.initReq.toPB()); err != nil {
						logger.Sugar().Debug("error when sending request", err)
					}
				}

				// reset for next iteration
				cur = nil

			case pbresp.Canceled:
				delete(cancelSet, pbresp.WatchId)
				if ws, ok := w.substreams[pbresp.WatchId]; ok {
					// signal to stream goroutine to update closingc
					close(ws.recvc)
					closing[ws] = struct{}{}
				}

				// reset for next iteration
				cur = nil

			default:
				// dispatch to appropriate watch stream
				ok := w.dispatchEvent(cur)

				// reset for next iteration
				cur = nil

				if ok {
					break
				}

				if _, ok := cancelSet[pbresp.WatchId]; ok {
					break
				}

				cancelSet[pbresp.WatchId] = struct{}{}
				cr := &pb2.WatchRequest_CancelRequest{
					CancelRequest: &pb2.WatchCancelRequest{
						WatchId: pbresp.WatchId,
					},
				}
				req := &pb2.WatchRequest{RequestUnion: cr}
				if err := wc.Send(req); err != nil {
					logger.Sugar().Debug("failed to send watch cancel request", pbresp.WatchId)

				}
			}

		// watch client failed on Recv; spawn another if possible
		case err := <-w.errc:
			if isHaltErr(w.ctx, err) {
				closeErr = err
				return
			}
			if wc, closeErr = w.newWatchClient(); closeErr != nil {
				return
			}
			if ws := w.nextResume(); ws != nil {
				if err := wc.Send(ws.initReq.toPB()); err != nil {
					logger.Sugar().Debug("error when sending request", err)
				}
			}
			cancelSet = make(map[int64]struct{})

		case <-w.ctx.Done():
			return

		case ws := <-w.closingc:
			w.closeSubstream(ws)
			delete(closing, ws)
			// no more watchers on this stream, shutdown, skip cancellation
			if len(w.substreams)+len(w.resuming) == 0 {
				return
			}
			if ws.id != InvalidWatchID {
				// client is closing an established watch; close it on the server proactively instead of waiting
				// to close when the next message arrives
				cancelSet[ws.id] = struct{}{}
				cr := &pb2.WatchRequest_CancelRequest{
					CancelRequest: &pb2.WatchCancelRequest{
						WatchId: ws.id,
					},
				}
				req := &pb2.WatchRequest{RequestUnion: cr}
				if err := wc.Send(req); err != nil {
					logger.Sugar().Debug("failed to send watch cancel request", ws.id)
				}
			}
		}
	}
}

// nextResume chooses the next resuming to register with the grpc stream. Abandoned
// streams are marked as nil in the queue since the head must wait for its inflight registration.
func (w *watchGrpcStream) nextResume() *watcherStream {
	for len(w.resuming) != 0 {
		if w.resuming[0] != nil {
			return w.resuming[0]
		}
		w.resuming = w.resuming[1:len(w.resuming)]
	}
	return nil
}
