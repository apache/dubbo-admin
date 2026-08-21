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
  <a-drawer
    v-model:open="localDrawerOpen"
    class="custom-class"
    :title="resolvedLabels.title"
    placement="right"
    :width="drawerWidth"
  >
    <!-- Prompt Messages Container - Modify the height according to your need -->
    <div class="flex w-full flex-col h-full">
      <!-- 使用消息列表组件 -->
      <MessageList
        ref="messageListRef"
        :messages="messages"
        :is-ai-thinking="isAiThinking"
        :is-loading="isLoading"
        :last-user-message="lastUserMessage"
        :md="md"
        :labels="resolvedLabels"
        :suggestions="suggestions"
        @retry-last-message="retryLastMessage"
        @suggestion-click="handleSuggestionClick"
      />

      <!-- Usage Information -->
      <div v-if="usageInfo" class="w-full mb-2 p-3 bg-gray-50 rounded-lg border border-gray-200">
        <div class="flex items-center justify-between">
          <div class="grid grid-cols-4 gap-x-6 gap-y-1 text-xs text-gray-600">
            <!-- Token 信息 -->
            <span class="flex items-center gap-1">
              <span class="w-2 h-2 bg-blue-400 rounded-full"></span>
              {{ resolvedLabels.inputTokens }}: {{ usageInfo?.inputTokens?.toLocaleString() || 0 }}
            </span>
            <span class="flex items-center gap-1">
              <span class="w-2 h-2 bg-green-400 rounded-full"></span>
              {{ resolvedLabels.outputTokens }}:
              {{ usageInfo?.outputTokens?.toLocaleString() || 0 }}
            </span>
            <!-- 字符信息 -->
            <!-- <span class="flex items-center gap-1">
              <span class="w-2 h-2 bg-cyan-400 rounded-full"></span>
              输入字符: {{ usageInfo?.inputCharacters?.toLocaleString() || 0 }}
            </span>
            <span class="flex items-center gap-1">
              <span class="w-2 h-2 bg-orange-400 rounded-full"></span>
              输出字符: {{ usageInfo?.outputCharacters?.toLocaleString() || 0 }}
            </span> -->
            <!-- 总计 -->
            <span class="flex items-center gap-1 font-medium">
              <span class="w-2 h-2 bg-purple-400 rounded-full"></span>
              {{ resolvedLabels.totalTokens }}: {{ usageInfo?.totalTokens?.toLocaleString() || 0 }}
            </span>
          </div>
          <button
            @click="usageInfo = null"
            class="text-gray-400 hover:text-gray-600 transition-colors ml-4"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M6 18L18 6M6 6l12 12"
              ></path>
            </svg>
          </button>
        </div>
      </div>

      <slot name="context" />

      <!-- 使用输入区域组件 -->
      <ChatInput
        ref="chatInputRef"
        :messages="messages"
        :is-loading="isLoading"
        :labels="resolvedLabels"
        :show-history="supportsHistory"
        @send-message="sendMessage"
        @handle-new-chat="handleNewChat"
        @handle-view-history="handleViewHistory"
        @clear-history="clearHistory"
        @keydown="handleKeyDown"
      />
    </div>
  </a-drawer>

  <!-- 使用会话历史模态框组件 -->
  <SessionHistoryModal
    v-if="supportsHistory"
    :visible="historyModalVisible"
    :sessions="sessions"
    :labels="resolvedLabels"
    @update:visible="handleHistoryModalVisibleUpdate"
    @load-session="loadSession"
    @delete-session="deleteSession"
  />
</template>

<script setup lang="ts">
import { computed, ref, watch, onMounted, nextTick } from 'vue'
import { message } from 'ant-design-vue'
import MarkdownIt from 'markdown-it'
import hljs from 'highlight.js'
import 'highlight.js/styles/github.css' // 使用 GitHub 风格的代码高亮主题
import MessageList from './MessageList.vue'
import ChatInput from './ChatInput.vue'
import SessionHistoryModal from './SessionHistoryModal.vue'
import {
  DEFAULT_AI_CHAT_LABELS,
  type AIChatLabels,
  type ChatMessage,
  type ChatService,
  type ChatSuggestion,
  type Session
} from './types'

