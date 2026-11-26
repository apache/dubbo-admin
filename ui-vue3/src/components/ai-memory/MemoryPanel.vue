<!-- Licensed to the Apache Software Foundation (ASF) under one or more
     contributor license agreements. See the NOTICE file distributed with
     this work for additional information regarding copyright ownership.
     The ASF licenses this file to You under the Apache License, Version 2.0
     (the "License"); you may not use this file except in compliance with
     the License. You may obtain a copy of the License at
     http://www.apache.org/licenses/LICENSE-2.0
     Unless required by applicable law or agreed to in writing, software
     distributed under the License is distributed on an "AS IS" BASIS,
     WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. -->
<template>
  <a-modal
    :open="open"
    :title="t('aiMemory.title')"
    :width="720"
    :footer="null"
    @cancel="emit('close')"
  >
    <p>{{ t('aiMemory.retentionHint') }}</p>
    <a-alert v-if="loadError" type="error" :message="loadError" show-icon class="mb-3" />
    <a-alert v-if="mutationError" type="error" :message="mutationError" show-icon class="mb-3" />
    <a-space wrap class="mb-3">
      <a-button data-testid="memory-refresh" :loading="loading" :disabled="busy" @click="load()">{{
        t('aiMemory.refresh')
      }}</a-button>
      <a-button
        data-testid="memory-add"
        type="primary"
        :disabled="!settings?.enabled || loading || busy"
        @click="edit()"
        >{{ t('aiMemory.add') }}</a-button
      >
      <a-button
        data-testid="memory-toggle"
        :disabled="!settings || loading || busy"
        @click="confirm('toggle')"
        >{{ t(settings?.enabled ? 'aiMemory.disable' : 'aiMemory.enable') }}</a-button
      >
      <a-button
        data-testid="memory-clear"
        danger
        :disabled="!settings || loading || busy"
        @click="confirm('clear')"
        >{{ t('aiMemory.clear') }}</a-button
      >
    </a-space>
    <a-alert
      v-if="settings && !settings.enabled"
      type="info"
      :message="t('aiMemory.disabledHint')"
      class="mb-3"
    />
    <a-empty v-if="!loading && records.length === 0" :description="t('aiMemory.empty')" />
    <article
      v-for="record in records"
      :key="record.id"
      class="memory-record"
      data-testid="memory-record"
    >
      <div class="flex justify-between gap-3">
        <strong>{{ categoryLabel(record) }}</strong>
        <a-space>
          <a-button
            size="small"
            data-testid="memory-edit"
            :disabled="
              !settings?.enabled ||
              busy ||
              loading ||
              !categoryOf(record) ||
              record.status === 'source_unavailable'
            "
            @click="edit(record)"
            >{{ t('aiMemory.edit') }}</a-button
          >
          <a-button
            size="small"
            danger
            data-testid="memory-delete"
            :disabled="busy || loading"
            @click="confirm('delete', record)"
            >{{ t('aiMemory.delete') }}</a-button
          >
        </a-space>
      </div>
      <p class="memory-value">{{ record.value }}</p>
      <a-tag v-if="record.status === 'source_unavailable'" color="orange">
        {{ t('aiMemory.sourceUnavailable') }}
      </a-tag>
      <div class="text-gray-500 text-xs">
        <div>
          {{ t('aiMemory.project') }}:
          {{ record.applicability.projectRef || t('aiMemory.noProject') }} ·
          {{ t('aiMemory.task') }}:
          {{
            record.applicability.taskType
              ? t(`aiMemory.tasks.${record.applicability.taskType}`)
              : t('aiMemory.noTask')
          }}
        </div>
        <div>
          {{ t('aiMemory.source') }}: {{ record.source_session_id || t('aiMemory.explicitSource') }}
        </div>
        <div>
          {{ t('aiMemory.expiry') }}:
          {{
            record.expires_at
              ? new Date(record.expires_at).toLocaleString()
              : t('aiMemory.noExpiry')
          }}
          <a-tag
            v-if="record.expires_at && Date.parse(record.expires_at) <= Date.now()"
            color="orange"
            >{{ t('aiMemory.expired') }}</a-tag
          >
        </div>
      </div>
    </article>
    <a-button
      v-if="nextCursor"
      data-testid="memory-more"
      :loading="loading"
      :disabled="busy"
      @click="load(nextCursor)"
      >{{ t('aiMemory.more') }}</a-button
    >
    <MemoryEditor
      v-if="editorOpen && settings"
      :record="editing"
      :settings="settings"
      :options="options"
      @close="editorOpen = false"
      @saved="saved"
      @stale="load()"
    />
    <a-modal
      :open="Boolean(pending)"
      :title="t('aiMemory.confirm')"
      :confirm-loading="busy"
      :closable="!busy"
      :mask-closable="false"
      @cancel="cancel"
      @ok="mutate"
    >
      <p>
        {{
          pending?.action === 'delete'
            ? t('aiMemory.deleteHint')
            : pending?.action === 'clear'
              ? t('aiMemory.clearHint')
              : t(pending?.settings.enabled ? 'aiMemory.disableHint' : 'aiMemory.enableHint')
        }}
      </p>
    </a-modal>
  </a-modal>
