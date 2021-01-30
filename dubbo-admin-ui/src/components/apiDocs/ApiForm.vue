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
    <div style="padding-left: 10px;padding-right: 10px;">
      <div>
        <v-timeline
          align-top
          dense
        >
          <v-timeline-item
            color="cyan"
            small
          >
            <div>
              <div class="font-weight-normal">
                <strong>{{ $t('apiDocsRes.apiForm.apiNameShowLabel') }}</strong>
              </div>
              <div>{{ this.apiInfoData.apiDocName }}</div>
            </div>
          </v-timeline-item>
          <v-timeline-item
            color="cyan"
            small
          >
            <div>
              <div class="font-weight-normal">
                <strong>{{ $t('apiDocsRes.apiForm.apiPathShowLabel') }}</strong>
              </div>
              <div>{{ this.apiInfoData.apiModelClass }}#{{ this.apiInfoData.apiName }}</div>
            </div>
          </v-timeline-item>
          <v-timeline-item
            color="cyan"
            small
          >
            <div>
              <div class="font-weight-normal">
                <strong>{{ $t('apiDocsRes.apiForm.apiMethodParamInfoLabel') }}</strong>
              </div>
              <div>{{ this.apiInfoData.methodParamInfo || $t('apiDocsRes.apiForm.none') }}</div>
            </div>
          </v-timeline-item>
          <v-timeline-item
            color="cyan"
            small
          >
            <div>
              <div class="font-weight-normal">
                <strong>{{ $t('apiDocsRes.apiForm.apiRespDecShowLabel') }}</strong>
              </div>
              <div>
                {{ this.apiInfoData.apiRespDec || $t('apiDocsRes.apiForm.none') }}
              </div>
            </div>
          </v-timeline-item>
          <v-timeline-item
            color="cyan"
            small
          >
            <div>
              <div class="font-weight-normal">
                <strong>{{ $t('apiDocsRes.apiForm.apiVersionShowLabel') }}</strong>
              </div>
              <div>
                {{ this.apiInfoData.apiVersion || $t('apiDocsRes.apiForm.none') }}
              </div>
            </div>
          </v-timeline-item>
          <v-timeline-item
            color="cyan"
            small
          >
            <div>
              <div class="font-weight-normal">
                <strong>{{ $t('apiDocsRes.apiForm.apiDescriptionShowLabel') }}</strong>
              </div>
              <div>
                {{ this.apiInfoData.description || $t('apiDocsRes.apiForm.none') }}
              </div>
            </div>
          </v-timeline-item>
        </v-timeline>
        <v-form ref="form">
          <v-select
            v-model="formItemAsync"
            :items="formItemAsyncSelectItems"
            :label="$t('apiDocsRes.apiForm.isAsyncFormLabel')"
            outline
            readonly
          ></v-select>
          <v-text-field
            v-model="formItemInterfaceClassName"
            :label="$t('apiDocsRes.apiForm.apiModuleFormLabel')"
            outline
            readonly
          ></v-text-field>
          <v-text-field
            v-model="formItemMethodName"
            :label="$t('apiDocsRes.apiForm.apiFunctionNameFormLabel')"
            outline
            readonly
          ></v-text-field>
          <v-text-field
            v-model="formItemRegistryCenterUrl"
            :label="$t('apiDocsRes.apiForm.registryCenterUrlFormLabel')"
            placeholder="nacos://127.0.0.1:8848"
            outline
          ></v-text-field>

          <div style="marginTop: 20px;"
            v-for="item in this.publicFormsArray"
            :key="item.get('name')">
            <v-layout row wrap>
              <v-flex lg4>
                <v-card style="height: 300px; overflowY: auto; overflowX: hidden">
                  <v-card-text>
                    <v-timeline
                      align-top
                      dense
                    >
                      <v-timeline-item
                        color="deep-purple lighten-1"
                        small
                      >
                        <div>
                          <div class="font-weight-normal">
                            <strong>{{ $t('apiDocsRes.apiForm.paramNameLabel') }}</strong>
                          </div>
                          <div style="wordBreak: break-word">{{item.get('name')}}</div>
                        </div>
                      </v-timeline-item>
                      <v-timeline-item
                        color="deep-purple lighten-1"
                        small
                      >
                        <div>
                          <div class="font-weight-normal">
                            <strong>{{ $t('apiDocsRes.apiForm.paramPathLabel') }}</strong>
                          </div>
                          <div style="wordBreak: break-word">[{{item.get('paramIndex')}}]{{item.get('paramType')}}#{{item.get('name')}}</div>
                        </div>
                      </v-timeline-item>
                      <v-timeline-item
                        color="deep-purple lighten-1"
                        small
                      >
                        <div>
                          <div class="font-weight-normal">
                            <strong>{{ $t('apiDocsRes.apiForm.paramDescriptionLabel') }}</strong>
                          </div>
                          <div style="wordBreak: break-word">{{item.get('description') || $t('apiDocsRes.apiForm.none')}}</div>
                        </div>
                      </v-timeline-item>
                    </v-timeline>
                  </v-card-text>
                </v-card>
              </v-flex>
              <v-flex lg8>
                  <apiFormItem :formItemInfo="item" :formValues="formValues" />
              </v-flex>
            </v-layout>
          </div>

          <div style="marginTop: 20px;">
            <v-btn
              block
              elevation="2"
              x-large
              color="info"
              @click="doTestApi()"
            >{{ $t('apiDocsRes.apiForm.doTestBtn') }}</v-btn>
          </div>
        </v-form>
      </div>
      <div>
        <v-system-bar
          window
          dark
          style="marginTop: 30px;"
        >
          <span>{{ $t('apiDocsRes.apiForm.responseLabel') }}</span>
        </v-system-bar>
        <v-layout row wrap>
          <v-flex lg6>
            <div>
              <v-system-bar
                window
                dark
                color="primary"
              >
                <span>{{ $t('apiDocsRes.apiForm.responseExampleLabel') }}</span>
              </v-system-bar>
            </div>
            <div style="marginTop: 10px;">
              <jsonViewer
                :value="getJsonOrString(this.apiInfoData.response)"
                copyable
                boxed
                sort></jsonViewer>
            </div>
          </v-flex>
          <v-flex lg6>
            <div>
              <v-system-bar
                window
                dark
                color="teal"
              >
                <span>{{ $t('apiDocsRes.apiForm.apiResponseLabel') }}</span>
              </v-system-bar>
            </div>
            <div style="marginTop: 10px;">
              <jsonViewer
                :value="responseData"
                copyable
                boxed
                sort></jsonViewer>
            </div>
          </v-flex>
        </v-layout>
      </div>
    </div>
  </div>
