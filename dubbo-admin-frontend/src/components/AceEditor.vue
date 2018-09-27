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
  <div :style="{height: myConfig.height, width: myConfig.width}"></div>
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
  tabSize: 2
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
      myConfig: Object.assign({}, defaultConfig, this.config),
      $ace: null
    }
  },
  watch: {
    config (newVal, oldVal) {
      if (newVal !== oldVal) {
        this.myConfig = Object.assign({}, defaultConfig, newVal)
        this.initAce(this.myConfig)
      }
    },
    value (newVal, oldVal) {
      if (newVal !== oldVal) {
        this.$ace.setValue(newVal, 1)
      }
    }
  },
  methods: {
    initAce (conf) {
      this.$ace = brace.edit(this.$el)
      this.$ace.$blockScrolling = Infinity // 消除警告
      let session = this.$ace.getSession()
      this.$emit('init', this.$ace)

      require(`brace/mode/${conf.lang}`)
      require(`brace/theme/${conf.theme}`)

      session.setMode(`ace/mode/${conf.lang}`) // 配置语言
      this.$ace.setTheme(`ace/theme/${conf.theme}`) // 配置主题
      this.$ace.setValue(this.value, 1) // 设置默认内容
      this.$ace.setReadOnly(conf.readonly) // 设置是否为只读模式
      this.$ace.setFontSize(conf.fontSize)
      session.setTabSize(conf.tabSize) // Tab大小
      session.setUseSoftTabs(true)

      this.$ace.setShowPrintMargin(false) // 不显示打印边距
      session.setUseWrapMode(true) // 自动换行

      // 绑定输入事件回调
      this.$ace.on('change', () => {
        var content = this.$ace.getValue()
        this.$emit('input', content)
      })
    }
  },
  mounted () {
    if (this.myConfig) {
      this.initAce(this.myConfig)
    } else {
      this.initAce(defaultConfig)
    }
  }
}
</script>
