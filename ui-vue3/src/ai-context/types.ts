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

export const AI_CONTEXT_VERSION = 1 as const
export const AI_CONTEXT_DEFAULT_MAX_BYTES = 8 * 1024

export interface AIContextGlobal {
  locale?: string
}

export interface AIContextPage {
  routeName?: string
  path: string
  fullPath?: string
  activeTab?: string
  params?: Record<string, unknown>
  query?: Record<string, unknown>
}

export interface AIContextScope {
  mesh: string
  application?: string
  service?: string
  instance?: string
  rule?: string
}

export interface AIContextState {
  filters?: Record<string, unknown>
  selection?: Record<string, unknown>
  unsavedChanges?: Record<string, unknown>
}

export interface AIContextSection {
  id: string
  source: string
  capturedAt?: string
  priority?: number
  data: Record<string, unknown>
}

export interface AIContextContribution {
  scope?: Partial<AIContextScope>
  state?: AIContextState
  evidence?: AIContextSection | AIContextSection[]
}

export interface AIContextTruncation {
  truncated: boolean
  omittedSections: string[]
}

export interface AIContextSnapshot {
  version: typeof AI_CONTEXT_VERSION
  capturedAt: string
  global: AIContextGlobal
  page: AIContextPage
  scope: AIContextScope
  state?: AIContextState
  evidence?: AIContextSection[]
  truncation?: AIContextTruncation
}

export interface AIContextBase {
  global: AIContextGlobal
  page: AIContextPage
  scope: AIContextScope
}

export interface AIContextProvider {
  id: string
  priority?: number
  routeKey?: () => string | undefined
  collect: () => AIContextContribution | undefined
}

export interface AIContextSnapshotOptions {
  maxBytes?: number
  now?: () => Date
}