// 初始化 markdown 解析器
const md: MarkdownIt = new MarkdownIt({
  html: false, // Escape raw HTML from untrusted model output.
  breaks: true, // 转换 \n 为 <br>
  linkify: true, // 自动转换 URL 为链接
  typographer: true, // 启用一些语言中性的替换 + 引号美化
  highlight: function (str: string, lang: string): string {
    if (lang && hljs.getLanguage(lang)) {
      try {
        return (
          '<pre class="code-block"><code class="hljs language-' +
          lang +
          '">' +
          hljs.highlight(str, { language: lang, ignoreIllegals: true }).value +
          '</code></pre>'
        )
      } catch (error) {
        console.warn('Failed to highlight code block:', error)
      }
    }
    return (
      '<pre class="code-block"><code class="hljs">' + md.utils.escapeHtml(str) + '</code></pre>'
    )
  }
})

// 定义本地响应式变量
const props = withDefaults(
  defineProps<{
    open?: boolean
    service: ChatService<unknown>
    collectContext?: () => unknown
    labels?: Partial<AIChatLabels>
    suggestions?: ChatSuggestion[]
    drawerWidth?: string | number
  }>(),
  {
    open: false,
    collectContext: undefined,
    labels: () => ({}),
    suggestions: () => [],
    drawerWidth: 600
  }
)

const emit = defineEmits<{
  (event: 'update:open', value: boolean): void
}>()

const resolvedLabels = computed<AIChatLabels>(() => ({
  ...DEFAULT_AI_CHAT_LABELS,
  ...props.labels
}))
const supportsHistory = computed(
  () =>
    Boolean(props.service.getSessions) &&
    Boolean(props.service.getSessionInfo) &&
    Boolean(props.service.deleteSession)
)

const localDrawerOpen = ref(props.open)
const messages = ref<ChatMessage[]>([])
const isLoading = ref(false)
const isAiThinking = ref(false) // AI是否正在思考（用于显示思考中的动画）
const currentSessionId = ref('')
const sessions = ref<Session[]>([])
const historyModalVisible = ref(false)

const lastUserMessage = ref<string>('') // 保存最后一次用户消息用于重试
const usageInfo = ref<any>(null) // 保存使用情况信息

// 子组件引用
const messageListRef = ref<InstanceType<typeof MessageList> | null>(null)
const chatInputRef = ref<InstanceType<typeof ChatInput> | null>(null)

// 节流滚动函数，避免频繁滚动影响性能
let scrollTimeout: any | null = null
const throttledScrollToBottom = () => {
  if (scrollTimeout) return
  scrollTimeout = setTimeout(async () => {
    await scrollToBottom()
    scrollTimeout = null
  }, 100) // 100ms 节流
}

// 监听 prop 变化同步到本地变量
watch(
  () => props.open,
  (newVal) => {
    localDrawerOpen.value = newVal
  }
)

// 监听本地变量变化并触发事件通知父组件
watch(localDrawerOpen, (newVal) => {
  emit('update:open', newVal)
})

// 滚动到底部的函数
const scrollToBottom = async () => {
  await nextTick() // 确保 DOM 更新完成
  if (messageListRef.value) {
    messageListRef.value.scrollToBottom()
  }
}

// 添加错误消息气泡
const addErrorMessage = (errorText: string) => {
  const errorMessage: ChatMessage = {
    id: Date.now().toString(),
    content: errorText,
    role: 'assistant',
    timestamp: Date.now(),
    type: 'error'
  }
  messages.value.push(errorMessage)
}

// 监听drawer打开状态，打开时滚动到底部
watch(localDrawerOpen, async (newVal) => {
  if (newVal) {
    await scrollToBottom()
  }
})

// 创建新会话
async function createNewSession() {
  try {
    const sessionId = await props.service.createSession()
    currentSessionId.value = sessionId
    messages.value = []
    return sessionId
  } catch (error) {
    message.error(resolvedLabels.value.createSessionFailed)
    console.error(resolvedLabels.value.createSessionFailed, error)
    return ''
  }
}

// 获取会话列表
async function fetchSessions() {
  if (!props.service.getSessions) return
  try {
    sessions.value = await props.service.getSessions()
  } catch (error) {
    console.error('获取会话列表失败:', error)
  }
}

// 获取特定会话信息
async function fetchSessionInfo(sessionId: string) {
  if (!props.service.getSessionInfo) return null
  try {
    const response = await props.service.getSessionInfo(sessionId)
    return response.data
  } catch (error) {
    console.error('获取会话信息失败:', error)
    return null
  }
}

