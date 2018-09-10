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
  <v-app :dark="dark">
    <drawer></drawer>
    <toolbar></toolbar>
    <v-content>
      <router-view/>
    </v-content>
    <v-footer height="auto" class="white pa-3 app--footer">
      <span class="caption">Copyright &copy; {{ new Date().getFullYear() }} The Apache Software Foundation. </span>
      <v-spacer></v-spacer>
      <span class="caption mr-1"> Make With Love </span>
      <v-icon color="pink" small>favorite</v-icon>
    </v-footer>
  </v-app>
</template>
<script>
  import Drawer from '@/components/Drawer'
  import Toolbar from '@/components/Toolbar'
  import Footers from '@/components/Footers'

  export default {
    components: {
      Drawer,
      Toolbar,
      Footers
    },
    data () {
      return {
        clipped: false,
        fixed: false,
        dark: false,
        items: [{
          icon: 'bubble_chart',
          title: 'Inspire'
        }],
        miniVariant: false,
        right: true
      }
    },
    created () {
      window.getApp = this
      window.getApp.$on('CHANGE_THEME', () => {
        this.dark = (!this.dark)
      })
    },
    computed: {
      appTitle () {
        return this.$store.state.appTitle
      }
    },
    name: 'App'
  }
</script>


<style lang="stylus" scoped>
  .setting-fab
    top: 50% !important;
    right: 0;
    border-radius: 0

  .page-wrapper
    min-height: calc(100vh - 64px - 50px - 81px);

</style>
