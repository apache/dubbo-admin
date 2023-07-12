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
              <Breadcrumb title="trafficGray" :items="breads"></breadcrumb>
            </v-flex>
            <v-flex lg12>
              可在这里了解应用 <a href="https://cn.dubbo.apache.org/zh-cn/overview/tasks/traffic-management/isolation/" target="_blank">灰度环境隔离</a> 的工作原理与使用方式！
            </v-flex>
      <v-flex lg12>
          <v-card flat color="transparent">
            <v-card-text>
              <v-form>
                <v-layout row wrap>
                  <v-flex xs6 sm3 md9>
                    <v-text-field
                      v-model="application"
                      flat
                      label="请输入应用名"
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
            <v-toolbar-title><span class="headline">{{$t('trafficGray')}}</span></v-toolbar-title>
            <v-spacer></v-spacer>
          </v-toolbar>
            <v-data-table :headers="headers" :items="tableData" hide-actions class="elevation-1">
              <template slot="items" slot-scope="props">
                <td >{{props.item.service}}</td>
                <td>{{props.item.Gary}}</td>
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
                   查看修改
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
          <span class="headline">新增灰度</span>
        </v-card-title>
        <v-layout row wrap>
          <v-flex lg12>
            可在这里了解如何为应用设置不同的 <a href="https://cn.dubbo.apache.org/zh-cn/overview/tasks/traffic-management/gray/" target="_blank">灰度流量隔离环境</a> ！
          </v-flex>
        </v-layout>
        <v-card>
          <v-card-text>
            <v-layout row warp>
              <v-flex xs6 sm3 md8>
              <v-text-field
                label="application"
                hint="请输入application"
                v-model="createGary.application"
              ></v-text-field>
            </v-flex>
            <v-flex xs6 sm3 md4>
               <v-btn
                style="margin-left: 20px;"
                depressed
                color="primary"
                @click="addCreateGary"
                  >
                    新增
                </v-btn>
            </v-flex>
            </v-layout>
          </v-card-text>
        <v-card-text v-for="(modal,index) in createGary.tags" :key="index">
            <v-flex  xs6 sm3 md6>
              <v-text-field
                label="灰度隔离环境名称"
                hint="请输入名称，该值将作为灰度流量的匹配条件"
                v-model="modal.name"
              ></v-text-field>
            </v-flex>
          <v-layout row wrap v-for="(item,idx) in modal.match" :key="idx">
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
                    新增灰度环境
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
          <span class="headline">修改灰度</span>
        </v-card-title>
        <v-layout row wrap>
          <v-flex lg12>
            可在这里了解如何为应用设置不同的 <a href="https://cn.dubbo.apache.org/zh-cn/overview/tasks/traffic-management/gray/" target="_blank">灰度流量隔离环境</a> ！
          </v-flex>
        </v-layout>
        <v-card>
          <v-card-text>
            <v-layout row warp>
              <v-flex xs6 sm3 md8>
              <v-text-field
                label="application"
                hint="请输入 Provider 应用名"
                disabled
                v-model="updateGary.application"
              ></v-text-field>
            </v-flex>
            <v-flex xs6 sm3 md4>
               <v-btn
                style="margin-left: 20px;"
                depressed
                color="primary"
                @click="addUpdateGary"
                  >
                    新增
                </v-btn>
            </v-flex>
            </v-layout>
          </v-card-text>
        <v-card-text v-for="(modal,index) in updateGary.tags" :key="index">
            <v-flex  xs6 sm3 md6>
              <v-text-field
                label="灰度隔离环境名称"
                hint="请输入名称，该值将作为灰度流量的匹配条件"
                v-model="modal.name"
              ></v-text-field>
            </v-flex>
          <v-layout row wrap v-for="(item,idx) in modal.match" :key="idx">
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
                    新增灰度环境
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
            @click="confirmDelete"
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
  name: 'Gray',
  components: { Breadcrumb },
  data: () => ({
    breads: [
      {
        text: 'trafficManagement',
        href: ''
      },
      {
        text: 'trafficGray',
        href: ''
      }
    ],
    typeAhead: [],
    input: null,
    searchLoading: false,
    timerID: null,
    service: '',
    Gary: '',
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
    selectedOption: [['exact']],
    selectedUpdateOption: [[]],
    headers: [
    ],
    items: ['empty', 'exact', 'noempty', 'prefix', 'regex', 'wildcard'],
    tableData: [],
    services: [],
    loading: false,
    updateDialog: false,
    application: '',
    updateGary: {},
    createGary:
      {
        application: '',
        tags: [
          {
            name: '',
            match: [
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
        ]
      }
  }),
  methods: {
    updateValue (index, idx) {
      const temp = {
        empty: '',
        exact: '',
        noempty: '',
        prefix: '',
        regex: '',
        wildcard: ''
      }
      this.updateGary.tags[index].match[idx].value = temp
    },
    submit () {
      this.search()
    },
    addCreateGary () {
      const temp = {
        name: '',
        match: [
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
      this.selectedOption.push(['exact'])
      this.createGary.tags.push(temp)
    },
    addUpdateGary () {
      const temp = {
        name: '',
        match: [
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
      this.selectedUpdateOption.push([])
      this.updateGary.tags.push(temp)
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
      this.selectedOption[index].push('exact')
      this.createGary.tags[index].match.push(temp)
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
      this.updateGary.tags[index].match.push(temp)
    },
    search () {
      this.$axios.get('/traffic/gray', {
        params: {
          application: this.application
        }
      }).then(response => {
        this.tableData = []
        response.data.forEach(element => {
          const array = []
          element.tags.forEach(item => {
            array.push(item.name)
          })
          const uniqueArray = Array.from(new Set(array))
          const Gary = uniqueArray.join('|')
          const result = {
            service: element.application,
            Gary,
            element
          }
          this.tableData.push(result)
        })
      })
    },
    saveUpdate () {
      this.updateDialog = false
      this.$axios.put('/traffic/gray', this.updateGary).then((res) => {
        if (res) {
          alert('操作成功')
        }
      })
      this.dialog = false
      setTimeout(() => {
        this.search()
      }, 1000)
    },
    setHeaders: function () {
      this.headers = [
        {
          text: '应用名',
          value: 'service'
        },
        {
          text: '灰度环境',
          value: 'Gary'
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
      this.createGary = {
        application: '',
        tags: [
          {
            name: '',
            match: [
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
        ]
      }
    },
    confirmDelete () {
      this.$axios.delete('/traffic/mock',
        {
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
      this.deleteDialog = true
      this.deleteService = props.service
    },
    update (props) {
      this.updateGary = props.element
      props.element.tags.forEach((item, index) => {
        this.selectedUpdateOption[index] = []
        item.match.forEach((it, idx) => {
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
      this.$axios.post('/traffic/gray', this.createGary).then((res) => {
        if (res) {
          alert('操作成功')
        }
      })
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
    this.application = '*'
    this.search()
  }
}

</script>
