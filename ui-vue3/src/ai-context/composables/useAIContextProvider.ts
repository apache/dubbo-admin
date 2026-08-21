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

import { onActivated, onDeactivated, onUnmounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { aiContextManager } from '../instance'
import type { AIContextProvider } from '../types'

export type PageAIContextProvider = Omit<AIContextProvider, 'routeKey'>

export const useAIContextProvider = (provider: PageAIContextProvider): (() => void) => {
  const route = useRoute()
  let routeKey = route.fullPath
  let unregister: (() => void) | undefined
  let active = true

  const register = () => {
    if (unregister) return
    routeKey = route.fullPath
    unregister = aiContextManager.register({
      ...provider,
      routeKey: () => routeKey
    })
  }

  const remove = () => {
    unregister?.()
    unregister = undefined
  }

  register()

  watch(
    () => route.fullPath,
    (value) => {
      if (active) routeKey = value
    }
  )

  onActivated(() => {
    active = true
    register()
  })

  onDeactivated(() => {
    active = false
    remove()
  })

  onUnmounted(remove)
  return remove
}
