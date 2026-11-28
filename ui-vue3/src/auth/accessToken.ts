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

import { issueAccessToken } from '@/api/service/login'

export class AccessTokenManager {
  private enabled = false
  private pending?: Promise<void>
  constructor(private readonly issue: () => Promise<void>) {}
  setEnabled(enabled: boolean) {
    this.enabled = enabled
  }
  isEnabled() {
    return this.enabled
  }
  refresh(): Promise<void> {
    if (!this.pending) {
      this.pending = this.issue().finally(() => {
        this.pending = undefined
      })
    }
    return this.pending
  }
}

export const accessTokenManager = new AccessTokenManager(async () => {
  await issueAccessToken()
})
