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

import type { AIContextContribution } from '../types'

const normalizeIdentifier = (value: unknown): string | undefined => {
  const candidate = Array.isArray(value) ? value[0] : value
  if (typeof candidate !== 'string') return undefined

  const normalized = candidate.trim()
  return normalized || undefined
}

export const createApplicationResourceContribution = (
  application: unknown
): AIContextContribution | undefined => {
  const normalizedApplication = normalizeIdentifier(application)
  if (!normalizedApplication) return undefined

  return {
    scope: {
      application: normalizedApplication
    }
  }
}

export const createServiceResourceContribution = (
  service: unknown,
  group?: unknown,
  version?: unknown
): AIContextContribution | undefined => {
  const normalizedService = normalizeIdentifier(service)
  if (!normalizedService) return undefined

  const normalizedGroup = normalizeIdentifier(group)
  const normalizedVersion = normalizeIdentifier(version)
  const selection = {
    ...(normalizedGroup ? { group: normalizedGroup } : {}),
    ...(normalizedVersion ? { version: normalizedVersion } : {})
  }

  return {
    scope: {
      service: normalizedService
    },
    ...(Object.keys(selection).length ? { state: { selection } } : {})
  }
}

export const createInstanceResourceContribution = (
  instance: unknown,
  application?: unknown
): AIContextContribution | undefined => {
  const normalizedInstance = normalizeIdentifier(instance)
  if (!normalizedInstance) return undefined

  const normalizedApplication = normalizeIdentifier(application)
  return {
    scope: {
      instance: normalizedInstance,
      ...(normalizedApplication ? { application: normalizedApplication } : {})
    }
  }
}
