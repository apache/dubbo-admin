<!--
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
-->

<template>
  <a-form layout="inline" @submit.prevent="submit">
    <a-form-item :label="$t('traceLog.traceId')">
      <a-input
        v-model:value="traceId"
        allow-clear
        :placeholder="$t('traceLog.traceIdPlaceholder')"
      />
    </a-form-item>
    <a-button type="primary" :loading="loading" @click="submit">
      {{ $t('traceLog.search') }}
    </a-button>
  </a-form>
</template>

<script setup lang="ts">
import { inject, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { message } from 'ant-design-vue'
import { getTraceLogDashboard, getTraceStartTime } from '@/api/service/trace'
import { isValidTraceId, resolveTraceLogRange } from './traceLogRange'
import { traceLogDashboardErrorMessage, traceLogSearchErrorMessage } from './traceLogSearchError'
import { PROVIDE_INJECT_KEY } from '@/base/enums/ProvideInject'
import type { GrafanaState } from '@/types/grafana'

const grafana = inject<GrafanaState>(PROVIDE_INJECT_KEY.GRAFANA)
const { t } = useI18n()
const traceId = ref('')
const loading = ref(false)

function showSearchError(text: string) {
  if (grafana) {
    grafana.showIframe = false
    grafana.error = text
  }
  message.error(text)
}

const submit = async () => {
  if (loading.value) return
  if (grafana) grafana.requestVersion = (grafana.requestVersion ?? 0) + 1
  const id = traceId.value.trim()
  if (!id) {
    showSearchError(t('traceLog.traceIdRequired'))
    return
  }
  if (!isValidTraceId(id)) {
    showSearchError(t('traceLog.invalidTraceId'))
    return
  }
  loading.value = true
  if (grafana) {
    grafana.showIframe = false
    grafana.error = ''
  }
  let stage: 'trace' | 'dashboard' = 'trace'
  try {
    const result = await getTraceStartTime(id)
    const range = resolveTraceLogRange({
      traceId: result.data?.traceId,
      startTime: result.data?.startTime
    })
    stage = 'dashboard'
    const dashboard = await getTraceLogDashboard(range)
    if (!grafana || !dashboard.data?.fullURL) throw new Error('Log dashboard is unavailable')
    grafana.url = dashboard.data.fullURL
    grafana.showIframe = true
  } catch (error) {
    showSearchError(
      t(
        stage === 'trace' ? traceLogSearchErrorMessage(error) : traceLogDashboardErrorMessage(error)
      )
    )
  } finally {
    loading.value = false
  }
}
</script>
