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

import { http, HttpResponse, type HttpHandler } from 'msw'
import { success, base } from '../utils'
import type {
  ServiceSearchItem,
  ServiceMethod,
  ServiceMethodDetail,
  ProviderInstance,
  ArgumentRouteConfig,
  PaginatedData
} from '@/types/api'

const serviceList: ServiceSearchItem[] = [
  {
    serviceName: 'org.apache.dubbo.samples.UserService',
    versionGroups: [
      { version: '1.0.0', group: 'group1' },
      { version: '1.0.0', group: null },
      { version: null, group: 'group1' },
      { version: null, group: null }
    ],
    avgQPS: 6,
    avgRT: '194ms',
    requestTotal: 200
  },
  {
    serviceName: 'org.apache.dubbo.samples.OrderService',
    versionGroups: [
      { version: '1.0.0', group: 'group1' },
      { version: '1.0.0', group: null },
      { version: null, group: 'group1' },
      { version: null, group: null }
    ],
    avgQPS: 13,
    avgRT: '189ms',
    requestTotal: 164
  },
  {
    serviceName: 'org.apache.dubbo.samples.DetailService',
    versionGroups: [
      { version: '1.0.0', group: 'group1' },
      { version: '1.0.0', group: null },
      { version: null, group: 'group1' },
      { version: null, group: null }
    ],
    avgQPS: 0.5,
    avgRT: '268ms',
    requestTotal: 1324
  },
  {
    serviceName: 'org.apache.dubbo.samples.PayService',
    versionGroups: [
      { version: '1.0.0', group: 'group1' },
      { version: '1.0.0', group: null },
      { version: null, group: 'group1' },
      { version: null, group: null }
    ],
    avgQPS: 9,
    avgRT: '346ms',
    requestTotal: 189
  },
  {
    serviceName: 'org.apache.dubbo.samples.CommentService',
    versionGroups: [
      { version: '1.0.0', group: 'group1' },
      { version: '1.0.0', group: null },
      { version: null, group: 'group1' },
      { version: null, group: null }
    ],
    avgQPS: 8,
    avgRT: '936ms',
    requestTotal: 200
  },
  {
    serviceName: 'org.apache.dubbo.samples.RepayService',
    versionGroups: [
      { version: '1.0.0', group: 'group1' },
      { version: '1.0.0', group: null },
      { version: null, group: 'group1' },
      { version: null, group: null }
    ],
    avgQPS: 17,
    avgRT: '240ms',
    requestTotal: 146
  },
  {
    serviceName: 'org.apche.dubbo.samples.TransportService',
    versionGroups: [
      { version: '1.0.0', group: 'group1' },
      { version: '1.0.0', group: null },
      { version: null, group: 'group1' },
      { version: null, group: null }
    ],
    avgQPS: 43,
    avgRT: '89ms',
    requestTotal: 367
  },
  {
    serviceName: 'org.apche.dubbo.samples.DistributionService',
    versionGroups: [
      { version: '1.0.0', group: 'group1' },
      { version: '1.0.0', group: null },
      { version: null, group: 'group1' },
      { version: null, group: null }
    ],
    avgQPS: 4,
    avgRT: '78ms',
    requestTotal: 145
  }
]

const userType: ServiceMethodDetail['types'][number] = {
  type: 'org.apache.demo.UserDTO',
  properties: { id: 'java.lang.String', name: 'java.lang.String', age: 'java.lang.Integer' },
  items: [],
  enums: []
}

const methodFixtures: { detail: ServiceMethodDetail; result: unknown }[] = [
  {
    detail: {
      methodName: 'query',
      signature: 'java.lang.String->org.apache.demo.UserDTO',
      parameterTypes: ['java.lang.String'],
      parameters: [{ name: 'id', type: 'java.lang.String' }],
      returnType: 'org.apache.demo.UserDTO',
      types: [userType]
    },
    result: { id: '1001', name: 'Alice', age: 18 }
  },
  {
    detail: {
      methodName: 'query',
      signature: 'char->org.apache.demo.UserDTO',
      parameterTypes: ['char'],
      parameters: [{ name: 'initial', type: 'char' }],
      returnType: 'org.apache.demo.UserDTO',
      types: [userType]
    },
    result: { id: '1002', name: 'Bob', age: 20 }
  },
  {
    detail: {
      methodName: 'create',
      signature: 'org.apache.demo.UserCreateReq->java.lang.Boolean',
      parameterTypes: ['org.apache.demo.UserCreateReq'],
      parameters: [{ name: 'req', type: 'org.apache.demo.UserCreateReq' }],
      returnType: 'java.lang.Boolean',
      types: [
        {
          type: 'org.apache.demo.UserCreateReq',
          properties: {
            name: 'java.lang.String',
            age: 'java.lang.Integer',
            email: 'java.lang.String'
          },
          items: [],
          enums: []
        }
      ]
    },
    result: true
  },
  {
    detail: {
      methodName: 'delete',
      signature: 'java.lang.String->java.lang.Boolean',
      parameterTypes: ['java.lang.String'],
      parameters: [{ name: 'id', type: 'java.lang.String' }],
      returnType: 'java.lang.Boolean',
      types: []
    },
    result: true
  }
]

const serviceMethods: ServiceMethod[] = methodFixtures.map(({ detail }) => ({
  methodName: detail.methodName,
  signature: detail.signature,
  parameterTypes: detail.parameterTypes
}))

