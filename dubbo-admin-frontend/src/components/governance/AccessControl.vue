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
  <v-container grid-list-xl
               fluid>
    <v-layout row
              wrap>
      <v-flex lg12>
        <v-card flat color="transparent">
          <v-card-text>
            <v-form>
              <v-layout row wrap>
                <v-combobox
                  id="serviceSearch"
                  v-model="filter"
                  flat
                  append-icon=""
                  hide-no-data
                  :suffix="queryBy"
                  label="Search Routing Rule"
                ></v-combobox>
                <v-menu class="hidden-xs-only">
                  <v-btn slot="activator" large icon>
                    <v-icon>unfold_more</v-icon>
                  </v-btn>

                  <v-list>
                    <v-list-tile
                      v-for="(item, i) in items"
                      :key="i"
                      @click="selected = i">
                      <v-list-tile-title>{{ item.title }}</v-list-tile-title>
                    </v-list-tile>
                  </v-list>
                </v-menu>
                <v-btn @click="search" color="primary" large>Search</v-btn>

              </v-layout>
            </v-form>
          </v-card-text>
        </v-card>
      </v-flex>
    </v-layout>

    <v-flex lg12>
      <v-card>
        <v-toolbar flat
                   color="transparent"
                   class="elevation-0">
          <v-toolbar-title>
            <span class="headline">Search Result</span>
          </v-toolbar-title>
          <v-spacer></v-spacer>
          <v-btn outline
                 color="primary"
                 @click.stop="toCreate"
                 class="mb-2">CREATE</v-btn>
        </v-toolbar>

        <v-card-text class="pa-0" v-if="selected == 0">
          <v-data-table :headers="serviceHeaders"
                        :items="accesses"
                        :loading="loading"
                        hide-actions
                        class="elevation-0">
            <template slot="items"
                      slot-scope="props">
              <td class="text-xs-left">{{ props.item.service }}</td>
              <td class="text-xs-center px-0">
                <v-tooltip bottom>
                  <v-icon small
                          class="mr-2"
                          color="blue"
                          slot="activator"
                          @click="toEdit(props.item)">edit</v-icon>
                  <span>Edit</span>
                </v-tooltip>
                <v-tooltip bottom>
                  <v-icon small
                          class="mr-2"
                          slot="activator"
                          color="red"
                          @click="toDelete(props.item)">delete</v-icon>
                  <span>Delete</span>
                </v-tooltip>
              </td>
            </template>
          </v-data-table>
        </v-card-text>
        <v-card-text class="pa-0" v-if="selected == 1">
          <v-data-table :headers="appHeaders"
                        :items="accesses"
                        :loading="loading"
                        hide-actions
                        class="elevation-0">
            <template slot="items"
                      slot-scope="props">
              <td class="text-xs-left">{{ props.item.application }}</td>
              <td class="text-xs-center px-0">
                <v-tooltip bottom>
                  <v-icon small
                          class="mr-2"
                          color="blue"
                          slot="activator"
                          @click="toEdit(props.item)">edit</v-icon>
                  <span>Edit</span>
                </v-tooltip>
                <v-tooltip bottom>
                  <v-icon small
                          class="mr-2"
                          slot="activator"
                          color="red"
                          @click="toDelete(props.item)">delete</v-icon>
                  <span>Delete</span>
                </v-tooltip>
              </td>
            </template>
          </v-data-table>
        </v-card-text>
      </v-card>
    </v-flex>

    <v-dialog v-model="modal.enable"
              width="800px"
              persistent>
      <v-card>
        <v-card-title class="justify-center">
          <span class="headline">{{ modal.title }} Access Control</span>
        </v-card-title>
        <v-card-text>
          <v-form ref="modalForm">
            <v-text-field label="Service Unique ID"
                          hint="A service ID in form of group/service:version, group and version are optional"
                          :readonly="modal.id != null"
                          v-model="modal.service" />
            <v-text-field
              label="Application Name"
              hint="Application name the service belongs to"
              :readonly="modal.id != null"
              v-model="modal.application"
            ></v-text-field>
            <v-subheader class="pa-0 mt-3">BLACK/WHITE LIST CONTENT</v-subheader>
            <ace-editor v-model="modal.content" />
          </v-form>
        </v-card-text>
        <v-card-actions>
          <v-spacer></v-spacer>
          <v-btn color="darken-1"
                 flat
                 @click="closeModal()">Close</v-btn>
          <v-btn color="primary"
                 depressed
                 @click="modal.click">{{ modal.saveBtn }}</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-dialog v-model="confirm.enable"
              persistent
              max-width="500px">
      <v-card>
        <v-card-title class="headline">{{this.confirm.title}}</v-card-title>
        <v-card-text>{{this.confirm.text}}</v-card-text>
        <v-card-actions>
          <v-spacer></v-spacer>
          <v-btn color="darken-1"
                 flat
                 @click="confirm.enable = false">Disagree</v-btn>
          <v-btn color="primary"
                 depressed
                 @click="deleteItem(confirm.id)">Agree</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </v-container>
</template>

<script>
import yaml from 'js-yaml'
import AceEditor from '@/components/public/AceEditor'
import Search from '@/components/public/Search'

