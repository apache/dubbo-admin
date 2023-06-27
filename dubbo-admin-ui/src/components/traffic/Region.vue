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
      <v-card-text >
        <v-layout wrap>
          <v-flex xs6 sm3 md3>
            <v-text-field
              label="服务名"
              hint="请输入服务名"
              v-model="createService"
            ></v-text-field>
          </v-flex>
          <v-flex xs6 sm3 md2>
            <v-text-field
              label="Group"
              hint="$t('groupInputPrompt')"
              v-model="createGroup"
            ></v-text-field>
          </v-flex>
          <v-flex xs6 sm3 md2>
            <v-text-field
              label="Version"
              hint="$t('versionInputPrompt')"
              v-model="createVersion"
            ></v-text-field>
          </v-flex>
        </v-layout>
        <v-text-field
          label="开启或关闭同区域优先（这里应该是一个开关，让用户选择打开或关闭同区域优先）"
          hint=""
          v-model="createRule"
        ></v-text-field>
        <v-text-field
          label="匹配以下条件的流量开启同区域优先（默认不显示，用户点击后才显示出来让用户输入）"
          hint="请输入流量匹配规则（默认不设置，则对所有流量生效），配置后只有匹配规则的流量才会执行同区域优先调用，如 method=sayHello"
          v-model="createRule"
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
      <v-card-text >
        <v-layout wrap>
          <v-flex>
            <v-text-field
              label="服务名"
              hint="请输入服务名"
              v-model="updateService"
            ></v-text-field>
          </v-flex>
        </v-layout>
        <v-text-field
          label="Group"
          hint="$t('groupInputPrompt')"
          v-model="updateGroup"
        ></v-text-field>
        <v-text-field
          label="Version"
          hint="$t('versionInputPrompt')"
          v-model="updateVersion"
        ></v-text-field>
        <v-text-field
          label="开启或关闭同区域优先（这里应该是一个开关，让用户选择打开或关闭同区域优先）"
          hint="这应该是一个 radio button，让用户选择是否开启同区域优先？"
          v-model="updateRule1"
        ></v-text-field>
        <v-text-field
          label="匹配以下条件的流量开启同区域优先（默认不显示，用户点击后才显示出来让用户输入）"
          hint="请输入流量匹配规则（默认不设置，则对所有流量生效），配置后只有匹配规则的流量才会执行同区域优先调用，如 method=sayHello"
          v-model="updateRule2"
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
      if (this.service && this.rule) {
        this.search()
      } else {
        this.$notify.error('service is needed')
        return false
      }
    },
    search () {
      this.$axios.get('/traffic/region', {
        params: {
          service: this.service,
          rule: this.rule,
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
      this.$axios.put('/traffic/region', {
        service: this.updateService,
        rule: this.updateRule,
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
          text: '应用',
          value: 'service'
        },
        {
          text: '应用规则',
          value: 'rule'
        },
        {
          text: 'Group',
          value: 'group'
        },
        {
          text: 'Version',
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
      console.log(this.deleteRegion)
      this.$axios.delete('/traffic/region', {
        service: this.deleteService,
        rule: this.deleteRule,
        group: this.deleteGroup,
        version: this.deleteVersion
      }).then((res) => {
        if (res) {
          alert('操作成功')
        }
      })
      this.deleteRegion = false
    },
    deleteItem (props) {
      this.deleteDialog = true
      this.deleteService = props.service
      this.deleteRule = props.rule
      this.deleteGroup = props.group
      this.deleteVersion = props.version
    },
    update (props) {
      this.updateService = props.service
      this.updateRule = props.rule
      this.updateGroup = props.group
      this.updateVersion = props.version
      this.updateDialog = true
    },
    save () {
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
