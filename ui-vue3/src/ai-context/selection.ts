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

import type { AIContextSnapshot } from './types'

export const AI_CONTEXT_UNSAVED_CHANGES_SECTION_ID = 'unsaved-changes'

export interface AIContextSelectionOptions {
  enabled?: boolean
  excludedSectionIds?: Iterable<string>
}

export const selectAIContextSnapshot = (
  snapshot: AIContextSnapshot,
  options: AIContextSelectionOptions = {}
): AIContextSnapshot | undefined => {
  if (options.enabled === false) return undefined

  const excludedSectionIds = new Set(options.excludedSectionIds)
  if (!excludedSectionIds.size) return snapshot

  const selected: AIContextSnapshot = {
    ...snapshot
  }

  if (snapshot.evidence?.length) {
    const evidence = snapshot.evidence.filter((section) => !excludedSectionIds.has(section.id))
    selected.evidence = evidence
    if (!evidence.length) delete selected.evidence
  }

  if (
    excludedSectionIds.has(AI_CONTEXT_UNSAVED_CHANGES_SECTION_ID) &&
    snapshot.state?.unsavedChanges
  ) {
    const state = { ...snapshot.state }
    delete state.unsavedChanges
    selected.state = state
    if (!Object.keys(state).length) delete selected.state
  }

  return selected
}