</template>
<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  aiMemoryService,
  type ExplicitMemory,
  type MemoryOptions,
  type MemorySettings
} from '@/api/service/aiMemory'
import { categoryOf, errorKey } from './model'
import MemoryEditor from './MemoryEditor.vue'

const props = defineProps<{ open: boolean; options: MemoryOptions }>()
const emit = defineEmits<{ (event: 'close'): void; (event: 'changed'): void }>()
const { t } = useI18n()
const settings = ref<MemorySettings>()
const records = ref<ExplicitMemory[]>([])
const nextCursor = ref('')
const loading = ref(false)
const busy = ref(false)
const loadError = ref('')
const mutationError = ref('')
const editorOpen = ref(false)
const editing = ref<ExplicitMemory>()
const pending = ref<{
  action: 'delete' | 'clear' | 'toggle'
  record?: ExplicitMemory
  settings: MemorySettings
}>()
let requestVersion = 0
watch(
  () => props.open,
  (open) => {
    requestVersion++
    editorOpen.value = false
    pending.value = undefined
    if (open) {
      records.value = []
      settings.value = undefined
      nextCursor.value = ''
      loadError.value = ''
      mutationError.value = ''
      void load()
    }
  },
  { immediate: true }
)
async function load(cursor = '') {
  const version = ++requestVersion
  loading.value = true
  try {
    const page = await aiMemoryService.list(cursor)
    if (version !== requestVersion) return
    // A clear/disable in another tab invalidates pagination and requires a fresh view.
    if (
      cursor &&
      settings.value &&
      (settings.value.generation !== page.generation ||
        settings.value.state_version !== page.state_version)
    ) {
      await load()
      return
    }
    loadError.value = ''
    settings.value = {
      generation: page.generation,
      state_version: page.state_version,
      enabled: page.enabled
    }
    records.value = cursor
      ? [
          ...new Map(
            [...records.value, ...page.records].map((record) => [record.id, record])
          ).values()
        ]
      : page.records
    nextCursor.value = page.next_cursor || ''
  } catch (cause) {
    if (version === requestVersion) loadError.value = t(errorKey(cause))
  } finally {
    if (version === requestVersion) loading.value = false
  }
}
function categoryLabel(record: ExplicitMemory) {
  const category = categoryOf(record)
  return category ? t(`aiMemory.categories.${category}`) : t('aiMemory.unknownCategory')
}
function edit(record?: ExplicitMemory) {
  if (record?.status === 'source_unavailable') return
  editing.value = record
  editorOpen.value = true
  mutationError.value = ''
}
function confirm(action: 'delete' | 'clear' | 'toggle', record?: ExplicitMemory) {
  if (settings.value) pending.value = { action, record, settings: { ...settings.value } }
}
function cancel() {
  if (!busy.value) pending.value = undefined
}
async function saved() {
  editorOpen.value = false
  mutationError.value = ''
  await load()
  emit('changed')
}
async function mutate() {
  if (!pending.value || busy.value) return
  const { action, record, settings: baseline } = pending.value
  busy.value = true
  mutationError.value = ''
  try {
    if (action === 'delete' && record)
      await aiMemoryService.delete(record.id, baseline, record.version)
    else if (action === 'clear') await aiMemoryService.clear(baseline)
    else await aiMemoryService.setEnabled(baseline, !baseline.enabled)
    emit('changed')
  } catch (cause) {
    mutationError.value = t(errorKey(cause))
  } finally {
    pending.value = undefined
    busy.value = false
    await load()
  }
}
</script>
<style scoped>
.memory-record {
  padding: 14px 0;
  border-top: 1px solid #e5e7eb;
}
.memory-value {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  margin: 8px 0;
}
</style>