// 删除会话
async function deleteSession(sessionId: string) {
  if (!props.service.deleteSession) return
  try {
    await props.service.deleteSession(sessionId)
    if (currentSessionId.value === sessionId) {
      currentSessionId.value = ''
      messages.value = []
    }
    await fetchSessions()
    message.success(resolvedLabels.value.sessionDeleted)
  } catch (error) {
    message.error(resolvedLabels.value.deleteSessionFailed)
    console.error(resolvedLabels.value.deleteSessionFailed, error)
  }
}

// 重试上一条消息
async function retryLastMessage() {
  if (!lastUserMessage.value) {
    message.warning(resolvedLabels.value.noRetryMessage)
    return
  }

  // 如果正在加载中，不允许重试
  if (isLoading.value) {
    message.warning(resolvedLabels.value.waitForResponse)
    return
  }

  // 删除最后的错误消息，如果有的话
  if (messages.value.length > 0 && messages.value[messages.value.length - 1].type === 'error') {
    messages.value.pop()
  }

  // 重新发送上一条消息
  if (chatInputRef.value) {
    chatInputRef.value.inputMessage = lastUserMessage.value
  }
  await sendMessage()
}

// 发送消息并接收流式响应
async function sendMessage() {
  // 获取输入消息
  let inputMsg = ''
  if (chatInputRef.value) {
    inputMsg = chatInputRef.value.inputMessage
  }

  if (!inputMsg.trim() || isLoading.value) return

  const context = props.collectContext?.()

  // 保存最后一条用户消息
  lastUserMessage.value = inputMsg

  // 如果没有当前会话ID，先创建一个新会话
  if (!currentSessionId.value) {
    const sessionId = await createNewSession()
    if (!sessionId) {
      addErrorMessage(resolvedLabels.value.createSessionUnavailable)
      isLoading.value = false
      return
    }
  }

  const userMessage: ChatMessage = {
    id: Date.now().toString(),
    content: inputMsg,
    role: 'user',
    timestamp: Date.now()
  }

  messages.value.push(userMessage)

  const aiMessage: ChatMessage = {
    id: (Date.now() + 1).toString(),
    content: '',
    role: 'assistant',
    timestamp: Date.now() + 1
  }

  messages.value.push(aiMessage)
  isLoading.value = true
  isAiThinking.value = true // 立即设置AI思考状态

  // 清空输入框
  if (chatInputRef.value) {
    chatInputRef.value.inputMessage = ''
  }

  // 清空之前的使用情况信息
  usageInfo.value = null

  // 发送消息后滚动到底部
  await scrollToBottom()

  try {
    // 发送消息并获取流式响应
    const stream = await props.service.sendChatMessage(
      userMessage.content,
      currentSessionId.value,
      context
    )

    // 处理SSE流
    const reader = stream.getReader()
    if (!reader) throw new Error('无法读取响应流')

    const decoder = new TextDecoder()
    let partialChunk = ''
    let hasError = false

    // eslint-disable-next-line no-constant-condition
    while (true) {
      const { done, value } = await reader.read()
      if (done) break

      // 解码并处理数据
      const chunk = decoder.decode(value, { stream: true })
      partialChunk += chunk

      // 处理可能的多行数据
      let lines = partialChunk.split('\n')

      // 如果最后一行不完整，保存到partialChunk中
      if (!chunk.endsWith('\n')) {
        partialChunk = lines.pop() || ''
      } else {
        partialChunk = ''
      }

      // 处理每一对事件和数据
      for (let i = 0; i < lines.length - 1; i++) {
        const line = lines[i]
        const nextLine = lines[i + 1]

        if (line.trim() === '') continue
        console.log('[ line ] >', line)

        // 解析事件类型和数据
        if (line.startsWith('event: ')) {
          const eventType = line.substring(7).trim()
          console.log('[ eventType ] >', eventType)

          // 确保下一行是数据行
          if (!nextLine || !nextLine.startsWith('data: ')) continue

          // 跳过已处理的数据行
          i++

          const dataStr = nextLine.substring(6)

          // 跳过空数据行
          if (!dataStr.trim()) continue

          try {
            const data = JSON.parse(dataStr)

            // 根据事件类型处理数据
            switch (eventType) {
              case 'message_start':
                console.log('开始新的消息')
                aiMessage.content = ''
                // isAiThinking.value = true // 已在函数开始时设置
                break

              case 'content_block_start':
                console.log('开始新的内容块', data.index)
                // 如果是新的内容块且已有内容，添加适当的间距
                if (aiMessage.content && data.index > 0) {
                  aiMessage.content += '\n\n'
                }
                break

              case 'content_block_delta':
                if (data.delta?.type === 'text_delta' && data.delta?.text) {
                  const textContent = data.delta.text
                  console.log('内容块 delta', data.index, textContent)
                  // 过滤掉可能包含的元数据或token信息
                  const isMetadata =
                    textContent.includes('"inputCharacters"') ||
                    textContent.includes('"outputTokens"') ||
                    textContent.includes('"usage"') ||
                    textContent.includes('"totalTokens"') ||
                    textContent.includes('"evidence"') ||
                    textContent.includes('"final_answer"') ||
                    textContent.includes('"summary"') ||
                    textContent.includes('"heartbeat"') ||
                    textContent.includes('"stop_reason"') ||
                    textContent.includes('"inputTokens"') ||
                    textContent.includes('"outputCharacters"')

                  if (!isMetadata) {
                    // 将新文本添加到消息内容中
                    aiMessage.content += textContent
                    // 直接更新最后一条消息，避免创建新数组
                    const lastIndex = messages.value.length - 1
                    if (lastIndex >= 0) {
                      messages.value[lastIndex] = { ...aiMessage }
                    }
                    // 实时滚动到底部（节流）
                    throttledScrollToBottom()
                  }
                }
                break

              case 'content_block_stop': {
                console.log('内容块结束')
                // 在内容块结束时添加水平分隔线
                if (aiMessage.content.trim()) {
                  aiMessage.content += '\n\n---\n\n'
                }
                // 创建新的消息数组以触发响应式更新
                const updatedBlockMessages = [...messages.value]
                // 更新最后一条消息
                updatedBlockMessages[updatedBlockMessages.length - 1] = { ...aiMessage }
                // 更新消息列表
                messages.value = updatedBlockMessages
                break
              }

              case 'message_delta':
                // 处理消息更新，比如处理建议的动作等
                if (data.delta?.suggested_actions) {
                  console.log('收到建议动作:', data.delta.suggested_actions)
                }
                console.log('lg', data)

                // 如果包含final字段，说明是最终的元数据，不需要显示
                if (data.final) {
                  console.log('收到最终元数据:', data.final)
                  // 保存使用情况信息
                  if (data.final.usage) {
                    usageInfo.value = data.final.usage
                  }
                  // 不处理final数据，避免显示在聊天界面
                }
                break

              case 'message_stop':
                console.log('消息结束')
                // 标记AI不再思考
                isAiThinking.value = false
                break

              case 'error':
                hasError = true
                isAiThinking.value = false
                console.error('SSE 流错误:', data)

                // 如果已经有内容，保留内容并添加单独的错误消息
                if (aiMessage.content.trim()) {
                  // 更新最后一条AI消息，保持原有内容
                  const lastIndex = messages.value.length - 1
                  if (lastIndex >= 0) {
                    messages.value[lastIndex] = { ...aiMessage }
                  }

                  // 添加单独的错误消息气泡
                  const errorMsg =
                    data.error?.message || data.message || resolvedLabels.value.unknownStreamError
                  addErrorMessage(errorMsg)
                } else {
                  // 如果没有内容，移除AI消息并添加纯错误消息
                  messages.value.pop()
                  const errorMsg =
                    data.error?.message || data.message || resolvedLabels.value.unknownStreamError
                  addErrorMessage(errorMsg)
                }
                break

              default:
                console.log(`收到事件: ${eventType}`, data)
            }
          } catch (e) {
            console.error(`解析 ${eventType} 事件数据失败:`, e, dataStr)
            // 如果解析失败，检查是否是错误事件
            if (eventType === 'error') {
              hasError = true
              isAiThinking.value = false
              messages.value.pop()
              addErrorMessage(resolvedLabels.value.parseStreamError)
            }
          }
        }
      }
    }

    // 如果处理过程中出现错误，添加错误提示
    if (hasError && !aiMessage.content) {
      aiMessage.content = resolvedLabels.value.processingError
    }
  } catch (error: any) {
    console.error('发送消息失败:', error)

    // 处理错误响应
    let errorMessage = resolvedLabels.value.requestFailed
    if (error.response?.data) {
      // JSON 格式的错误响应，直接使用 message 字段
      errorMessage = error.response.data.message || error.response.data.error || errorMessage
    } else if (error.message) {
      // 普通的 Error 对象
      if (error.message.includes('Failed to fetch') || error.message.includes('NetworkError')) {
        errorMessage = resolvedLabels.value.networkError
      } else if (error.message.includes('HTTP error')) {
        errorMessage = resolvedLabels.value.serverError
      } else {
        errorMessage = error.message
      }
    }

    // 检查是否有部分内容需要保留
    if (aiMessage.content.trim()) {
      // 如果有内容，保留AI消息并添加单独的错误消息
      const lastIndex = messages.value.length - 1
      if (lastIndex >= 0) {
        messages.value[lastIndex] = { ...aiMessage }
      }
      // 添加单独的错误消息气泡
      addErrorMessage(errorMessage)
    } else {
      // 如果没有内容，移除AI消息并添加纯错误消息
      messages.value.pop()
      addErrorMessage(errorMessage)
    }
  } finally {
    isLoading.value = false
    isAiThinking.value = false
  }
}

