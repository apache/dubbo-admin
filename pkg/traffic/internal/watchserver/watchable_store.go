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
	"sync"
	"time"

	"github.com/apache/dubbo-admin/pkg/traffic/internal/pb"
)

var (
	// chanBufLen is the length of the buffered chan
	// for sending out watched events.
	chanBufLen = 128
)

// cancelFunc updates synced maps when running
// cancel operations.
type cancelFunc func()

type watchable interface {
	watch(key []byte, id WatchID, ch chan<- WatchResponse) (*watcher, cancelFunc)
}

type WatchableStore struct {
	// mu protects watcher groups and batches. It should never be locked
	// before locking store.mu to avoid deadlock.
	mu sync.RWMutex

	victims []watcherBatch

	victimc chan struct{}

	synced watcherGroup

	stopc chan struct{}
	wg    sync.WaitGroup
}

func (s *WatchableStore) NewWatchStream() WatchStream {
	return &watchStream{
		watchable: s,
		ch:        make(chan WatchResponse, chanBufLen),
		cancels:   make(map[WatchID]cancelFunc),
		watchers:  make(map[WatchID]*watcher),
	}
}

func NewWatchableStore() *WatchableStore {
	s := &WatchableStore{
		victimc: make(chan struct{}, 1),
		synced:  newWatcherGroup(),
		stopc:   make(chan struct{}),
	}
	s.wg.Add(1)
	go s.syncVictimsLoop()
	return s
}

func (s *WatchableStore) watch(key []byte, id WatchID, ch chan<- WatchResponse) (*watcher, cancelFunc) {
	wa := &watcher{
		key: key,
		id:  id,
		ch:  ch,
	}

	s.mu.Lock()
	s.synced.add(wa)
	s.mu.Unlock()

	return wa, func() { s.cancelWatcher(wa) }
}

// cancelWatcher removes references of the watcher from the WatchableStore
func (s *WatchableStore) cancelWatcher(wa *watcher) {
	for {
		s.mu.Lock()
		if s.synced.delete(wa) {
			break
		} else if wa.ch == nil {
			// already canceled (e.g., cancel/close race)
			break
		}
		if !wa.victim {
			panic("watcher not victim but not in watch groups")
		}

		var victimBatch watcherBatch
		for _, wb := range s.victims {
			if wb[wa] != nil {
				victimBatch = wb
				break
			}
		}
		if victimBatch != nil {
			delete(victimBatch, wa)
			break
		}

		s.mu.Unlock()
		time.Sleep(time.Millisecond)
	}
	wa.ch = nil
	s.mu.Unlock()
}

func (s *WatchableStore) syncVictimsLoop() {
	defer s.wg.Done()
	for {
		for s.moveVictims() != 0 {
			// try to update all victim watchers
		}
		s.mu.RLock()
		isEmpty := len(s.victims) == 0
		s.mu.RUnlock()

		var tickc <-chan time.Time
		if !isEmpty {
			tickc = time.After(10 * time.Millisecond)
		}

		select {
		case <-tickc:
		case <-s.victimc:
		case <-s.stopc:
			return
		}
	}
}

// moveVictims tries to update watches with already pending event data
func (s *WatchableStore) moveVictims() (moved int) {
	s.mu.Lock()
	victims := s.victims
	s.victims = nil
	s.mu.Unlock()

	var newVictim watcherBatch
	for _, wb := range victims {
		// try to send responses again
		for w, eb := range wb {
			if !w.send(WatchResponse{WatchID: w.id, Events: eb.evs}) {
				if newVictim == nil {
					newVictim = make(watcherBatch)
				}
				newVictim[w] = eb
				continue
			}
			moved++
		}
		// assign completed victim watchers to sync
		s.mu.Lock()
		for w := range wb {
			if newVictim != nil && newVictim[w] != nil {
				// couldn't send watch response; stays victim
				continue
			}
			w.victim = false
			s.synced.add(w)
		}
		s.mu.Unlock()
	}
	if len(newVictim) > 0 {
		s.mu.Lock()
		s.victims = append(s.victims, newVictim)
		s.mu.Unlock()
	}
	return moved
}

func (s *WatchableStore) addVictim(victim watcherBatch) {
	if victim == nil {
		return
	}
	s.victims = append(s.victims, victim)
	select {
	case s.victimc <- struct{}{}:
	default:
	}
}

// Notify notifies the fact that given event at the given rev just happened to
// watchers that watch on the key of the event.
func (s *WatchableStore) Notify(evs []pb.Event) {
	var victim watcherBatch
	for w, eb := range newWatcherBatch(&s.synced, evs) {
		if !w.send(WatchResponse{WatchID: w.id, Events: eb.evs}) {
			// move slow watcher to victims
			if victim == nil {
				victim = make(watcherBatch)
			}
			w.victim = true
			victim[w] = eb
			s.synced.delete(w)
		}
	}
	s.addVictim(victim)
}

type watcher struct {
	key []byte

	victim bool

	id WatchID

	// a chan to send out the watch response.
	// The chan might be shared with other watchers.
	ch chan<- WatchResponse
}

func (w *watcher) send(wr WatchResponse) bool {
	if len(wr.Events) == 0 {
		return true
	}
	select {
	case w.ch <- wr:
		return true
	default:
		return false
	}
}
