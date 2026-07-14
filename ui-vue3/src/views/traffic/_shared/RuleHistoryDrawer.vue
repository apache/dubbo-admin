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
    width="min(92vw, 480px)"
    :destroyOnClose="false"
    @close="$emit('update:open', false)"
  >
    <template #title>
      <div class="drawer-title">
        <a-typography-text strong>{{ title }}</a-typography-text>
        <a-tag v-if="latestRecordedVersionNo !== undefined" color="blue">{{
          t('ruleVersionDomain.latestRecordedVersionBadge', {
            versionNo: latestRecordedVersionNo
          })
        }}</a-tag>
      </div>
    </template>

    <a-spin :spinning="loading">
      <a-empty v-if="!items.length" :description="t('ruleVersionDomain.empty')" />
      <div v-else class="history-list">
        <div
          v-for="item in items"
          :key="item.versionNo"
          class="history-item"
          :class="{ current: item.isLatestRecorded }"
        >
          <div class="history-head">
            <a-space wrap>
              <a-tag color="geekblue">v{{ item.versionNo }}</a-tag>
              <a-tag>{{ sourceLabel(item.source) }}</a-tag>
              <a-tag v-if="item.isLatestRecorded" color="green">{{
                t('ruleVersionDomain.latestRecorded')
              }}</a-tag>
            </a-space>
          </div>
          <div class="history-body">
            <div>{{ authorLabel(item.author) }}</div>
            <div>{{ t('ruleVersionDomain.modifiedAt') }}: {{ createdAtLabel(item.createdAt) }}</div>
            <div v-if="item.reason">{{ reasonLabel(item) }}: {{ item.reason }}</div>
          </div>
          <a-space>
            <a-button type="link" @click="$emit('view-json', item)">{{
              t('ruleVersionDomain.view')
            }}</a-button>
            <a-button type="link" @click="$emit('diff-current', item)">{{
              t('ruleVersionDomain.diffCurrent')
            }}</a-button>
            <a-tooltip v-if="isRollbackDisabled(item)" :title="rollbackDisabledReason(item)">
              <span>
                <a-button type="link" disabled>{{ t('ruleVersionDomain.rollback') }}</a-button>
              </span>
            </a-tooltip>
            <a-button v-else type="link" @click="$emit('rollback', item)">{{
              t('ruleVersionDomain.rollback')
            }}</a-button>
          </a-space>
        </div>
      </div>
    </a-spin>
  </a-drawer>
</template>

<script setup lang="ts">
import type { RuleVersion } from '@/api/service/traffic'
import dayjs from 'dayjs'
import { useI18n } from 'vue-i18n'

defineProps<{
  open: boolean
  title: string
  items: RuleVersion[]
  latestRecordedVersionNo?: number
  loading?: boolean
}>()

defineEmits(['update:open', 'view-json', 'diff-current', 'rollback'])

const { t } = useI18n()

const sourceLabels: Record<string, string> = {
  ADMIN: 'ruleVersionDomain.sourceAdmin',
  UPSTREAM: 'ruleVersionDomain.sourceUpstream',
  BOOTSTRAP: 'ruleVersionDomain.sourceBootstrap',
  ROLLBACK: 'ruleVersionDomain.sourceRollback'
}

const sourceLabel = (source: string) => (sourceLabels[source] ? t(sourceLabels[source]) : source)

const authorLabel = (author: string) => author.replace(/^system:/, '')

const createdAtLabel = (createdAt: string) => dayjs(createdAt).format('YYYY/M/D HH:mm:ss')

const isRollbackDisabled = (item: RuleVersion) => item.operation === 'DELETE'

const rollbackDisabledReason = (item: RuleVersion) => {
  if (item.operation === 'DELETE') return t('ruleVersionDomain.rollbackDeleteDisabled')
  return ''
}

const reasonLabel = (item: RuleVersion) => {
  if (item.source === 'ROLLBACK') return t('ruleVersionDomain.rollbackReason')
  if (item.operation === 'CREATE' || item.operation === 'UPDATE') {
    return t('ruleVersionDomain.changeReason')
  }
  return t('ruleVersionDomain.reason')
}
</script>

<style scoped lang="less">
.drawer-title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.history-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.history-item {
  padding: 12px 14px;
  background: #fff;
  border: 1px solid #f0f0f0;
  border-radius: 6px;
  box-shadow: 0 1px 2px rgb(0 0 0 / 4%);
}

.history-item.current {
  border-color: #b7eb8f;
  background: #fcfffa;
}

.history-head,
.history-body {
  margin-bottom: 8px;
}

.history-body {
  line-height: 1.7;
}
</style>
