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
        <breadcrumb title="meshRule" :items="breads"></breadcrumb>
      </v-flex>
    </v-layout>
    <v-flex lg12>
      <v-card flat color="transparent">
        <v-card-text>
          <v-form>
            <v-layout row wrap>
              <v-combobox
                id="serviceSearch"
                :loading="searchLoading"
                :items="typeAhead"
                :search-input.sync="input"
                @keyup.enter="submit"
                v-model="filter"
                flat
                append-icon=""
                hide-no-data
                :label="$t('searchMeshRule')"
              ></v-combobox>
              <v-btn @click="submit" color="primary" large>{{$t('search')}}</v-btn>

            </v-layout>
          </v-form>
        </v-card-text>
      </v-card>
    </v-flex>
    <v-flex lg12>
      <v-card>
        <v-toolbar flat color="transparent" class="elevation-0">
          <v-toolbar-title><span class="headline">{{$t('searchResult')}}</span></v-toolbar-title>
          <v-spacer></v-spacer>
          <v-btn outline color="primary" @click.stop="openDialog" class="mb-2">{{$t('create')}}</v-btn>
        </v-toolbar>

        <v-card-text class="pa-0">
          <v-data-table
            :headers="headers"
            :items="meshRoutingRules"
            hide-actions
            class="elevation-0"
          >
            <template slot="items" slot-scope="props">
              <td class="text-xs-left">{{ props.item.application }}</td>
              <td class="text-xs-center px-0">
                <v-tooltip bottom>
                  <v-icon small
                          class="mr-2"
                          color="blue"
                          slot="activator"
                          @click="itemOperation('visibility', props.item)">visibility
                  </v-icon>
                  <span>{{$t('view')}}</span>
                </v-tooltip>
                <v-tooltip bottom>
                  <v-icon small
                          class="mr-2"
                          color="blue"
                          slot="activator"
                          @click="itemOperation('edit', props.item)">edit
                  </v-icon>
                  <span>Edit</span>
                </v-tooltip>
                <v-tooltip bottom>
                  <v-icon small
                          class="mr-2"
                          slot="activator"
                          color="red"
                          @click="itemOperation('delete', props.item)">delete
                  </v-icon>
                  <span>Delete</span>
                </v-tooltip>
              </td>
            </template>
          </v-data-table>
        </v-card-text>
      </v-card>
    </v-flex>

    <v-dialog v-model="dialog" width="800px" persistent>
      <v-card>
        <v-card-title class="justify-center">
          <span class="headline">{{$t('createNewMeshRule')}}</span>
        </v-card-title>
        <v-card-text>
          <v-text-field
            :label="$t('appName')"
            :hint="$t('appNameHint')"
            v-model="application"
          ></v-text-field>

          <v-subheader class="pa-0 mt-3">{{$t('ruleContent')}}</v-subheader>
          <ace-editor v-model="ruleText" :readonly="readonly"></ace-editor>

        </v-card-text>
        <v-card-actions>
          <v-spacer></v-spacer>
          <v-btn flat @click.native="closeDialog">{{$t('close')}}</v-btn>
          <v-btn depressed color="primary" @click.native="saveItem">{{$t('save')}}</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-dialog v-model="warn.display" persistent max-width="500px">
      <v-card>
        <v-card-title class="headline">{{$t(this.warn.title)}}</v-card-title>
        <v-card-text>{{this.warn.text}}</v-card-text>
        <v-card-actions>
          <v-spacer></v-spacer>
          <v-btn flat @click.native="closeWarn">CANCLE</v-btn>
          <v-btn depressed color="primary" @click.native="deleteItem(warn.status)">{{$t('confirm')}}</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

  </v-container>

