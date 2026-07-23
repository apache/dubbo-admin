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
  <div class="__container_AppTabHeaderSlot">
    <a-row>
      <a-col :span="12">
        <span class="header-desc"> {{ t('routingRuleDomain.createNewRoutingRule') }} </span>
      </a-col>
    </a-row>
  </div>
</template>

<script setup lang="ts">
import { inject } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { createTrafficDraftContribution, useAIContextProvider } from '@/ai-context'
import { PROVIDE_INJECT_KEY } from '@/base/enums/ProvideInject'

const { t } = useI18n()
const route = useRoute()
const TAB_STATE: any = inject(PROVIDE_INJECT_KEY.TAB_LAYOUT_STATE)

useAIContextProvider({
  id: 'condition-rule-draft',
  priority: 100,
  collect: () =>
    createTrafficDraftContribution({
      kind: 'condition-rule',
      mode: 'create',
      representation: String(route.name).endsWith('YAMLView') ? 'yaml' : 'form',
      draft: TAB_STATE?.conditionRule,
      version: TAB_STATE?.addConditionRuleSate?.version,
      group: TAB_STATE?.addConditionRuleSate?.group
    })
})
</script>
<style lang="less" scoped>
.__container_AppTabHeaderSlot {
  .header-desc {
    line-height: 30px;
    vertical-align: center;
  }
}
</style>
