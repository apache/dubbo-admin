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
  ApplicationSearchItem,
  ApplicationDetail,
  ApplicationInstanceStatistics,
  ApplicationInstanceInfoItem,
  ApplicationEventItem,
  PaginatedData,
  FlowWeightItem,
  GrayItem
} from '@/types/api'

const appNames = ['QuickStartApplication', 'shop-comment', 'shop-detail', 'shop-order', 'shop-user']
const deployClusters = ['default', 'prod', 'test']

function randomInt(min: number, max: number): number {
  return Math.floor(Math.random() * (max - min + 1)) + min
}

function randomPick<T>(arr: T[]): T {
  return arr[Math.floor(Math.random() * arr.length)]
}

export const appHandlers: HttpHandler[] = [
  http.get(`${base}/application/metric-dashboard`, () =>
    success(
      'http://8.147.104.101:3000/d/a0b114ca-edf7-4dfe-ac2c-34a4fc545fed/application?orgId=1&refresh=1m&from=1711855893859&to=1711877493859&theme=light'
    )
  ),

  http.get(`${base}/application/trace-dashboard`, () =>
    success(
      'http://8.147.104.101:3000/d/e968a89b-f03d-42e3-8ad3-930ae815cb0f/application?orgId=1&refresh=1m'
    )
  ),

  http.get(`${base}/application/config/operatorLog`, () => success({ operatorLog: true })),

  http.put(`${base}/application/config/operatorLog`, () => success(null)),

  http.get(`${base}/application/config/flowWeight`, () =>
    success({
      flowWeightSets: [
        { version: '1.0.0', weight: 80 },
        { version: '2.0.0', weight: 20 }
      ]
    } as { flowWeightSets: FlowWeightItem[] })
  ),

  http.put(`${base}/application/config/flowWeight`, () => success(null)),

  http.get(`${base}/application/config/gray`, () =>
    success({
      graySets: [
        { tag: 'v1', weight: 100 },
        { tag: 'v2', weight: 0 }
      ]
    } as { graySets: GrayItem[] })
  ),

  http.put(`${base}/application/config/gray`, () => success(null)),

  http.get(`${base}/application/search`, () => {
    const total = randomInt(3, 20)
    const list: ApplicationSearchItem[] = Array.from({ length: total }, () => ({
      appName: randomPick(appNames),
      deployClusters: [randomPick(deployClusters)],
      instanceCount: randomInt(1, 5),
      registryClusters: [
        `${randomInt(1, 255)}.${randomInt(0, 255)}.${randomInt(0, 255)}.${randomInt(1, 254)}:8848`
      ]
    }))
    return success<PaginatedData<ApplicationSearchItem>>({
      list,
      pageInfo: { Total: total, NextOffset: '' }
    })
  }),

  http.get(`${base}/application/instance/statistics`, () =>
    success<ApplicationInstanceStatistics>({
      instanceTotal: 43,
      versionTotal: 4,
      cpuTotal: '56c',
      memoryTotal: '108.2GB'
    })
  ),

  http.get(`${base}/application/instance/info`, () => {
    const total = randomInt(8, 100)
    const list: ApplicationInstanceInfoItem[] = Array.from({ length: total }, () => ({
      ip: '121.90.211.162',
      name: 'shop-user',
      deployState: randomPick(['Running', 'Pending', 'Terminating', 'Crashing']),
      deployCluster: 'tx-shanghai-1',
      registerState: 'Registed',
      registerClusters: ['ali-hangzhou-1', 'ali-hangzhou-2'],
      cpu: '1.2c',
      memory: '2349MB',
      startTime: '2023-06-09 03:47:10',
      registerTime: '2023-06-09 03:48:20',
      labels: { region: 'beijing', version: 'v1' }
    }))
    return success<PaginatedData<ApplicationInstanceInfoItem>>({
      pageInfo: { Total: list.length, NextOffset: '0' },
      list
    })
  }),

  http.get(`${base}/application/detail`, () =>
    success<ApplicationDetail>({
      appName: 'sample-app',
      appTypes: ['web', 'rpc'],
      deployClusters: ['default', 'prod', 'test'],
      dubboPorts: [20880, 20881],
      dubboVersions: ['3.0.0', '3.1.0'],
      images: ['apache/dubbo-samples:v1'],
      registerClusters: ['nacos-1', 'nacos-2'],
      registerModes: ['instance', 'application'],
      rpcProtocols: ['triple', 'dubbo'],
      serialProtocols: ['hessian2', 'fastjson2'],
      workloads: ['deployment-1', 'deployment-2']
    })
  ),

  http.get(`${base}/application/event`, () => {
    const list: ApplicationEventItem[] = Array.from({ length: 10 }, () => ({
      desc: `Scaled down replica set shop-detail-v1-5847b7cdfd to ${randomInt(3, 10)} from ${randomInt(3, 10)}`,
      time: '2024-03-31 12:00:00',
      type: 'deployment-controller'
    }))
    return success({ list })
  }),

  http.get(`${base}/application/service/form`, () =>
    success<PaginatedData>({
      list: [],
      pageInfo: { Total: 0, NextOffset: '' }
    })
  )
]
