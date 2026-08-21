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
import { createInstanceDetailContribution } from '../providers/instance'
import {
  createConfigurationStateContribution,
  createDashboardStateContribution,
  createEventListContribution,
  createResourceDetailsContribution,
  createServiceDebugContribution,
  createTopologyStateContribution
} from '../providers/page'
import {
  createApplicationResourceContribution,
  createInstanceResourceContribution,
  createServiceResourceContribution
} from '../providers/resource'
import {
  createSearchFiltersContribution,
  createSearchResultsContribution
} from '../providers/search'
import {
  createTrafficDraftContribution,
  createTrafficRuleContentContribution,
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

  it('creates a whitelisted instance detail summary', () => {
    const contribution = createInstanceDetailContribution({
      appName: 'shop-user',
      ip: '10.0.0.8',
      rpcPort: 20880,
      lifecycleState: 'Running',
      registerState: 'Registered',
      deployState: 'Ready',
      registerTime: '2026-07-24T08:00:00Z',
      startTime: '2026-07-24T07:59:00Z',
      deployCluster: 'prod-k8s',
      registerClusters: ['nacos-a', 'nacos-b'],
      node: 'node-1',
      workloadName: 'shop-user(deployment)',
      image: 'registry.example.com/shop-user:v2',
      labels: {
        app: 'shop-user',
        token: 'must-not-be-included',
        version: 'v2'
      },
      probes: {
        startupProbe: { open: true, type: 'http', port: 8080, command: 'ignored' },
        readinessProbe: { open: true, type: 'tcp', port: 20880 },
        livenessProbe: { open: false, type: 'http', port: 8080 }
      },
      kubeconfig: 'must-not-be-included',
      internalMetadata: { password: 'must-not-be-included' }
    })

    expect(contribution).toEqual({
      evidence: {
        id: 'instance-detail',
        source: 'instance-detail-api',
        data: {
          application: 'shop-user',
          ip: '10.0.0.8',
          rpcPort: 20880,
          lifecycleState: 'Running',
          registerState: 'Registered',
          deployState: 'Ready',
          registeredAt: '2026-07-24T08:00:00Z',
          startedAt: '2026-07-24T07:59:00Z',
          deployCluster: 'prod-k8s',
          registerClusters: 'nacos-a, nacos-b',
          node: 'node-1',
          workloadName: 'shop-user(deployment)',
          image: 'registry.example.com/shop-user:v2',
          labels: '{"app":"shop-user","token":"[REDACTED]","version":"v2"}',
          startupProbe: 'open=true, type=http, port=8080',
          readinessProbe: 'open=true, type=tcp, port=20880',
          livenessProbe: 'open=false, type=http, port=8080'
        }
      }
    })
    expect(JSON.stringify(contribution)).not.toContain('must-not-be-included')
    expect(createInstanceDetailContribution({})).toBeUndefined()
  })

  it('collects only declared, non-empty search filters', () => {
    expect(
      createSearchFiltersContribution(
        [{ param: 'keywords' }, { param: 'status' }, { param: 'labels' }, { param: 'options' }],
        {
          keywords: '  shop-user  ',
          status: false,
          labels: ['prod', '', 'gray'],
          options: { includeOffline: true },
          internalToken: 'must-not-be-included'
        }
      )
    ).toEqual({
      state: {
        filters: {
          keywords: 'shop-user',
          status: false,
          labels: ['prod', 'gray']
        }
      }
    })
    expect(
      createSearchFiltersContribution([{ param: 'keywords' }], { keywords: ' ' })
    ).toBeUndefined()
  })

  it('summarizes only visible table columns and limits rows', () => {
    const rows = Array.from({ length: 12 }, (_, index) => ({
      appName: `shop-${index}`,
      status: index % 2 ? 'Running' : 'Stopped',
      hiddenToken: 'must-not-be-included',
      metadata: { internal: true }
    }))

    const contribution = createSearchResultsContribution(
      rows,
      [
        { dataIndex: 'appName' },
        { key: 'status' },
        { dataIndex: 'hiddenToken', __hide: true },
        { dataIndex: 'metadata' }
      ],
      { curPage: 2, pageSize: 10, total: 25 }
    )

    expect(contribution?.evidence).toMatchObject({
      id: 'search-results',
      data: {
        total: 25,
        currentPage: 2,
        pageSize: 10,
        displayedCount: 12,
        includedCount: 10,
        truncated: true
      }
    })
    const data = (contribution?.evidence as any).data
    expect(data.rows).toHaveLength(10)
    expect(data.rows[0]).toEqual({ appName: 'shop-0', status: 'Stopped' })
    expect(JSON.stringify(contribution)).not.toContain('must-not-be-included')
    expect(createSearchResultsContribution(undefined, [])).toBeUndefined()
  })

  it('summarizes configuration and dashboard state without exposing dashboard credentials', () => {
    const configuration = createConfigurationStateContribution('accesslog', {
      enabled: true,
      token: 'must-not-be-included',
      nested: { retries: 3 }
    })
    expect(configuration?.evidence).toMatchObject({
      id: 'configuration-state',
      data: {
        section: 'accesslog',
        values: {
          enabled: true,
          token: '[REDACTED]',
          nested: { retries: 3 }
        }
      }
    })

    const dashboard = createDashboardStateContribution(
      'instanceDomain.monitor',
      { instanceName: 'shop-user-1' },
      'https://admin:secret@grafana.example.com/d/instance?from=now-1h&to=now&refresh=1m&token=secret'
    )
    expect(dashboard.evidence).toEqual({
      id: 'dashboard-state',
      source: 'dashboard-page',
      data: {
        dashboard: 'instanceDomain.monitor',
        loaded: true,
        parameters: { instanceName: 'shop-user-1' },
        timeRange: { from: 'now-1h', to: 'now', refresh: '1m' }
      }
    })
    expect(JSON.stringify(dashboard)).not.toContain('admin:secret')
    expect(JSON.stringify(dashboard)).not.toContain('token=secret')
  })

  it('creates whitelisted resource, topology, and event summaries', () => {
    const details = createResourceDetailsContribution(
      'application-detail',
      'application-detail-api',
      {
        appName: 'shop-user',
        dubboVersions: ['3.3.0'],
        token: 'must-not-be-included'
      },
      ['appName', 'dubboVersions']
    )
    expect(details?.evidence).toMatchObject({
      id: 'application-detail',
      data: { appName: 'shop-user', dubboVersions: ['3.3.0'] }
    })
    expect(JSON.stringify(details)).not.toContain('must-not-be-included')

    const topology = createTopologyStateContribution(
      {
        nodes: [{ id: 'shop-user', label: 'shop-user', type: 'application', secret: 'hidden' }],
        edges: [{ source: 'shop-user', target: 'DemoService', metadata: 'hidden' }]
      },
      { key: 'shop-user', type: 'application', detail: { deployState: 'Running' } }
    )
    expect(topology?.evidence).toMatchObject({
      id: 'topology-state',
      data: {
        nodeCount: 1,
        edgeCount: 1,
        nodes: [{ id: 'shop-user', label: 'shop-user', type: 'application' }],
        edges: [{ source: 'shop-user', target: 'DemoService' }],
        selectedNode: 'shop-user',
        selectedNodeDetail: { deployState: 'Running' }
      }
    })

    expect(
      createEventListContribution([
        { type: 'Registered', desc: 'instance registered', time: '2026-07-24', raw: 'hidden' }
      ])?.evidence
    ).toMatchObject({
      id: 'event-list',
      data: {
        total: 1,
        events: [{ type: 'Registered', description: 'instance registered', time: '2026-07-24' }]
      }
    })
  })

  it('summarizes service debug input and output with parsed sensitive fields redacted', () => {
    const contribution = createServiceDebugContribution({
      methods: [
        { methodName: 'sayHello', signature: 'sayHello(java.lang.String)', internal: 'hidden' }
      ],
      providers: [{ name: 'shop-user-1', appName: 'shop-user', ip: '10.0.0.8' }],
      selectedInstance: 'shop-user-1',
      method: {
        methodName: 'sayHello',
        parameterTypes: ['java.lang.String'],
        returnType: 'java.lang.String'
      },
      request: '[{"token":"must-not-be-included","name":"Dubbo"}]',
      response: '{"message":"hello"}',
      elapsedMs: 12,
      timeoutMs: 3000,
      attachmentKeys: ['traceId']
    })

    expect(contribution.evidence).toMatchObject({
      id: 'service-debug',
      data: {
        methodCount: 1,
        providerCount: 1,
        selectedInstance: 'shop-user-1',
        request: [{ token: '[REDACTED]', name: 'Dubbo' }],
        response: { message: 'hello' },
        elapsedMs: 12,
        timeoutMs: 3000,
        attachmentKeys: ['traceId']
      }
    })
    expect(JSON.stringify(contribution)).not.toContain('must-not-be-included')
  })

  it('collects a condition rule draft summary and selectable content', () => {
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

    expect(contribution).toMatchObject({
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
      },
      evidence: {
        id: 'rule-content',
        data: {
          kind: 'condition-rule',
          content: {
            scope: 'service',
            key: 'DemoService',
            conditions: ['host=10.0.0.1 => address=10.0.0.2']
          }
        }
      }
    })
    expect(JSON.stringify(contribution)).not.toContain('must-not-be-included')
  })

  it('summarizes tag and dynamic config drafts with safe rule content', () => {
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
      state: { unsavedChanges: { kind: 'tag-rule', entryCount: 1 } },
      evidence: {
        data: {
          content: {
            tags: [{ name: 'gray', match: [{ key: 'env', value: { exact: 'secret-value' } }] }]
          }
        }
      }
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

  it('creates selectable content for an existing rule', () => {
    expect(
      createTrafficRuleContentContribution('condition-rule', {
        key: 'DemoService',
        enabled: true,
        conditions: ['host=10.0.0.1 => address=10.0.0.2'],
        internalMetadata: 'ignored'
      })
    ).toEqual({
      evidence: {
        id: 'rule-content',
        source: 'traffic-rule-page',
        data: {
          kind: 'condition-rule',
          content: {
            key: 'DemoService',
            enabled: true,
            conditions: ['host=10.0.0.1 => address=10.0.0.2']
          }
        }
      }
    })
  })

  it('collects an existing traffic rule identifier', () => {
    expect(createTrafficRuleResourceContribution(' shop-user.tag-router ')).toEqual({
      scope: { rule: 'shop-user.tag-router' }
    })
    expect(createTrafficRuleResourceContribution('_tmp')).toBeUndefined()
  })
})
