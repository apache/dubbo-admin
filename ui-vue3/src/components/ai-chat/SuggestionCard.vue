<template>
    <div class="border rounded-lg p-4 hover:bg-gray-50 cursor-pointer transition-all duration-200 hover:shadow-md"
        @click="handleClick">
        <div class="flex items-center gap-2 mb-2">
            <div :class="suggestion.iconColor">{{ suggestion.icon }}</div>
            <div class="font-medium">{{ suggestion.title }}</div>
        </div>
        <div class="text-gray-500 text-sm">{{ suggestion.content }}</div>
    </div>
</template>

<script setup lang="ts">
import type { ChatSuggestion } from './types'

// 定义props
const props = defineProps<{
    suggestion: ChatSuggestion
}>()

// 定义emits
const emit = defineEmits<{
    (e: 'click', suggestion: string): void
}>()

// 处理点击事件
const handleClick = () => {
    emit('click', props.suggestion.content)
}
</script>
