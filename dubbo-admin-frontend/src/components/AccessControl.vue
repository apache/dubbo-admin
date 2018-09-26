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
      <v-flex xs12>
        <v-card flat
                color="transparent">
          <v-card-text>
            <v-layout row
                      wrap>
              <v-text-field label="Search Access Controls by service name"
                            v-model="filter"
                            clearable></v-text-field>
              <v-btn @click="submit"
                     color="primary"
                     large>Search</v-btn>
            </v-layout>
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

        <v-card-text class="pa-0">
          <v-data-table v-model="selected"
                        :headers="headers"
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
                          hint="A service ID in form of service"
                          required
                          :readonly="modal.id != null"
                          v-model="modal.service" />
            <v-subheader class="pa-0 mt-3">BLACK/WHITE LIST CONTENT</v-subheader>
            <ace-editor v-model="modal.content"
                        :config="modal.aceConfig" />
          </v-form>
        </v-card-text>
        <v-card-actions>
          <v-spacer></v-spacer>
          <v-btn color="darken-1"
                 flat
                 @click="closeModal()">Close</v-btn>
          <v-btn color="green darken-1"
                 flat
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
          <v-btn color="red darken-1"
                 flat
                 @click="confirm.enable = false">Disagree</v-btn>
          <v-btn color="green darken-1"
                 flat
                 @click="deleteItem(confirm.id)">Agree</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-snackbar v-model="snackbar.enable"
                :color="snackbar.color">
      {{ snackbar.text }}
      <v-btn dark
             flat
             @click="snackbar.enable = false">
        Close
      </v-btn>
    </v-snackbar>
  </v-container>
</template>

<script>
import yaml from 'js-yaml'
import { AXIOS } from './http-common'
import AceEditor from '@/components/AceEditor'

export default {
  name: 'AccessControl',
  data: () => ({
    selected: [],
    filter: '',
    loading: false,
    headers: [
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
    accesses: [],
    modal: {
      enable: false,
      title: 'Create New',
      saveBtn: 'Create',
      click: () => {},
      id: null,
      service: null,
      content: '',
      template:
        'blacklist:\n' +
        '  - 1.1.1.1\n' +
        '  - 2.2.2.2\n' +
        'whitelist:\n' +
        '  - 3.3.3.3\n' +
        '  - 4.4.*\n',
      aceConfig: {}
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
    submit () {
      if (this.filter == null) {
        this.filter = ''
      }
      this.search(this.filter)
    },
    search (filter) {
      this.loading = true
      this.$router.push({path: 'access', query: (filter !== '' ? {service: filter} : null)})
      AXIOS.post('/access/search', {
        service: filter
      }).then(response => {
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
      AXIOS.post('/access/create', {
        service: this.modal.service,
        whitelist: doc.whitelist,
        blacklist: doc.blacklist
      }).then(response => {
        this.closeModal()
        this.search(this.filter)
        this.showSnackbar('success', 'Create success')
      }).catch(error => this.showSnackbar('error', error.response.data.message))
    },
    toEdit (item) {
      Object.assign(this.modal, {
        enable: true,
        title: 'Edit',
        saveBtn: 'Update',
        click: this.editItem,
        id: item.id,
        service: item.service,
        content: yaml.safeDump({blacklist: item.blacklist, whitelist: item.whitelist})
      })
    },
    editItem () {
      let doc = yaml.load(this.modal.content)
      AXIOS.post('/access/update', {
        id: this.modal.id,
        whitelist: doc.whitelist,
        blacklist: doc.blacklist
      }).then(response => {
        this.closeModal()
        this.search(this.filter)
        this.showSnackbar('success', 'Update success')
      }).catch(error => this.showSnackbar('error', error.response.data.message))
    },
    toDelete (item) {
      Object.assign(this.confirm, {
        enable: true,
        title: 'Are you sure to Delete Access Control',
        text: `Service: ${item.service}`,
        id: item.id
      })
    },
    deleteItem (id) {
      AXIOS.post('/access/delete', {
        id: id
      }).then(response => {
        this.showSnackbar('success', 'Delete success')
        this.search(this.filter)
      }).catch(error => this.showSnackbar('error', error.response.data.message))
    },
    showSnackbar (color, message) {
      Object.assign(this.snackbar, {
        enable: true,
        color: color,
        text: message
      })
      this.confirm.enable = false
      this.selected = []
    }
  },
  mounted () {
    let query = this.$route.query
    let service = ''
    Object.keys(query).forEach(function (key) {
      if (key === 'service') {
        service = query[key]
      }
    })
    if (service !== '') {
      this.filter = service
      this.search(this.filter)
    }
  },
  components: {
    AceEditor
  }
}
</script>
