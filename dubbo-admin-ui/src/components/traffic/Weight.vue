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
  <v-container grid-list-xl fluid>
      <v-layout row wrap>
          <v-flex lg12>
            <Breadcrumb title="trafficWeight" :items="breads"></breadcrumb>
          </v-flex>
          <v-flex lg12>
            可在这里了解 <a href="https://cn.dubbo.apache.org/zh-cn/overview/tasks/traffic-management/weight/" target="_blank">服务权重</a> 配置的工作原理与使用方式！
          </v-flex>
    <v-flex lg12>
        <v-card flat color="transparent">
          <v-card-text>
            <v-form>
              <v-layout row wrap>
                <v-flex xs6 sm3 md9>
                  <v-text-field
                    v-model="searchService"
                    label="Service"
                    flat
                    hint="请输入service,如有group和version，请按照group/service:version格式输入"
                  ></v-text-field>
                </v-flex>
                <v-btn @click="submit" color="primary" large>{{$t('search')}}</v-btn>
                <v-btn @click="create" color="primary" large>新建</v-btn>
              </v-layout>
            </v-form>
          </v-card-text>
        </v-card>
      </v-flex>
    <v-flex xs12>
      <v-card>
        <v-toolbar flat color="transparent" class="elevation-0">
          <v-toolbar-title><span class="headline">{{$t('trafficweight')}}</span></v-toolbar-title>
          <v-spacer></v-spacer>
        </v-toolbar>
          <v-data-table :headers="headers" :items="tableData" hide-actions class="elevation-1">
            <template slot="items" slot-scope="props">
              <td >{{props.item.service}}</td>
              <td>{{props.item.weight}}</td>
              <td>{{props.item.matchRule}}</td>
              <td class="text-xs-center px-0" nowrap>
                <!-- <v-btn
                  class="tiny"
                  color='success'
                  @click="Check(props.item)"
                >
                 查看
                </v-btn> -->
                <v-btn
                  class="tiny"
                  color='success'
                  @click="update(props.item)"
                >
                 修改
                </v-btn>
                <v-btn
                  class="tiny"
                  outline
                  @click="deleteItem(props.item)"
                >
                  删除
                </v-btn>
              </td>
                </template>
          </v-data-table>
      </v-card>
    </v-flex>
    <v-dialog v-model="dialog" width="800px" persistent >
    <v-card>
      <v-card-title class="justify-center">
        <span class="headline">新增权重</span>
      </v-card-title>
      <v-layout row wrap>
        <v-flex lg12>
          可在这里了解如何动态调整服务的 <a href="https://cn.dubbo.apache.org/zh-cn/overview/tasks/traffic-management/weight/" target="_blank">权重值配置</a>！
        </v-flex>
      </v-layout>
      <v-card>
        <v-card-text>
          <v-layout row warp>
            <v-flex xs6 sm3 md9>
            <v-text-field
              label="service"
              hint="请输入service,如有group和version，请按照group/service:version格式输入"
              v-model="createWeight.service"
            ></v-text-field>
          </v-flex>
          <v-flex style="margin-left: 20px;" xs6 sm3 md2>
             <v-btn
              depressed
              color="primary"
              @click="addCreateWeight"
                >
                  新增
              </v-btn>
          </v-flex>
          </v-layout>
        </v-card-text>
      <v-card-text v-for="(modal,index) in createWeight.weights" :key="index">
          <v-flex  xs6 sm3 md6>
            <v-text-field
              label="请输入匹配实例的目标权重"
              hint="所有实例的默认权重为 100，如想要目标实例的流量为普通实例的 20%，则可以设置值为 25"
              type="number"
              v-model="modal.weight"
              @input="handleInputWeight(index)"
            ></v-text-field>
          </v-flex>
        <v-layout row wrap v-for="(item,idx) in modal.match.param" :key="idx">
          <v-flex xs6 sm3 md3>
            <v-text-field
              label="key"
              hint="请输入key"
              v-model="item.key"
            ></v-text-field>
          </v-flex>
          <v-flex xs6 sm3 md3>
            <v-select
              style="margin-left: 20px;"
              :items="items"
              label="Outlined style"
              v-model="selectedOption[index][idx]"
              @change="updateValue"
              outlined
            ></v-select>
          </v-flex>
          <v-flex xs6 sm3 md>
            <v-text-field
              style="margin-left: 20px;"
              label="value"
              hint="请输入匹配的值"
              v-model="item.value[selectedOption[index][idx]]"
            ></v-text-field>
          </v-flex>
          <v-flex xs6 sm3 md3>
             <v-btn
              style="margin-left: 20px;"
                  class="tiny"
                  color='success'
                  outline
                  @click="addItem(index)"
                >
                  新增权重条件
              </v-btn>
          </v-flex>
    </v-layout>
      </v-card-text>
    </v-card>
      <v-card-actions>
        <v-spacer></v-spacer>
        <v-btn flat @click.native="closeDialog">{{$t('close')}}</v-btn>
        <v-btn depressed color="primary" @click.native="save">{{$t('save')}}</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
  <v-dialog v-model="updateDialog" width="800px" persistent >
    <v-card>
      <v-card-title class="justify-center">
        <span class="headline">修改权重</span>
      </v-card-title>
      <v-layout row wrap>
        <v-flex lg12>
          可在这里了解如何动态调整服务的 <a href="https://cn.dubbo.apache.org/zh-cn/overview/tasks/traffic-management/weight/" target="_blank">权重值配置</a>！
        </v-flex>
      </v-layout>
      <v-card>
        <v-card-text>
          <v-layout row warp>
            <v-flex xs6 sm3 md9>
              <v-text-field
                label="service"
                disabled
                v-model="updateWeight.service"
            ></v-text-field>
            </v-flex>
          <v-flex xs6 sm3 md4>
             <v-btn
              style="margin-left: 20px;"
              depressed
              color="primary"
              @click="addUpdateWeight"
                >
                  新增
              </v-btn>
          </v-flex>
          </v-layout>
        </v-card-text>
      <v-card-text v-for="(modal,index) in updateWeight.weights" :key="index">
          <v-flex  xs6 sm3 md6>
            <v-text-field
              label="请输入匹配实例的目标权重"
              hint="所有实例的默认权重为 100，如想要目标实例的流量为普通实例的 20%，则可以设置值为 25"
              type="number"
              v-model="modal.weight"
              @input="handleUpdateInputWeight(index)"
            ></v-text-field>
          </v-flex>
        <v-layout row wrap v-for="(item,idx) in modal.match.param" :key="idx">
          <v-flex xs6 sm3 md3>
            <v-text-field
              label="key"
              hint="请输入key"
              v-model="item.key"
            ></v-text-field>
          </v-flex>
          <v-flex xs6 sm3 md3>
            <v-select
              style="margin-left: 20px;"
              :items="items"
              label="Outlined style"
              v-model="selectedUpdateOption[index][idx]"
              @change="updateValue(index, idx)"
              outlined
            ></v-select>
          </v-flex>
          <v-flex xs6 sm3 md>
            <v-text-field
              style="margin-left: 20px;"
              label="value"
              hint="请输入匹配的值"
              v-model="item.value[selectedUpdateOption[index][idx]]"
            ></v-text-field>
          </v-flex>
          <v-flex xs6 sm3 md3>
             <v-btn
              style="margin-left: 20px;"
                  class="tiny"
                  color='success'
                  outline
                  @click="addUpdateItem(index)"
                >
                  新增权重条件
              </v-btn>
          </v-flex>
    </v-layout>
      </v-card-text>
    </v-card>
      <v-card-actions>
        <v-spacer></v-spacer>
        <v-btn flat @click.native="closeUpdateDialog">{{$t('close')}}</v-btn>
        <v-btn depressed color="primary" @click.native="saveUpdate">{{$t('save')}}</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
  <v-dialog
    v-model="deleteDialog"
    persistent
    max-width="290"
  >
    <v-card>
      <v-card-title class="text-h5">
        您确认删除这条数据嘛?
      </v-card-title>
      <v-card-actions>
        <v-spacer></v-spacer>
        <v-btn
          color="green darken-1"
          text
          @click="deleteDialog = false"
        >
          取消
        </v-btn>
        <v-btn
          color="green darken-1"
          text
          @click="confirmDelete()"
        >
        确定
        </v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
    </v-layout>
  </v-container>
