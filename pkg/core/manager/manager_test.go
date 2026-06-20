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

package manager

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	corelock "github.com/apache/dubbo-admin/pkg/core/lock"
	locallock "github.com/apache/dubbo-admin/pkg/lock/local"
)

func TestValidateMutationContext(t *testing.T) {
	require.ErrorContains(t, validateMutationContext(nil), "resource mutation context is required")
	require.NoError(t, validateMutationContext(context.Background()))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, validateMutationContext(ctx), context.Canceled)
}

func TestValidateMutationContextRejectsLostLease(t *testing.T) {
	lockMgr := locallock.NewLocalLock()

	err := corelock.WithLock(context.Background(), lockMgr, "manager-test-lost-lease", time.Millisecond, func(leaseCtx context.Context) error {
		lease, err := corelock.RequireLease(leaseCtx)
		require.NoError(t, err)
		select {
		case <-lease.Lost():
		case <-time.After(time.Second):
			t.Fatal("lease was not reported lost after its TTL expired")
		}

		err = validateMutationContext(leaseCtx)
		require.ErrorIs(t, err, corelock.ErrLockLeaseLost)
		return err
	})
	require.ErrorIs(t, err, corelock.ErrLockLeaseLost)
}
