<!--
  - Licensed to the Apache Software Foundation (ASF) under one or more
  - contributor license agreements.  See the NOTICE file distributed with
  - this work for additional information regarding copyright ownership.
  - The ASF licenses this file to You under the Apache License, Version 2.0
  - (the "License"); you may not use this file except in compliance with
  - the License.  You may obtain a copy of the License at
  -
  -   http://www.apache.org/licenses/LICENSE-2.0
  -
  - Unless required by applicable law or agreed to in writing, software
  - distributed under the License is distributed on an "AS IS" BASIS,
  - WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  - See the License for the specific language governing permissions and
  - limitations under the License.
  -->
<template>
    <div ref="jsoneditor">
    </div>
</template>

<script>
import JSONEditor from 'jsoneditor/dist/jsoneditor-minimalist.js'
import 'jsoneditor/dist/jsoneditor.min.css'
import _ from 'lodash'
export default {
  name: 'json-editor2',
  data () {
    return {
      editor: null,
      maxed: false,
      jsoneditorModes: null
    }
  },
  props: {
    json: {
      required: true
    },
    options: {
      type: Object,
      default: () => {
        return {}
      }
    },
    onChange: {
      type: Function
    }
  },
  watch: {
    json: {
      handler (newJson) {
        if (this.editor) {
          this.editor.set(newJson)
        }
      },
      deep: true
    }
  },
  methods: {
    _onChange (e) {
      if (this.onChange && this.editor) {
        this.onChange(this.editor.get())
      }
    },
    _onModeChange (newMode, oldMode) {
      const container = this.$refs.jsoneditor
      if (container.getElementsByClassName('jsoneditor-modes') && container.getElementsByClassName('jsoneditor-modes')[0]) {
        this.jsoneditorModes = container.getElementsByClassName('jsoneditor-modes')[0]
      }
      if (newMode === 'code') {
        this.addMaxBtn()
      }
    },
    addMaxBtn () {
      const container = this.$refs.jsoneditor
      var jsoneditorMunusDiv = container.getElementsByClassName('jsoneditor-menu')[0]
      var maxBtn = document.createElement('button')
      maxBtn.type = 'button'
      maxBtn.classList.add('jsoneditor-max-btn')
      maxBtn.jsoneditor = {}
      maxBtn.jsoneditor.maxed = this.maxed
      maxBtn.jsoneditor.editor = this.$refs.jsoneditor
      var _this = this
      maxBtn.onclick = function () {
        if (this.jsoneditor.maxed) {
          if (!container.getElementsByClassName('jsoneditor-modes')[0]) {
            maxBtn.before(_this.jsoneditorModes)
          }
          this.jsoneditor.editor.classList.remove('jsoneditor-max')
          this.jsoneditor.maxed = false
        } else {
          if (container.getElementsByClassName('jsoneditor-modes') && container.getElementsByClassName('jsoneditor-modes')[0]) {
            container.getElementsByClassName('jsoneditor-modes')[0].remove()
          }
          this.jsoneditor.editor.classList.add('jsoneditor-max')
          this.jsoneditor.maxed = true
        }
      }
      jsoneditorMunusDiv.appendChild(maxBtn)
    }
  },
  mounted () {
    const container = this.$refs.jsoneditor
    const options = _.extend({
      onChange: this._onChange,
      onModeChange: this._onModeChange
    }, this.options)
    this.editor = new JSONEditor(container, options)
    this.editor.set(this.json)

    if (container.getElementsByClassName('jsoneditor-modes') && container.getElementsByClassName('jsoneditor-modes')[0]) {
      this.jsoneditorModes = container.getElementsByClassName('jsoneditor-modes')[0]
    }
  },
  beforeDestroy () {
    if (this.editor) {
      this.editor.destroy()
      this.editor = null
    }
  }
}
</script>

<style>

.jsoneditor-max-btn {
  position: absolute;
  top: 6px;
  width: 18px !important;
  height: 18px !important;
  background: rgba(0, 0, 0, 0) url(./assets/max_btn.svg) no-repeat !important;
  background-position: 3px;
}

.jsoneditor-max {
  position: fixed;
  top: 0px;
  left: 0px;
  width: 100%;
  height: 100% !important;
  z-index: 10000;
}

</style>
