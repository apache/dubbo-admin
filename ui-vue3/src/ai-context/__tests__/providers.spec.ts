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
import {
  createApplicationResourceContribution,
  createInstanceResourceContribution,
  createServiceResourceContribution
} from '../providers/resource'
import {
  createTrafficDraftContribution,
  createTrafficRuleResourceContribution
} from '../providers/traffic'

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

  it('collects stable application, service, and instance identifiers', () => {
    expect(createApplicationResourceContribution(' shop-user ')).toEqual({
      scope: { application: 'shop-user' }
    })
    expect(
      createServiceResourceContribution('org.apache.dubbo.ShopService', 'prod', '1.0.0')
    ).toEqual({
      scope: { service: 'org.apache.dubbo.ShopService' },
      state: { selection: { group: 'prod', version: '1.0.0' } }
    })
    expect(createInstanceResourceContribution('shop-user-7f9d', 'shop-user')).toEqual({
      scope: { instance: 'shop-user-7f9d', application: 'shop-user' }
    })
  })

  it('omits missing resource identifiers and empty service qualifiers', () => {
    expect(createApplicationResourceContribution(' ')).toBeUndefined()
    expect(createServiceResourceContribution(undefined)).toBeUndefined()
    expect(createInstanceResourceContribution([])).toBeUndefined()
    expect(createServiceResourceContribution(['DemoService'], '', undefined)).toEqual({
      scope: { service: 'DemoService' }
    })
  })

  it('collects only a whitelisted condition rule draft summary', () => {
    const contribution = createTrafficDraftContribution({
      kind: 'condition-rule',
      mode: 'update',
      representation: 'form',
      rule: 'DemoService:1.0.0:prod',
      draft: {
        scope: 'service',
        key: 'DemoService',
        enabled: true,
        runtime: false,
        force: true,
        conditions: ['host=10.0.0.1 => address=10.0.0.2'],
        password: 'must-not-be-included'
      }
    })

    expect(contribution).toEqual({
      scope: {
        rule: 'DemoService:1.0.0:prod',
        service: 'DemoService'
      },
      state: {
        unsavedChanges: {
          kind: 'condition-rule',
          mode: 'update',
          representation: 'form',
          scope: 'service',
          key: 'DemoService',
          enabled: true,
          runtime: false,
          force: true,
          entryCount: 1
        }
      }
    })
    expect(JSON.stringify(contribution)).not.toContain('10.0.0.1')
    expect(JSON.stringify(contribution)).not.toContain('must-not-be-included')
  })

  it('summarizes tag and dynamic config drafts without values', () => {
    expect(
      createTrafficDraftContribution({
        kind: 'tag-rule',
        mode: 'create',
        representation: 'yaml',
        draft: {
          scope: 'application',
          key: 'shop-user',
          tags: [{ name: 'gray', match: [{ key: 'env', value: { exact: 'secret-value' } }] }]
        }
      })
    ).toMatchObject({
      scope: { rule: 'shop-user', application: 'shop-user' },
      state: { unsavedChanges: { kind: 'tag-rule', entryCount: 1 } }
    })

    const dynamicConfig = createTrafficDraftContribution({
      kind: 'dynamic-config',
      mode: 'create',
      representation: 'form',
      rule: '_tmp',
      draft: {
        basicInfo: { ruleName: '_tmp', scope: 'service', key: 'DemoService', enabled: true },
        config: [{ parametersValue: { token: 'must-not-be-included' } }]
      }
    })
    expect(dynamicConfig).toMatchObject({
      scope: { rule: 'DemoService.configurators', service: 'DemoService' },
      state: { unsavedChanges: { kind: 'dynamic-config', entryCount: 1 } }
    })
    expect(JSON.stringify(dynamicConfig)).not.toContain('must-not-be-included')
  })

  it('collects an existing traffic rule identifier', () => {
    expect(createTrafficRuleResourceContribution(' shop-user.tag-router ')).toEqual({
      scope: { rule: 'shop-user.tag-router' }
    })
    expect(createTrafficRuleResourceContribution('_tmp')).toBeUndefined()
  })
})
