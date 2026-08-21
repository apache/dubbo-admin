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
  <div class="__container_app_topology">
    <a-flex>
      <a-card class="topology-warpper">
        <!-- G6 mount point (application topology) -->
        <div id="topology"></div>
      </a-card>
    </a-flex>
    <!-- Right drawer: show application details after node click -->
    <a-drawer v-model:open="detailDrawerOpen" :title="detailTitle" placement="right" width="520">
      <a-spin :spinning="detailLoading">
        <a-typography-text v-if="detailError" type="danger">{{ detailError }}</a-typography-text>
        <a-descriptions
          v-else
          :column="1"
          size="small"
          bordered
          :labelStyle="{ fontWeight: 'bold', width: '160px' }"
        >
          <a-descriptions-item v-for="item in detailEntries" :key="item.key">
            <template #label>{{ item.key }}</template>
            {{ formatValueForDisplay(item.value) }}
          </a-descriptions-item>
        </a-descriptions>
      </a-spin>
    </a-drawer>
  </div>
</template>

<script setup lang="tsx">
import { PRIMARY_COLOR } from '@/base/constants'
import { getApplicationDetail, getApplicationGraph } from '@/api/service/app'
import { HTTP_STATUS } from '@/base/http/constants'
import { computed, defineComponent, onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue'
import type { PropType } from 'vue'
import { useRoute } from 'vue-router'
import { ExtensionCategory, register, Graph, NodeEvent } from '@antv/g6'
import { VueNode } from 'g6-extension-vue'
import { createTopologyStateContribution, useAIContextProvider } from '@/ai-context'

const route = useRoute()

// G6 graph instance for this view (resize/destroy lifecycle)
const graphRef = shallowRef<Graph | null>(null)
const topologyData = shallowRef<{ nodes: any[]; edges: any[] }>()

// Right-side detail drawer state
const detailDrawerOpen = ref(false)
const detailLoading = ref(false)
const detailError = ref('')
const detailData = shallowRef<Record<string, unknown>>({})
const currentDetailKey = ref('')
const selectedNodeId = ref('')

useAIContextProvider({
  id: 'topology-state',
  priority: 70,
  collect: () =>
    createTopologyStateContribution(topologyData.value, {
      key: currentDetailKey.value,
      type: 'application',
      detail: detailData.value
    })
})

// Clear selection when the drawer closes to avoid stale highlight
const clearSelectedNode = () => {
  const id = selectedNodeId.value
  if (id && graphRef.value) {
    graphRef.value.setElementState(id, [])
  }
  selectedNodeId.value = ''
}

watch(detailDrawerOpen, (open) => {
  if (!open) {
    clearSelectedNode()
    currentDetailKey.value = ''
  }
})

type VueNodeViewData = {
  id?: string | number
  label?: string
  states?: string[]
  data?: Record<string, unknown>
}

// Node renderer: render a TSX component into a G6 node via VueNode
const StatefulNode = defineComponent({
  props: {
    data: { type: Object as PropType<VueNodeViewData>, required: true }
  },
  setup(props) {
    const label = computed(() => {
      return String(props.data?.label ?? props.data?.data?.label ?? props.data?.id ?? '')
    })

    const isSelected = computed(() => {
      const states = props.data?.states
      if (!Array.isArray(states)) return false
      return states.includes('selected') || states.includes('active')
    })

    return () => (
      <div
        style={{
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center'
        }}
      >
        <div
          style={{
            color: isSelected.value ? PRIMARY_COLOR.value : 'rgba(0,0,0,0.65)',
            filter: isSelected.value ? `drop-shadow(0 0 6px ${PRIMARY_COLOR.value}88)` : 'none',
            transform: `scale(${isSelected.value ? 1.06 : 1})`,
            transition: 'all 0.12s ease-in-out'
          }}
        >
          <span
            class={['iconfont', 'icon-yingyong']}
            style={{ fontSize: '40px', lineHeight: '40px' }}
          ></span>
        </div>
        <div
          style={{
            paddingTop: '4px',
            userSelect: 'none',
            textAlign: 'center',
            fontSize: '12px',
            lineHeight: '16px'
          }}
          onPointerdown={(e: any) => e.stopPropagation()}
          onMousedown={(e: any) => e.stopPropagation()}
          onClick={(e: any) => e.stopPropagation()}
        >
          {label.value}
        </div>
      </div>
    )
  }
})

// Register the VueNode extension type in G6
register(ExtensionCategory.NODE, 'vue-node', VueNode)

// Drawer title: current node (application name)
const detailTitle = computed(() => {
  return currentDetailKey.value ? `应用详情：${currentDetailKey.value}` : '应用详情'
})

// Expand detail object into description entries and filter empty values
const detailEntries = computed(() => {
  const data = detailData.value ?? {}
  return Object.entries(data)
    .filter(([, v]) => v !== undefined && v !== null && String(v) !== '')
    .map(([key, value]) => ({ key, value }))
})

// Format values for display (primitives/arrays/objects)
const formatValueForDisplay = (v: unknown): string => {
  if (v === null || v === undefined) return ''
  if (typeof v === 'string' || typeof v === 'number' || typeof v === 'boolean') return String(v)
  if (Array.isArray(v))
    return v
      .map((x) => formatValueForDisplay(x))
      .filter(Boolean)
      .join(', ')
  if (typeof v === 'object') {
    try {
      return JSON.stringify(v)
    } catch {
      return String(v)
    }
  }
  return String(v)
}

// Convert backend topology payload into G6 nodes/edges
const buildGraphData = (raw: any) => {
  const nodes = Array.isArray(raw?.nodes)
    ? raw.nodes.map((n: any) => ({
        id: String(n?.id ?? ''),
        label: n?.label ?? n?.id,
        data: n?.data
      }))
    : []

  const edges = Array.isArray(raw?.edges)
    ? raw.edges.map((e: any, idx: number) => ({
        id: e?.id ?? `edge-${idx}`,
        source: String(e?.source ?? ''),
        target: String(e?.target ?? '')
      }))
    : []

  return { nodes, edges }
}

// Render topology: create Graph, configure layout/styles/behaviors, bind node click
const renderTopology = (graphData: any) => {
  const root = document.getElementById('topology')
  if (!root) return

  graphRef.value?.destroy()
  graphRef.value = null

  const primaryColor = PRIMARY_COLOR.value
  const graph = new Graph({
    container: root,
    width: root.clientWidth || 800,
    height: Math.max(root.clientHeight || 500, 500),
    autoFit: 'view',
    padding: 20,
    data: graphData,
    layout: {
      type: 'd3-force',
      link: { distance: 180, strength: 1 },
      collide: { radius: 40 }
    },
    node: {
      type: 'vue-node',
      style: {
        component: (data: any) => <StatefulNode data={Object.assign({}, data)} />
      }
    },
    edge: {
      style: {
        stroke: primaryColor,
        endArrow: true,
        lineWidth: 1.2,
        strokeOpacity: 0.8
      }
    },

    behaviors: [
      'drag-canvas',
      'zoom-canvas',
      'click-select',
      { type: 'drag-element-force', fixed: true }
    ]
  })

  // On node click, fetch application detail and show it in the drawer
  const handleNodeClick = async (e: any) => {
    const rawId = e?.target?.id
    const appName = rawId == null ? '' : String(rawId)
    if (!appName) return

    selectedNodeId.value = appName
    detailDrawerOpen.value = true
    currentDetailKey.value = appName
    detailError.value = ''

    detailLoading.value = true
    try {
      const res = await getApplicationDetail(appName)
      if (res?.code !== HTTP_STATUS.SUCCESS) {
        detailError.value = String(res?.message ?? '请求失败')
        detailData.value = {}
        return
      }

      const data = (res?.data ?? {}) as Record<string, unknown>
      detailData.value = data
    } catch {
      detailError.value = '请求失败'
      detailData.value = {}
    } finally {
      detailLoading.value = false
    }
  }
  graph.on(NodeEvent.CLICK, handleNodeClick)
  graph.render()
  graphRef.value = graph
}

let resizeHandler: (() => void) | null = null
onMounted(async () => {
  try {
    // Fetch application graph by route param and render with force layout
    const appName = String(route.params?.pathId ?? '')
    const res = await getApplicationGraph(appName)
    if (res?.code !== HTTP_STATUS.SUCCESS) return

    const graphData = buildGraphData(res?.data)
    topologyData.value = graphData
    renderTopology(graphData)

    // Resize canvas on window resize (keep container width responsive)
    resizeHandler = () => {
      const root = document.getElementById('topology')
      if (!root || !graphRef.value) return
      graphRef.value.resize(root.clientWidth || 800, 500)
    }
    window.addEventListener('resize', resizeHandler)
  } catch {}
})

onBeforeUnmount(() => {
  if (resizeHandler) window.removeEventListener('resize', resizeHandler)
  // Destroy Graph on unmount to release events/resources
  graphRef.value?.destroy()
  graphRef.value = null
})
</script>
<style lang="less" scoped>
.topology-warpper {
  width: 100%;
}
#topology {
  width: 100%;
}
</style>