export default {
  name: 'AccessControl',
  data: () => ({
    items: [
      {id: 0, title: 'service name', value: 'serviceName'},
      {id: 1, title: 'application', value: 'application'}
    ],
    selected: 0,
    filter: null,
    loading: false,
    serviceHeaders: [
      {
        text: 'Service Name',
        value: 'service',
        align: 'left'
      },
      {
        text: 'Operation',
        value: 'operation',
        sortable: false,
        width: '115px'
      }
    ],
    appHeaders: [
      {
        text: 'Application Name',
        value: 'application',
        align: 'left'
      },
      {
        text: 'Operation',
        value: 'operation',
        sortable: false,
        width: '115px'
      }
    ],
    accesses: [],
    modal: {
      enable: false,
      title: 'Create New',
      saveBtn: 'Create',
      click: () => {},
      id: null,
      service: null,
      application: null,
      content: '',
      template:
        'blacklist:\n' +
        '  - 1.1.1.1\n' +
        '  - 2.2.2.2\n' +
        'whitelist:\n' +
        '  - 3.3.3.3\n' +
        '  - 4.4.*\n'
    },
    services: [],
    confirm: {
      enable: false,
      title: '',
      text: '',
      id: null
    },
    snackbar: {
      enable: false,
      text: ''
    }
  }),
  methods: {
    search () {
      if (this.filter == null) {
        return
      }
      let type = this.items[this.selected].value
      this.loading = true
      if (this.selected === 0) {
        this.$router.push({
          path: 'access',
          query: {service: this.filter}
        })
      } else if (this.selected === 1) {
        this.$router.push({
          path: 'access',
          query: {application: this.filter}
        })
      }
      let url = '/rules/access/?' + type + '=' + this.filter
      this.$axios.get(url)
        .then(response => {
          this.accesses = response.data
          this.loading = false
        }).catch(error => {
          this.showSnackbar('error', error.response.data.message)
          this.loading = false
        })
    },
    closeModal () {
      this.modal.enable = false
      this.modal.id = null
      this.$refs.modalForm.reset()
    },
    toCreate () {
      Object.assign(this.modal, {
        enable: true,
        title: 'Create New',
        saveBtn: 'Create',
        content: this.modal.template,
        click: this.createItem
      })
    },
    createItem () {
      let doc = yaml.load(this.modal.content)
      this.filter = ''
      if (this.modal.service === '' && this.modal.service === null) {
        this.$notify.error('Either service or application is needed')
        return
      }
      var vm = this
      this.$axios.post('/rules/access', {
        service: this.modal.service,
        application: this.modal.application,
        whitelist: doc.whitelist,
        blacklist: doc.blacklist
      }).then(response => {
        if (response.status === 201) {
          if (vm.modal.service !== null) {
            vm.selected = 0
            vm.filter = vm.modal.service
          } else {
            vm.selected = 1
            vm.filter = vm.modal.application
          }
          this.search()
          this.closeModal()
        }
        this.showSnackbar('success', 'Create success')
      }).catch(error => this.showSnackbar('error', error.response.data.message))
    },
    toEdit (item) {
      let itemId = null
      if (this.selected === 0) {
        itemId = item.service
      } else {
        itemId = item.application
      }
      Object.assign(this.modal, {
        enable: true,
        title: 'Edit',
        saveBtn: 'Update',
        click: this.editItem,
        id: itemId,
        service: item.service,
        application: item.application,
        content: yaml.safeDump({blacklist: item.blacklist, whitelist: item.whitelist})
      })
    },
    editItem () {
      let doc = yaml.load(this.modal.content)
      let vm = this
      this.$axios.put('/rules/access/' + this.modal.id, {
        whitelist: doc.whitelist,
        blacklist: doc.blacklist,
        application: this.modal.application,
        service: this.modal.service

      }).then(response => {
        if (response.status === 200) {
          if (vm.modal.service !== null) {
            vm.selected = 0
            vm.filter = vm.modal.service
          } else {
            vm.selected = 1
            vm.filter = vm.modal.application
          }
          vm.closeModal()
          vm.search()
        }
        this.showSnackbar('success', 'Update success')
      }).catch(error => this.showSnackbar('error', error.response.data.message))
    },
    toDelete (item) {
      let itemId = null
      if (this.selected === 0) {
        itemId = item.service
      } else {
        itemId = item.application
      }
      Object.assign(this.confirm, {
        enable: true,
        title: 'Are you sure to Delete Access Control',
        text: `Id: ${itemId}`,
        id: itemId
      })
    },
    deleteItem (id) {
      this.$axios.delete('/rules/access/' + id)
      .then(response => {
        this.showSnackbar('success', 'Delete success')
        this.search(this.filter)
      }).catch(error => this.showSnackbar('error', error.response.data.message))
    },
    showSnackbar (color, message) {
      this.$notify(message, color)
      this.confirm.enable = false
    }
  },
  computed: {
    queryBy () {
      return 'by ' + this.items[this.selected].title
    }
  },
  mounted () {
    let query = this.$route.query
    if ('service' in query) {
      this.filter = query['service']
      this.selected = 0
    }
    if ('application' in query) {
      this.filter = query['application']
      this.selected = 1
    }
    this.search()
  },
  components: {
    AceEditor,
    Search
  }
}
</script>
