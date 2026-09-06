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
  <div ref="editorEl" class="rule-diff-editor" :style="{ height: editorHeight }"></div>
</template>

<script lang="ts" setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import * as monaco from 'monaco-editor'

const props = defineProps({
  original: {
    type: String,
    default: ''
  },
  modified: {
    type: String,
    default: ''
  },
  height: {
    type: [String, Number],
    default: '420px'
  }
})

let diffEditor: monaco.editor.IStandaloneDiffEditor | null = null
const editorEl = ref<HTMLElement | null>(null)
const editorHeight = computed(() =>
  typeof props.height === 'number' ? `${props.height}px` : props.height
)
const isJsonDocument = (value: string) => {
  const trimmed = value.trim()
  if (!trimmed) {
    return false
  }
  try {
    JSON.parse(trimmed)
    return true
  } catch {
    return false
  }
}
const editorLanguage = computed(() =>
  isJsonDocument(props.original || '') && isJsonDocument(props.modified || '')
    ? 'json'
    : 'plaintext'
)

const disposeModels = () => {
  // The diff editor does not own models passed through setModel. Dispose the
  // previous pair before each rebuild so Monaco workers do not retain snapshots.
  const model = diffEditor?.getModel()
  model?.original.dispose()
  model?.modified.dispose()
}

const render = () => {
  if (!diffEditor) {
    return
  }
  disposeModels()
  // Models are recreated when language changes; a disposed model must never be
  // reused after Monaco has detached it from the editor.
  const originalModel = monaco.editor.createModel(props.original || '', editorLanguage.value)
  const modifiedModel = monaco.editor.createModel(props.modified || '', editorLanguage.value)
  diffEditor.setModel({ original: originalModel, modified: modifiedModel })
}

onMounted(() => {
  if (!editorEl.value) {
    return
  }
  diffEditor = monaco.editor.createDiffEditor(editorEl.value, {
    automaticLayout: true,
    renderSideBySide: true,
    readOnly: true,
    minimap: { enabled: false },
    renderOverviewRuler: false,
    overviewRulerLanes: 0,
    overviewRulerBorder: false
  })
  render()
})

watch(
  () => [props.original, props.modified, editorLanguage.value],
  () => render()
)

onBeforeUnmount(() => {
  disposeModels()
  diffEditor?.dispose()
  diffEditor = null
})
</script>

<style scoped lang="less">
.rule-diff-editor {
  width: 100%;
}
</style>
