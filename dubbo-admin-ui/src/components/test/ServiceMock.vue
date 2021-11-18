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
                  id="mockRule"
                  :loading="searchLoading"
                  :search-input.sync="input"
                  v-model="filter"
                  flat
                  append-icon=""
                  hide-no-data
                  :hint="$t('testModule.searchServiceHint')"
                  :label="$t('placeholders.searchService')"
                  @update:searchInput="updateFilter"
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
            <v-toolbar-title><span class="headline">{{$t('ruleList')}}</span></v-toolbar-title>
            <v-spacer></v-spacer>
            <v-btn outline color="primary" @click.stop="openDialog" class="mb-2">{{$t('create')}}</v-btn>
          </v-toolbar>
          <v-card-text class="pa-0">
            <v-data-table :headers="headers"
                          :items="mockRules"
                          :pagination.sync="pagination"
                          :total-items="totalItems"
                          :loading="loadingRules"
                          class="elevation-1">
              <template slot="items" slot-scope="props">
                <td>{{ props.item.serviceName }}</td>
                <td>
                  <v-chip label>{{ props.item.methodName }}</v-chip>
                </td>
                <td>{{ props.item.rule }}
                </td>
                <td>
                  <v-switch v-model="props.item.enable" inset @change="enableOrDisableMockRule(props.item)"></v-switch>
                </td>
                <td>
                  <v-btn class="tiny" color="primary" @click="editMockRule(props.item)"> {{$t('edit')}} </v-btn>
                  <v-btn class="tiny" color="error" @click="openDeleteDialog(props.item)"> {{$t('delete')}} </v-btn>
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
          <span class="headline">{{dialogType === 1 ? $t('createMockRule') : $t('editMockRule')}}</span>
        </v-card-title>
        <v-card-text >
          <v-text-field
            :label="$t('serviceName')"
            :hint="$t('dataIdClassHint')"
            v-model="mockRule.serviceName"
          ></v-text-field>

          <v-text-field
            :label="$t('methodName')"
            :hint="$t('methodNameHint')"
            v-model="mockRule.methodName"
          ></v-text-field>

          <v-subheader class="pa-0 mt-3">{{$t('ruleContent')}}</v-subheader>
          <ace-editor v-model="mockRule.rule"></ace-editor>

        </v-card-text>
        <v-card-actions>
          <v-spacer></v-spacer>
          <v-btn flat @click.native="closeDialog">{{$t('close')}}</v-btn>
          <v-btn depressed color="primary" @click.native="saveOrUpdateMockRule">{{$t('save')}}</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-dialog v-model="warnDialog" persistent max-width="500px">
      <v-card>
        <v-card-title class="headline">{{$t('deleteRuleTitle')}}</v-card-title>
        <v-card-actions>
          <v-spacer></v-spacer>
          <v-btn color="darken-1" flat @click.native="closeDeleteDialog">{{$t('cancel')}}</v-btn>
          <v-btn color="primary darken-1" depressed @click.native="deleteMockRule">{{$t('confirm')}}</v-btn>
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
        dialogType: 1,
        warnDialog: false,
        deleteRule: null
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
            text: this.$t('mockData'),
            value: 'rule',
            sortable: false
          },
          {
            text: this.$t('enabled'),
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
        const page = this.pagination.page - 1;
        const size = this.pagination.rowsPerPage === -1 ? this.totalItems : this.pagination.rowsPerPage;
        this.loadingRules = true;
        this.$axios.get('/mock/rule/list', {
          params: {
            page,
            size,
            filter
          }
        }).then(res => {
          this.mockRules = res.data.content;
          this.totalItems = res.data.totalElements
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
        this.dialog = false;
        this.dialogType = 1;
        this.mockRule = {
          serviceName: '',
          methodName: '',
          rule: '',
          enable: true
        }
      },
      saveOrUpdateMockRule() {
        this.$axios.post("/mock/rule", this.mockRule).then(res => {
          this.$notify(this.$t('saveRuleSuccess'), 'success');
          this.closeDialog();
          this.listMockRules()
        }).catch(e => this.showSnackbar('error', e.response.data.message))
      },
      deleteMockRule() {
        const id = this.deleteRule.id
        this.$axios.delete('/mock/rule', {
          data: {id}}
          ).then(res => {
            this.$notify(this.$t('deleteRuleSuccess'), 'success');
            this.closeDeleteDialog()
            this.listMockRules(this.filter)
        })
        .catch(e => this.$notify(e.response.data.message, 'error'))
      },
      editMockRule(mockRule) {
        this.mockRule = mockRule;
        this.openDialog();
        this.dialogType = 2
      },
      enableOrDisableMockRule(mockRule) {
        this.$axios.post('/mock/rule', mockRule)
        .then(res => this.$notify(mockRule.enable ? this.$t('enableRuleSuccess') : this.$t('disableRuleSuccess'), 'success'))
        .catch(e => this.$notify(e.data.response.message, 'error'))
      },
      updateFilter() {
        this.filter = document.querySelector('#mockRule').value.trim();
      },
      closeDeleteDialog() {
        this.warnDialog = false
        this.deleteRule = null
      },
      openDeleteDialog(rule) {
        this.warnDialog = true
        this.deleteRule = rule
      }
    },
    mounted() {
      this.setHeaders();
      this.listMockRules(this.filter);
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
      },
      pagination: {
        handler (newVal, oldVal) {
          if (newVal.page === oldVal.page && newVal.rowsPerPage === oldVal.rowsPerPage) {
            return
          }
          const filter = this.filter;
          this.listMockRules(filter)
        },
        deep: true
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
