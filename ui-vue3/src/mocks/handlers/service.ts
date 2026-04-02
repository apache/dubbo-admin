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

import { http, type HttpHandler } from 'msw'
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

const serviceMethods: ServiceMethod[] = [
  {
    methodName: 'query',
    parameterTypes: ['java.lang.String'],
    signature: 'java.lang.String->org.apache.demo.UserDTO'
  },
  {
    methodName: 'create',
    parameterTypes: ['org.apache.demo.UserCreateReq'],
    signature: 'org.apache.demo.UserCreateReq->java.lang.Boolean'
  },
  {
    methodName: 'delete',
    parameterTypes: ['java.lang.String'],
    signature: 'java.lang.String->java.lang.Boolean'
  }
]

const methodDetails: Record<string, ServiceMethodDetail> = {
  query: {
    methodName: 'query',
    signature: 'java.lang.String->org.apache.demo.UserDTO',
    parameterTypes: ['java.lang.String'],
    parameters: [{ name: 'id', type: 'java.lang.String' }],
    returnType: 'org.apache.demo.UserDTO',
    types: [
      {
        type: 'org.apache.demo.UserDTO',
        properties: { id: 'java.lang.String', name: 'java.lang.String', age: 'java.lang.Integer' },
        items: [],
        enums: []
      }
    ]
  },
  create: {
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
  delete: {
    methodName: 'delete',
    signature: 'java.lang.String->java.lang.Boolean',
    parameterTypes: ['java.lang.String'],
    parameters: [{ name: 'id', type: 'java.lang.String' }],
    returnType: 'java.lang.Boolean',
    types: []
  }
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
    const methodName = url.searchParams.get('methodName') || 'query'
    return success(methodDetails[methodName] || methodDetails.query)
  }),

  http.post(`${base}/service/generic/invoke`, () =>
    success({ elapsedMs: 12, rawResult: { id: '1001', name: 'Alice', age: 18 } })
  )
]
