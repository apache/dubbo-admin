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
  <div ref="pageElement" class="grafana-page">
    <!--    <div class="option">-->
    <!--      <a-button class="btn" @click="refresh"> refresh </a-button>-->
    <!--      <a-button class="btn" @click="newPageForGrafana"> grafana </a-button>-->
    <!--    </div>-->
    <a-alert v-if="grafana.error" type="warning" show-icon :message="grafana.error" />
    <a-spin v-else class="spin" :spinning="!grafana.showIframe">
      <div class="grafana-frame-container" :style="{ height: `${frameHeight}px` }">
        <iframe
          v-if="grafana.showIframe"
          :src="dashboardUrl"
          title="Grafana dashboard"
          frameborder="0"
        ></iframe>
      </div>
    </a-spin>
  </div>
</template>

<script setup lang="ts">
import { computed, inject, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { message } from 'ant-design-vue'
import { PROVIDE_INJECT_KEY } from '@/base/enums/ProvideInject'
import type { GrafanaState } from '@/types/grafana'
import { embedDashboardUrl } from './grafanaEmbed'

const grafana = inject<GrafanaState>(PROVIDE_INJECT_KEY.GRAFANA)
const { t } = useI18n()

if (!grafana) {
  throw new Error('Grafana state not provided')
}

const pageElement = ref<HTMLElement | null>(null)
const frameHeight = ref(0)
const dashboardUrl = computed(() => embedDashboardUrl(grafana.url || ''))
let resizeObserver: ResizeObserver | undefined

function updateFrameHeight() {
  const page = pageElement.value
  if (!page) return
  const container = page.querySelector('.grafana-frame-container')
  const boundary = page.closest('#layout-tab-body, .layout-content')
  if (!container || !boundary) return

  const bottomPadding = boundary.id === 'layout-tab-body' ? 20 : 24
  frameHeight.value = Math.max(
    0,
    Math.floor(
      boundary.getBoundingClientRect().bottom -
        container.getBoundingClientRect().top -
        bottomPadding
    )
  )
}

onMounted(async () => {
  const boundary = pageElement.value?.closest('#layout-tab-body, .layout-content')
  if (boundary) {
    resizeObserver = new ResizeObserver(updateFrameHeight)
    resizeObserver.observe(boundary)
  }
  window.addEventListener('resize', updateFrameHeight)
  updateFrameHeight()
  const requestVersion = grafana.requestVersion ?? 0
  try {
    let res = await grafana.api(grafana.params || {})
    if (requestVersion !== (grafana.requestVersion ?? 0)) return
    if (res?.data?.fullURL) {
      grafana.url = res.data.fullURL
      grafana.showIframe = true
      await nextTick()
      updateFrameHeight()
    } else {
      grafana.error = t('grafana.dashboardNotConfigured')
      message.error(grafana.error)
    }
  } catch (error) {
    if (requestVersion !== (grafana.requestVersion ?? 0)) return
    grafana.error = t('grafana.dashboardUnavailable')
    console.error('Grafana dashboard request failed:', error)
    message.error(grafana.error)
  }
})

onUnmounted(() => {
  resizeObserver?.disconnect()
  window.removeEventListener('resize', updateFrameHeight)
})
</script>
<style lang="less" scoped>
.grafana-page,
.grafana-frame-container,
.grafana-frame-container iframe {
  width: 100%;
}

.grafana-frame-container {
  overflow: hidden;
}

.grafana-frame-container iframe {
  display: block;
  height: 100%;
}
</style>