// 清空历史消息
function clearHistory() {
  messages.value = []
  usageInfo.value = null
  message.success(resolvedLabels.value.historyCleared)
}

// 处理新对话按钮点击
async function handleNewChat() {
  // 创建新对话
  await createNewSession()
  // 清空使用情况信息
  usageInfo.value = null
  message.success(resolvedLabels.value.newChatCreated)
}

// 处理对话历史按钮点击
async function handleViewHistory() {
  if (!supportsHistory.value) return
  await fetchSessions()
  historyModalVisible.value = true
}

// 处理键盘事件
function handleKeyDown(event: KeyboardEvent) {
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault()
    sendMessage()
  }
}

// 处理建议问题的点击
function handleSuggestionClick(suggestion: string) {
  if (chatInputRef.value) {
    chatInputRef.value.inputMessage = suggestion
    nextTick(() => chatInputRef.value?.focus())
  }
}

// 处理会话历史模态框可见性更新
function handleHistoryModalVisibleUpdate(value: boolean) {
  historyModalVisible.value = value
}

onMounted(() => {
  // 确保消息容器正确初始化并滚动到底部
  scrollToBottom()
})

// 加载选定的会话
async function loadSession(sessionId: string) {
  try {
    isLoading.value = true
    const sessionInfo = await fetchSessionInfo(sessionId)
    if (sessionInfo) {
      currentSessionId.value = sessionId
      messages.value = sessionInfo.messages || []
      historyModalVisible.value = false
      // 加载会话后滚动到底部
      await scrollToBottom()
    }
  } catch (error) {
    console.error('加载会话失败:', error)
    message.error(resolvedLabels.value.loadSessionFailed)
  } finally {
    isLoading.value = false
  }
}
</script>

