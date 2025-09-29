<template>
    <div class="flex-1 space-y-6 overflow-y-auto rounded-xl bg-white p-4 text-sm leading-6 text-slate-900 sm:text-base sm:leading-7"
        ref="messagesScrollContainer" style="height: calc(100vh - 200px); max-height: 90%">
        <template v-if="messages.length === 0">
            <div class="flex flex-col items-center justify-center h-full gap-4">
                <h1 class="text-2xl font-bold">Dubbo Admin AI</h1>
                <p class="text-gray-500">
                    我是Dubbo Admin的AI小助手，你可以问我任何关于kubernetes的问题，我尽量给你提供最准确的答案。
                </p>
                <p class="text-lg text-amber-300 font-medium">✨ 奇思妙想和创新的火花</p>
                <div class="grid grid-cols-2 gap-4 w-full max-w-2xl mt-4">
                    <SuggestionCard v-for="(suggestion, index) in suggestions" :key="index" :suggestion="suggestion"
                        @click="handleSuggestionClick" />
                </div>
            </div>
        </template>

        <template v-else>
            <MessageItem v-for="msg in messages" :key="msg.id" :message="msg" :is-ai-thinking="isAiThinking"
                :is-loading="isLoading" :messages="messages" @retry-last-message="retryLastMessage" :md="md" />
        </template>
    </div>
</template>

<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'
import type { ChatMessage } from '@/api/service/ai'
import MessageItem from './MessageItem.vue'
import SuggestionCard from './SuggestionCard.vue'

// 定义props
const props = defineProps<{
    messages: ChatMessage[]
    isAiThinking: boolean
    isLoading: boolean
    lastUserMessage: string
    md: any
}>()

// 定义emits
const emit = defineEmits<{
    (e: 'retryLastMessage'): void
    (e: 'suggestionClick', suggestion: string): void
}>()

// 消息滚动容器的引用
const messagesScrollContainer = ref<HTMLElement | null>(null)

// 建议问题列表
const suggestions = [
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

// 滚动到底部的函数
const scrollToBottom = async () => {
    await nextTick() // 确保 DOM 更新完成
    if (messagesScrollContainer.value) {
        messagesScrollContainer.value.scrollTo({
            top: messagesScrollContainer.value.scrollHeight,
            behavior: 'smooth'
        })
    }
}

// 监听消息列表变化，自动滚动到底部
watch(
    () => props.messages,
    scrollToBottom,
    { deep: true } // 深度监听，捕获消息内容的变化
)

// 重试上一条消息
const retryLastMessage = () => {
    emit('retryLastMessage')
}

// 处理建议问题的点击
const handleSuggestionClick = (suggestion: string) => {
    emit('suggestionClick', suggestion)
}

// 定义暴露给父组件的方法
defineExpose({
    scrollToBottom
})
</script>