<!--
  - Licensed to the Apache Software Foundation (ASF) under one or more
  - contributor license agreements.  See the NOTICE file distributed with
  - this work for additional information regarding copyright ownership.
  - The ASF licenses this file to You under the Apache License, Version 2.0
  - (the "License"); you may not use this file except in compliance with
  -  he License.  You may obtain a copy of the License at
  -
  -      http://www.apache.org/licenses/LICENSE-2.0
  -
  -  Unless required by applicable law or agreed to in writing, software
  -  distributed under the License is distributed on an "AS IS" BASIS,
  -  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  -  See the License for the specific language governing permissions and
  -  limitations under the License.
  -->

<template>
  <v-container grid-list-xl fluid>
    <v-layout row wrap>
      <v-flex xs12>
        <v-card flat color="transparent">
          <v-card-text>
            <v-layout row wrap>
              <v-text-field label="Search dubbo service" v-model="filter" clearable></v-text-field>
              <v-btn @click="submit" color="primary" large>Search</v-btn>
            </v-layout>
          </v-card-text>
        </v-card>
      </v-flex>
    </v-layout>

    <v-flex lg12>
      <v-card>
        <v-toolbar flat color="transparent" class="elevation-0">
          <v-toolbar-title>
            <span class="headline">Search Result</span>
          </v-toolbar-title>
          <v-divider class="mx-2" inset vertical></v-divider>
          <v-spacer></v-spacer>
          <v-btn :disabled="selected.length == 0" outline color="error" @click.stop="toDelete(selected)" class="mb-2">BATCH DELETE</v-btn>
          <v-btn outline color="primary" @click.stop="openDialog" class="mb-2">CREATE</v-btn>
        </v-toolbar>

        <v-card-text class="pa-0">
          <v-data-table v-model="selected" :headers="headers" :items="accesses" hide-actions select-all class="elevation-0">
            <template slot="items" slot-scope="props">
              <td>
                <v-checkbox v-model="props.selected" primary hide-details></v-checkbox>
              </td>
              <td class="text-xs-left">{{ props.item.address }}</td>
              <td class="text-xs-left">{{ props.item.service }}</td>
              <td class="text-xs-left">
                <span v-if="props.item.allow" class="green--text">Whitelist</span>
                <span v-else class="red--text">Blacklist</span>
              </td>
              <td class="text-xs-center px-0">
                <v-tooltip bottom>
                  <v-icon small class="mr-2" slot="activator" @click="toDelete([props.item])">delete</v-icon>
                  <span>Delete</span>
                </v-tooltip>
              </td>
            </template>
          </v-data-table>
        </v-card-text>
      </v-card>
    </v-flex>

    <v-dialog v-model="create.enable" width="800px" persistent>
      <v-card>
        <v-card-title class="justify-center">
          <span class="headline">Create New Access Control</span>
        </v-card-title>
        <v-card-text>
        </v-card-text>
        <v-card-actions>
          <v-spacer></v-spacer>
          <v-btn color="darken-1" flat @click="create.enable = false">Close</v-btn>
          <v-btn color="green darken-1" flat @click="saveItem()">Save</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-dialog v-model="confirm.enable" persistent max-width="500px">
      <v-card>
        <v-card-title class="headline">{{this.confirm.title}}</v-card-title>
        <v-card-text>{{this.confirm.text}}</v-card-text>
        <v-card-actions>
          <v-spacer></v-spacer>
          <v-btn color="red darken-1" flat @click="confirm.enable = false">Disagree</v-btn>
          <v-btn color="green darken-1" flat @click="deleteItems(confirm.accesses)">Agree</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-snackbar v-model="snackbar.enable" :color="snackbar.color">
      {{ snackbar.text }}
      <v-btn dark flat @click="snackbar.enable = false">
        Close
      </v-btn>
    </v-snackbar>
  </v-container>
</template>

<script>
import { AXIOS } from './http-common'

export default {
  name: 'AccessControl',
  data: () => ({
    selected: [],
    filter: '',
    headers: [
      {
        text: 'Consumer Address',
        value: 'address',
        align: 'left'
      },
      {
        text: 'Service Name',
        value: 'service',
        align: 'left'
      },
      {
        text: 'Type',
        value: 'allow',
        sortable: false
      },
      {
        text: 'Operation',
        value: 'operation',
        sortable: false
      }
    ],
    accesses: [],
    create: {
      enable: false
    },
    confirm: {
      enable: false,
      title: '',
      text: '',
      accesses: []
    },
    snackbar: {
      enable: false,
      text: ''
    }
  }),
  methods: {
    submit () {
      this.search(this.filter)
    },
    search (filter) {
      AXIOS.get('/accesses/search?service=' + filter)
        .then(response => {
          this.accesses = response.data
        })
    },
    openDialog () {
      this.create.enable = true
    },
    saveItem () {

    },
    toDelete (items) {
      let text = items.length === 1
        ? ('Address: ' + items[0].address + ' Service: ' + items[0].service)
        : ('Delete ' + items.length + ' Access Controls')
      Object.assign(this.confirm, {
        enable: true,
        title: 'Are you sure to Delete Access Control(s)',
        text: text,
        accesses: items
      })
    },
    deleteItems (accesses) {
      AXIOS.post('/accesses/delete', accesses)
        .then(response => {
          this.showSnackbar('success', 'Delete success')
          this.search(this.filter)
        })
        .catch(error => this.showSnackbar('error', error.response.data.message))
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
    this.search(this.filter)
  }
}
</script>
