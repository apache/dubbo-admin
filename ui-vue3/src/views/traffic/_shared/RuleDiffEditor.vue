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
  <div :id="editorId" class="rule-diff-editor" :style="{ height: editorHeight }"></div>
</template>

<script lang="ts" setup>
import { computed, onBeforeUnmount, onMounted, watch } from 'vue'
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
  editorId: {
    type: String,
    default: 'rule-diff-editor'
  },
  height: {
    type: [String, Number],
    default: '420px'
  }
})

let diffEditor: monaco.editor.IStandaloneDiffEditor | null = null
const editorHeight = computed(() =>
  typeof props.height === 'number' ? `${props.height}px` : props.height
)

const render = () => {
  if (!diffEditor) {
    return
  }
  const originalModel = monaco.editor.createModel(props.original || '', 'json')
  const modifiedModel = monaco.editor.createModel(props.modified || '', 'json')
  diffEditor.setModel({ original: originalModel, modified: modifiedModel })
}

onMounted(() => {
  const el = document.getElementById(props.editorId)
  if (!el) {
    return
  }
  diffEditor = monaco.editor.createDiffEditor(el, {
    automaticLayout: true,
    renderSideBySide: true,
    readOnly: true,
    minimap: { enabled: false }
  })
  render()
})

watch(
  () => [props.original, props.modified],
  () => render()
)

onBeforeUnmount(() => {
  diffEditor?.dispose()
  diffEditor = null
})
</script>

<style scoped lang="less">
.rule-diff-editor {
  width: 100%;
}
</style>
