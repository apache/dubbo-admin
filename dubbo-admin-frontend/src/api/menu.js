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
  { title: 'Service Search', path: '/service', icon: 'search' },
  {
    title: 'Service Governance',
    icon: 'edit',
    group: 'governance',
    items: [
      { title: 'Routing Rule', path: '/governance/routingRule' },
      { title: 'Tag Rule', path: '/governance/tagRule', badge: 'new'},
      { title: 'Dynamic Config', path: '/governance/config' },
      { title: 'Access Control', path: '/governance/access' },
      { title: 'Weight Adjust', path: '/governance/weight' },
      { title: 'Load Balance', path: '/governance/loadbalance' }
    ]
  },
  { title: 'Service Test', path: '/test', icon: 'code', badge: 'feature' },
  { title: 'Service Mock', path: '/mock', icon: 'build', badge: 'feature' },
  { title: 'Metrics', path: '/metrics', icon: 'show_chart', badge: 'feature' }

]

export default Menu
