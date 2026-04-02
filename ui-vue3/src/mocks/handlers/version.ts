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
import type { VersionInfo } from '@/types/api'

const versionInfo: VersionInfo = {
  gitVersion: 'dubbo-admin-',
  gitCommit: '$Format:%H$',
  gitTreeState: '',
  buildDate: '1970-01-01T00:00:00Z',
  goVersion: 'go1.20.4',
  compiler: 'gc',
  platform: 'darwin/arm64'
}

export const versionHandlers: HttpHandler[] = [
  http.get(`${base}/version`, () => success(versionInfo))
]
