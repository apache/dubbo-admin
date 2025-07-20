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

package watchdog

import (
	"context"
	"time"

	"github.com/pkg/errors"
)

type Watchdog interface {
	Start(stop <-chan struct{})
}

type SimpleWatchdog struct {
	NewTicker func() *time.Ticker
	OnTick    func(context.Context) error
	OnError   func(error)
	OnStop    func()
}

func (w *SimpleWatchdog) Start(stop <-chan struct{}) {
	ticker := w.NewTicker()
	defer ticker.Stop()

	for {
		ctx, cancel := context.WithCancel(context.Background())
		// cancel is called at the end of the loop
		go func() {
			select {
			case <-stop:
				cancel()
			case <-ctx.Done():
			}
		}()
		select {
		case <-ticker.C:
			select {
			case <-stop:
			default:
				if err := w.onTick(ctx); err != nil && !errors.Is(err, context.Canceled) {
					w.OnError(err)
				}
			}
		case <-stop:
			if w.OnStop != nil {
				w.OnStop()
			}
			// cancel will be called by the above goroutine
			return
		}
		cancel()
	}
}

func (w *SimpleWatchdog) onTick(ctx context.Context) error {
	defer func() {
		if cause := recover(); cause != nil {
			if w.OnError != nil {
				var err error
				switch typ := cause.(type) {
				case error:
					err = errors.WithStack(typ)
				default:
					err = errors.Errorf("%v", cause)
				}
				w.OnError(err)
			}
		}
	}()
	return w.OnTick(ctx)
}