</template>

<script>
import JsonViewer from 'vue-json-viewer'
import ApiFormItem from '@/components/apiDocs/ApiFormItem'
export default {
  name: 'ApiForm',
  components: {
    JsonViewer,
    ApiFormItem
  },
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
      formItemInterfaceClassName: '',
      formItemMethodName: '',
      formItemRegistryCenterUrl: '',
      apiInfoData: {},
      publicFormsArray: [],
      responseData: '',
      formValues: new Map()
    }
  },
  watch: {
    formInfo: 'changeFormInfo'
  },
  methods: {
    getJsonOrString (str) {
      if (!str) return ''
      try {
        return JSON.parse(str)
      } catch (error) {
        return str
      }
    },
    changeFormInfo (curVal) {
      this.publicFormsArray = []
      this.formValues = new Map()
      this.responseData = ''
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
            this.formItemInterfaceClassName = this.apiInfoData.apiModelClass
            this.formItemMethodName = this.apiInfoData.apiName
            var params = this.apiInfoData.params
            const formsArray = []
            for (var i = 0; i < params.length; i++) {
              var paramItem = params[i]
              if (paramItem.htmlType) {
                // Has htmlType, that's a basic type
                var formItem = new Map()
                formItem.set('name', paramItem.name)
                formItem.set('htmlType', paramItem.htmlType)
                formItem.set('paramType', paramItem.paramType)
                formItem.set('javaType', paramItem.paramType)
                formItem.set('paramIndex', paramItem.paramIndex)
                formItem.set('docName', paramItem.docName)
                formItem.set('description', paramItem.description)
                formItem.set('example', paramItem.example)
                formItem.set('defaultValue', paramItem.defaultValue)
                formItem.set('allowableValues', paramItem.allowableValues)
                formItem.set('subParamsJson', paramItem.subParamsJson)
                formItem.set('required', paramItem.required)
                formItem.set('methodParam', true)
                formsArray.push(formItem)
              } else {
                // No htmltype, that's an object
                var paramInfoArray = paramItem.paramInfo
                for (var j = 0; j < paramInfoArray.length; j++) {
                  var paramInfoItem = paramInfoArray[j]
                  // eslint-disable-next-line no-redeclare
                  var formItem = new Map()
                  formItem.set('name', paramInfoItem.name)
                  formItem.set('htmlType', paramInfoItem.htmlType)
                  formItem.set('paramType', paramItem.paramType)
                  formItem.set('javaType', paramInfoItem.javaType)
                  formItem.set('paramIndex', paramItem.paramIndex)
                  formItem.set('docName', paramInfoItem.docName)
                  formItem.set('description', paramInfoItem.description)
                  formItem.set('example', paramInfoItem.example)
                  formItem.set('defaultValue', paramInfoItem.defaultValue)
                  formItem.set('allowableValues', paramInfoItem.allowableValues)
                  formItem.set('subParamsJson', paramInfoItem.subParamsJson)
                  formItem.set('required', paramInfoItem.required)
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
    },
    doTestApi () {
      if (!this.$refs.form.validate()) {
        return false
      }
      console.log(this.formValues)
      var tempMap = new Map()
      this.formValues.forEach((value, key) => {
        var elementIdSplited = key.split('@@')
        var tempMapKey = elementIdSplited[0] + '@@' + elementIdSplited[1]
        if (elementIdSplited[5]) {
          tempMapKey = tempMapKey + '@@' + elementIdSplited[5]
        }
        var tempMapValueArray = tempMap.get(tempMapKey)
        if (!tempMapValueArray) {
          tempMapValueArray = new Array()
          tempMap.set(tempMapKey, tempMapValueArray)
        }
        var element = {}
        element.key = key
        element.value = value
        tempMapValueArray.push(element)
      })
      var postData = []
      tempMap.forEach((value, key) => {
        var postDataItem = {}
        postData[key.split('@@')[1]] = postDataItem
        postDataItem.paramType = key.split('@@')[0]
        if (key.split('@@')[2]) {
          postDataItem.paramValue = value[0].value
        } else {
          var postDataItemValue = {}
          postDataItem.paramValue = postDataItemValue
          value.forEach(element => {
            var elementKeySplited = element.key.split('@@')
            var elementName = elementKeySplited[3]
            if (elementKeySplited[4] === 'TEXT_AREA') {
              if (element.value !== '') {
                postDataItemValue[elementName] = element.value
              }
            } else {
              postDataItemValue[elementName] = element.value
            }
          })
        }
      })
      if (this.formItemRegistryCenterUrl === '') {
        this.formItemRegistryCenterUrl = 'dubbo://' + this.formInfo.dubboIp + ':' + this.formInfo.dubboPort
      }
      this.$axios({
        url: '/docs/requestDubbo',
        method: 'post',
        params: {
          async: this.formItemAsync,
          interfaceClassName: this.formItemInterfaceClassName,
          methodName: this.formItemMethodName,
          registryCenterUrl: this.formItemRegistryCenterUrl,
          version: this.apiInfoData.apiVersion || ''
        },
        headers: {
          'Content-Type': 'application/json; charset=UTF-8'
        },
        data: JSON.stringify(postData)
      }).catch(error => {
        console.log(error)
      }).then(response => {
        this.responseData = response.data
      })
    }
  },
  mounted () {

  }
}
</script>

<style scoped>
</style>
