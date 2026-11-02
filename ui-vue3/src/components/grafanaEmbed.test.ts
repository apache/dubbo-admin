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
import { embedDashboardUrl } from './grafanaEmbed'

describe('embedDashboardUrl', () => {
  it('hides Grafana navigation while preserving log dashboard variables', () => {
    const result = new URL(
      embedDashboardUrl('http://grafana/d/logs?var-application=shop-user&from=123&to=456')
    )
    expect(result.searchParams.get('kiosk')).toBe('1')
    expect(result.searchParams.get('var-application')).toBe('shop-user')
    expect(result.searchParams.get('from')).toBe('123')
    expect(result.searchParams.get('to')).toBe('456')
    expect(result.searchParams.has('var-adminOrigin')).toBe(false)
  })

  it('preserves an existing Grafana kiosk setting', () => {
    const result = new URL(embedDashboardUrl('http://grafana/d/metrics?kiosk=tv'))
    expect(result.searchParams.get('kiosk')).toBe('tv')
    expect(result.searchParams.has('var-adminOrigin')).toBe(false)
  })
})
