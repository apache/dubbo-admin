<!--
  Licensed to the Apache Software Foundation (ASF) under one or more
  contributor license agreements.  See the NOTICE file distributed with
  this work for additional information regarding copyright ownership.
  The ASF licenses this file to You under the Apache License, Version 2.0
  (the "License"); you may not use this file except in compliance with
  the License.  You may obtain a copy of the License at

      http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.
-->

<template>
  <a-space direction="vertical" size="middle" style="width: 100%">
    <a-alert :message="t('routingRuleDomain.structuredConditionHint')" type="info" show-icon />
    <a-card v-for="(condition, conditionIndex) in modelValue" :key="conditionIndex">
      <template #title>
        <a-flex justify="space-between" align="center">
          <span>{{ t('routingRuleDomain.route') }}【{{ conditionIndex + 1 }}】</span>
          <a-button v-if="!readonly" danger type="text" @click="removeCondition(conditionIndex)">
            {{ t('delete') }}
          </a-button>
        </a-flex>
      </template>

      <a-form layout="vertical">
        <a-form-item :label="t('routingRuleDomain.sourceMatch')" required>
          <a-input
            v-model:value="condition.from.match"
            :readonly="readonly"
            placeholder="method=SayHello & arguments[0]=gray"
          />
        </a-form-item>

        <a-form-item :label="t('routingRuleDomain.destinations')" required>
          <a-table
            :columns="columns"
            :data-source="condition.to"
            :pagination="false"
            row-key="match"
          >
            <template #bodyCell="{ column, record, index }">
              <template v-if="column.key === 'match'">
                <a-input
                  v-model:value="record.match"
                  :readonly="readonly"
                  placeholder="region=hangzhou"
                />
              </template>
              <template v-else-if="column.key === 'weight'">
                <a-input-number
                  v-model:value="record.weight"
                  :disabled="readonly"
                  :min="0"
                  style="width: 120px"
                />
              </template>
              <template v-else-if="column.key === 'operation'">
                <a-button
                  v-if="!readonly"
                  danger
                  type="text"
                  @click="removeDestination(conditionIndex, index)"
                >
                  {{ t('delete') }}
                </a-button>
              </template>
            </template>
          </a-table>
          <a-button
            v-if="!readonly"
            type="dashed"
            style="margin-top: 12px"
            @click="addDestination(conditionIndex)"
          >
            {{ t('routingRuleDomain.addDestination') }}
          </a-button>
        </a-form-item>
      </a-form>
    </a-card>
    <a-button v-if="!readonly" type="primary" @click="addCondition">
      {{ t('routingRuleDomain.addRoute') }}
    </a-button>
  </a-space>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  newStructuredConditionRule,
  type StructuredConditionRule
} from '../model/ConditionRuleModel'

const props = withDefaults(
  defineProps<{
    modelValue: StructuredConditionRule[]
    readonly?: boolean
  }>(),
  { readonly: false }
)
const emit = defineEmits<{ 'update:modelValue': [value: StructuredConditionRule[]] }>()
const { t } = useI18n()

const columns = computed(() => {
  const result = [
    { key: 'match', dataIndex: 'match', title: t('routingRuleDomain.destinationMatch') },
    { key: 'weight', dataIndex: 'weight', title: t('routingRuleDomain.weight'), width: 160 }
  ]
  if (!props.readonly) {
    result.push({ key: 'operation', dataIndex: 'operation', title: t('operation'), width: 120 })
  }
  return result
})

function update(mutator: (conditions: StructuredConditionRule[]) => void) {
  const conditions = props.modelValue.map((condition) => ({
    from: { match: condition.from.match },
    to: condition.to.map((destination) => ({ ...destination }))
  }))
  mutator(conditions)
  emit('update:modelValue', conditions)
}

function addCondition() {
  update((conditions) => conditions.push(newStructuredConditionRule()))
}

function removeCondition(index: number) {
  update((conditions) => conditions.splice(index, 1))
}

function addDestination(conditionIndex: number) {
  update((conditions) => conditions[conditionIndex].to.push({ match: '', weight: 0 }))
}

function removeDestination(conditionIndex: number, destinationIndex: number) {
  update((conditions) => conditions[conditionIndex].to.splice(destinationIndex, 1))
}
</script>
