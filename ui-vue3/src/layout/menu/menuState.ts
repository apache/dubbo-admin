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

import type { RouterMeta } from '@/router/RouterMeta'
import type { RouteRecordType } from '@/router/defaultRoutes'

export function findRouteByPath(
  routes: readonly RouteRecordType[],
  path: string
): RouteRecordType | undefined {
  for (const route of routes) {
    if (route.path === path) return route
    const child = findRouteByPath(route.children || [], path)
    if (child) return child
  }
}

export function activeMenuRoute(
  routes: readonly RouteRecordType[],
  meta: RouterMeta
): RouteRecordType | undefined {
  if (meta.back) {
    const route = findRouteByPath(routes, meta.back)
    if (route) return route
  }
  let route = findRouteByKey(routes, meta._router_key)
  while (route?.meta?.hidden || route?.meta?.tab) {
    route = route.meta.parent
  }
  return route
}

function findRouteByKey(
  routes: readonly RouteRecordType[],
  key: string | undefined
): RouteRecordType | undefined {
  if (!key) return undefined
  for (const route of routes) {
    if (route.meta?._router_key === key) return route
    const child = findRouteByKey(route.children || [], key)
    if (child) return child
  }
}

export function ancestorMenuKeys(route: RouteRecordType | undefined): string[] {
  const keys: string[] = []
  let current = route?.meta?.parent
  while (current) {
    if (current.meta?._router_key) keys.push(current.meta._router_key)
    current = current.meta?.parent
  }
  return keys
}
