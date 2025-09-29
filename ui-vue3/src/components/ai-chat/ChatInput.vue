<template>
    <div class="w-full mt-2">
        <div class="w-full flex flex-row gap-2" v-show="messages.length > 0">
            <a-button class="flex items-center" style="
          background-image: linear-gradient(
            97deg,
            rgb(242, 249, 254) 0%,
            rgb(247, 243, 255) 100%
          );
        " @click="handleNewChat">
                <PlusOutlined />
                新对话
            </a-button>
            <a-button class="flex items-center" style="
          background-image: linear-gradient(
            97deg,
            rgb(242, 249, 254) 0%,
            rgb(247, 243, 255) 100%
          );
        " @click="handleViewHistory">
                <ClockCircleOutlined />
                对话历史
            </a-button>
            <a-button class="flex items-center" style="
          background-image: linear-gradient(
            97deg,
            rgb(242, 249, 254) 0%,
            rgb(247, 243, 255) 100%
          );
        " @click="clearHistory">
                <DeleteOutlined />
                清空历史
            </a-button>
        </div>
        <div class="mt-2">
            <div class="relative">
                <div
                    class="flex w-full rounded-lg border border-gray-200 bg-white shadow-lg transition-all duration-200 hover:shadow-xl focus-within:border-blue-400 focus-within:ring-4 focus-within:ring-blue-100">
                    <textarea id="chat-input" v-model="inputMessage" @keydown="handleKeyDown" rows="1"
                        class="block w-full resize-none rounded-lg border-0 bg-transparent px-4 py-3 text-gray-900 placeholder:text-gray-400 focus:outline-none focus:ring-0 sm:text-sm transition-colors duration-200 hover:bg-gray-50"
                        placeholder="输入你的问题，Shift + Enter 换行" :disabled="isLoading" required></textarea>
                    <div class="flex items-end gap-2 p-2">
                        <a-button shape="circle" type="primary" :loading="isLoading" @click="sendMessage"
                            class="flex h-8 w-8 items-center justify-center">
                            <ArrowUpOutlined v-if="!isLoading" />
                        </a-button>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import {
    ClockCircleOutlined,
    DeleteOutlined,
    PlusOutlined,
    ArrowUpOutlined
} from '@ant-design/icons-vue'
import type { ChatMessage } from '@/api/service/ai'

// 定义props
const props = defineProps<{
    messages: ChatMessage[]
    isLoading: boolean
}>()

// 定义emits
const emit = defineEmits<{
    (e: 'update:inputMessage', value: string): void
    (e: 'sendMessage'): void
    (e: 'handleNewChat'): void
    (e: 'handleViewHistory'): void
    (e: 'clearHistory'): void
    (e: 'keydown', event: KeyboardEvent): void
}>()

// 输入消息
const inputMessage = ref('')

// 监听输入消息变化并通知父组件
watch(inputMessage, (newVal) => {
    emit('update:inputMessage', newVal)
})

// 发送消息
const sendMessage = () => {
    emit('sendMessage')
}

// 处理新对话按钮点击
const handleNewChat = () => {
    emit('handleNewChat')
}

// 处理对话历史按钮点击
const handleViewHistory = () => {
    emit('handleViewHistory')
}

// 清空历史消息
const clearHistory = () => {
    emit('clearHistory')
}

// 处理键盘事件
const handleKeyDown = (event: KeyboardEvent) => {
    emit('keydown', event)
}

// 定义暴露给父组件的属性
defineExpose({
    inputMessage
})
</script>