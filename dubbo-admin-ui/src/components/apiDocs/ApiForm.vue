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
  <div v-if="showForm">
    <v-form>
      <h3>
        {{ $t('apiDocsRes.apiForm.apiNameShowLabel') }}:
        {{ formInfo.moduleClassName }}.{{ formInfo.apiName }}
      </h3>
      <h3>
        {{ $t('apiDocsRes.apiForm.apiRespDecShowLabel') }}:
        {{ formInfo.apiRespDec }}
      </h3>
      <h3>
        {{ $t('apiDocsRes.apiForm.apiVersionShowLabel') }}:
        {{ formInfo.apiVersion }}
      </h3>
      <h3>
        {{ $t('apiDocsRes.apiForm.apiDescriptionShowLabel') }}:
        {{ formInfo.description }}
      </h3>
      <v-select
        v-model="formItemAsync"
        :items="formItemAsyncSelectItems"
        :label="$t('apiDocsRes.apiForm.isAsyncFormLabel')"
        outline
        readonly
      ></v-select>
      <v-text-field
        :label="$t('apiDocsRes.apiForm.apiModuleFormLabel')"
        :value="this.apiInfoData.apiModelClass"
        outline
        readonly
      ></v-text-field>
      <v-text-field
        :label="$t('apiDocsRes.apiForm.apiFunctionNameFormLabel')"
        :value="this.apiInfoData.apiName"
        outline
        readonly
      ></v-text-field>
      <v-text-field
        :label="$t('apiDocsRes.apiForm.registryCenterUrlFormLabel')"
        placeholder="nacos://127.0.0.1:8848"
        outline
      ></v-text-field>

      <div style="marginTop: 20px">
        {{ $t('apiDocsRes.apiForm.prarmNameLabel') }}:xxx
        <v-layout row wrap>
          <v-flex lg3>
            <v-card>
              {{ $t('apiDocsRes.apiForm.paramDescriptionShowLabel') }}: <br />
              xxxx
            </v-card>
          </v-flex>
          <v-flex lg9>
              <v-text-field
                label="xxx"
                placeholder="127.0.0.1"
                value="127.0.0.1"
                outline
              ></v-text-field>
          </v-flex>
        </v-layout>
      </div>

      <div style="marginTop: 20px">
        {{ $t('apiDocsRes.apiForm.prarmNameLabel') }}:xxx
        <v-layout row wrap>
          <v-flex lg3>
            <v-card>
              {{ $t('apiDocsRes.apiForm.paramDescriptionShowLabel') }}: <br />
              xxxx
            </v-card>
          </v-flex>
          <v-flex lg9>
              <v-select
                v-model="formItemAsync"
                :items="formItemAsyncSelectItems"
                :label="$t('apiDocsRes.apiForm.isAsyncFormLabel')"
                outline
              ></v-select>
          </v-flex>
        </v-layout>
      </div>

      <div style="marginTop: 20px">
        {{ $t('apiDocsRes.apiForm.prarmNameLabel') }}:xxx
        <v-layout row wrap>
          <v-flex lg3>
            <v-card>
              {{ $t('apiDocsRes.apiForm.paramDescriptionShowLabel') }}: <br />
              xxxx
            </v-card>
          </v-flex>
          <v-flex lg9>
              <v-textarea
                outline
                name="input-7-4"
                label="Outline textarea"
                value="The Woodman set to work at once, and so sharp was his axe that the tree was soon chopped nearly through."
              ></v-textarea>
          </v-flex>
        </v-layout>
      </div>
    </v-form>
  </div>
</template>

<script>
export default {
  name: 'ApiForm',
  props: {
    formInfo: {
      type: Object,
      default: function () {
        return {}
      }
    }
  },
  data: () => {
    return {
      showForm: false,
      formItemAsyncSelectItems: [true, false],
      formItemAsync: false,
      apiInfoData: {},
      publicFormsArray: []
    }
  },
  watch: {
    formInfo: 'changeFormInfo'
  },
  methods: {
    changeFormInfo (curVal) {
      this.$axios
        .get('/docs/apiParamsResp', {
          params: {
            dubboIp: curVal.dubboIp,
            dubboPort: curVal.dubboPort,
            apiName: curVal.moduleClassName + '.' + curVal.apiName
          }
        })
        .then((response) => {
          if (response && response.data && response.data !== '') {
            this.apiInfoData = JSON.parse(response.data)
            this.formItemAsync = this.apiInfoData.async
            var params = this.apiInfoData.params
            const formsArray = []
            for (var i = 0; i < params.length; i++) {
              var paramItem = params[i]
              if (paramItem.htmlType) {
                // Has htmlType, that's a basic type
                var formItem = new Map()
                formItem.set('name', paramItem.name)
                formItem.set('htmlType', paramItem.htmlType)
                formItem.set('paramType', paramItem.prarmType)
                formItem.set('javaType', paramItem.prarmType)
                formItem.set('paramIndex', paramItem.prarmIndex)
                formItem.set('nameCh', paramItem.nameCh)
                formItem.set('description', paramItem.description)
                formItem.set('example', paramItem.example)
                formItem.set('defaultValue', paramItem.defaultValue)
                formItem.set('allowableValues', paramItem.allowableValues)
                formItem.set('required', paramItem.required)
                formsArray.push(formItem)
              } else {
                // No htmltype, that's an object
                var prarmInfoArray = paramItem.prarmInfo
                for (var j = 0; j < prarmInfoArray.length; j++) {
                  var prarmInfoItem = prarmInfoArray[j]
                  // eslint-disable-next-line no-redeclare
                  var formItem = new Map()
                  formItem.set('name', prarmInfoItem.name)
                  formItem.set('htmlType', prarmInfoItem.htmlType)
                  formItem.set('paramType', paramItem.prarmType)
                  formItem.set('javaType', prarmInfoItem.javaType)
                  formItem.set('paramIndex', paramItem.prarmIndex)
                  formItem.set('nameCh', prarmInfoItem.nameCh)
                  formItem.set('description', prarmInfoItem.description)
                  formItem.set('example', prarmInfoItem.example)
                  formItem.set('defaultValue', prarmInfoItem.defaultValue)
                  formItem.set(
                    'allowableValues',
                    prarmInfoItem.allowableValues
                  )
                  formItem.set('subParamsJson', prarmInfoItem.subParamsJson)
                  formItem.set('required', prarmInfoItem.required)
                  formsArray.push(formItem)
                }
              }
            }
            this.publicFormsArray = formsArray
          }
        })
        .catch((error) => {
          console.log('error', error.message)
        })
      this.showForm = true
    }
  },
  mounted () {}
}
</script>

<style scoped>
</style>
