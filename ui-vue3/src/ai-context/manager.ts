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

import { createAIContextSnapshot, type PrioritizedContribution } from './snapshot'
import type {
  AIContextBase,
  AIContextProvider,
  AIContextSnapshot,
  AIContextSnapshotOptions
} from './types'

export type AIContextBaseCollector = () => AIContextBase

export class AIContextManager {
  private readonly providers = new Map<symbol, AIContextProvider>()

  constructor(private readonly collectBase: AIContextBaseCollector) {}

  register(provider: AIContextProvider): () => void {
    const token = Symbol(provider.id)
    this.providers.set(token, provider)

    return () => {
      this.providers.delete(token)
    }
  }

  snapshot(options: AIContextSnapshotOptions = {}): AIContextSnapshot {
    const base = this.collectBase()
    const routeKey = base.page.fullPath || base.page.path
    const contributions: PrioritizedContribution[] = []

    for (const provider of this.providers.values()) {
      const providerRouteKey = provider.routeKey?.()
      if (providerRouteKey && providerRouteKey !== routeKey) continue

      try {
        const contribution = provider.collect()
        if (!contribution) continue
        contributions.push({
          id: provider.id,
          priority: provider.priority ?? 0,
          contribution
        })
      } catch {
        // A page provider must not prevent the user from sending a message.
      }
    }

    return createAIContextSnapshot(base, contributions, options)
  }

  clear(): void {
    this.providers.clear()
  }

  size(): number {
    return this.providers.size
  }
}
