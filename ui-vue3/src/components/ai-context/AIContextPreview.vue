<!--
  ~ Licensed to the Apache Software Foundation (ASF) under one or more
  ~ contributor license agreements.  See the NOTICE file distributed with
  ~ this work for additional information regarding copyright ownership.
  ~ The ASF licenses this file to You under the Apache License, Version 2.0
  ~ (the "License"); you may not use this file except in compliance with
  ~ the License.  You may obtain a copy of the License at
  ~
  ~     http://www.apache.org/licenses/LICENSE-2.0
  ~
  ~ Unless required by applicable law or agreed to in writing, software
  ~ distributed under the License is distributed on an "AS IS" BASIS,
  ~ WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  ~ See the License for the specific language governing permissions and
  ~ limitations under the License.
-->

<template>
  <div class="ai-context-toolbar">
    <a-popover placement="topLeft" trigger="click" @open-change="handleOpenChange">
      <template #content>
        <div class="ai-context-popover">
          <div class="ai-context-header">
            <div>
              <div class="ai-context-title">{{ t('aiContext.title') }}</div>
              <div class="ai-context-subtitle">{{ t('aiContext.nextMessage') }}</div>
            </div>
            <a-switch
              size="small"
              :checked="enabled"
              :aria-label="t('aiContext.include')"
              @update:checked="updateEnabled"
            />
          </div>

          <template v-if="snapshot">
            <div class="ai-context-details" :class="{ disabled: !enabled }">
              <div class="ai-context-row">
                <span class="ai-context-label">{{ t('aiContext.page') }}</span>
                <span
                  class="ai-context-value"
                  :title="snapshot.page.fullPath || snapshot.page.path"
                >
                  {{ snapshot.page.fullPath || snapshot.page.path }}
                </span>
              </div>
              <div class="ai-context-row">
                <span class="ai-context-label">{{ t('aiContext.mesh') }}</span>
                <span class="ai-context-value">{{
                  snapshot.scope.mesh || t('aiContext.notSet')
                }}</span>
              </div>
              <div class="ai-context-row">
                <span class="ai-context-label">{{ t('aiContext.locale') }}</span>
                <span class="ai-context-value">{{
                  snapshot.global.locale || t('aiContext.notSet')
                }}</span>
              </div>
            </div>

            <div v-if="scopeEntries.length" class="ai-context-group">
              <div class="ai-context-group-title">{{ t('aiContext.resourceScope') }}</div>
              <div class="ai-context-tags">
                <a-tag v-for="entry in scopeEntries" :key="entry.key" color="blue">
                  {{ entry.label }}: {{ entry.value }}
                </a-tag>
              </div>
            </div>

            <div class="ai-context-group">
              <div class="ai-context-group-title">{{ t('aiContext.sections') }}</div>
              <div v-if="optionalSectionIds.length" class="ai-context-sections">
                <label
                  v-for="sectionId in optionalSectionIds"
                  :key="sectionId"
                  class="ai-context-section"
                >
                  <a-checkbox
                    :checked="!excludedSectionIds.includes(sectionId)"
                    :disabled="!enabled"
                    @change="toggleSection(sectionId)"
                  />
                  <span class="ai-context-section-copy">
                    <span class="ai-context-section-name">{{ getSectionLabel(sectionId) }}</span>
                  </span>
                </label>
              </div>
              <div v-else class="ai-context-empty">{{ t('aiContext.noSections') }}</div>
            </div>

            <div v-if="snapshot.truncation?.truncated" class="ai-context-warning">
              {{ t('aiContext.truncated') }}
            </div>
          </template>

          <div v-else class="ai-context-empty">{{ t('aiContext.unavailable') }}</div>
        </div>
      </template>

      <button
        type="button"
        class="ai-context-trigger"
        :class="{ disabled: !enabled }"
        :aria-pressed="enabled"
      >
        <PaperClipOutlined />
        <span>{{ t('aiContext.title') }}</span>
        <span v-if="includedSectionCount" class="ai-context-count">{{ includedSectionCount }}</span>
      </button>
    </a-popover>

    <span class="ai-context-summary" :title="contextSummary">{{ contextSummary }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { PaperClipOutlined } from '@ant-design/icons-vue'
import { AI_CONTEXT_UNSAVED_CHANGES_SECTION_ID } from '@/ai-context'
import type { AIContextSnapshot } from '@/ai-context'

const props = defineProps<{
  snapshot?: AIContextSnapshot
  enabled: boolean
  excludedSectionIds: string[]
}>()

const emit = defineEmits<{
  (event: 'update:enabled', value: boolean): void
  (event: 'update:excludedSectionIds', value: string[]): void
  (event: 'refresh'): void
}>()

const { t } = useI18n()

const optionalSectionIds = computed(() => {
  const sectionIds = props.snapshot?.evidence?.map((section) => section.id) || []
  if (props.snapshot?.state?.unsavedChanges) {
    sectionIds.unshift(AI_CONTEXT_UNSAVED_CHANGES_SECTION_ID)
  }
  return [...new Set(sectionIds)]
})

const includedSectionCount = computed(() => {
  return optionalSectionIds.value.filter((id) => !props.excludedSectionIds.includes(id)).length
})

const contextSummary = computed(() => {
  if (!props.enabled) return t('aiContext.disabled')
  if (!props.snapshot) return t('aiContext.unavailable')

  const page = props.snapshot.page.routeName
    ? t(props.snapshot.page.routeName)
    : props.snapshot.page.path
  const mesh = props.snapshot.scope.mesh || t('aiContext.notSet')
  return `${page} · ${mesh}`
})

const scopeEntries = computed(() => {
  if (!props.snapshot) return []

  return Object.entries(props.snapshot.scope)
    .filter(([key, value]) => key !== 'mesh' && value !== undefined && value !== '')
    .map(([key, value]) => ({
      key,
      label: t(`aiContext.scope.${key}`),
      value: String(value)
    }))
})

const updateEnabled = (value: boolean) => {
  emit('update:enabled', value)
}

const getSectionLabel = (sectionId: string): string => {
  const translationKey = `aiContext.section.${sectionId}`
  const translated = t(translationKey)
  return translated === translationKey ? sectionId : translated
}

const toggleSection = (sectionId: string) => {
  const excluded = new Set(props.excludedSectionIds)
  if (excluded.has(sectionId)) excluded.delete(sectionId)
  else excluded.add(sectionId)
  emit('update:excludedSectionIds', [...excluded])
}

const handleOpenChange = (open: boolean) => {
  if (open) emit('refresh')
}
</script>

<style scoped>
.ai-context-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  height: 32px;
  margin-top: 8px;
}

