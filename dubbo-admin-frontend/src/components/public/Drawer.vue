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
  <v-navigation-drawer
    id="appDrawer"
    :mini-variant.sync="mini"
    fixed
    :dark="$vuetify.dark"
    app
    v-model="drawer"
  >

    <v-toolbar color="primary darken-1" dark>
      <img src="@/assets/logo.png" width="24" height="24"/>
      <v-toolbar-title class="ml-0 pl-3">
        <span class="hidden-sm-and-down white--text">Dubbo Admin</span>
      </v-toolbar-title>
    </v-toolbar>

    <v-list expand>
      <template v-for="(item, i) in menus">
        <v-list-group v-if="item.items" :group="item.group" :prepend-icon="item.icon" no-action>
          <v-list-tile slot="activator" ripple>
            <v-list-tile-content>
              <v-list-tile-title>{{ item.title }}</v-list-tile-title>
            </v-list-tile-content>
          </v-list-tile>

          <template v-for="(subItem, i) in item.items">
            <v-list-tile :to="subItem.path" ripple>
              <v-list-tile-content>
                <v-list-tile-title>{{ subItem.title }}</v-list-tile-title>
              </v-list-tile-content>
            </v-list-tile>
          </template>
        </v-list-group>

        <v-list-tile v-else :key="item.title" :to="item.path" ripple>
          <v-list-tile-action>
            <v-icon>{{ item.icon }}</v-icon>
          </v-list-tile-action>
          <v-list-tile-content>{{ item.title }}</v-list-tile-content>
        </v-list-tile>
      </template>
    </v-list>
  </v-navigation-drawer>
</template>

<script>
  import menu from '@/api/menu'

  export default {
    name: 'drawer',
    data: () => ({
      mini: false,
      drawer: true,
      menus: menu
    }),
    created () {
      window.getApp.$on('DRAWER_TOGGLED', () => {
        this.drawer = (!this.drawer)
      })
    },
    computed: {
      sideToolbarColor () {
        return this.$vuetify.options.extra.sideNav
      }
    }
  }
</script>