function findMethodFixture(
  methodName: string | null | undefined,
  signature: string | null | undefined
) {
  return methodFixtures.find(
    ({ detail }) => detail.methodName === methodName && detail.signature === signature
  )
}

function invalidArgument(message: string, field?: string) {
  return HttpResponse.json({
    code: 'InvalidArgument',
    message,
    data: null,
    ...(field ? { field, expected: 'non-empty', actual: '' } : {})
  })
}

function methodNotFound(serviceName: string, methodName: string, signature?: string) {
  // collectMethods in the backend sorts signatures before building this field.
  const expected = methodFixtures
    .filter(({ detail }) => detail.methodName === methodName)
    .map(({ detail }) => detail.signature)
    .sort()
    .join(' | ')
  return HttpResponse.json({
    code: 'NotFoundError',
    message: `method ${methodName} not found for service ${serviceName}`,
    data: null,
    ...(signature !== undefined && expected
      ? { field: 'signature', expected, actual: signature }
      : {})
  })
}

const providerInstances: ProviderInstance[] = [
  { name: 'dubbo-provider-0', appName: 'dubbo-sample-provider', ip: '10.20.30.11' },
  { name: 'dubbo-provider-1', appName: 'dubbo-sample-provider', ip: '10.20.30.12' }
]

const argumentRouteConfig: ArgumentRouteConfig = {
  args: [
    {
      type: 'header',
      key: 'X-User-Id',
      operator: '=',
      value: '123',
      serviceName: 'org.apache.dubbo.samples.UserService'
    }
  ]
}

export const serviceHandlers: HttpHandler[] = [
  http.get(`${base}/service/search`, () =>
    success<PaginatedData<ServiceSearchItem>>({
      pageInfo: { Total: 8, NextOffset: '0' },
      list: serviceList
    })
  ),

  http.get(`${base}/service/metric-dashboard`, () =>
    success(
      'http://8.147.104.101:3000/d/a0b114ca-edf7-4dfe-ac2c-34a4fc545fed/service?orgId=1&refresh=1m'
    )
  ),

  http.get(`${base}/service/trace-dashboard`, () =>
    success(
      'http://8.147.104.101:3000/d/e968a89b-f03d-42e3-8ad3-930ae815cb0f/service?orgId=1&refresh=1m'
    )
  ),

  http.get(`${base}/service/config/timeout`, () => success({ timeout: 3000 })),

  http.put(`${base}/service/config/timeout`, () => success(null)),

  http.get(`${base}/service/config/retry`, () => success({ retry: 3 })),

  http.put(`${base}/service/config/retry`, () => success(null)),

  http.get(`${base}/service/config/regionPriority`, () => success({ enable: true })),

  http.put(`${base}/service/config/regionPriority`, () => success(null)),

  http.get(`${base}/service/config/argumentRoute`, () => success(argumentRouteConfig)),

  http.put(`${base}/service/config/argumentRoute`, () => success(null)),

  http.get(`${base}/service/provider-instances`, () => success(providerInstances)),

  http.get(`${base}/service/methods`, () => success(serviceMethods)),

  http.get(`${base}/service/method/detail`, ({ request }) => {
    const url = new URL(request.url)
    const serviceName = (url.searchParams.get('serviceName') || '').trim()
    const methodName = (url.searchParams.get('methodName') || '').trim()
    const signature = (url.searchParams.get('signature') || '').trim()
    if (!methodName) return invalidArgument('method name is empty')
    if (!signature) return invalidArgument('signature is empty')
    const fixture = findMethodFixture(methodName, signature)
    return fixture ? success(fixture.detail) : methodNotFound(serviceName, methodName)
  }),

  http.post(`${base}/service/generic/invoke`, async ({ request }) => {
    let body: { serviceName?: string; methodName?: string; signature?: string } | null
    try {
      body = (await request.json()) as typeof body
    } catch {
      // Match ShouldBindJSON's business error envelope, without emulating Go's parser diagnostics.
      return invalidArgument('invalid JSON request body')
    }
    const serviceName = typeof body?.serviceName === 'string' ? body.serviceName.trim() : ''
    const methodName = typeof body?.methodName === 'string' ? body.methodName.trim() : ''
    const signature = typeof body?.signature === 'string' ? body.signature.trim() : ''
    if (!methodName) return invalidArgument('method name is empty', 'methodName')
    if (!signature) return invalidArgument('signature is empty', 'signature')
    const fixture = findMethodFixture(methodName, signature)
    return fixture
      ? success({ elapsedMs: 12, rawResult: fixture.result })
      : methodNotFound(serviceName, methodName, signature)
  }),

  http.get(`${base}/service/event`, () => {
    const sources = ['deployment-controller', 'nacos', 'zookeeper', 'kubelet']
    const messages = [
      'Service provider metadata updated: org.apache.dubbo.samples.UserService:v1',
      'Service consumer registered: shop-user app',
      'Instance registered via Nacos: 10.20.30.11:20880',
      'Condition route rule applied to org.apache.dubbo.samples.UserService',
      'Instance deregistered from Zookeeper: 10.20.30.12:20880'
    ]
    const list = Array.from({ length: 5 }, (_, i) => ({
      time: `2024/2/17 ${String(20 - i).padStart(2, '0')}:04:38`,
      type: (i === 4 ? 'warning' : 'normal') as 'normal' | 'warning',
      message: messages[i],
      source: sources[i % sources.length]
    }))
    return success({ list, total: list.length })
  })
]
