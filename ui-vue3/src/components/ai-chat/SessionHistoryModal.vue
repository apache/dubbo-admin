<template>
    <!-- 对话历史Modal -->
    <a-modal v-model:visible="localVisible" :title="labels.historyTitle" :footer="null" width="600px">
        <div class="max-h-[400px] overflow-y-auto">
            <a-empty v-if="sessions.length === 0" :description="labels.emptyHistory" />
            <a-list v-else>
                <a-list-item v-for="session in sessions" :key="session.session_id"
                    class="cursor-pointer hover:bg-gray-100 rounded p-2">
                    <div class="flex justify-between w-full" @click="loadSession(session.session_id)">
                        <div>
                            <div class="font-medium">
                                {{ labels.session }} #{{ session.session_id ? session.session_id.substring(0, 8) : '' }}
                            </div>
                            <div class="text-gray-500 text-sm">
                                {{ session.message_count ? `${labels.messageCount}: ${session.message_count}` : labels.newSession }}
                            </div>
                        </div>
                        <div>
                            <div class="text-gray-500 text-sm">
                                {{ new Date(session.created_at).toLocaleString() }}
                            </div>
                            <a-button type="link" size="small" @click.stop="deleteSession(session.session_id)" danger>
                                <DeleteOutlined /> {{ labels.delete }}
                            </a-button>
                        </div>
                    </div>
                </a-list-item>
            </a-list>
        </div>
    </a-modal>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { DeleteOutlined } from '@ant-design/icons-vue'
import type { AIChatLabels, Session } from './types'

// 定义props
const props = defineProps<{
    visible: boolean
    sessions: Session[]
    labels: AIChatLabels
}>()

// 定义emits
const emit = defineEmits<{
    (e: 'update:visible', value: boolean): void
    (e: 'loadSession', sessionId: string): void
    (e: 'deleteSession', sessionId: string): void
}>()

// 本地响应式变量
const localVisible = ref(props.visible)

// 监听prop变化
watch(
    () => props.visible,
    (newVal) => {
        localVisible.value = newVal
    }
)

// 监听本地变量变化并触发事件通知父组件
watch(localVisible, (newVal) => {
    emit('update:visible', newVal)
})

// 加载选定的会话
const loadSession = (sessionId: string) => {
    emit('loadSession', sessionId)
}

// 删除会话
const deleteSession = (sessionId: string) => {
    emit('deleteSession', sessionId)
}
</script>
