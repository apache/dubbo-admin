<template>
  <!-- GrafanaPage owns loading, Not Configured, and error states before showing the iframe. -->
  <div class="instance-log-tab">
    <TraceLogSearch />
    <GrafanaPage />
  </div>
</template>

<script setup lang="ts">
import { provide, reactive } from 'vue'
import { useRoute } from 'vue-router'
import GrafanaPage from '@/components/GrafanaPage.vue'
import TraceLogSearch from '@/components/TraceLogSearch.vue'
import { getInstanceLogDashboard } from '@/api/service/instance'
import { PROVIDE_INJECT_KEY } from '@/base/enums/ProvideInject'
import type { GrafanaState } from '@/types/grafana'

const route = useRoute()
provide<GrafanaState>(
  PROVIDE_INJECT_KEY.GRAFANA,
  reactive({
    api: getInstanceLogDashboard,
    showIframe: false,
    params: { instanceName: route.params.name as string }
  })
)
</script>
