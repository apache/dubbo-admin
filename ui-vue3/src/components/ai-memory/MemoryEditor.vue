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
    :open="true"
    :title="t(record ? 'aiMemory.edit' : 'aiMemory.add')"
    :footer="null"
    :closable="!busy"
    :mask-closable="false"
    @cancel="emit('close')"
  >
    <a-alert v-if="error" type="error" :message="error" show-icon class="mb-3" />
    <a-alert v-if="conflict" type="warning" :message="t('aiMemory.conflictHint')" class="mb-3" />
    <a-form layout="vertical" @submit="save">
      <a-form-item :label="t('aiMemory.category')">
        <a-select v-model:value="category" data-testid="memory-category" :disabled="busy">
          <a-select-option v-for="key in Object.keys(memoryCategories)" :key="key" :value="key">{{
            t(`aiMemory.categories.${key}`)
          }}</a-select-option>
        </a-select>
      </a-form-item>
      <template v-if="!preference">
        <a-form-item :label="t('aiMemory.project')" :required="category === 'background'">
          <a-select
            v-model:value="projectRef"
            data-testid="memory-project"
            allow-clear
            :disabled="busy"
            :placeholder="t('aiMemory.noProject')"
          >
            <a-select-option
              v-for="project in options.projects"
              :key="project.id"
              :value="project.id"
              >{{ project.id }}</a-select-option
            >
          </a-select>
        </a-form-item>
        <a-form-item :label="t('aiMemory.task')">
          <a-select
            v-model:value="taskType"
            data-testid="memory-task"
            allow-clear
            :disabled="busy || diagnostic"
            :placeholder="t('aiMemory.noTask')"
          >
            <a-select-option v-for="task in options.task_types" :key="task" :value="task">{{
              t(`aiMemory.tasks.${task}`)
            }}</a-select-option>
          </a-select>
        </a-form-item>
      </template>
      <a-form-item :label="t('aiMemory.value')" required>
        <a-textarea
          v-model:value="value"
          data-testid="memory-value"
          :rows="4"
          :disabled="busy || category === 'readOnly'"
        />
        <span v-if="category === 'readOnly'" class="text-gray-500">{{
          t('aiMemory.readOnlyHint')
        }}</span>
      </a-form-item>
      <a-form-item :label="t('aiMemory.expiry')">
        <input
          v-model="expiry"
          data-testid="memory-expiry"
          type="datetime-local"
          :disabled="busy"
          class="expiry-input"
        />
        <div class="text-gray-500">{{ t('aiMemory.expiryHint') }}</div>
      </a-form-item>
      <p>{{ t('aiMemory.confirmSave') }}</p>
      <a-space>
        <a-button @click="emit('close')" :disabled="busy">{{ t('aiMemory.cancel') }}</a-button>
        <a-button
          type="primary"
          html-type="submit"
          data-testid="memory-save"
          :loading="busy"
          :disabled="conflict || !settings.enabled || !category"
          >{{ t('aiMemory.save') }}</a-button
        >
      </a-space>
    </a-form>
  </a-modal>
</template>
<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  AIAPIError,
  aiMemoryService,
  type ExplicitMemory,
  type MemoryOptions,
  type MemorySettings,
  type TaskType
} from '@/api/service/aiMemory'
import {
  applicabilityFor,
  categoryOf,
  errorKey,
  memoryCategories,
  memoryIdentity,
  type MemoryCategory
} from './model'

const props = defineProps<{
  record?: ExplicitMemory
  settings: MemorySettings
  options: MemoryOptions
}>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'saved'): void
  (event: 'stale'): void
}>()
const { t } = useI18n()
const category = ref<MemoryCategory>(
  props.record ? categoryOf(props.record) || 'language' : 'language'
)
const value = ref(props.record?.value || '')
const projectRef = ref(props.record?.applicability.projectRef)
const taskType = ref<TaskType | undefined>(props.record?.applicability.taskType)
const expiry = ref(toLocalDate(props.record?.expires_at))
const busy = ref(false)
const conflict = ref(false)
const error = ref('')
// Keep the editor's original control versions until the user reviews a fresh form.
const baseline = { ...props.settings }
const preference = computed(() => category.value === 'language' || category.value === 'style')
const diagnostic = computed(() => category.value === 'priority' || category.value === 'readOnly')
watch(category, () => {
  projectRef.value = undefined
  taskType.value = diagnostic.value ? 'diagnosis' : undefined
  value.value = category.value === 'readOnly' ? 'true' : ''
})
function toLocalDate(date?: string | null) {
  if (!date) return ''
  const parsed = new Date(date)
  return new Date(parsed.getTime() - parsed.getTimezoneOffset() * 60000).toISOString().slice(0, 16)
}
async function save() {
  if (busy.value || conflict.value) return
  error.value = ''
  if (
    !value.value.trim() ||
    (category.value === 'background' && !projectRef.value) ||
    (expiry.value && new Date(expiry.value).getTime() <= Date.now())
  ) {
    error.value = t('aiMemory.validation')
    return
  }
  busy.value = true
  try {
    await aiMemoryService.save(
      {
        generation: baseline.generation,
        state_version: baseline.state_version,
        ...(props.record ? { version: props.record.version } : {}),
        scope_type: 'user',
        ...memoryIdentity(category.value),
        value: value.value.trim(),
        applicability: applicabilityFor(category.value, {
          projectRef: projectRef.value,
          taskType: taskType.value
        }),
        ...(props.record?.source_session_id
          ? { source_session_id: props.record.source_session_id }
          : {}),
        expires_at: expiry.value ? new Date(expiry.value).toISOString() : null
      },
      props.record?.id
    )
    emit('saved')
  } catch (cause) {
    error.value = t(errorKey(cause))
    if (
      cause instanceof AIAPIError &&
      ['MEMORY_VERSION_CONFLICT', 'MEMORY_NOT_FOUND', 'MEMORY_DISABLED'].includes(cause.code)
    ) {
      conflict.value = true
      emit('stale')
    }
  } finally {
    busy.value = false
  }
}
</script>
<style scoped>
.expiry-input {
  width: 100%;
  padding: 5px 10px;
  border: 1px solid #d9d9d9;
  border-radius: 6px;
}
</style>
