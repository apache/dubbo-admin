<!--
  - Licensed to the Apache Software Foundation (ASF) under one or more
  - contributor license agreements.  See the NOTICE file distributed with
  - this work for additional information regarding copyright ownership.
  - The ASF licenses this file to You under the Apache License, Version 2.0
  - (the "License"); you may not use this file except in compliance with
  - the License.  You may obtain a copy of the License at
  -
  -     http://www.apache.org/licenses/LICENSE-2.0
  -
  - Unless required by applicable law or agreed to in writing, software
  - distributed under the License is distributed on an "AS IS" BASIS,
  - WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  - See the License for the specific language governing permissions and
  - limitations under the License.
  -->
<template>
  <div :style="{height: instanceConfig.height, width: instanceConfig.width}"></div>
</template>

<script>
import brace from 'brace'

let defaultConfig = {
  width: '100%',
  height: '300px',
  lang: 'yaml',
  theme: 'monokai',
  readonly: false,
  fontSize: 14,
  tabSize: 2,
  overrideValueHistory: true
}

export default {
  name: 'ace-editor',
  props: {
    value: String,
    config: {
      type: Object,
      default () {
        return {}
      }
    }
  },
  data () {
    return {
      $ace: null,
      instanceConfig: Object.assign({}, defaultConfig, this.config),
      _content: ''
    }
  },
  watch: {
    value (newVal, oldVal) {
      if (newVal !== oldVal) {
        if (this._content !== newVal) {
          this._content = newVal
          if (this.instanceConfig.overrideValueHistory) {
            this.$ace.getSession().setValue(newVal)
          } else {
            this.$ace.setValue(newVal, 1)
          }
        }
      }
    },
    config (newVal, oldVal) {
      if (newVal !== oldVal) {
        this.instanceConfig = Object.assign({}, defaultConfig, newVal)
        this.initAce(this.instanceConfig)
      }
    }
  },
  methods: {
    initAce (conf) {
      this.$ace = brace.edit(this.$el)
      this.$ace.$blockScrolling = Infinity

      this.$emit('init', this.$ace)

      require(`brace/mode/${conf.lang}`)
      require(`brace/theme/${conf.theme}`)

      let session = this.$ace.getSession()
      session.setMode(`ace/mode/${conf.lang}`)
      session.setTabSize(conf.tabSize)
      session.setUseSoftTabs(true)
      session.setUseWrapMode(true)

      if (conf.overrideValueHistory) {
        session.setValue(this.value)
      } else {
        this.$ace.setValue(this.value, 1)
      }

      this.$ace.setTheme(`ace/theme/${conf.theme}`)
      this.$ace.setReadOnly(conf.readonly)
      this.$ace.setFontSize(conf.fontSize)
      this.$ace.setShowPrintMargin(false)

      this.$ace.on('change', () => {
        var aceValue = this.$ace.getValue()
        this.$emit('input', aceValue)
        this._content = aceValue
      })
    }
  },
  mounted () {
    if (this.instanceConfig) {
      this.initAce(this.instanceConfig)
    } else {
      this.initAce(defaultConfig)
    }
  }
}
</script>
