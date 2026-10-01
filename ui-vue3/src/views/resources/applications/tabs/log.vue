<template>
  <!-- GrafanaPage owns loading, Not Configured, and error states before showing the iframe. -->
  <div class="application-log-tab">
    <TraceLogSearch />
    <GrafanaPage />
  </div>
</template>

<script setup lang="ts">
import { provide, reactive } from 'vue'
import { useRoute } from 'vue-router'
import GrafanaPage from '@/components/GrafanaPage.vue'
import TraceLogSearch from '@/components/TraceLogSearch.vue'
import { getApplicationLogDashboard } from '@/api/service/app'
import { PROVIDE_INJECT_KEY } from '@/base/enums/ProvideInject'
import type { GrafanaState } from '@/types/grafana'

const route = useRoute()
provide<GrafanaState>(
  PROVIDE_INJECT_KEY.GRAFANA,
  reactive({
    api: getApplicationLogDashboard,
    showIframe: false,
    params: { appName: route.params.pathId as string }
  })
)
</script>
