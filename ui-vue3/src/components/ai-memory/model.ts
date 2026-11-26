/* Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements. See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License. You may obtain a copy of the License at
 * http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 */
import {
  AIAPIError,
  type ExplicitMemory,
  type MatchContext,
  type MemoryWrite
} from '@/api/service/aiMemory'

export const memoryCategories = {
  language: { type: 'preference', key: 'response.language' },
  style: { type: 'preference', key: 'response.style' },
  background: { type: 'task_context', key: 'project.background' },
  priority: { type: 'diagnostic_constraint', key: 'diagnostic.priority' },
  readOnly: { type: 'diagnostic_constraint', key: 'diagnostic.read_only' }
} as const
export type MemoryCategory = keyof typeof memoryCategories
export function categoryOf(record: ExplicitMemory): MemoryCategory | undefined {
  return (Object.keys(memoryCategories) as MemoryCategory[]).find(
    (key) =>
      memoryCategories[key].type === record.memory_type &&
      memoryCategories[key].key === record.fact_key
  )
}
export function applicabilityFor(category: MemoryCategory, match: MatchContext): MatchContext {
  if (category === 'language' || category === 'style') return {}
  return {
    ...(match.projectRef ? { projectRef: match.projectRef } : {}),
    ...(category === 'priority' || category === 'readOnly'
      ? { taskType: 'diagnosis' as const }
      : match.taskType
        ? { taskType: match.taskType }
        : {})
  }
}
export function errorKey(error: unknown): string {
  const code = error instanceof AIAPIError ? error.code : 'NETWORK_ERROR'
  return `aiMemory.errors.${
    [
      'AUTHENTICATION_REQUIRED',
      'MEMORY_INVALID',
      'MEMORY_VERSION_CONFLICT',
      'MEMORY_DISABLED',
      'MEMORY_NOT_FOUND',
      'RESOURCE_NOT_ACCESSIBLE',
      'RESOURCE_EXHAUSTED',
      'MEMORY_UNAVAILABLE',
      'MEMORY_OPTIONS_UNAVAILABLE'
    ].includes(code)
      ? code
      : 'NETWORK_ERROR'
  }`
}
export function memoryIdentity(
  category: MemoryCategory
): Pick<MemoryWrite, 'memory_type' | 'fact_key'> {
  return { memory_type: memoryCategories[category].type, fact_key: memoryCategories[category].key }
}
