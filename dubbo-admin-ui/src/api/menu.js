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

const Menu = [
  { title: 'serviceSearch', path: '/service', icon: 'search' },
  {
    title: 'trafficManagement',
    icon: 'show_chart',
    group: 'traffic',
    items: [
      { title: 'trafficTimeout', path: '/traffic/timeout' },
      { title: 'trafficRetry', path: '/traffic/retry' },
      { title: 'trafficRegion', path: '/traffic/region' },
      { title: 'trafficIsolation', path: '/traffic/isolation' },
      { title: 'trafficWeight', path: '/traffic/weight' },
      { title: 'trafficArguments', path: '/traffic/arguments' },
      { title: 'trafficMock', path: '/traffic/mock' },
      { title: 'trafficAccesslog', path: '/traffic/accesslog' },
      { title: 'trafficHost', path: '/traffic/host' }
    ]
  },
  {
    title: 'serviceGovernance',
    icon: 'edit',
    group: 'governance',
    items: [
      { title: 'routingRule', path: '/governance/routingRule' },
      { title: 'tagRule', path: '/governance/tagRule' },
      { title: 'dynamicConfig', path: '/governance/config' }
    ]
  },
  { title: 'serviceTest', path: '/test', icon: 'code' },
  { title: 'serviceMock', path: '/mock/rule', icon: 'build' },
  { title: 'serviceMetrics', path: '/metrics/index', icon: 'show_chart' }
  // { title: 'configManage', path: '/management', icon: 'build' }
  // { title: 'apiDocs', path: '/apiDocs', icon: 'code' },
]

export default Menu
