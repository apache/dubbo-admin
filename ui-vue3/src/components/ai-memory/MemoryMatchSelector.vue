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
  <div class="memory-toolbar">
    <a-space wrap>
      <a-select
        :value="modelValue.projectRef"
        data-testid="chat-project"
        :aria-label="t('aiMemory.project')"
        :placeholder="t('aiMemory.noProject')"
        allow-clear
        :disabled="!available"
        style="min-width: 150px"
        @update:value="updateProject"
      >
        <a-select-option
          v-for="project in options.projects"
          :key="project.id"
          :value="project.id"
          >{{ project.id }}</a-select-option
        >
      </a-select>
      <a-select
        :value="modelValue.taskType"
        data-testid="chat-task"
        :aria-label="t('aiMemory.task')"
        :placeholder="t('aiMemory.noTask')"
        allow-clear
        :disabled="!available"
        style="min-width: 130px"
        @update:value="updateTask"
      >
        <a-select-option v-for="task in options.task_types" :key="task" :value="task">{{
          t(`aiMemory.tasks.${task}`)
        }}</a-select-option>
      </a-select>
      <a-button data-testid="manage-memory" @click="panelOpen = true">{{
        t('aiMemory.title')
      }}</a-button>
    </a-space>
    <p class="text-xs text-gray-500 mt-1">{{ t('aiMemory.selectionHint') }}</p>
    <a-alert v-if="error" :message="error" type="warning" show-icon>
      <template #action
        ><a-button size="small" @click="load">{{ t('aiMemory.refresh') }}</a-button></template
      >
    </a-alert>
    <MemoryPanel :open="panelOpen" :options="options" @close="panelOpen = false" />
  </div>
</template>
<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  aiMemoryService,
  type MatchContext,
  type MemoryOptions,
  type TaskType
} from '@/api/service/aiMemory'
import { errorKey } from './model'
import MemoryPanel from './MemoryPanel.vue'
const props = defineProps<{ modelValue: MatchContext; active: boolean }>()
const emit = defineEmits<{ (event: 'update:modelValue', value: MatchContext): void }>()
const { t } = useI18n()
const options = ref<MemoryOptions>({ projects: [], task_types: [] })
const panelOpen = ref(false)
const error = ref('')
const available = ref(false)
let revision = 0
watch(
  () => props.active,
  (active) => {
    if (active) void load()
    else {
      revision++
      panelOpen.value = false
    }
  },
  { immediate: true }
)
function update(value: MatchContext) {
  emit('update:modelValue', { ...props.modelValue, ...value })
}
function updateProject(projectRef?: string) {
  update({ projectRef })
}
function updateTask(taskType?: TaskType) {
  update({ taskType })
}
async function load() {
  const current = ++revision
  available.value = false
  error.value = ''
  try {
    const result = await aiMemoryService.options()
    if (current !== revision) return
    options.value = result
    available.value = true
    if (
      props.modelValue.projectRef &&
      !result.projects.some((p) => p.id === props.modelValue.projectRef)
    )
      emit('update:modelValue', {})
  } catch (cause) {
    if (current !== revision) return
    options.value = { projects: [], task_types: [] }
    emit('update:modelValue', {})
    error.value = t(errorKey(cause))
  }
}
</script>
<style scoped>
.memory-toolbar {
  padding: 8px 0;
  border-top: 1px solid #e5e7eb;
}
</style>
