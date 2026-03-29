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
import type { InstanceSearchItem, InstanceDetail, PaginatedData } from '@/types/api'

function randomInt(min: number, max: number): number {
  return Math.floor(Math.random() * (max - min + 1)) + min
}

function randomPick<T>(arr: T[]): T {
  return arr[Math.floor(Math.random() * arr.length)]
}

const instanceDetail: InstanceDetail = {
  deployState: 'Running',
  registerStates: 'Unregisted',
  ip: '45.7.37.227',
  rpcPort: '20880',
  appName: 'shop-user',
  workloadName: 'shop-user-prod(deployment)',
  labels: { app: 'shop-user', version: 'v1', region: 'beijing' },
  createTime: '2023/12/19 22:09:34',
  readyTime: '2023/12/19  22:12:34',
  registerTime: '2023/12/19   22:16:56',
  registerClusters: ['sz-ali-zk-f8otyo4r', 'hz-ali-zk-oqgiq9gq'],
  deployCluster: 'tx-shanghai-1',
  node: 'hz-ali-30.33.0.1',
  image: 'apache/org.apahce.dubbo.samples.shop-user:v1',
  probes: {
    startupProbe: { type: 'http', open: true },
    readinessProbe: { type: 'http', open: true },
    livenessProbe: { type: 'http', open: true }
  }
}

export const instanceHandlers: HttpHandler[] = [
  http.get(`${base}/instance/search`, () => {
    const total = randomInt(8, 1000)
    const list: InstanceSearchItem[] = Array.from({ length: total }, () => ({
      ip: '121.90.211.162',
      name: 'shop-user',
      deployState: randomPick(['Running', 'Pending', 'Terminating', 'Crashing']),
      deployCluster: 'tx-shanghai-1',
      registerState: 'Registed',
      registerClusters: ['ali-hangzhou-1', 'ali-hangzhou-2'],
      cpu: '1.2c',
      memory: '2349MB',
      startTime_k8s: '2023-06-09 03:47:10',
      registerTime: '2023-06-09 03:48:20',
      labels: { region: 'beijing', version: 'v1' }
    }))
    return success<PaginatedData<InstanceSearchItem>>({
      pageInfo: { Total: total, NextOffset: '0' },
      list
    })
  }),

  http.get(`${base}/instance/detail`, () => success(instanceDetail)),

  http.get(`${base}/instance/metric-dashboard`, () =>
    success(
      'http://8.147.104.101:3000/d/dcf5defe-d198-4704-9edf-6520838880e9/instance?orgId=1&refresh=1m&from=1710644821536&to=1710731221536&theme=light'
    )
  ),

  http.get(`${base}/instance/trace-dashboard`, () =>
    success(
      'http://8.147.104.101:3000/d/e968a89b-f03d-42e3-8ad3-930ae815cb0f/instance?orgId=1&refresh=1m'
    )
  ),

  http.get(`${base}/instance/config/operatorLog`, () => success({ operatorLog: true })),

  http.put(`${base}/instance/config/operatorLog`, () => success(null)),

  http.get(`${base}/instance/config/trafficDisable`, () => success({ trafficDisable: false })),

  http.put(`${base}/instance/config/trafficDisable`, () => success(null))
]
