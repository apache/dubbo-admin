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
import axios from 'axios'
import { aiClient, AIAPIError } from './aiTransport'
export { AIAPIError } from './aiTransport'

export type TaskType = 'diagnosis' | 'configuration' | 'inspection'
export interface MatchContext {
  projectRef?: string
  taskType?: TaskType
}
export interface AIChatRequestContext {
  matchContext?: MatchContext
}
export interface MemorySettings {
  generation: number
  state_version: number
  enabled: boolean
}
export interface MemoryOptions {
  projects: { id: string }[]
  task_types: TaskType[]
}
export interface MemoryWrite {
  generation: number
  state_version: number
  version?: number
  scope_type: 'user'
  memory_type: 'preference' | 'task_context' | 'diagnostic_constraint'
  fact_key: string
  value: string
  applicability: MatchContext
  source_session_id?: string
  expires_at?: string | null
}
export interface ExplicitMemory {
  id: string
  memory_type: MemoryWrite['memory_type']
  fact_key: string
  value: string
  applicability: MatchContext
  status: string
  version: number
  provenance: string
  source_session_id?: string
  expires_at?: string | null
  updated_at: string
}
export interface MemoryPage extends MemorySettings {
  records: ExplicitMemory[]
  next_cursor?: string
}

// AI endpoints use their own envelope, unlike the Console request interceptor.
async function request<T>(method: string, url: string, data?: unknown): Promise<T> {
  try {
    const response = await aiClient.request<{ data: T }>({
      method,
      url,
      data,
      timeout: 30000
    })
    return response.data.data
  } catch (error) {
    if (axios.isAxiosError(error)) {
      throw new AIAPIError(
        error.response?.data?.code || 'NETWORK_ERROR',
        error.response?.status || 0
      )
    }
    throw error
  }
}
export const aiMemoryService = {
  options: () => request<MemoryOptions>('GET', '/memory-options'),
  settings: () => request<MemorySettings>('GET', '/memory-settings'),
  list: (cursor = '') =>
    request<MemoryPage>('GET', `/memories?limit=32&cursor=${encodeURIComponent(cursor)}`),
  save: (data: MemoryWrite, id?: string) =>
    request<ExplicitMemory>(
      id ? 'PUT' : 'POST',
      id ? `/memories/${encodeURIComponent(id)}` : '/memories',
      data
    ),
  delete: (id: string, settings: MemorySettings, version: number) =>
    request<void>('DELETE', `/memories/${encodeURIComponent(id)}`, {
      generation: settings.generation,
      state_version: settings.state_version,
      version
    }),
  clear: (settings: MemorySettings) =>
    request<MemorySettings>('POST', '/memories/clear', {
      generation: settings.generation,
      state_version: settings.state_version
    }),
  setEnabled: (settings: MemorySettings, enabled: boolean) =>
    request<MemorySettings>('PUT', '/memory-settings', {
      generation: settings.generation,
      state_version: settings.state_version,
      enabled
    })
}
