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
  <v-container grid-list-xl fluid >
    <v-layout row wrap>
      <v-flex lg12>
        <breadcrumb title="routingRule" :items="breads"></breadcrumb>
      </v-flex>
      <v-flex lg12>
        <v-card flat color="transparent">
          <v-card-text>
            <v-form>
              <v-layout row wrap>
                <v-flex>
                  <v-combobox
                    id="serviceSearch"
                    v-model="filter"
                    :loading="searchLoading"
                    :items="typeAhead"
                    :search-input.sync="input"
                    @keyup.enter="submit"
                    flat
                    append-icon=""
                    hide-no-data
                    :suffix="queryBy"
                    :label="$t('searchRoutingRule')"
                    @input="split($event)"
                  ></v-combobox>
                </v-flex>
                <v-menu class="hidden-xs-only">
                  <v-btn slot="activator" large icon>
                    <v-icon>unfold_more</v-icon>
                  </v-btn>

                  <v-list>
                    <v-list-tile
                      v-for="(item, i) in items"
                      :key="i"
                      @click="selected = i">
                      <v-list-tile-title>{{ $t(item.title) }}</v-list-tile-title>
                    </v-list-tile>
                  </v-list>
                </v-menu>
                <v-flex xs6 sm3 md2 v-show="selected === 0">
                  <v-text-field
                    v-show="selected === 0"
                    label="Version"
                    :hint="$t('dataIdVersionHint')"
                    v-model="serviceVersion4Search"
                  ></v-text-field>
                </v-flex>
                <v-flex xs6 sm3 md2 v-show="selected === 0">
                  <v-text-field
                    label="Group"
                    :hint="$t('dataIdGroupHint')"
                    v-model="serviceGroup4Search"
                  ></v-text-field>
                </v-flex>
                <v-btn @click="submit" color="primary" large>{{$t('search')}}</v-btn>

              </v-layout>
            </v-form>
          </v-card-text>
        </v-card>
      </v-flex>
    </v-layout>
    <v-flex lg12>
      <v-card>
        <v-toolbar flat color="transparent" class="elevation-0">
          <v-toolbar-title><span class="headline">{{$t('searchResult')}}</span></v-toolbar-title>
          <v-spacer></v-spacer>
          <v-btn outline color="primary" @click.stop="openDialog" class="mb-2">{{$t('create')}}</v-btn>
        </v-toolbar>

        <v-card-text class="pa-0" v-show="selected === 0">
          <v-data-table
            :headers="serviceHeaders"
            :items="serviceRoutingRules"
            hide-actions
            class="elevation-0"
          >
            <template slot="items" slot-scope="props">
              <td class="text-xs-left">{{ props.item.service }}</td>
              <td class="text-xs-left">{{ props.item.serviceGroup }}</td>
              <td class="text-xs-left">{{ props.item.serviceVersion }}</td>
              <td class="text-xs-left">{{ props.item.enabled }}</td>
              <td class="text-xs-center px-0">
                <v-tooltip bottom v-for="op in operations" :key="op.id">
                  <v-icon small class="mr-2" slot="activator" @click="itemOperation(op.icon(props.item), props.item)">
                    {{op.icon(props.item)}}
                  </v-icon>
                  <span>{{$t(op.tooltip(props.item))}}</span>
                </v-tooltip>
              </td>
            </template>
          </v-data-table>
        </v-card-text>
        <v-card-text class="pa-0" v-show="selected === 1">
          <v-data-table
            :headers="appHeaders"
            :items="appRoutingRules"
            hide-actions
            class="elevation-0"
          >
            <template slot="items" slot-scope="props">
              <td class="text-xs-left">{{ props.item.application }}</td>
              <td class="text-xs-left">{{ props.item.enabled }}</td>
              <td class="text-xs-center px-0">
                <v-tooltip bottom v-for="op in operations" :key="op.id">
                  <v-icon small class="mr-2" slot="activator" @click="itemOperation(op.icon(props.item), props.item)">
                    {{op.icon(props.item)}}
                  </v-icon>
                  <span>{{$t(op.tooltip(props.item))}}</span>
                </v-tooltip>
              </td>
            </template>
          </v-data-table>
        </v-card-text>
      </v-card>
    </v-flex>

    <v-dialog v-model="dialog" width="800px" persistent >
      <v-card>
        <v-card-title class="justify-center">
          <span class="headline">{{$t('createNewRoutingRule')}}</span>
        </v-card-title>
        <v-card-text >
          <v-layout wrap>
            <v-flex xs24 sm12 md8>
              <v-text-field
                label="Service class"
                :hint="$t('dataIdClassHint')"
                v-model="service"
              ></v-text-field>
            </v-flex>
            <v-flex xs6 sm3 md2>
              <v-text-field
                label="Version"
                :hint="$t('dataIdVersionHint')"
                v-model="serviceVersion"
              ></v-text-field>
            </v-flex>
            <v-flex xs6 sm3 md2>
              <v-text-field
                label="Group"
                :hint="$t('dataIdGroupHint')"
                v-model="serviceGroup"
              ></v-text-field>
            </v-flex>
          </v-layout>
          <v-text-field
            label="Application Name"
            hint="Application name the service belongs to"
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

    <v-dialog v-model="warn" persistent max-width="500px">
      <v-card>
        <v-card-title class="headline">{{$t(this.warnTitle)}}</v-card-title>
        <v-card-text >{{this.warnText}}</v-card-text>
        <v-card-actions>
          <v-spacer></v-spacer>
          <v-btn flat @click.native="closeWarn">{{$t('cancel')}}</v-btn>
          <v-btn depressed color="primary" @click.native="deleteItem(warnStatus)">{{$t('confirm')}}</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

  </v-container>

