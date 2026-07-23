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
  <AiChatDrawer
    v-model:open="localOpen"
    :service="aiService"
    :collect-context="consumeSelectedContext"
    :labels="labels"
    :suggestions="suggestions"
  >
    <template #context>
      <AIContextPreview
        v-model:enabled="contextEnabled"
        v-model:excluded-section-ids="excludedContextSectionIds"
        :snapshot="contextSnapshot"
        @refresh="refreshContextSnapshot"
      />
    </template>
  </AiChatDrawer>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { aiService } from '@/api/service/ai'
import {
  AI_CONTEXT_UNSAVED_CHANGES_SECTION_ID,
  aiContextManager,
  selectAIContextSnapshot
} from '@/ai-context'
import type { AIContextSnapshot } from '@/ai-context'
import AIContextPreview from './ai-context/AIContextPreview.vue'
import { AiChatDrawer, type AIChatLabels, type ChatSuggestion } from './ai-chat'

const props = withDefaults(
  defineProps<{
    agentDrawerOpen?: boolean
  }>(),
  {
    agentDrawerOpen: false
  }
)

const emit = defineEmits<{
  (event: 'update:agentDrawerOpen', value: boolean): void
}>()

const localOpen = ref(props.agentDrawerOpen)
const contextSnapshot = ref<AIContextSnapshot>()
const contextEnabled = ref(true)
const excludedContextSectionIds = ref<string[]>([])

const labels: Partial<AIChatLabels> = {
  title: 'Dubbo Admin AI',
  welcomeMessage:
    '我是Dubbo Admin的AI小助手，你可以问我任何关于kubernetes的问题，我尽量给你提供最准确的答案。',
  welcomeTagline: '✨ 奇思妙想和创新的火花',
  inputTokens: '输入 Tokens',
  outputTokens: '输出 Tokens',
  totalTokens: '总 Tokens',
  newChat: '新对话',
  history: '对话历史',
  clearHistory: '清空历史',
  placeholder: '输入你的问题，Shift + Enter 换行',
  thinking: '正在思考...',
  errorTitle: '出现错误',
  retry: '重试',
  historyTitle: '对话历史',
  emptyHistory: '暂无对话历史',
  session: '会话',
  messageCount: '消息数',
  newSession: '新会话',
  delete: '删除',
  createSessionFailed: '创建会话失败',
  createSessionUnavailable: '无法创建会话，请稍后再试',
  sessionDeleted: '会话已删除',
  deleteSessionFailed: '删除会话失败',
  noRetryMessage: '没有可重试的消息',
  waitForResponse: '请等待当前消息处理完成',
  unknownStreamError: '处理消息时发生未知错误',
  parseStreamError: '解析服务器响应时发生错误',
  requestFailed: '发送消息失败，请稍后重试',
  networkError: '网络连接失败，请检查网络后重试',
  serverError: '服务器响应错误，请稍后重试',
  processingError: '抱歉，处理消息时发生错误，请稍后再试。',
  historyCleared: '历史记录已清空',
  newChatCreated: '已创建新对话',
  loadSessionFailed: '加载会话失败'
}

const suggestions: ChatSuggestion[] = [
  {
    icon: '💡',
    iconColor: 'text-yellow-500',
    title: 'yaml编写',
    content: '请给我一个基本的nginx 部署yaml如何配置?'
  },
  {
    icon: 'ℹ️',
    iconColor: 'text-blue-500',
    title: '网络',
    content: '请解释下Deploy中的HostNetwork如何配置?'
  },
  {
    icon: '🔔',
    iconColor: 'text-purple-500',
    title: '自动应用',
    content: '请给我一个基本的nginx 部署yaml, 并部署到集群中'
  },
  {
    icon: '✅',
    iconColor: 'text-green-500',
    title: 'Yaml模板',
    content: '请给我一个基本的nginx 部署yaml, 并保存为模板'
  }
]

const refreshContextSnapshot = () => {
  try {
    const snapshot = aiContextManager.snapshot()
    const availableSectionIds = new Set(snapshot.evidence?.map((section) => section.id) || [])
    if (snapshot.state?.unsavedChanges) {
      availableSectionIds.add(AI_CONTEXT_UNSAVED_CHANGES_SECTION_ID)
    }
    excludedContextSectionIds.value = excludedContextSectionIds.value.filter((id) =>
      availableSectionIds.has(id)
    )
    contextSnapshot.value = snapshot
  } catch (error) {
    contextSnapshot.value = undefined
    console.warn('Failed to collect page context:', error)
  }
}

const consumeSelectedContext = (): AIContextSnapshot | undefined => {
  refreshContextSnapshot()
  const selected = contextSnapshot.value
    ? selectAIContextSnapshot(contextSnapshot.value, {
        enabled: contextEnabled.value,
        excludedSectionIds: excludedContextSectionIds.value
      })
    : undefined

  contextEnabled.value = true
  excludedContextSectionIds.value = []
  return selected
}

watch(
  () => props.agentDrawerOpen,
  (open) => {
    localOpen.value = open
    if (open) refreshContextSnapshot()
  },
  { immediate: true }
)

watch(localOpen, (open) => {
  emit('update:agentDrawerOpen', open)
})
</script>
