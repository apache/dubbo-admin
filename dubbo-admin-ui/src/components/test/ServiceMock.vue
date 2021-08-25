<!--
  - Licensed to the Apache Software Foundation (ASF) under one or more
  - contributor license agreements.  See the NOTICE file distributed with
  - this work for additional information regarding copyright ownership.
  - The ASF licenses this file to You under the Apache License, Version 2.0
  - (the "License"); you may not use this file except in compliance with
  - the License.  You may obtain a copy of the License at
  -
  -     http://www.apache.org/licenses/LICENSE-2.0
  -
  - Unless required by applicable law or agreed to in writing, software
  - distributed under the License is distributed on an "AS IS" BASIS,
  - WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  - See the License for the specific language governing permissions and
  - limitations under the License.
  -->

<template>
  <v-container grid-list-xl fluid>
    <v-layout row wrap>
      <v-flex lg12>
        <breadcrumb title="serviceMock" :items="breads"></breadcrumb>
      </v-flex>
      <v-flex lg12>
        <v-card flat color="transparent">
          <v-card-text>
            <v-form>
              <v-layout row wrap>
                <v-combobox
                  id="serviceTestSearch"
                  :loading="searchLoading"
                  :search-input.sync="input"
                  :value=filter
                  flat
                  append-icon=""
                  hide-no-data
                  :hint="$t('testModule.searchServiceHint')"
                  :label="$t('placeholders.searchService')"
                  @keyup.enter="submitSearch"
                ></v-combobox>
                <v-btn @click="submitSearch" color="primary" large>{{ $t('search') }}</v-btn>
              </v-layout>
            </v-form>
          </v-card-text>
        </v-card>
      </v-flex>
      <v-flex xs12>
        <v-card>
          <v-toolbar flat color="transparent" class="elevation-0">
            <v-toolbar-title><span class="headline">{{'规则列表'}}</span></v-toolbar-title>
            <v-spacer></v-spacer>
            <v-btn outline color="success" class="mb-2">全局禁用</v-btn>

            <v-btn outline color="primary" @click.stop="openDialog" class="mb-2">{{$t('create')}}</v-btn>
          </v-toolbar>
          <v-card-text class="pa-0">
            <v-data-table :headers="headers"
                          :items="mockRules"
                          :pagination.sync="pagination"
                          :total-items="totalItems"
                          class="elevation-1">
              <template slot="items" slot-scope="props">
                <td>{{ props.item.serviceName }}</td>
                <td>
                  <v-chip label>{{ props.item.methodName }}</v-chip>
                </td>
                <td>{{ props.item.rule }}
                </td>
                <td>
                  <v-switch v-model="props.item.enable" inset></v-switch>
                </td>
                <td>
                  <v-btn class="tiny" color="primary" @click="editMockRule(props.item)"> 编辑 </v-btn>
                  <v-btn class="tiny" color="error" @click="deleteMockRule(props.item.id)"> 删除 </v-btn>
                </td>
              </template>
            </v-data-table>
          </v-card-text>
        </v-card>
      </v-flex>
    </v-layout>

    <v-dialog v-model="dialog" width="800px" persistent>
      <v-card>
        <v-card-title class="justify-center">
          <span class="headline">{{$t('createNewRoutingRule')}}</span>
        </v-card-title>
        <v-card-text >
          <v-text-field
            label="Service Name"
            :hint="$t('dataIdClassHint')"
            v-model="mockRule.serviceName"
          ></v-text-field>

          <v-text-field
            label="Method Name"
            hint="Application name the service belongs to"
            v-model="mockRule.methodName"
          ></v-text-field>

          <v-subheader class="pa-0 mt-3">{{$t('ruleContent')}}</v-subheader>
          <ace-editor v-model="mockRule.rule" :readonly="readonly"></ace-editor>

        </v-card-text>
        <v-card-actions>
          <v-spacer></v-spacer>
          <v-btn flat @click.native="closeDialog">{{$t('close')}}</v-btn>
          <v-btn depressed color="primary" @click.native="saveOrUpdateMockRule">{{$t('save')}}</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </v-container>
</template>

<script>
  import JsonEditor from '@/components/public/JsonEditor'
  import Search from '@/components/public/Search'
  import Breadcrumb from '@/components/public/Breadcrumb'
  import yaml from 'js-yaml'
  import AceEditor from '@/components/public/AceEditor'

  export default {
    name: 'ServiceMock',
    components: {
      JsonEditor,
      Search,
      Breadcrumb,
      yaml,
      AceEditor
    },
    data() {
      return {
        headers: [],
        mockRules: [],
        breads: [
          {
            text: 'mockRule',
            href: '/mock'
          }
        ],
        pagination: {
          page: 1,
          rowsPerPage: 10 // -1 for All
        },
        loadingRules: false,
        searchLoading: false,
        filter: null,
        totalItems: 0,
        dialog: false,
        mockRule: {
          serviceName: '',
          methodName: '',
          rule: '',
          enable: true
        },
        dialogType: 1
      }
    },
    methods: {
      setHeaders() {
        this.headers = [
          {
            text: this.$t('serviceName'),
            value: 'serviceName',
            sortable: false
          },
          {
            text: this.$t('methodName'),
            value: 'methodName',
            sortable: false
          },
          {
            text: '返回数据',
            value: 'rule',
            sortable: false
          },
          {
            text: '是否启用',
            value: 'enable',
            sortable: false
          },
          {
            text: this.$t('operation'),
            value: 'operation',
            sortable: false
          }
        ]
      },
      listMockRules(filter) {
        const page = this.pagination.page - 1
        const size = this.pagination.rowsPerPage === -1 ? this.totalItems : this.pagination.rowsPerPage
        this.loadingRules = true
        this.$axios.get('/mock/rule/list', {
          params: {
            page,
            size,
            filter
          }
        }).then(res => {
          this.mockRules = res.data.content
        }).catch(e => {
          this.showSnackbar('error', e.response.data.message)
        }).finally(() => this.loadingRules = false)
      },
      submitSearch() {
        this.listMockRules(this.filter)
      },
      openDialog() {
        this.dialog = true
      },
      closeDialog() {
        this.dialog = false
        this.dialogType = 1
        this.mockRule = {
          serviceName: '',
          methodName: '',
          rule: '',
          enable: true
        }
      },
      saveOrUpdateMockRule() {
        this.$axios.post("/mock/rule", this.mockRule).then(res => {
          this.$notify('保存规则成功', 'success')
          this.closeDialog()
          this.listMockRules()
        }).catch(e => this.showSnackbar('error', e.response.data.message))
      },
      deleteMockRule(id) {
        this.$axios.delete('/mock/rule', {
          data: {id}}
          ).then(res => {
            this.$notify('删除成功', 'success')
            this.listMockRules(this.filter)
        })
        .catch(e => this.$notify(e.response.data.message, 'error'))
      },
      editMockRule(mockRule) {
        this.mockRule = mockRule
        this.openDialog()
        this.dialogType = 2
      }
    },
    mounted() {
      this.setHeaders()
      this.listMockRules(this.filter)
    },
    computed: {
      area () {
        return this.$i18n.locale
      }
    },
    watch: {
      input (val) {
        this.querySelections(val)
      },
      area () {
        this.setHeaders()
      }
    }
  }
</script>

<style scoped>

  .tiny {
    min-width: 30px;
    height: 20px;
    font-size: 8px;
  }

  .tiny-icon {
    font-size: 18px;
  }
</style>
