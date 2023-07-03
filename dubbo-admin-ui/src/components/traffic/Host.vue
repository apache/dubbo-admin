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
            <Breadcrumb title="trafficHost" :items="breads"></breadcrumb>
          </v-flex>
          <v-flex lg12>
            可在这里了解如何 <a href="https://cn.dubbo.apache.org/zh-cn/overview/tasks/traffic-management/host/" target="_blank">将流量转发到指定机器</a> 的工作原理与使用方式！
          </v-flex>
    <v-flex lg12>
        <v-card flat color="transparent">
          <v-card-text>
            <v-form>
              <v-layout row wrap>
                <v-combobox
                  :loading="searchLoading"
                  :items="typeAhead"
                  :search-input.sync="application"
                  flat
                  append-icon=""
                  hide-no-data
                  label="请输入application"
                  hint="请输入application"
                ></v-combobox>
                  <v-combobox
                  style="margin-left: 20px;"
                  :loading="searchLoading"
                  :items="typeAhead"
                  :search-input.sync="accesslog"
                  flat
                  append-icon=""
                  hide-no-data
                  label="请输入accesslog"
                  hint="请输入accesslog"
                ></v-combobox>

                <v-btn @click="submit" color="primary" large>搜索</v-btn>
                <v-btn @click="create" color="primary" large>新建</v-btn>
              </v-layout>
            </v-form>
          </v-card-text>
        </v-card>
      </v-flex>
    <v-flex xs12>
      <v-card>
        <v-toolbar flat color="transparent" class="elevation-0">
          <v-toolbar-title><span class="headline">{{$t('trafficHost')}}</span></v-toolbar-title>
          <v-spacer></v-spacer>
        </v-toolbar>
          <v-data-table :headers="headers" :items="tableData" hide-actions class="elevation-1">
            <template slot="items" slot-scope="props">
              <td >{{props.item.application}}</td>
              <td>{{props.item.accesslog}}</td>
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
                <v-btn
                  class="tiny"
                  outline
                >
                  启用
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
          可在这里了解如何配置 <a href="https://cn.dubbo.apache.org/zh-cn/overview/tasks/traffic-management/host/" target="_blank">将服务流量转发到固定机器</a> ！
        </v-flex>
      </v-layout>
      <v-card-text >
        <v-layout wrap>
          <v-flex>
            <v-text-field
              label="Application Name"
              hint="请输入Application Name"
              v-model="createApplication"
            ></v-text-field>
          </v-flex>
        </v-layout>
        <v-text-field
          label="Accesslog"
          hint="请输入Accesslog"
          v-model="createAccesslog"
        ></v-text-field>
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
          可在这里了解如何配置 <a href="https://cn.dubbo.apache.org/zh-cn/overview/tasks/traffic-management/host/" target="_blank">将服务流量转发到固定机器</a> ！
        </v-flex>
      </v-layout>
      <v-card-text >
        <v-layout wrap>
          <v-flex>
            <v-text-field
              label="Application Name"
              hint="请输入Application Name"
              v-model="updateApplication"
            ></v-text-field>
          </v-flex>
        </v-layout>
        <v-text-field
          label="Accesslog"
          hint="请输入Accesslog"
          v-model="updateAccesslog"
        ></v-text-field>
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
  name: 'Accesslog',
  components: { Breadcrumb },
  data: () => ({
    breads: [
      {
        text: 'trafficManagement',
        href: ''
      },
      {
        text: 'trafficHost',
        href: ''
      }
    ],
    typeAhead: [],
    input: null,
    searchLoading: false,
    timerID: null,
    application: '',
    accesslog: '',
    deleteDialog: false,
    createApplication: '',
    createAccesslog: '',
    deleteApplication: '',
    deleteAccesslog: '',
    dialog: false,
    headers: [
    ],
    service: null,
    tableData: [],
    services: [],
    loading: false,
    updateDialog: false,
    updateApplication: '',
    updateAccesslog: ''
  }),
  methods: {
    submit () {
      if (this.accesslog && this.application) {
        this.search()
      } else {
        this.$notify.error('service is needed')
        return false
      }
    },
    search () {
      this.$axios.get('/traffic/accesslog', {
        params: {
          application: this.application,
          accesslog: this.accesslog
        }
      }).then(response => {
        console.log(response)
        this.tableData = []
        response.data.forEach(element => {
          this.tableData.push(element)
        })
        console.log(this.tableData)
      })
    },
    saveUpdate () {
      console.log(this.updateAccesslog)
      this.updateDialog = false
      this.$axios.put('/traffic/accesslog', {
        application: this.updateApplication,
        accesslog: this.updateAccesslog
      }).then((res) => {
        if (res) {
          alert('操作成功')
        }
      })
    },
    setHeaders: function () {
      this.headers = [
        {
          text: '服务',
          value: 'application'
        },
        {
          text: 'accesslog',
          value: 'accesslog'
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
    },
    confirmDelete () {
      console.log(this.deleteAccesslog)
      this.$axios.delete('/traffic/accesslog', {
        application: this.deleteApplication,
        accesslog: this.deleteAccesslog
      }).then((res) => {
        if (res) {
          alert('操作成功')
        }
      })
      this.deleteAccesslog = false
    },
    deleteItem (props) {
      this.deleteDialog = true
      this.deleteAccesslog = props.accesslog
      this.deleteApplication = props.application
    },
    update (props) {
      console.log(props)
      this.updateApplication = props.application
      this.updateAccesslog = props.accesslog
      this.updateDialog = true
      console.log(this.updateApplication)
      console.log(this.updateAccesslog)
    },
    save () {
      this.$axios.post('/traffic/accesslog', {
        application: this.createApplication,
        accesslog: this.createAccesslog
      }).then((res) => {
        if (res) {
          alert('操作成功')
        }
      })
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
  }
}

</script>
