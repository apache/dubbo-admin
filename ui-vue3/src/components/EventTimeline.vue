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
  <div class="event-timeline-container">
    <a-spin :spinning="loading">
      <a-timeline mode="left" class="event-timeline">
        <a-timeline-item
          v-for="(item, index) in events"
          :key="index"
          :color="item.type === 'warning' ? '#faad14' : '#1890ff'"
        >
          <!-- Time label on the left -->
          <template #label>
            <span class="event-time">{{ item.time }}</span>
          </template>

          <!-- Custom dot -->
          <template #dot>
            <div class="event-dot" :class="item.type">
              <CheckCircleOutlined v-if="item.type !== 'warning'" class="dot-icon normal" />
              <WarningOutlined v-else class="dot-icon warning" />
            </div>
          </template>

          <!-- Event card -->
          <div class="event-card" :class="item.type">
            <span class="event-message">{{ item.message }}</span>
            <a-tag :color="item.type === 'warning' ? 'orange' : 'blue'" class="event-source-tag">
              {{ item.source }}
            </a-tag>
          </div>
        </a-timeline-item>

        <!-- Bottom hint -->
        <a-timeline-item>
          <template #dot>
            <ArrowDownOutlined class="bottom-arrow" />
          </template>
          <div class="bottom-hint">
            <span>{{ $t('eventExpiryHint') || '过期事件不会存储' }}</span>
            <a-spin v-if="loadingMore" size="small" class="load-more-spinner" />
          </div>
        </a-timeline-item>
      </a-timeline>
      <div ref="loadMoreTriggerRef" class="load-more-trigger" />
    </a-spin>

    <a-empty v-if="!loading && events.length === 0" description="暂无事件" />
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import type { EventItem } from '@/types/api'
import { CheckCircleOutlined, WarningOutlined, ArrowDownOutlined } from '@ant-design/icons-vue'

const props = defineProps<{
  events: EventItem[]
  loading: boolean
  loadingMore?: boolean
  hasMore?: boolean
}>()

const emit = defineEmits<{
  (e: 'loadMore'): void
}>()

const loadMoreTriggerRef = ref<HTMLElement>()
let observer: IntersectionObserver | null = null

const tryLoadMore = () => {
  if (!props.loading && !props.loadingMore && props.hasMore) {
    emit('loadMore')
  }
}

onMounted(() => {
  if (!window.IntersectionObserver || !loadMoreTriggerRef.value) {
    return
  }
  observer = new IntersectionObserver((entries) => {
    if (entries.some((entry) => entry.isIntersecting)) {
      tryLoadMore()
    }
  })
  observer.observe(loadMoreTriggerRef.value)
})

onBeforeUnmount(() => {
  observer?.disconnect()
  observer = null
})
</script>

<style lang="less" scoped>
.event-timeline-container {
  padding: 40px 20px 20px;

  .event-timeline {
    :deep(.ant-timeline-item-label) {
      width: 180px;
    }

    :deep(.ant-timeline-item-tail) {
      border-left: 2px solid #e8f4ff;
    }
  }

  .event-time {
    font-size: 13px;
    color: #8c8c8c;
    white-space: nowrap;
    display: inline-block;
    width: 160px;
    text-align: right;
  }

  .event-dot {
    display: flex;
    align-items: center;
    justify-content: center;

    .dot-icon {
      font-size: 16px;

      &.normal {
        color: #1890ff;
      }

      &.warning {
        color: #faad14;
      }
    }
  }

  .event-card {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 16px;
    border-radius: 6px;
    border-left: 4px solid #1890ff;
    background: #fff;
    box-shadow: 0 1px 4px rgba(0, 0, 0, 0.08);
    transition: box-shadow 0.2s;

    &:hover {
      box-shadow: 0 2px 8px rgba(0, 0, 0, 0.12);
    }

    &.warning {
      border-left-color: #faad14;
      background: #fff7e6;
    }

    .event-message {
      flex: 1;
      font-size: 14px;
      color: #262626;
      margin-right: 12px;
      line-height: 1.5;
    }

    .event-source-tag {
      flex-shrink: 0;
      font-size: 12px;
      border-radius: 4px;
    }
  }

  .bottom-arrow {
    font-size: 14px;
    color: #bfbfbf;
  }

  .bottom-hint {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 13px;
    color: #bfbfbf;
    padding: 8px 0;
  }

  .load-more-spinner {
    flex-shrink: 0;
  }

  .load-more-trigger {
    width: 100%;
    height: 1px;
  }
}
</style>
