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

import {
  getAuthProviders,
  getUserInfo,
  type AuthConfiguration,
  type Principal
} from '@/api/service/login'
import { updateAuthState } from '@/utils/AuthUtil'

let configurationPromise: Promise<AuthConfiguration> | undefined

export function providerLoginURL(providerID: string): string {
  return `/api/v1/auth/providers/${encodeURIComponent(providerID)}/login`
}

export async function loadAuthConfiguration(force = false): Promise<AuthConfiguration> {
  if (!configurationPromise || force) {
    configurationPromise = getAuthProviders()
      .then(({ data }) => data)
      .catch((error) => {
        configurationPromise = undefined
        throw error
      })
  }
  return configurationPromise
}

export async function syncAuthenticatedPrincipal(): Promise<Principal> {
  const { data } = await getUserInfo()
  updateAuthState(true, data.username)
  return data
}
