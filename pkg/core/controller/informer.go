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

package controller

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/go-logr/zapr"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/core/events"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
)

// Informer is transferred from cache.SharedInformer, and modified to support event distribution in events.EventBus
type Informer interface {
	// Run starts and runs the shared informer, returning after it stops.
	// The informer will be stopped when stopCh is closed.
	Run(stopCh <-chan struct{})
	// IsStopped reports whether the informer has already been stopped.
	// Adding event handlers to already stopped informers is not possible.
	// An informer already stopped will never be started again.
	IsStopped() bool

	// SetTransform The TransformFunc is called for each object which is about to be stored.
	//
	// This function is intended for you to take the opportunity to
	// remove, transform, or normalize fields. One use case is to strip unused
	// metadata fields out of objects to save on RAM cost.
	//
	// Must be set before starting the informer.
	//
	// Please see the comment on TransformFunc for more details.
	SetTransform(handler cache.TransformFunc) error
}

// Options configures an informer.
type Options struct {
	// ResyncPeriod is the default event handler resync period and resync check
	// period. If unset/unspecified, these are defaulted to 0 (do not resync).
	ResyncPeriod time.Duration
}

// informer implements Informer and has three
// main components.  One is the cache.Indexer which provides curd operations for objects.
// The second main component is a cache.Controller that pulls
// objects/notifications using the ListerWatcher and pushes them into
// a cache.DeltaFIFO --- whose knownObjects is the informer's indexer
// --- while concurrently Popping Deltas values from that fifo and
// processing them with informer.HandleDeltas.  Each
// invocation of HandleDeltas, which is done with the fifo's lock
// held, processes each Delta in turn.  For each cache.Delta this both
// updates the store and emit the event to the events.EventBus
// The third main component is emitter, which is responsible for
// event distribution
type informer struct {
	// see store.ResourceStore
	indexer cache.Indexer
	// controller is the underlying cache.Controller that pop cache.Delta from the fifo queue
	controller cache.Controller
	// listerWatcher is where we got our initial list of objects and where we perform a watch from.
	listerWatcher cache.ListerWatcher
	// emitter is used to emit events to events.EventBus
	emitter events.Emitter
	// objectType is an example object of the type this informer is expected to handle. If set, an event
	// with an object with a mismatching type is dropped instead of being delivered to listeners.
	objectType runtime.Object
	// resyncCheckPeriod is how often we want the reflector's resync timer to fire so it can call
	// ShouldResync to check if any of our listeners need a resync.
	resyncCheckPeriod time.Duration

	started, stopped bool
	startedLock      sync.Mutex
	// blockDeltas gives a way to stop all event distribution so that a late event handler
	// can safely join the shared informer.
	blockDeltas sync.Mutex
	// Called whenever the ListAndWatch drops the connection with an error.
	watchErrorHandler cache.WatchErrorHandler
	// transform is an optional function that is called on each object before it is pushed into the queue.
	transform cache.TransformFunc
	// keyFunc see cache.KeyFunc
	keyFunc cache.KeyFunc
}

func NewInformerWithOptions(
	lw cache.ListerWatcher,
	emitter events.Emitter,
	store store.ResourceStore,
	keyFunc cache.KeyFunc,
	options Options) Informer {
	return &informer{
		indexer:           store,
		listerWatcher:     lw,
		emitter:           emitter,
		keyFunc:           keyFunc,
		resyncCheckPeriod: options.ResyncPeriod,
	}
}

func (s *informer) SetWatchErrorHandler(handler cache.WatchErrorHandler) error {
	s.startedLock.Lock()
	defer s.startedLock.Unlock()

	if s.started {
		return fmt.Errorf("informer has already started")
	}

	s.watchErrorHandler = handler
	return nil
}

func (s *informer) SetTransform(handler cache.TransformFunc) error {
	s.startedLock.Lock()
	defer s.startedLock.Unlock()

	if s.started {
		return fmt.Errorf("informer has already started")
	}

	s.transform = handler
	return nil
}

