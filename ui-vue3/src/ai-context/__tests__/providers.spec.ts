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

import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { i18n } from '@/base/i18n'
import { useMeshStore } from '@/stores/mesh'
import { collectGlobalAIContext } from '../providers/global'
import { createHomeOverviewContribution } from '../providers/home'

const initialLocale = i18n.global.locale.value

describe('AI context providers', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  afterEach(() => {
    i18n.global.locale.value = initialLocale
  })

  it('reads the current mesh and locale for every collection', () => {
    const meshStore = useMeshStore()
    meshStore.mesh = 'mesh-a'
    i18n.global.locale.value = 'en'

    expect(collectGlobalAIContext()).toEqual({
      global: { locale: 'en' },
      scope: { mesh: 'mesh-a' }
    })

    meshStore.mesh = 'mesh-b'
    i18n.global.locale.value = 'cn'

    expect(collectGlobalAIContext()).toEqual({
      global: { locale: 'cn' },
      scope: { mesh: 'mesh-b' }
    })
  })

  it('creates a small whitelisted home overview', () => {
    const contribution = createHomeOverviewContribution({
      appCount: 3,
      serviceCount: 8,
      insCount: 12,
      releases: { '3.2.0': 4, '3.3.0': 8 },
      protocols: { dubbo: 10, tri: 2 },
      discoveries: { kubernetes: 12 },
      token: 'must-not-be-included'
    })

    expect(contribution?.evidence).toMatchObject({
      id: 'cluster-overview',
      source: 'cluster-overview-api',
      data: {
        applicationCount: 3,
        serviceCount: 8,
        instanceCount: 12,
        releases: ['3.2.0', '3.3.0'],
        protocols: ['dubbo', 'tri'],
        discoveries: ['kubernetes']
      }
    })
    expect(JSON.stringify(contribution)).not.toContain('must-not-be-included')
  })
})
