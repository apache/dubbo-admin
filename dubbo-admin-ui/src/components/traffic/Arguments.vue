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
            <Breadcrumb title="trafficArguments" :items="breads"></breadcrumb>
          </v-flex>
          <v-flex lg12>
            可在这里了解服务 <a href="https://cn.dubbo.apache.org/zh-cn/overview/tasks/traffic-management/arguments/" target="_blank">参数路由</a> 的工作原理与使用方式！
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
          <v-toolbar-title><span class="headline">{{$t('trafficArguments')}}</span></v-toolbar-title>
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
        <span class="headline">{{$t('createArgumentRule')}}</span>
      </v-card-title>
      <v-layout row wrap>
        <v-flex lg12>
          可在这里了解如何配置服务的 <a href="https://dubbo.apache.org/zh-cn/overview/tasks/traffic-management/arguments/" target="_blank">参数路由</a>！
        </v-flex>
      </v-layout>
      <v-card-text >
        <v-layout wrap>
          <v-flex xs6 sm3 md3>
            <v-text-field
              label="服务名"
              hint="请输入服务名"
              v-model="createService"
            ></v-text-field>
          </v-flex>
          <v-flex style="margin-left: 10px;" xs6 sm3 md2>
            <v-text-field
              label="Group"
              hint="请输入服务分组group（可选）"
              v-model="createGroup"
            ></v-text-field>
          </v-flex>
          <v-flex style="margin-left: 10px;" xs6 sm3 md2>
            <v-text-field
              label="Version"
              hint="请输入服务版本version（可选）"
              v-model="createVersion"
            ></v-text-field>
          </v-flex>
        </v-layout>
        <v-layout wrap>
          <v-flex lg12>
            符合以下条件的参数调用：
          </v-flex>
        </v-layout>
        <v-layout wrap>
          <v-flex xs6 sm3 md2>
            <v-text-field
              label="方法名"
              hint="请输入方法名"
              v-model="createRuleMethod"
            ></v-text-field>
          </v-flex>
          <v-flex style="margin-left: 10px;" xs6 sm3 md2>
            <v-text-field
              label="参数索引"
              hint="如第一个参数，请输入0"
              type="number"
              v-model="createRuleIndex"
            ></v-text-field>
          </v-flex>
          <v-flex style="margin-left: 10px;" xs6 sm3 md2>
            <v-text-field
              label="参数匹配条件"
              hint="请输入参数匹配条件（仅支持字符串类型参数）"
              v-model="createRuleMatch"
            ></v-text-field>
          </v-flex>
        </v-layout>
        <v-layout wrap>
          <v-flex lg12>
            将被路由到符合以下条件的目标机器上：
          </v-flex>
        </v-layout>
        <v-layout wrap>
          <v-flex xs6 sm3 md5>
            <v-text-field
              label="输入目标机器过滤条件"
              hint="可以使用 URL 上的任意参数进行匹配，如 orderVersion=v2 & region=hangzhou，具体可参见文档说明。"
              v-model="createFilterCondition"
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
        <span class="headline">{{$t('createArgumentRule')}}</span>
      </v-card-title>
      <v-layout row wrap>
        <v-flex lg12>
          可在这里了解如何配置服务的 <a href="https://dubbo.apache.org/zh-cn/overview/tasks/traffic-management/arguments/" target="_blank">参数路由</a>！
        </v-flex>
      </v-layout>
      <v-card-text >
        <v-layout wrap>
          <v-flex xs6 sm3 md3>
            <v-text-field
              label="服务名"
              hint="请输入服务名"
              v-model="updateService"
            ></v-text-field>
          </v-flex>
          <v-flex style="margin-left: 10px;" xs6 sm3 md2>
            <v-text-field
              label="Group"
              hint="请输入服务分组group（可选）"
              v-model="updateGroup"
            ></v-text-field>
           </v-flex>
          <v-flex style="margin-left: 10px;" xs6 sm3 md2>
            <v-text-field
              label="Version"
              hint="请输入服务版本version（可选）"
              v-model="updateVersion"
            ></v-text-field>
           </v-flex>
        </v-layout>
        <v-layout wrap>
          <v-flex lg12>
            符合以下条件的参数调用：
          </v-flex>
        </v-layout>
        <v-layout wrap>
          <v-flex xs6 sm3 md2>
            <v-text-field
              label="方法名"
              hint="请输入方法名"
              v-model="updateRuleMethod"
            ></v-text-field>
          </v-flex>
          <v-flex style="margin-left: 10px;" xs6 sm3 md2>
            <v-text-field
              label="参数索引"
              hint="如第一个参数，请输入0"
              type="number"
              v-model="updateRuleIndex"
            ></v-text-field>
          </v-flex>
          <v-flex style="margin-left: 10px;" xs6 sm3 md2>
            <v-text-field
              label="参数匹配条件"
              hint="请输入参数匹配条件（仅支持字符串类型参数）"
              v-model="updateRuleMatch"
            ></v-text-field>
          </v-flex>
        </v-layout>
        <v-layout wrap>
          <v-flex lg12>
            将被路由到符合以下条件的目标机器上：
          </v-flex>
        </v-layout>
        <v-layout wrap>
          <v-flex xs6 sm3 md5>
            <v-text-field
              label="输入目标机器过滤条件"
              hint="可以是使用 URL 上的任意参数进行匹配，如 orderVersion=v2 & region=hangzhou，具体可参见文档说明。"
              v-model="updateFilterCondition"
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
  name: 'Arguments',
  components: { Breadcrumb },
  data: () => ({
    breads: [
      {
        text: 'trafficManagement',
        href: ''
      },
      {
        text: 'trafficArguments',
        href: ''
      }
    ],
    typeAhead: [],
    input: null,
    searchLoading: false,
    timerID: null,
    service: '',
    group: '',
    version: '',
    createGroup: '',
    createVersion: '',
    createRuleMethod: '',
    createRuleIndex: '',
    createRuleMatch: '',
    createFilterCondition: '',
    updateService: '',
    updateRuleMethod: '',
    updateRuleIndex: '',
    updateRuleMatch: '',
    updateFilterCondition: '',
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
      this.search()
    },
    search () {
      this.$axios.get('/traffic/argument', {
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
      if (!this.updateRuleMethod || !this.updateRuleMatch || !this.updateRuleIndex || !this.updateFilterCondition) {
        alert('请分别输入方法匹配条件和机器过滤条件')
      } else {
        const matchCondition = `method=${this.updateRuleMethod} & arguments[${this.updateRuleIndex}]=${this.updateRuleMatch}`
        const filterCondition = ` => ${this.updateFilterCondition}`
        this.$axios.put('/traffic/argument', {
          service: this.updateService,
          rule: matchCondition + filterCondition,
          group: this.updateGroup,
          version: this.updateVersion
        }).then((res) => {
          if (res) {
            alert('操作成功')
          }
        })
      }
    },
    setHeaders: function () {
      this.headers = [
        {
          text: '服务',
          value: 'service'
        },
        {
          text: '参数路由条件',
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
    },
    confirmDelete () {
      console.log(this.deleteArguments)
      this.$axios.delete('/traffic/argument', {
        service: this.deleteService,
        group: this.deleteGroup,
        version: this.deleteVersion
      }).then((res) => {
        if (res) {
          alert('操作成功')
        }
      })
      this.deleteArguments = false
    },
    deleteItem (props) {
      this.deleteDialog = true
      this.deleteService = props.service
      this.deleteGroup = props.group
      this.deleteVersion = props.version
    },
    update (props) {
      this.updateService = props.service
      var parts = props.rule.split(/(\w+)\[(\w+)\]=(\w+)/)
      this.updateRuleMethod = parts[1]
      this.updateRuleIndex = parts[2]
      this.updateRuleMatch = parts[3]
      this.updateGroup = props.group
      this.updateVersion = props.version
      this.updateDialog = true
    },
    save () {
      // if (!this.createRuleMethod || !this.createRuleMatch || !this.createRuleIndex || !this.createFilterCondition) {
      //  alert('请分别输入方法匹配条件和机器过滤条件')
      // } else {
      const matchCondition = `method=${this.createRuleMethod} & arguments[${this.createRuleIndex}]=${this.createRuleMatch}`
      const filterCondition = ` => ${this.createFilterCondition}`
      this.$axios.post('/traffic/argument', {
        service: this.createService,
        rule: matchCondition + filterCondition,
        group: this.createGroup,
        version: this.createVersion
      }).then((res) => {
        if (res) {
          alert('操作成功')
        }
      })
      // }
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