.ai-context-trigger {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 6px;
  height: 28px;
  padding: 0 9px;
  color: #334155;
  font-size: 12px;
  line-height: 1;
  background: #f8fafc;
  border: 1px solid #cbd5e1;
  border-radius: 6px;
  cursor: pointer;
}

.ai-context-trigger:hover {
  color: #1677ff;
  background: #eff6ff;
  border-color: #91caff;
}

.ai-context-trigger.disabled {
  color: #94a3b8;
  background: #f8fafc;
  border-color: #e2e8f0;
}

.ai-context-count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  color: #ffffff;
  font-size: 11px;
  background: #1677ff;
  border-radius: 9px;
}

.ai-context-summary {
  min-width: 0;
  overflow: hidden;
  color: #64748b;
  font-size: 12px;
  line-height: 20px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ai-context-popover {
  width: min(360px, calc(100vw - 48px));
}

.ai-context-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24px;
  padding-bottom: 12px;
  border-bottom: 1px solid #e2e8f0;
}

.ai-context-title {
  color: #0f172a;
  font-size: 14px;
  font-weight: 600;
  line-height: 20px;
}

.ai-context-subtitle {
  color: #64748b;
  font-size: 12px;
  line-height: 18px;
}

.ai-context-details {
  padding: 10px 0;
  border-bottom: 1px solid #e2e8f0;
}

.ai-context-details.disabled {
  opacity: 0.5;
}

.ai-context-row {
  display: grid;
  grid-template-columns: 64px minmax(0, 1fr);
  gap: 12px;
  min-height: 26px;
  align-items: center;
}

.ai-context-label {
  color: #64748b;
  font-size: 12px;
}

.ai-context-value {
  overflow: hidden;
  color: #1e293b;
  font-size: 12px;
  text-align: right;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ai-context-group {
  padding-top: 10px;
}

.ai-context-group-title {
  margin-bottom: 6px;
  color: #475569;
  font-size: 12px;
  font-weight: 600;
}

.ai-context-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.ai-context-sections {
  display: grid;
  gap: 4px;
}

.ai-context-section {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 38px;
  padding: 4px 6px;
  border-radius: 4px;
  cursor: pointer;
}

.ai-context-section:hover {
  background: #f8fafc;
}

.ai-context-section-copy {
  display: flex;
  min-width: 0;
  flex-direction: column;
}

.ai-context-section-name {
  overflow: hidden;
  color: #1e293b;
  font-size: 12px;
  line-height: 18px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ai-context-empty {
  padding: 12px 0 4px;
  color: #94a3b8;
  font-size: 12px;
  text-align: center;
}

.ai-context-warning {
  margin-top: 10px;
  padding: 6px 8px;
  color: #92400e;
  font-size: 12px;
  background: #fffbeb;
  border-left: 3px solid #f59e0b;
}
</style>