</template>
<script>
import yaml from 'js-yaml'
import AceEditor from '@/components/public/AceEditor'
import operations from '@/api/operation'
import Breadcrumb from '@/components/public/Breadcrumb'
export default {
  components: {
    AceEditor,
    Breadcrumb
  },
  data: () => ({
    items: [
      { id: 0, title: 'serviceName', value: 'service' },
      { id: 1, title: 'app', value: 'application' }
    ],
    breads: [
      {
        text: 'serviceGovernance',
        href: ''
      },
      {
        text: 'routingRule',
        href: ''
      }
    ],
    selected: 0,
    dropdown_font: ['Service', 'App', 'IP'],
    ruleKeys: ['enabled', 'force', 'runtime', 'group', 'version', 'rule'],
    pattern: 'Service',
    filter: '',
    serviceVersion4Search: '',
    serviceGroup4Search: '',
    dialog: false,
    warn: false,
    updateId: '',
    application: '',
    service: '',
    serviceVersion: '',
    serviceGroup: '',
    warnTitle: '',
    warnText: '',
    warnStatus: {},
    height: 0,
    searchLoading: false,
    typeAhead: [],
    input: null,
    timerID: null,
    operations: operations,
    serviceRoutingRules: [
    ],
    appRoutingRules: [
    ],
    template:
        'enabled: true\n' +
        'runtime: false\n' +
        'force: true\n' +
        'conditions:\n' +
        ' - \'=> host != 172.22.3.91\'\n',
    ruleText: '',
    readonly: false,
    appHeaders: [],
    serviceHeaders: []
  }),
  methods: {
    setAppHeaders: function () {
      this.appHeaders = [
        {
          text: this.$t('appName'),
          value: 'application',
          align: 'left'
        },
        {
          text: this.$t('enabled'),
          value: 'enabled',
          sortable: false
        },
        {
          text: this.$t('operation'),
          value: 'operation',
          sortable: false,
          width: '115px'
        }
      ]
    },
    setServiceHeaders: function () {
      this.serviceHeaders = [
        {
          text: this.$t('serviceName'),
          value: 'service',
          align: 'left'
        },
        {
          text: this.$t('group'),
          value: 'group',
          align: 'left'

        },
        {
          text: this.$t('version'),
          value: 'group',
          align: 'left'

        },
        {
          text: this.$t('enabled'),
          value: 'enabled',
          sortable: false
        },
        {
          text: this.$t('operation'),
          value: 'operation',
          sortable: false,
          width: '115px'
        }
      ]
    },
    querySelections (v) {
      if (this.timerID) {
        clearTimeout(this.timerID)
      }
      // Simulated ajax query
      this.timerID = setTimeout(() => {
        if (v && v.length >= 4) {
          this.searchLoading = true
          if (this.selected === 0) {
            this.typeAhead = this.$store.getters.getServiceItems(v)
          } else if (this.selected === 1) {
            this.typeAhead = this.$store.getters.getConsumerItems(v)
          }
          this.searchLoading = false
          this.timerID = null
        } else {
          this.typeAhead = []
        }
      }, 500)
    },
    submit: function () {
      this.filter = document.querySelector('#serviceSearch').value.trim()
      this.search(true)
    },
    split: function (service) {
      if (this.selected === 0) {
        const groupSplit = service.split('/')
        const versionSplit = service.split(':')
        console.log(groupSplit)
        console.log(versionSplit)
        if (groupSplit.length > 1) {
          this.serviceGroup4Search = groupSplit[0]
        } else {
          this.serviceGroup4Search = ''
        }
        if (versionSplit.length > 1) {
          this.serviceVersion4Search = versionSplit[1]
        } else {
          this.serviceVersion4Search = ''
        }
        const serviceSplit = versionSplit[0].split('/')
        if (serviceSplit.length === 1) {
          this.filter = serviceSplit[0]
        } else {
          this.filter = serviceSplit[1]
        }
      }
    },
    search: function (rewrite) {
      if (!this.filter) {
        this.$notify.error('Either service or application is needed')
        return
      }
      const type = this.items[this.selected].value
      const url = '/rules/route/condition/?' + type + '=' + this.filter + '&serviceVersion=' + this.serviceVersion4Search + '&serviceGroup=' + this.serviceGroup4Search
      this.$axios.get(url)
        .then(response => {
          if (this.selected === 0) {
            this.serviceRoutingRules = response.data
          } else {
            this.appRoutingRules = response.data
          }
          if (rewrite) {
            if (this.selected === 0) {
              this.$router.push({ path: 'routingRule', query: { service: this.filter, serviceVersion: this.serviceVersion4Search, serviceGroup: this.serviceGroup4Search } })
            } else if (this.selected === 1) {
              this.$router.push({ path: 'routingRule', query: { application: this.filter } })
            }
          }
        })
    },
    closeDialog: function () {
      this.ruleText = this.template
      this.updateId = ''
      this.service = ''
      this.serviceVersion = ''
      this.serviceGroup = ''
      this.application = ''
      this.dialog = false
      this.readonly = false
    },
    openDialog: function () {
      this.dialog = true
    },
    openWarn: function (title, text) {
      this.warnTitle = title
      this.warnText = text
      this.warn = true
    },
    closeWarn: function () {
      this.warnTitle = ''
      this.warnText = ''
      this.warn = false
    },
    saveItem: function () {
      const rule = yaml.safeLoad(this.ruleText)
      if (!this.service && !this.application) {
        this.$notify.error('Either service or application is needed')
        return
      }
      if (this.service && this.application) {
        this.$notify.error('You can not set both service ID and application name')
        return
      }
      const vm = this
      rule.service = this.service
      const serviceVersion = this.serviceVersion == null ? '' : this.serviceVersion
      const serviceGroup = this.serviceGroup == null ? '' : this.serviceGroup
      rule.application = this.application
      rule.serviceVersion = serviceVersion
      rule.serviceGroup = serviceGroup
      if (this.updateId !== '') {
        if (this.updateId === 'close') {
          this.closeDialog()
        } else {
          rule.id = this.updateId
          this.$axios.put('/rules/route/condition/' + rule.id, rule)
            .then(response => {
              if (response.status === 200) {
                if (vm.service) {
                  vm.selected = 0
                  vm.search(vm.service, true)
                  vm.filter = vm.service
                } else {
                  vm.selected = 1
                  vm.search(vm.application, true)
                  vm.filter = vm.application
                }
                this.closeDialog()
                this.$notify.success('Update success')
              }
            })
        }
      } else {
        this.$axios.post('/rules/route/condition/', rule)
          .then(response => {
            if (response.status === 201) {
              if (vm.service) {
                vm.selected = 0
                vm.search(vm.service, true)
                vm.filter = vm.service
              } else {
                vm.selected = 1
                vm.search(vm.application, true)
                vm.filter = vm.application
              }
              this.closeDialog()
              this.$notify.success('Create success')
            }
          })
          .catch(error => {
            console.log(error)
          })
      }
    },
    itemOperation: function (icon, item) {
      const itemId = item.id
      const serviceVersion = item.serviceVersion == null ? '' : item.serviceVersion
      const serviceGroup = item.serviceGroup == null ? '' : item.serviceGroup
      const scope = item.scope == null ? '' : item.scope
      switch (icon) {
        case 'visibility':
          this.$axios.get('/rules/route/condition/' + itemId)
            .then(response => {
              const conditionRoute = response.data
              this.serviceVersion = conditionRoute.serviceVersion
              this.serviceGroup = conditionRoute.serviceGroup
              this.scope = conditionRoute.scope
              delete conditionRoute.serviceVersion
              delete conditionRoute.serviceGroup
              delete conditionRoute.scope
              this.handleBalance(conditionRoute, true)
              this.updateId = 'close'
            })
          break
        case 'edit':
          this.$axios.get('/rules/route/condition/' + itemId)
            .then(response => {
              const conditionRoute = response.data
              this.serviceVersion = conditionRoute.serviceVersion
              this.serviceGroup = conditionRoute.serviceGroup
              this.scope = conditionRoute.scope
              delete conditionRoute.serviceVersion
              delete conditionRoute.serviceGroup
              delete conditionRoute.scope
              this.handleBalance(conditionRoute, false)
              this.updateId = itemId
            })
          break
        case 'block':
          this.openWarn(' Are you sure to block Routing Rule', 'service: ' + itemId)
          this.warnStatus.operation = 'disable'
          this.warnStatus.id = itemId
          this.warnStatus.serviceVersion = serviceVersion
          this.warnStatus.serviceGroup = serviceGroup
          this.warnStatus.scope = scope
          break
        case 'check_circle_outline':
          this.openWarn(' Are you sure to enable Routing Rule', 'service: ' + itemId)
          this.warnStatus.operation = 'enable'
          this.warnStatus.id = itemId
          this.warnStatus.serviceVersion = serviceVersion
          this.warnStatus.serviceGroup = serviceGroup
          this.warnStatus.scope = scope
          break
        case 'delete':
          this.openWarn('warnDeleteRouteRule', 'service: ' + itemId)
          this.warnStatus.operation = 'delete'
          this.warnStatus.id = itemId
          this.warnStatus.serviceVersion = serviceVersion
          this.warnStatus.serviceGroup = serviceGroup
          this.warnStatus.scope = scope
      }
    },
    handleBalance: function (conditionRoute, readonly) {
      this.service = conditionRoute.service
      this.application = conditionRoute.application
      delete conditionRoute.service
      delete conditionRoute.id
      delete conditionRoute.app
      delete conditionRoute.group
      delete conditionRoute.application
      delete conditionRoute.priority
      this.ruleText = yaml.safeDump(conditionRoute)
      this.readonly = readonly
      this.dialog = true
    },
    setHeight: function () {
      this.height = window.innerHeight * 0.5
    },
    deleteItem: function (warnStatus) {
      const id = warnStatus.id
      const operation = warnStatus.operation
      const serviceVersion = warnStatus.serviceVersion
      const serviceGroup = warnStatus.serviceGroup
      const scope = warnStatus.scope
      if (operation === 'delete') {
        this.$axios.delete('/rules/route/condition/' + id + '?serviceVersion=' + serviceVersion + '&serviceGroup=' + serviceGroup + '&scope=' + scope)
          .then(response => {
            if (response.status === 200) {
              this.warn = false
              this.search(this.filter, false)
              this.$notify.success('Delete success')
            }
          })
      } else if (operation === 'disable') {
        this.$axios.put('/rules/route/condition/disable/' + id + '?serviceVersion=' + serviceVersion + '&serviceGroup=' + serviceGroup + '&scope=' + scope)
          .then(response => {
            if (response.status === 200) {
              this.warn = false
              this.search(this.filter, false)
              this.$notify.success('Disable success')
            }
          })
      } else if (operation === 'enable') {
        this.$axios.put('/rules/route/condition/enable/' + id + '?serviceVersion=' + serviceVersion + '&serviceGroup=' + serviceGroup + '&scope=' + scope)
          .then(response => {
            if (response.status === 200) {
              this.warn = false
              this.search(this.filter, false)
              this.$notify.success('Enable success')
            }
          })
      }
    }
  },
  created () {
    this.setHeight()
  },
  computed: {
    queryBy () {
      return this.$t('by') + this.$t(this.items[this.selected].title)
    },
    area () {
      return this.$i18n.locale
    }
  },
  watch: {
    input (val) {
      this.querySelections(val)
    },
    area () {
      this.setAppHeaders()
      this.setServiceHeaders()
    }
  },
  mounted: function () {
    this.setAppHeaders()
    this.setServiceHeaders()
    this.$store.dispatch('loadServiceItems')
    this.$store.dispatch('loadConsumerItems')
    this.ruleText = this.template
    const query = this.$route.query
    let filter = null
    let queryServiceVersion = null
    let queryServiceGroup = null
    const vm = this
    Object.keys(query).forEach(function (key) {
      if (key === 'service') {
        filter = query[key]
        if (query.serviceVersion) {
          queryServiceVersion = query.serviceVersion
        }
        if (query.serviceGroup) {
          queryServiceGroup = query.serviceGroup
        }
        vm.selected = 0
      }
      if (key === 'application') {
        filter = query[key]
        vm.selected = 1
      }
    })
    if (queryServiceVersion != null) {
      this.serviceVersion4Search = query.serviceVersion
    }
    if (queryServiceGroup != null) {
      this.serviceGroup4Search = query.serviceGroup
    }
    if (filter !== null) {
      this.filter = filter
      this.search(false)
    }
  }

}
</script>
