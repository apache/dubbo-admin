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

import { createPinia, setActivePinia } from 'pinia'
import { describe, expect, it, vi } from 'vitest'
import { useMeshStore } from '@/stores/mesh'
import { AIContextManager } from '../manager'
import { collectGlobalAIContext } from '../providers/global'
import type { AIContextBase } from '../types'

const baseContext = (): AIContextBase => ({
  global: { locale: 'cn' },
  page: {
    routeName: 'applicationDomain.detail',
    path: '/resources/applications/detail/shop-user',
    fullPath: '/resources/applications/detail/shop-user'
  },
  scope: { mesh: 'nacos2.5' }
})

describe('AIContextManager', () => {
  it('collects only providers for the active route', () => {
    const manager = new AIContextManager(baseContext)
    manager.register({
      id: 'active',
      routeKey: () => '/resources/applications/detail/shop-user',
      collect: () => ({ scope: { application: 'shop-user' } })
    })
    manager.register({
      id: 'stale',
      routeKey: () => '/resources/applications/detail/shop-order',
      collect: () => ({ scope: { application: 'shop-order' } })
    })

    expect(manager.snapshot().scope).toEqual({
      mesh: 'nacos2.5',
      application: 'shop-user'
    })
  })

  it('stops collecting after a provider is unregistered', () => {
    const manager = new AIContextManager(baseContext)
    const collect = vi.fn(() => ({ scope: { service: 'UserService' } }))
    const unregister = manager.register({ id: 'service', collect })

    manager.snapshot()
    unregister()
    const snapshot = manager.snapshot()

    expect(collect).toHaveBeenCalledTimes(1)
    expect(snapshot.scope.service).toBeUndefined()
    expect(manager.size()).toBe(0)
  })

  it('lets higher-priority providers override lower-priority scope', () => {
    const manager = new AIContextManager(baseContext)
    manager.register({
      id: 'low-priority',
      priority: 1,
      collect: () => ({ scope: { application: 'fallback' } })
    })
    manager.register({
      id: 'high-priority',
      priority: 100,
      collect: () => ({ scope: { application: 'shop-user' } })
    })

    expect(manager.snapshot().scope.application).toBe('shop-user')
  })

  it('ignores provider failures so chat can continue', () => {
    const manager = new AIContextManager(baseContext)
    manager.register({
      id: 'broken',
      collect: () => {
        throw new Error('page is not ready')
      }
    })

    expect(() => manager.snapshot()).not.toThrow()
    expect(manager.snapshot().scope.mesh).toBe('nacos2.5')
  })

  it('reads the current mesh again for the next snapshot', () => {
    setActivePinia(createPinia())
    const meshStore = useMeshStore()
    const manager = new AIContextManager(() => ({
      ...collectGlobalAIContext(),
      page: baseContext().page
    }))

    meshStore.mesh = 'mesh-a'
    expect(manager.snapshot().scope.mesh).toBe('mesh-a')

    meshStore.mesh = 'mesh-b'
    expect(manager.snapshot().scope.mesh).toBe('mesh-b')
  })
})
