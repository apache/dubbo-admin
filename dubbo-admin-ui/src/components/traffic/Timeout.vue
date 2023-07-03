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
            <Breadcrumb title="trafficTimeout" :items="breads"></breadcrumb>
          </v-flex>
          <v-flex lg12>
            可在这里了解 <a href="https://cn.dubbo.apache.org/zh-cn/overview/tasks/traffic-management/timeout/" target="_blank">超时时间</a> 配置的工作原理与使用方式！
          </v-flex>
    <v-flex lg12>
        <v-card flat color="transparent">
          <v-card-text>
            <v-form>
              <v-layout row wrap>
                <v-flex xs6 sm3 md3>
                  <v-text-field
                    v-model="service"
                    flat
                    label="请输入服务名"
                  ></v-text-field>
                </v-flex>
                <v-flex xs6 sm3 md2 >
                  <v-text-field
                    label="Version"
                    :hint="$t('dataIdVersionHint')"
                    v-model="group"
                  ></v-text-field>
                </v-flex>
                <v-flex xs6 sm3 md2 >
                  <v-text-field
                    label="Group"
                    :hint="$t('dataIdGroupHint')"
                    v-model="version"
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
          <v-toolbar-title><span class="headline">{{$t('trafficTimeout')}}</span></v-toolbar-title>
          <v-spacer></v-spacer>
        </v-toolbar>
          <v-data-table :headers="headers" :items="tableData" hide-actions class="elevation-1">
            <template slot="items" slot-scope="props">
              <td >{{props.item.service}}</td>
              <td>{{props.item.timeout}}</td>
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
        <span class="headline">{{$t('createTimeoutRule')}}</span>
      </v-card-title>
      <v-layout row wrap>
        <v-flex lg12>
          可在这里了解如何动态调整服务的 <a href="https://cn.dubbo.apache.org/zh-cn/overview/tasks/traffic-management/timeout/" target="_blank">超时时间配置</a>！
        </v-flex>
      </v-layout>
      <v-card-text >
        <v-layout row wrap>
          <v-flex xs6 sm3 md2>
            <v-text-field
              label="服务名"
              hint="请输入服务名"
              v-model="createService"
            ></v-text-field>
          </v-flex>
          <v-flex style="margin-left: 10px;" xs6 sm3 md2>
            <v-text-field
                label="服务分组"
              hint="请输入服务group(可选)"
              v-model="createGroup"
            ></v-text-field>
           </v-flex>
          <v-flex style="margin-left: 10px;" xs6 sm3 md2>
            <v-text-field
              label="服务版本"
              hint="请输入服务version(可选)"
              v-model="createVersion"
            ></v-text-field>
           </v-flex>
        </v-layout>
        <v-layout row wrap>
          <v-flex style="margin-left: 10px;" xs6 sm3 md2>
            <v-text-field
              label="超时时间"
              hint="请输入一个整数值作为超时时间(单位ms)"
              type="number"
              v-model="createTimeout"
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
        <span class="headline">{{$t('createTimeoutRule')}}</span>
      </v-card-title>
      <v-layout row wrap>
        <v-flex lg12>
          可在这里了解如何动态调整服务的 <a href="https://cn.dubbo.apache.org/zh-cn/overview/tasks/traffic-management/timeout/" target="_blank">超时时间配置</a>！
        </v-flex>
      </v-layout>
      <v-card-text >
        <v-layout wrap>
          <v-flex xs6 sm3 md2>
            <v-text-field
              label="服务名"
              hint="请输入服务名"
              v-model="updateService"
            ></v-text-field>
          </v-flex>
          <v-flex style="margin-left: 10px;" xs6 sm3 md2>
            <v-text-field
              label="服务分组"
              hint="请输入服务group(可选)"
              v-model="updateGroup"
            ></v-text-field>
          </v-flex>
          <v-flex style="margin-left: 10px;" xs6 sm3 md2>
            <v-text-field
              label="服务版本"
              hint="请输入服务version(可选)"
              v-model="updateVersion"
            ></v-text-field>
          </v-flex>
        <v-layout wrap>
          <v-flex style="margin-left: 10px;" xs6 sm3 md2>
              <v-text-field
                label="超时时间"
                hint="请输入一个整数值作为超时时间(单位ms)"
                type="number"
                v-model="updateTimeout"
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
  name: 'Timeout',
  components: { Breadcrumb },
  data: () => ({
    breads: [
      {
        text: 'trafficManagement',
        href: ''
      },
      {
        text: 'trafficTimeout',
        href: ''
      }
    ],
    typeAhead: [],
    input: null,
    searchLoading: false,
    timerID: null,
    service: '',
    timeout: null,
    group: '',
    version: '',
    createGroup: '',
    createVersion: '',
    updateService: '',
    updateTimeout: NaN,
    updateGroup: '',
    updateVersion: '',
    deleteDialog: false,
    createService: '',
    createTimeout: NaN,
    deleteService: '',
    deleteTimeout: NaN,
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
      if (this.service) {
        this.search()
      } else {
        this.$notify.error('service is needed')
        return false
      }
    },
    search () {
      this.$axios.get('/traffic/timeout', {
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
      this.$axios.put('/traffic/timeout', {
        service: this.updateService,
        timeout: parseInt(this.updateTimeout),
        group: this.updateGroup,
        version: this.updateVersion
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
          value: 'service'
        },
        {
          text: '超时时间',
          value: 'timeout'
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
    },
    confirmDelete () {
      console.log(this.deleteTimeout)
      this.$axios.delete('/traffic/timeout', {
        service: this.deleteService,
        group: this.deleteGroup,
        version: this.deleteVersion
      }).then((res) => {
        if (res) {
          alert('操作成功')
        }
      })
      this.deleteTimeout = false
    },
    deleteItem (props) {
      this.deleteDialog = true
      this.deleteService = props.service
      this.deleteTimeout = props.timeout
      this.deleteGroup = props.group
      this.deleteVersion = props.version
    },
    update (props) {
      this.updateService = props.service
      this.updateTimeout = props.timeout
      this.updateGroup = props.group
      this.updateVersion = props.version
      this.updateDialog = true
    },
    save () {
      this.$axios.post('/traffic/timeout', {
        service: this.createService,
        timeout: parseInt(this.createTimeout),
        group: this.createGroup,
        version: this.createVersion
      }).then((res) => {
        if (res) {
          alert('操作成功')
        }
      })
      this.dialog = false
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
