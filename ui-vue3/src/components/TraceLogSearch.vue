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
import { message } from 'ant-design-vue'
import { getTraceLogDashboard, getTraceStartTime } from '@/api/service/trace'
import { isValidTraceId, resolveTraceLogRange } from './traceLogRange'
import { traceLogDashboardErrorMessage, traceLogSearchErrorMessage } from './traceLogSearchError'
import { PROVIDE_INJECT_KEY } from '@/base/enums/ProvideInject'
import type { GrafanaState } from '@/types/grafana'

const grafana = inject<GrafanaState>(PROVIDE_INJECT_KEY.GRAFANA)
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
    showSearchError('Trace ID is required')
    return
  }
  if (!isValidTraceId(id)) {
    showSearchError('Trace ID must be 16 or 32 hexadecimal characters')
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
      stage === 'trace'
        ? traceLogSearchErrorMessage(error)
        : traceLogDashboardErrorMessage(error)
    )
  } finally {
    loading.value = false
  }
}
</script>