</template>
<script>
  import yaml from 'js-yaml'
  import AceEditor from '@/components/public/AceEditor'
  import operations from '@/api/operation'
  import Search from '@/components/public/Search'
  import Breadcrumb from '@/components/public/Breadcrumb'

  export default {
    components: {
      AceEditor,
      Search,
      Breadcrumb
    },
    data: () => ({
      dropdown_font: ['Service', 'App', 'IP'],
      ruleKeys: ['enabled', 'force', 'dynamic', 'runtime', 'group', 'version', 'rule'],
      pattern: 'Service',
      filter: '',
      dialog: false,
      updateId: '',
      application: '',
      searchLoading: false,
      typeAhead: [],
      input: null,
      timerID: null,
      warn: {
        display: false,
        title: '',
        text: '',
        status: {}
      },
      breads: [
        {
          text: 'serviceGovernance',
          href: ''
        },
        {
          text: 'meshRule',
          href: ''
        }
      ],
      height: 0,
      operations: operations,
      meshRoutingRules: [],
      template: 'apiVersion: service.dubbo.apache.org/v1alpha1\n' +
        'kind: DestinationRule\n' +
        'metadata: { name: demo-route }\n' +
        'spec:\n' +
        '  host: demo\n' +
        '  subsets:\n' +
        '    - labels: { env-sign: xxx, tag1: hello }\n' +
        '      name: isolation\n' +
        '    - labels: { env-sign: yyy }\n' +
        '      name: testing-trunk\n' +
        '    - labels: { env-sign: zzz }\n' +
        '      name: testing\n' +
        '  trafficPolicy:\n' +
        '    loadBalancer: { simple: ROUND_ROBIN }\n' +
        '\n' +
        '---\n' +
        '\n' +
        'apiVersion: service.dubbo.apache.org/v1alpha1\n' +
        'kind: VirtualService\n' +
        'metadata: {name: demo-route}\n' +
        'spec:\n' +
        '  dubbo:\n' +
        '    - routedetail:\n' +
        '        - match:\n' +
        '            - sourceLabels: {trafficLabel: xxx}\n' +
        '          name: xxx-project\n' +
        '          route:\n' +
        '            - destination: {host: demo, subset: isolation}\n' +
        '        - match:\n' +
        '            - sourceLabels: {trafficLabel: testing-trunk}\n' +
        '          name: testing-trunk\n' +
        '          route:\n' +
        '            - destination: {host: demo, subset: testing-trunk}\n' +
        '        - name: testing\n' +
        '          route:\n' +
        '            - destination: {host: demo, subset: testing}\n' +
        '      services:\n' +
        '        - {regex: ccc}\n' +
        '  hosts: [demo]',
      ruleText: '',
      readonly: false,
      headers: []
    }),
    methods: {
      setHeaders: function () {
        this.headers = [
          {
            text: this.$t('appName'),
            value: 'application',
            align: 'left'
          },
          {
            text: this.$t('operation'),
            value: 'operation',
            sortable: false,
            width: '115px'
          }
        ]
      },
      querySelections(v) {
        if (this.timerID) {
          clearTimeout(this.timerID)
        }
        // Simulated ajax query
        this.timerID = setTimeout(() => {
          if (v && v.length >= 4) {
            this.searchLoading = true
            this.typeAhead = this.$store.getters.getAppItems(v)
            this.searchLoading = false
            this.timerID = null
          } else {
            this.typeAhead = []
          }
        }, 500)
      },
      submit: function () {
        if (!this.filter) {
          this.$notify.error('application is needed')
          return
        }
        this.filter = this.filter.trim()
        this.search(true)
      },
      search: function (rewrite) {
        let url = '/rules/route/mesh/?application' + '=' + this.filter
        this.$axios.get(url)
          .then(response => {
            this.meshRoutingRules = response.data
            if (rewrite) {
              this.$router.push({path: 'meshRule', query: {application: this.filter}})
            }
          })
      },
      closeDialog: function () {
        this.ruleText = this.template
        this.updateId = ''
        this.application = ''
        this.dialog = false
        this.readonly = false
      },
      openDialog: function () {
        this.dialog = true
      },
      openWarn: function (title, text) {
        this.warn.title = title
        this.warn.text = text
        this.warn.display = true
      },
      closeWarn: function () {
        this.warn.title = ''
        this.warn.text = ''
        this.warn.display = false
      },
      saveItem: function () {
        const rule = {}
        rule.meshRule = this.ruleText
        if (!this.application) {
          this.$notify.error('application is required')
          return
        }
        rule.application = this.application
        let vm = this
        if (this.updateId) {
          if (this.updateId === 'close') {
            this.closeDialog()
          } else {
            rule.id = this.updateId
            this.$axios.put('/rules/route/mesh/' + rule.id, rule)
              .then(response => {
                if (response.status === 200) {
                  vm.search(vm.application, true)
                  vm.closeDialog()
                  vm.$notify.success('Update success')
                }
              })
          }
        } else {
          this.$axios.post('/rules/route/mesh/', rule)
            .then(response => {
              if (response.status === 201) {
                vm.search(vm.application, true)
                vm.filter = vm.application
                vm.closeDialog()
                vm.$notify.success('Create success')
              }
            })
            .catch(error => {
              console.log(error)
            })
        }
      },
      itemOperation: function (icon, item) {
        let itemId = item.application
        switch (icon) {
          case 'visibility':
            this.$axios.get('/rules/route/mesh/' + itemId)
              .then(response => {
                let meshRoute = response.data
                this.handleBalance(meshRoute, true)
                this.updateId = 'close'
              })
            break
          case 'edit':
            let id = {}
            id.id = itemId
            this.$axios.get('/rules/route/mesh/' + itemId)
              .then(response => {
                let meshRoute = response.data
                this.handleBalance(meshRoute, false)
                this.updateId = itemId
              })
            break
          case 'delete':
            this.openWarn('warnDeleteMeshRule', 'application: ' + item.application)
            this.warn.status.operation = 'delete'
            this.warn.status.id = itemId
        }
      },
      handleBalance: function (meshRoute, readonly) {
        this.application = meshRoute.application
        delete meshRoute.id
        delete meshRoute.application
        this.ruleText = meshRoute.meshRule
        this.readonly = readonly
        this.dialog = true
      },
      setHeight: function () {
        this.height = window.innerHeight * 0.5
      },
      deleteItem: function (warnStatus) {
        let id = warnStatus.id
        let operation = warnStatus.operation
        if (operation === 'delete') {
          this.$axios.delete('/rules/route/mesh/' + id)
            .then(response => {
              if (response.status === 200) {
                this.warn.display = false
                this.search(this.filter, false)
                this.$notify.success('Delete success')
              }
            })
        }
      }
    },
    created() {
      this.setHeight()
    },
    computed: {
      area() {
        return this.$i18n.locale
      }
    },
    watch: {
      input(val) {
        this.querySelections(val)
      },
      area() {
        this.setHeaders()
      }
    },
    mounted: function () {
      this.setHeaders()
      this.$store.dispatch('loadInstanceAppItems')
      this.ruleText = this.template
      let query = this.$route.query
      let filter = null
      Object.keys(query).forEach(function (key) {
        if (key === 'application') {
          filter = query[key]
        }
      })
      if (filter !== null) {
        this.filter = filter
        this.search(false)
      }
    }

  }
</script>