<style>
/* 聊天区域滚动条样式优化 */
.flex-1.space-y-6.overflow-y-auto {
  /* Webkit浏览器滚动条样式 */
  &::-webkit-scrollbar {
    width: 6px;
    height: 6px;
  }

  &::-webkit-scrollbar-track {
    background: #f1f1f1;
    border-radius: 10px;
  }

  &::-webkit-scrollbar-thumb {
    background: #c1c1c1;
    border-radius: 10px;
  }

  &::-webkit-scrollbar-thumb:hover {
    background: #a8a8a8;
  }

  /* Firefox浏览器滚动条样式 */
  scrollbar-width: thin;
  scrollbar-color: #c1c1c1 #f1f1f1;
}

/* 消息内容区域滚动条样式 */
.markdown-body {
  font-family:
    -apple-system,
    BlinkMacSystemFont,
    Segoe UI,
    Helvetica,
    Arial,
    sans-serif;
  font-size: 14px;
  line-height: 1.6;
  word-wrap: break-word;
  width: 100%;
  max-width: 100%;
  overflow-x: auto;
  color: #24292e;

  /* Webkit浏览器滚动条样式 */
  &::-webkit-scrollbar {
    width: 6px;
    height: 6px;
  }

  &::-webkit-scrollbar-track {
    background: #f6f8fa;
    border-radius: 10px;
  }

  &::-webkit-scrollbar-thumb {
    background: #c1c1c1;
    border-radius: 10px;
  }

  &::-webkit-scrollbar-thumb:hover {
    background: #a8a8a8;
  }

  /* Firefox浏览器滚动条样式 */
  scrollbar-width: thin;
  scrollbar-color: #c1c1c1 #f6f8fa;
}

