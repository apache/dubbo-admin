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

import { describe, expect, it } from 'vitest'
import { activeMenuRoute, ancestorMenuKeys, findRouteByPath } from './menuState'

describe('traffic menu state', () => {
  const traffic: any = {
    path: '/traffic',
    meta: { _router_key: 'traffic' },
    children: []
  }
  const routes: any[] = [traffic]
  for (const [name, key] of [
    ['affinityRule', 'affinity'],
    ['scriptRule', 'script'],
    ['routingRule', 'condition']
  ]) {
    traffic.children.push({
      path: `/traffic/${name}`,
      meta: { _router_key: key, parent: traffic }
    })
  }

  it.each([
    ['/traffic/affinityRule/edit/:ruleName?', '/traffic/affinityRule'],
    ['/traffic/scriptRule/edit/:ruleName?', '/traffic/scriptRule'],
    ['/traffic/updateRoutingRule/updateByFormView/:ruleName', '/traffic/routingRule']
  ])('maps hidden editor %s to its own menu item %s', (editorPath, menuPath) => {
    const editorRoute: any = {
      path: editorPath,
      meta: { hidden: true, back: menuPath, parent: traffic, _router_key: `${menuPath}-editor` }
    }
    traffic.children.push(editorRoute)
    const editor = findRouteByPath(routes, editorPath)
    const menu = findRouteByPath(routes, menuPath)
    expect(editor).toBeDefined()
    expect(activeMenuRoute(routes, editor!.meta!)).toBe(menu)
    expect(ancestorMenuKeys(menu)).toContain(findRouteByPath(routes, '/traffic')?.meta?._router_key)
  })
})
