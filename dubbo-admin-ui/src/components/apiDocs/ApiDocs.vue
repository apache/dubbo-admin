<!--
  - Licensed to the Apache Software Foundation (ASF) under one or more
  - contributor license agreements.  See the NOTICE file distributed with
  - this work for additional information regarding copyright ownership.
  - The ASF licenses this file to You under the Apache License, Version 2.0
  - (the "License"); you may not use this file except in compliance with
  - the License.  You may obtain a copy of the License at
  -
  -   http://www.apache.org/licenses/LICENSE-2.0
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
    <breadcrumb title="apiDocs" :items="breads"></breadcrumb>
    </v-flex>
    <v-flex lg12>
    <v-card flat color="transparent">
      <v-card-text>
      <v-form>
        <v-layout row wrap>
        <v-text-field
          id="dubboProviderIP"
          :label="$t('apiDocsRes.dubboProviderIP')"
          :rules="rules"
          placeholder="127.0.0.1"
          value="127.0.0.1"
          outline
        ></v-text-field>
        <v-text-field style="marginLeft: 10px;"
          id="dubboProviderPort"
          :label="$t('apiDocsRes.dubboProviderPort')"
          :rules="rules"
          placeholder="20880"
          value="20881"
          outline
        ></v-text-field>
        <v-btn @click="submit" color="primary" large>{{ $t('apiDocsRes.loadApiList') }}</v-btn>
        </v-layout>
      </v-form>
      </v-card-text>
    </v-card>
    </v-flex>
  </v-layout>

  <v-layout row wrap>
    <v-flex lg3>
      <v-card id="apiListDiv"
        class="mx-auto"
      >
        <v-toolbar>
          <v-toolbar-side-icon></v-toolbar-side-icon>
          <v-toolbar-title>{{ $t('apiDocsRes.apiListText') }}</v-toolbar-title>
          <v-spacer></v-spacer>
        </v-toolbar>
        <v-list>
          <v-list-group
            v-for="item in apiModules"
            :key="item.title"
            no-action
          >
            <template v-slot:activator>
            <v-list-tile>
              <v-list-tile-content>
              <v-list-tile-title>{{ item.title }}</v-list-tile-title>
              </v-list-tile-content>
            </v-list-tile>
            </template>

            <v-list-tile
            class="apiListListTile"
            v-for="child in item.apis"
            :key="child.title"
            @click="showApiForm(child.formInfo, $event)"
            >
            <v-list-tile-content>
              <v-list-tile-title>{{ child.title }}</v-list-tile-title>
            </v-list-tile-content>
            </v-list-tile>
          </v-list-group>
        </v-list>
      </v-card>
    </v-flex>
    <v-flex lg9>
      <v-card id="apiFormDiv">
        <apiForm :formInfo="formInfo" />
      </v-card>
    </v-flex>
  </v-layout>
  </v-container>
</template>

<script>
import Breadcrumb from '@/components/public/Breadcrumb'
import ApiForm from '@/components/apiDocs/ApiForm'
export default {
  name: 'ApiDocs',
  components: {
    Breadcrumb,
    ApiForm
  },
  data: () => ({
    breads: [
      {
        text: 'apiDocs',
        href: '/apiDocs'
      }
    ],
    rules: [
      value => !!value || 'Required.'
    ],
    apiModules: [],
    formInfo: {},
    isApiListDivFixed: false
  }),
  methods: {
    submit () {
      const dubboProviderIP = document.querySelector('#dubboProviderIP').value.trim()
      const dubboProviderPort = document.querySelector('#dubboProviderPort').value.trim()
      this.$axios.get('/docs/apiModuleList', {
        params: {
          dubboIp: dubboProviderIP,
          dubboPort: dubboProviderPort
        }
      }).then(response => {
        const resultData = []
        if (response && response.data && response.data !== '') {
          const menuData = JSON.parse(response.data)
          menuData.sort((a, b) => {
            return a.moduleDocName > b.moduleDocName
          })
          for (let i = 0; i < menuData.length; i++) {
            const menu = menuData[i]
            menu.moduleApiList.sort((a, b) => {
              return a.apiName > b.apiName
            })
            const menu2 = {
              title: menu.moduleDocName,
              apis: []
            }
            const menuItems = menu.moduleApiList
            for (let j = 0; j < menuItems.length; j++) {
              const menuItem = menuItems[j]
              const menuItem2 = {
                title: menuItem.apiDocName,
                formInfo: {
                  moduleClassName: menu.moduleClassName,
                  dubboIp: dubboProviderIP,
                  dubboPort: dubboProviderPort,
                  apiName: menuItem.apiName,
                  apiRespDec: menuItem.apiRespDec,
                  apiDocName: menuItem.apiDocName,
                  description: menuItem.description,
                  apiVersion: menuItem.apiVersion
                }
              }
              menu2.apis.push(menuItem2)
            }
            resultData.push(menu2)
          }
        }
        this.apiModules = resultData
      }).catch(error => {
        console.log('error', error.message)
      })
    },
    showApiForm (formInfo, e) {
      this.formInfo = formInfo
      const apiListListTileList = document.getElementsByClassName('apiListListTile')
      for (var i = 0; i < apiListListTileList.length; i++) {
        apiListListTileList[i].childNodes.forEach(function (curr, index, arr) {
          curr.classList.remove('primary--text')
        })
      }
      e.currentTarget.classList.add('primary--text')
    },
    fixedApiListDiv () {
      var scrollTop = document.documentElement.scrollTop || document.body.scrollTop
      var apiListDivTop = document.getElementById('apiFormDiv').offsetTop
      var apiListDivWidth = document.getElementById('apiListDiv').offsetWidth
      if (!this.isApiListDivFixed && scrollTop >= apiListDivTop) {
        this.isApiListDivFixed = true
        document.getElementById('apiListDiv').classList.add('apiListDiv-fixed')
        document.getElementById('apiListDiv').style.top = '75px'
        document.getElementById('apiListDiv').style.width = apiListDivWidth + 'px'
      }
      if (this.isApiListDivFixed && scrollTop <= apiListDivTop) {
        this.isApiListDivFixed = false
        document.getElementById('apiListDiv').classList.remove('apiListDiv-fixed')
        document.getElementById('apiListDiv').style.top = '0px'
      }
    }
  },
  mounted () {
    window.addEventListener('scroll', this.fixedApiListDiv)
  }
}
</script>
<style scoped>

  .apiListDiv-fixed{
    position: fixed;
  }

</style>