</template>
<script>
import Breadcrumb from '../public/Breadcrumb.vue'
export default {
  name: 'weight',
  components: { Breadcrumb },
  data: () => ({
    breads: [
      {
        text: 'trafficManagement',
        href: ''
      },
      {
        text: 'trafficWeight',
        href: ''
      }
    ],
    typeAhead: [],
    input: null,
    searchLoading: false,
    timerID: null,
    searchService: '',
    service: '',
    weight: '',
    mock: '',
    group: '',
    version: '',
    createGroup: '',
    createVersion: '',
    updateService: '',
    updateMock: '',
    updateGroup: '',
    updateVersion: '',
    deleteDialog: false,
    createService: '',
    createMock: '',
    deleteService: '',
    deleteMock: '',
    deleteVersion: '',
    deleteGroup: '',
    dialog: false,
    selectedOption: [[]],
    selectedUpdateOption: [[]],
    headers: [
    ],
    items: ['empty', 'exact', 'noempty', 'prefix', 'regex', 'wildcard'],
    tableData: [],
    services: [],
    loading: false,
    updateDialog: false,
    updateWeight: {},
    createWeight:
    {
      service: '',
      group: '',
      version: '',
      weights: [
        {
          weight: '',
          match: {
            param: [
              {
                key: '',
                value: {
                  empty: '',
                  exact: '',
                  noempty: '',
                  prefix: '',
                  regex: '',
                  wildcard: ''
                }
              }
            ]
          }
        }
      ]
    }
  }),
  methods: {
    handleInputWeight (index) {
      this.createWeight.weights[index].weight = Number(this.createWeight.weights[index].weight)
    },
    handleUpdateInputWeight (index) {
      this.updateWeight.weights[index].weight = Number(this.updateWeight.weights[index].weight)
    },
    updateValue (index, idx) {
      const temp = {
        empty: '',
        exact: '',
        noempty: '',
        prefix: '',
        regex: '',
        wildcard: ''
      }
      this.updateWeight.weights[index].match[idx].value = temp
    },
    submit () {
      this.search()
    },
    addCreateWeight () {
      const temp = {
        weight: '',
        match: {
          param: [
            {
              key: '',
              value: {
                empty: '',
                exact: '',
                noempty: '',
                prefix: '',
                regex: '',
                wildcard: ''
              }
            }
          ]
        }
      }
      this.selectedOption.push([])
      this.createWeight.weights.push(temp)
    },
    addUpdateWeight () {
      const temp = {
        weight: '',
        match: {
          param: [
            {
              key: '',
              value: {
                empty: '',
                exact: '',
                noempty: '',
                prefix: '',
                regex: '',
                wildcard: ''
              }
            }
          ]
        }
      }
      this.selectedUpdateOption.push([])
      this.updateWeights.push(temp)
    },
    addItem (params) {
      const temp = {
        key: '',
        value: {
          empty: '',
          exact: '',
          noempty: '',
          prefix: '',
          regex: '',
          wildcard: ''
        }
      }
      const index = parseInt(params)
      this.createWeight.weights[index].match.param.push(temp)
    },
    addUpdateItem (params) {
      const temp = {
        key: '',
        value: {
          empty: '',
          exact: '',
          noempty: '',
          prefix: '',
          regex: '',
          wildcard: ''
        }
      }
      const index = parseInt(params)
      this.updateWeight.weights[index].match.param.push(temp)
    },
    search () {
      if (this.searchService === '*') {
        this.service = '*'
      } else {
        const matches = this.searchService.split(/^(.*?)\/(.*?):(.*)$/)
        if (matches.length === 1) {
          this.service = matches[0]
        } else {
          this.group = matches[1]
          this.service = matches[2]
          this.version = matches[3]
        }
      }
      this.$axios.get('/traffic/weight', {
        params: {
          service: this.service,
          version: this.version,
          group: this.group
        }
      }).then(response => {
        this.tableData = []
        response.data.forEach(element => {
          let sum = 0
          let matchRule = ''
          element.weights.forEach(item => {
            sum += item.weight
            const matchParams = [] // 用于存储参数的数组
            item.match.param.forEach(it => {
              let param = `${it.key}-`
              if (it.value.empty !== '') {
                param += 'empty-' + it.value.empty
              } else if (it.value.exact !== '') {
                param += 'exact-' + it.value.exact
              } else if (it.value.noempty !== '') {
                param += 'noempty-' + it.value.noempty
              } else if (it.value.prefix !== '') {
                param += 'prefix-' + it.value.prefix
              } else if (it.value.regex !== '') {
                param += 'regex-' + it.value.regex
              } else if (it.value.wildcard !== '') {
                param += 'wildcard-' + it.value.wildcard
              }
              matchParams.push(param) // 将参数添加到数组中
            })
            matchRule += (matchParams.length > 0) ? (matchParams.join(',') + ',') : ''
          })
          matchRule = matchRule.slice(0, -1)
          const weight = sum / element.weights.length
          const result = {
            service: element.service,
            weight,
            matchRule,
            element
          }
          this.tableData.push(result)
        })
      })
    },
    saveUpdate () {
      this.updateDialog = false
      this.updateWeight.service = this.tempService
      if (this.updateWeight) {
        this.$axios.put('/traffic/weight', this.updateWeight).then((res) => {
          if (res) {
            alert('操作成功')
          }
        })
      } else {
        alert('请输入权重值')
      }
      setTimeout(() => {
        this.search()
      }, 1000)
    },
    setHeaders: function () {
      this.headers = [
        {
          text: '服务',
          value: 'service'
        },
        {
          text: '权重',
          value: 'weight'
        },
        {
          text: '匹配条件',
          value: 'matchRule'
        },
        {
          text: '操作',
          value: 'version'
        }
      ]
    },
    closeUpdateDialog () {
      this.updateDialog = false
    },
    create () {
      this.dialog = true
      this.createWeight = {
        service: '',
        group: '',
        version: '',
        weights: [
          {
            weight: '',
            match: {
              param: [
                {
                  key: '',
                  value: {
                    empty: '',
                    exact: '',
                    noempty: '',
                    prefix: '',
                    regex: '',
                    wildcard: ''
                  }
                }
              ]
            }
          }
        ]
      }
    },
    confirmDelete () {
      console.log(this.deleteService)
      this.$axios.delete('/traffic/mock', {
        params: {
          service: this.deleteService,
          group: this.deleteGroup,
          version: this.deleteVersion
        }
      }).then((res) => {
        if (res) {
          alert('操作成功')
        }
      })
      this.deleteDialog = false
      setTimeout(() => {
        this.search()
      }, 1000)
    },
    deleteItem (props) {
      console.log(props)
      this.deleteDialog = true
      this.deleteService = props.element.service
      this.deleteGroup = props.element.group
      this.deleteVersion = props.element.version
    },
    update (props) {
      this.updateWeight = props.element
      this.tempService = this.updateWeight.service
      if (this.updateWeight.group && this.updateWeight.version) {
        this.updateWeight.service = `${this.updateWeight.group}/${this.updateWeight.service}:${this.updateWeight.version}`
      }
      props.element.weights.forEach((item, index) => {
        this.selectedUpdateOption[index] = []
        item.match.param.forEach((it, idx) => {
          if (it.value.empty !== '') {
            this.selectedUpdateOption[index][idx] = 'empty'
          } else if (it.value.exact !== '') {
            this.selectedUpdateOption[index][idx] = 'exact'
          } else if (it.value.noempty !== '') {
            this.selectedUpdateOption[index][idx] = 'noempty'
          } else if (it.value.prefix !== '') {
            this.selectedUpdateOption[index][idx] = 'prefix'
          } else if (it.value.regex !== '') {
            this.selectedUpdateOption[index][idx] = 'regex'
          } else if (it.value.wildcard !== '') {
            this.selectedUpdateOption[index][idx] = 'wildcard'
          }
        })
      })
      this.updateDialog = true
    },
    save () {
      const matches = this.createWeight.service.split(/^(.*?)\/(.*?):(.*)$/)
      if (matches.length === 1) {
        this.createWeight.service = matches[0]
      } else {
        this.createWeight.group = matches[1]
        this.createWeight.service = matches[2]
        this.createWeight.version = matches[3]
      }
      if (this.createWeight) {
        this.$axios.post('/traffic/weight', this.createWeight).then((res) => {
          if (res) {
            alert('操作成功')
          }
        })
      } else {
        alert('请输入权重值')
      }
      this.dialog = false
      setTimeout(() => {
        this.search()
      }, 1000)
    },
    closeDialog () {
      this.dialog = false
    }
  },
  watch: {
    area () {
      this.setHeaders()
    }
  },
  mounted () {
    this.setHeaders()
    this.searchService = '*'
    this.search()
  }
}

</script>