/* 代码块滚动条样式 */
.markdown-body .code-block,
.markdown-body pre {
  /* Webkit浏览器滚动条样式 */
  &::-webkit-scrollbar {
    width: 6px;
    height: 6px;
  }

  &::-webkit-scrollbar-track {
    background: #f6f8fa;
    border-radius: 10px;
  }

  &::-webkit-scrollbar-thumb {
    background: #c1c1c1;
    border-radius: 10px;
  }

  &::-webkit-scrollbar-thumb:hover {
    background: #a8a8a8;
  }

  /* Firefox浏览器滚动条样式 */
  scrollbar-width: thin;
  scrollbar-color: #c1c1c1 #f6f8fa;
}

.markdown-body > *:first-child {
  margin-top: 0 !important;
}

.markdown-body > *:last-child {
  margin-bottom: 0 !important;
}

.markdown-body > p:first-of-type {
  position: relative;
  padding-left: 1.5em;
}

.markdown-body > p:first-of-type::before {
  content: '💡';
  position: absolute;
  left: 0;
  top: 0;
}

.markdown-body .code-block {
  margin: 0;
  padding: 16px;
  overflow: auto;
  font-size: 85%;
  line-height: 1.45;
  background-color: #f6f8fa;
  border-radius: 6px;
}

.markdown-body .hljs {
  background: transparent;
  padding: 0;
}

.markdown-body a {
  color: #0366d6;
  text-decoration: none;
}

.markdown-body a:hover {
  text-decoration: underline;
}

.markdown-body hr {
  height: 1px;
  padding: 0;
  margin: 1.5em 0;
  background-color: #e1e4e8;
  border: 0;
}

.markdown-body blockquote {
  padding: 0 1em;
  color: #6a737d;
  border-left: 0.25em solid #dfe2e5;
  margin: 0 0 16px 0;
}

.markdown-body ul,
.markdown-body ol {
  padding-left: 2em;
  margin-top: 0;
  margin-bottom: 16px;
}

.markdown-body img {
  max-width: 100%;
  box-sizing: content-box;
  background-color: #fff;
  border-radius: 3px;
}

.markdown-body pre {
  background-color: #f6f8fa;
  border-radius: 6px;
  padding: 16px;
  overflow-x: auto;
  white-space: pre-wrap;
  word-wrap: break-word;
}

.markdown-body code {
  background-color: rgba(175, 184, 193, 0.2);
  border-radius: 6px;
  padding: 0.2em 0.4em;
  font-family:
    ui-monospace,
    SFMono-Regular,
    SF Mono,
    Menlo,
    Consolas,
    Liberation Mono,
    monospace;
}

.markdown-body pre code {
  background-color: transparent;
  padding: 0;
}

.markdown-body h1,
.markdown-body h2,
.markdown-body h3,
.markdown-body h4,
.markdown-body h5,
.markdown-body h6 {
  margin-top: 24px;
  margin-bottom: 16px;
  font-weight: 600;
  line-height: 1.25;
}

.markdown-body h1 {
  font-size: 2em;
}

.markdown-body h2 {
  font-size: 1.5em;
}

.markdown-body h3 {
  font-size: 1.25em;
}

.markdown-body h4 {
  font-size: 1em;
}

.markdown-body ul,
.markdown-body ol {
  padding-left: 2em;
  margin-top: 0;
  margin-bottom: 16px;
}

.markdown-body blockquote {
  padding: 0 1em;
  color: #57606a;
  border-left: 0.25em solid #d0d7de;
  margin: 0 0 16px;
}

.markdown-body table {
  display: block;
  width: 100%;
  width: max-content;
  max-width: 100%;
  overflow: auto;
  margin-top: 0;
  margin-bottom: 16px;
  border-spacing: 0;
  border-collapse: collapse;
}

.markdown-body table th,
.markdown-body table td {
  padding: 6px 13px;
  border: 1px solid #d0d7de;
}

.markdown-body table tr {
  background-color: #ffffff;
  border-top: 1px solid #d0d7de;
}

.markdown-body table tr:nth-child(2n) {
  background-color: #f6f8fa;
}

.markdown-body img {
  max-width: 100%;
  box-sizing: content-box;
  background-color: #ffffff;
}

.markdown-body p {
  margin-top: 0;
  margin-bottom: 16px;
}
</style>
