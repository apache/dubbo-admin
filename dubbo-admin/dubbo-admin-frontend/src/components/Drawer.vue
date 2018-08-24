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
  <v-navigation-drawer
    v-model="drawer"
    fixed
    app
    dark
  >
    <v-toolbar
    color="primary">
      <img src="@/assets/logo.png" width="24" height="24"/>
      <v-toolbar-title class="ml-0 pl-3">
        <span class="hidden-sm-and-down">Dubbo Admin</span>
      </v-toolbar-title>
    </v-toolbar>
    <v-list>
      <v-list-tile href="/#/service">
        <v-list-tile-action>
          <v-icon>search</v-icon>
        </v-list-tile-action>
        <v-list-tile-title >Service Search</v-list-tile-title>
      </v-list-tile>

      <v-list-group
        prepend-icon="edit"
        value="true"
        no-action
      >
        <v-list-tile slot="activator">
          <v-list-tile-title>Service Governance</v-list-tile-title>
        </v-list-tile>
        <v-list-tile href="/#/routingRule">
          <v-list-tile-title>Routing Rule</v-list-tile-title>
        </v-list-tile>
        <v-list-tile >
          <v-list-tile-title>Dynamic Config</v-list-tile-title>
        </v-list-tile>
        <v-list-tile >
          <v-list-tile-title>Access Control</v-list-tile-title>
        </v-list-tile>
        <v-list-tile >
          <v-list-tile-title>Weight Adjust</v-list-tile-title>
        </v-list-tile>
        <v-list-tile >
          <v-list-tile-title>Load Balance</v-list-tile-title>
        </v-list-tile>
      </v-list-group>

      <v-list-tile>
        <v-list-tile-action>
          <v-icon>computer</v-icon>
        </v-list-tile-action>
        <v-list-tile-title>QoS</v-list-tile-title>
      </v-list-tile>

      <v-list-group
        prepend-icon="info"
        value="true"
        no-action
      >
        <v-list-tile slot="activator">
          <v-list-tile-title>Service Info</v-list-tile-title>
        </v-list-tile>
        <v-list-tile >
          <v-list-tile-title>Version</v-list-tile-title>
        </v-list-tile>
      </v-list-group>
    </v-list>
  </v-navigation-drawer>
</template>
<script>
  export default {
    name: 'drawer',
    data: () => ({
      drawer: true
    }),
    created () {
      window.getApp.$on('DRAWER_TOGGLED', () => {
        this.drawer = (!this.drawer)
      })
    }
  }
</script>
