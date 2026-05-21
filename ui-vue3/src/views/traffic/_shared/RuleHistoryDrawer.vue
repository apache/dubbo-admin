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
    :open="open"
    placement="right"
    width="520"
    :destroyOnClose="false"
    @close="$emit('update:open', false)"
  >
    <template #title>
      <div class="drawer-title">
        <a-typography-text strong>{{ title }}</a-typography-text>
        <a-tag v-if="currentVersionNo !== undefined" color="blue"
          >current v{{ currentVersionNo }}</a-tag
        >
      </div>
    </template>

    <a-spin :spinning="loading">
      <a-empty v-if="disabled" description="版本功能未启用" />
      <a-empty v-else-if="!items.length" description="暂无版本记录" />
      <a-timeline v-else>
        <a-timeline-item v-for="item in items" :key="item.id">
          <div class="history-item" :class="{ current: item.isCurrent }">
            <div class="history-head">
              <a-space wrap>
                <a-tag color="geekblue">v{{ item.versionNo }}</a-tag>
                <a-tag>{{ item.source }}</a-tag>
                <a-tag>{{ item.operation }}</a-tag>
                <a-tag v-if="item.isCurrent" color="green">current</a-tag>
              </a-space>
            </div>
            <div class="history-body">
              <div>{{ item.author }}</div>
              <div>{{ item.createdAt }}</div>
              <div v-if="item.reason">{{ item.reason }}</div>
            </div>
            <a-space>
              <a-button type="link" @click="$emit('view-json', item)">查看</a-button>
              <a-button type="link" @click="$emit('diff-current', item)">对比当前</a-button>
              <a-popconfirm title="确认回滚到该版本？" @confirm="$emit('rollback', item)">
                <a-button type="link">回滚</a-button>
              </a-popconfirm>
            </a-space>
          </div>
        </a-timeline-item>
      </a-timeline>
    </a-spin>
  </a-drawer>
</template>

<script setup lang="ts">
import type { RuleVersion } from '@/api/service/traffic'

defineProps<{
  open: boolean
  title: string
  items: RuleVersion[]
  currentVersionNo?: number
  loading?: boolean
  disabled?: boolean
}>()

defineEmits(['update:open', 'view-json', 'diff-current', 'rollback'])
</script>

<style scoped lang="less">
.drawer-title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.history-item {
  padding: 4px 0 12px;

  &.current {
    color: inherit;
  }
}

.history-head,
.history-body {
  margin-bottom: 6px;
}
</style>
