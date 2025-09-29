<template>
    <div :class="{
        'flex items-start': message.role === 'assistant' && message.type !== 'error',
        'flex flex-row-reverse items-start': message.role === 'user',
        'flex justify-center': message.type === 'error'
    }">
        <img v-if="message.role === 'assistant' && message.type !== 'error'" class="mr-2 h-8 w-8 rounded-full"
            src="https://dummyimage.com/128x128/fde3cf/f56a00&text=AI" />
        <img v-else-if="message.role === 'user'" class="ml-2 h-8 w-8 rounded-full"
            src="https://dummyimage.com/128x128/87d068/ffffff&text=U" />

        <div :class="{
            'flex flex-col rounded-xl bg-[#0000000f] text-[#000000e0] p-4 max-w-[480px] break-words':
                message.role === 'assistant' && message.type !== 'error',
            'flex h-fit rounded-xl bg-[#0000000f] text-[#000000e0] p-4 max-w-[480px] break-words':
                message.role === 'user',
            'flex flex-col rounded-xl bg-red-50 border border-red-200 text-red-800 p-4 min-w-[480px] max-w-[480px] break-words':
                message.type === 'error'
        }">
            <template v-if="message.role === 'assistant'">
                <template v-if="message === messages[messages.length - 1] && isAiThinking && !message.content">
                    <div class="flex items-center text-gray-400 text-sm">
                        <LoadingOutlined class="mr-2" />
                        <span class="animate-pulse">正在思考...</span>
                    </div>
                </template>
                <template v-else-if="message.type === 'error'">
                    <div class="flex flex-col">
                        <div class="flex items-center text-red-500 text-sm mb-2">
                            <span class="mr-2">❌</span>
                            <span class="font-medium">出现错误</span>
                        </div>
                        <div class="text-red-600 text-sm mb-3" v-html="message.content.replace(/\n/g, '<br />')"></div>
                        <div class="flex justify-end">
                            <a-button size="small" type="primary" @click="retryLastMessage" :loading="isLoading"
                                class="flex items-center">
                                <RedoOutlined v-if="!isLoading" class="mr-1" />
                                重试
                            </a-button>
                        </div>
                    </div>
                </template>

                <div v-else-if="message.content" class="markdown-body">
                    <div v-html="md.render(message.content)"></div>
                    <div v-if="message === messages[messages.length - 1] && isAiThinking"
                        class="flex items-center text-gray-400 text-xs mt-2">
                        <LoadingOutlined class="mr-1" />
                        <span class="animate-pulse">正在思考...</span>
                    </div>
                </div>
            </template>
            <template v-else>
                <p v-html="message.content.replace(/\n/g, '<br />')"></p>
            </template>
        </div>
    </div>
</template>

<script setup lang="ts">
import { LoadingOutlined, RedoOutlined } from '@ant-design/icons-vue'
import type { ChatMessage } from '@/api/service/ai'

// 定义props
const props = defineProps<{
    message: ChatMessage
    isAiThinking: boolean
    isLoading: boolean
    messages: ChatMessage[]
    md: any
}>()

// 定义emits
const emit = defineEmits<{
    (e: 'retryLastMessage'): void
}>()

// 重试上一条消息
const retryLastMessage = () => {
    emit('retryLastMessage')
}
</script>