func (s *informer) SetObjectType(objectType runtime.Object) error {
	s.startedLock.Lock()
	defer s.startedLock.Unlock()

	if s.started {
		return fmt.Errorf("informer has already started")
	}
	s.objectType = objectType
	return nil
}

func (s *informer) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer func() {
		s.startedLock.Lock()
		defer s.startedLock.Unlock()
		s.stopped = true // Don't want any new listeners
	}()

	if s.HasStarted() {
		klog.Warningf("The informer has started, run more than once is not allowed")
		return
	}

	func() {
		s.startedLock.Lock()
		defer s.startedLock.Unlock()
		zapLogr := zapr.NewLogger(logger.Logger())
		fifo := cache.NewDeltaFIFOWithOptions(cache.DeltaFIFOOptions{
			KnownObjects:          s.indexer,
			EmitDeltaTypeReplaced: true,
			Transformer:           s.transform,
			Logger:                &zapLogr,
			KeyFunction:           s.keyFunc,
		})

		// We turn off the resync mechanism because we don't want to re-list all objects.
		cfg := &cache.Config{
			Queue:             fifo,
			ListerWatcher:     s.listerWatcher,
			ObjectType:        s.objectType,
			FullResyncPeriod:  s.resyncCheckPeriod,
			ShouldResync:      s.ShouldResync,
			Process:           s.HandleDeltas,
			WatchErrorHandler: s.watchErrorHandler,
		}

		s.controller = cache.New(cfg)
		s.started = true
	}()

	s.controller.Run(stopCh)
}

func (s *informer) HasStarted() bool {
	s.startedLock.Lock()
	defer s.startedLock.Unlock()
	return s.started
}

// ShouldResync if the informer's resyncPeriod is non-zero, resync will be periodically triggered.
func (s *informer) ShouldResync() bool {
	return s.resyncCheckPeriod != 0
}

// HandleDeltas is called for each delta when pop out from queue.
func (s *informer) HandleDeltas(obj interface{}, _ bool) error {
	s.blockDeltas.Lock()
	defer s.blockDeltas.Unlock()

	deltas, ok := obj.(cache.Deltas)
	if !ok {
		return errors.New("object given as Process argument is not Deltas")
	}
	// from oldest to newest
	for _, d := range deltas {
		var resource model.Resource
		var object interface{}
		if o, ok := d.Object.(cache.DeletedFinalStateUnknown); ok {
			object = o.Obj
		} else {
			object = d.Object
		}
		resource, ok := object.(model.Resource)
		if !ok {
			logger.Errorf("object from ListWatcher is not conformed to Resource, obj: %v", obj)
			return bizerror.NewAssertionError("Resource", reflect.TypeOf(obj).Name())
		}
		switch d.Type {
		case cache.Sync, cache.Replaced, cache.Added, cache.Updated:
			if old, exists, err := s.indexer.Get(resource); err == nil && exists {
				if err := s.indexer.Update(resource); err != nil {
					logger.Errorf("failed to update resource in informer, cause: %v, resource: %s,", err, resource.String())
					return err
				}
				s.EmitEvent(d.Type, old.(model.Resource), resource)
			} else {
				if err := s.indexer.Add(resource); err != nil {
					logger.Errorf("failed to add resource to informer, cause %v, resource: %s,", err, resource.String())
					return err
				}
				s.EmitEvent(d.Type, nil, resource)
			}
		case cache.Deleted:
			if err := s.indexer.Delete(resource); err != nil {
				logger.Errorf("failed to delete resource from informer, cause %v, resource: %s,", err, resource.String())
				return err
			}
			s.EmitEvent(d.Type, resource, nil)
		}
	}
	return nil
}

// EmitEvent emits an event to the event bus.
func (s *informer) EmitEvent(typ cache.DeltaType, oldObj model.Resource, newObj model.Resource) {
	event := events.NewResourceChangedEvent(typ, oldObj, newObj)
	s.emitter.Send(event)
}

// IsStopped reports whether the informer has already been stopped.
func (s *informer) IsStopped() bool {
	s.startedLock.Lock()
	defer s.startedLock.Unlock()
	return s.stopped
}
