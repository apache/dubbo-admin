<template>
  <div :style="{height: myConfig.height + 'px', width: myConfig.width + 'px'}"></div>
</template>

<script>
import brace from 'brace'

let defaultConfig = {
  width: '100%',
  height: 300,
  lang: 'yaml',
  theme: 'monokai',
  readonly: false,
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
      this.$ace.setFontSize(14)
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
