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

package retry

import (
	"errors"
	"fmt"
	"time"

	"github.com/apache/dubbo-admin/test"
)

const (
	// DefaultTimeout the default timeout for the entire retry operation
	DefaultTimeout = time.Second * 30

	// DefaultDelay the default delay between successive retry attempts
	DefaultDelay = time.Millisecond * 10

	// DefaultConverge the default converge, requiring something to succeed one time
	DefaultConverge = 1
)

var defaultConfig = config{
	timeout:  DefaultTimeout,
	delay:    DefaultDelay,
	converge: DefaultConverge,
}

type config struct {
	error    string
	timeout  time.Duration
	delay    time.Duration
	converge int
}

// Option for a retry operation.
type Option func(cfg *config)

// Timeout sets the timeout for the entire retry operation.
func Timeout(timeout time.Duration) Option {
	return func(cfg *config) {
		cfg.timeout = timeout
	}
}

// Delay sets the delay between successive retry attempts.
func Delay(delay time.Duration) Option {
	return func(cfg *config) {
		cfg.delay = delay
	}
}

// Converge sets the number of successes in a row needed to count a success.
// This is useful to avoid the case where tests like `coin.Flip() == HEADS` will always
// return success due to random variance.
func Converge(successes int) Option {
	return func(cfg *config) {
		cfg.converge = successes
	}
}

// Message defines a more detailed error message to use when failing
func Message(errorMessage string) Option {
	return func(cfg *config) {
		cfg.error = errorMessage
	}
}

// RetriableFunc a function that can be retried.
type RetriableFunc func() (result interface{}, completed bool, err error)

// UntilSuccess retries the given function until success, timeout, or until the passed-in function returns nil.
func UntilSuccess(fn func() error, options ...Option) error {
	_, e := Do(func() (interface{}, bool, error) {
		err := fn()
		if err != nil {
			return nil, false, err
		}

		return nil, true, nil
	}, options...)

	return e
}

// UntilSuccessOrFail calls UntilSuccess, and fails t with Fatalf if it ends up returning an error
func UntilSuccessOrFail(t test.Failer, fn func() error, options ...Option) {
	t.Helper()
	err := UntilSuccess(fn, options...)
	if err != nil {
		t.Fatalf("retry.UntilSuccessOrFail: %v", err)
	}
}

var ErrConditionNotMet = errors.New("expected condition not met")

// Until retries the given function until it returns true or hits the timeout timeout
func Until(fn func() bool, options ...Option) error {
	return UntilSuccess(func() error {
		if !fn() {
			return getErrorMessage(options)
		}
		return nil
	}, options...)
}

// UntilOrFail calls Until, and fails t with Fatalf if it ends up returning an error
func UntilOrFail(t test.Failer, fn func() bool, options ...Option) {
	t.Helper()
	err := Until(fn, options...)
	if err != nil {
		t.Fatalf("retry.UntilOrFail: %v", err)
	}
}

func getErrorMessage(options []Option) error {
	cfg := defaultConfig
	for _, option := range options {
		option(&cfg)
	}
	if cfg.error == "" {
		return ErrConditionNotMet
	}
	return errors.New(cfg.error)
}

// Do retries the given function, until there is a timeout, or until the function indicates that it has completed.
func Do(fn RetriableFunc, options ...Option) (interface{}, error) {
	cfg := defaultConfig
	for _, option := range options {
		option(&cfg)
	}

	successes := 0
	var lasterr error
	to := time.After(cfg.timeout)
	for {
		select {
		case <-to:
			return nil, fmt.Errorf("timeout while waiting (last error: %v)", lasterr)
		default:
		}

		result, completed, err := fn()
		if completed {
			if err == nil {
				successes++
			} else {
				successes = 0
			}
			if successes >= cfg.converge {
				return result, err
			}

			// Skip delay if we have a success
			continue
		} else {
			successes = 0
		}
		if err != nil {
			lasterr = err
		}

		<-time.After(cfg.delay)
	}
}
