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
              <Breadcrumb title="trafficAccesslog" :items="breads"></breadcrumb>
            </v-flex>
            <v-flex lg12>
              可在这里了解如何开启/关闭应用的 <a href="https://cn.dubbo.apache.org/zh-cn/overview/tasks/traffic-management/accesslog/" target="_blank">访问日志</a>！
            </v-flex>
      <v-flex lg12>
          <v-card flat color="transparent">
            <v-card-text>
              <v-form>
                <v-layout row wrap>
                  <v-text-field
                    label="Application Name"
                    hint="请输入application"
                    v-model="application"
                  ></v-text-field>
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
            <v-toolbar-title><span class="headline">{{$t('trafficAccesslog')}}</span></v-toolbar-title>
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
                </td>
                  </template>
            </v-data-table>
        </v-card>
      </v-flex>
      <v-dialog v-model="dialog" width="800px" persistent >
      <v-card>
        <v-card-title class="justify-center">
          <span class="headline">{{$t('createAccesslogRule')}}</span>
        </v-card-title>
        <v-layout row wrap>
          <v-flex lg12>
            可在这里了解如何动态开启/关闭应用的 <a href="https://dubbo.apache.org/zh-cn/overview/tasks/traffic-management/accesslog/" target="_blank">访问日志</a>！
          </v-flex>
        </v-layout>
        <v-card-text >
          <v-layout wrap>
            <v-flex xs6 sm3 md5>
              <v-text-field
                label="Application Name"
                hint="请输入应用名"
                v-model="createApplication"
              ></v-text-field>
            </v-flex>
          </v-layout>
          <v-layout row wrap>
          <v-switch label="开启或关闭访问日志" v-model="handleAccesslog"></v-switch>
        </v-layout>
        <v-layout v-if="handleAccesslog" row wrap>
          <v-flex xs6 sm3 md5>
            <v-text-field
              label="（可选）访问日志已开启，可继续调整存储路径"
              hint="请参考文档开启日志路径修改权限后再配置，否则日志仍会输入到默认路径。请输入目标文件绝对路径（如/home/user1/access.log）"
              v-model="createAccesslog"
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
          <span class="headline">{{$t('createAccesslogRule')}}</span>
        </v-card-title>
        <v-layout row wrap>
          <v-flex lg12>
            可在这里了解如何动态开启/关闭应用的 <a href="https://dubbo.apache.org/zh-cn/overview/tasks/traffic-management/accesslog/" target="_blank">访问日志</a>！
          </v-flex>
        </v-layout>
        <v-card-text >
          <v-layout wrap>
            <v-flex xs6 sm3 md5>
              <v-text-field
                label="Application Name"
                hint="请输入应用名"
                disabled
                v-model="updateApplication"
              ></v-text-field>
            </v-flex>
          </v-layout>
          <v-layout row wrap>
          <v-switch label="开启或关闭访问日志" v-model="handleUpdateAccesslog"></v-switch>
        </v-layout>
        <v-layout v-if="handleUpdateAccesslog" row wrap>
          <v-flex xs6 sm3 md5>
            <v-text-field
              label="（可选）访问日志已开启，可继续调整存储路径"
              hint="请参考文档开启日志路径修改权限后再配置，否则日志仍会输入到默认路径。请输入目标文件绝对路径（如/home/user1/access.log）"
               v-model="updateAccesslog"
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
  name: 'Accesslog',
  components: { Breadcrumb },
  data: () => ({
    breads: [
      {
        text: 'trafficManagement',
        href: ''
      },
      {
        text: 'trafficAccesslog',
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
    handleUpdateAccesslog: '',
    handleAccesslog: false,
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
      this.search()
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
      this.updateDialog = false
      if (this.handleUpdateAccesslog) {
        this.$axios.put('/traffic/accesslog', {
          application: this.updateApplication,
          accesslog: this.updateAccesslog === '' ? 'true' : this.updateAccesslog
        }).then((res) => {
          if (res) {
            alert('操作成功')
          }
        })
      } else {
        this.$axios.put('/traffic/accesslog', {
          application: this.updateApplication,
          accesslog: '' // 删除
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
          text: '应用名',
          value: 'application'
        },
        {
          text: '访问日志(状态)',
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
      this.createAccesslog = ''
      this.createApplication = ''
    },
    confirmDelete () {
      console.log(this.deleteApplication)
      this.$axios.delete('/traffic/accesslog', {
        params: {
          application: this.deleteApplication,
          group: this.group,
          version: this.version
        }
      }
      ).then((res) => {
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
      this.deleteAccesslog = props.accesslog
      this.deleteApplication = props.application
    },
    update (props) {
      this.updateApplication = props.application
      this.handleUpdateAccesslog = props.accesslog !== 'false'
      this.updateAccesslog = props.accesslog === 'false' ? '' : props.accesslog
      this.updateDialog = true
    },
    save () {
      if (this.handleAccesslog) {
        this.$axios.post('/traffic/accesslog', {
          application: this.createApplication,
          accesslog: this.createAccesslog === '' ? 'true' : this.createAccesslog
        }).then((res) => {
          if (res) {
            alert('操作成功')
          }
        })
      } else {
        alert('访问日志未开启，请选中开关后再保存！')
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
    this.application = '*'
    this.search()
  }
}

</script>
