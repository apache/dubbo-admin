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
            <Breadcrumb title="trafficRegion" :items="breads"></breadcrumb>
          </v-flex>
          <v-flex lg12>
            可在这里了解 <a href="https://dubbo.apache.org/zh-cn/overview/tasks/traffic-management/region/" target="_blank">同区域优先</a> 配置的工作原理与使用方式！
          </v-flex>
    <v-flex lg12>
        <v-card flat color="transparent">
          <v-card-text>
            <v-form>
              <v-layout row wrap>
                <v-flex xs6 sm3 md9>
                  <v-text-field
                    v-model="searchService"
                    flat
                    label="请输入服务名"
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
          <v-toolbar-title><span class="headline">{{$t('trafficRegion')}}</span></v-toolbar-title>
          <v-spacer></v-spacer>
        </v-toolbar>
          <v-data-table :headers="headers" :items="tableData" hide-actions class="elevation-1">
            <template slot="items" slot-scope="props">
              <td >{{props.item.service}}</td>
              <td>{{props.item.rule}}</td>
              <td>{{props.item.group}}</td>
              <td>{{props.item.version}}</td>
              <td class="text-xs-center px-0" nowrap>
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
        <span class="headline">{{$t('createNewRoutingRule')}}</span>
      </v-card-title>
      <v-layout row wrap>
        <v-flex lg12>
          可在这里了解如何让服务调用遵循 <a href="https://cn.dubbo.apache.org/zh-cn/overview/tasks/traffic-management/region/" target="_blank">同区域优先</a> 规则！
        </v-flex>
      </v-layout>
      <v-card-text >
        <v-layout row wrap>
          <v-flex xs6 sm3 md9>
            <v-text-field
              label="服务名"
              hint="请输入service,如有group和version，请按照group/service:version格式输入"
              v-model="createService"
            ></v-text-field>
          </v-flex>
        </v-layout>
        <v-layout row wrap>
          <v-switch label="开启或关闭同区域优先" v-model="handleRule"></v-switch>
        </v-layout>
        <v-layout v-if="handleRule" row wrap>
          <v-flex xs6 sm3 md5>
            <v-text-field
               label="请输入表示同区域的 key 值"
               hint="每个公司的区域标识可能不同，比如对于以下地址：tri://host:port/service?timeout=1000&region=hangzhou&...，则同区域key值为：region。"
               v-model="createRule"
             ></v-text-field>
          </v-flex>
        </v-layout>
      </v-card-text>
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
        <span class="headline">{{$t('createNewRoutingRule')}}</span>
      </v-card-title>
       <v-layout row wrap>
          <v-flex lg12>
            可在这里了解如何让服务调用遵循 <a href="https://cn.dubbo.apache.org/zh-cn/overview/tasks/traffic-management/region/" target="_blank">同区域优先</a> 规则！
          </v-flex>
       </v-layout>
      <v-card-text >
        <v-layout row wrap>
          <v-flex xs6 sm3 md9>
            <v-text-field
              label="服务名"
              hint="请输入service,如有group和version，请按照group/service:version格式输入"
              disabled
              v-model="updateService"
            ></v-text-field>
          </v-flex>
        </v-layout>
        <v-layout row wrap>
          <v-switch label="开启或关闭同区域优先" v-model="handleUpdateRule"></v-switch>
        </v-layout>
        <v-layout v-if="handleUpdateRule" row wrap>
          <v-flex xs6 sm3 md5>
            <v-text-field
              label="请输入表示同区域的 key 值"
              hint="每个公司的区域标识可能不同，比如对于以下地址：tri://host:port/service?timeout=1000&region=hangzhou&...，则同区域key值为：region。"
              v-model="updateRule"
        ></v-text-field>
          </v-flex>
        </v-layout>
      </v-card-text>
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
  name: 'Region',
  components: { Breadcrumb },
  data: () => ({
    breads: [
      {
        text: 'trafficManagement',
        href: ''
      },
      {
        text: 'trafficRegion',
        href: ''
      }
    ],
    typeAhead: [],
    input: null,
    searchLoading: false,
    timerID: null,
    service: '',
    searchService: '',
    rule: '',
    group: '',
    version: '',
    createGroup: '',
    createVersion: '',
    updateService: '',
    updateRule: '',
    updateGroup: '',
    updateVersion: '',
    deleteDialog: false,
    handleRule: false,
    handleUpdateRule: false,
    createService: '',
    createRule: '',
    deleteService: '',
    deleteRule: '',
    deleteVersion: '',
    deleteGroup: '',
    dialog: false,
    headers: [
    ],
    tableData: [],
    services: [],
    loading: false,
    updateDialog: false
  }),
  methods: {
    submit () {
      this.search()
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
      this.$axios.get('/traffic/region', {
        params: {
          service: this.service,
          group: this.group,
          version: this.version
        }
      }).then(response => {
        this.tableData = []
        response.data.forEach(element => {
          this.tableData.push(element)
        })
        console.log(this.tableData)
      })
    },
    saveUpdate () {
      this.updateDialog = false
      if (this.handleUpdateRule) {
        this.$axios.put('/traffic/region', {
          service: this.tempService,
          rule: this.updateRule,
          group: this.updateGroup,
          version: this.updateVersion
        }).then((res) => {
          if (res) {
            alert('操作成功')
          }
        })
      } else {
        this.$axios.put('/traffic/region', {
          service: this.updateService,
          rule: '',
          group: this.updateGroup,
          version: this.updateVersion
        }).then((res) => {
          if (res) {
            alert('操作成功')
          }
        })
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
          text: '同区域Key',
          value: 'rule'
        },
        {
          text: '分组',
          value: 'group'
        },
        {
          text: '版本',
          value: 'version'
        },
        {
          text: '操作',
          value: ''
        }
      ]
    },
    closeUpdateDialog () {
      this.updateDialog = false
    },
    create () {
      this.dialog = true
      this.createService = ''
      this.handleRule = false
      this.createRule = ''
    },
    confirmDelete () {
      this.$axios.delete('/traffic/region',
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
      this.deleteRule = props.rule
      this.deleteGroup = props.group
      this.deleteVersion = props.version
    },
    update (props) {
      if (props.version && props.group) {
        this.updateService = `${props.group}/${props.service}:${props.version}`
      } else {
        this.updateService = props.service
      }
      this.tempService = props.service
      if (props.rule === 'false') {
        this.handleUpdateRule = false
        this.updateRule = ''
      } else {
        this.handleUpdateRule = true
        this.updateRule = props.rule
      }
      this.updateGroup = props.group
      this.updateVersion = props.version
      this.updateDialog = true
    },
    save () {
      const matches = this.createService.split(/^(.*?)\/(.*?):(.*)$/)
      if (matches.length === 1) {
        this.createService = matches[0]
      } else {
        this.createGroup = matches[1]
        this.createService = matches[2]
        this.createVersion = matches[3]
      }
      if (this.handleRule) {
        this.$axios.post('/traffic/region', {
          service: this.createService,
          rule: this.createRule,
          group: this.createGroup,
          version: this.createVersion
        }).then((res) => {
          if (res) {
            alert('操作成功')
          }
        })
      } else {
        alert('同区域优先未开启，请选中开关后再保存！')
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
