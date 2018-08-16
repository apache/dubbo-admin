<template>
  <v-toolbar fixed app >
    <v-toolbar-side-icon @click.stop="handleDrawerToggle"></v-toolbar-side-icon>
    <v-text-field
      flat
      solo-inverted
      prepend-icon="search"
      label="Search"
      class="hidden-sm-and-down"
    >
    </v-text-field>
    <v-spacer></v-spacer>
    <v-spacer></v-spacer>
    <v-spacer></v-spacer>
    <v-switch
    ></v-switch>
  </v-toolbar>
</template>
<script>
  export default {
    name: 'toolbar',
    methods: {
      handleDrawerToggle () {
        window.getApp.$emit('DRAWER_TOGGLED')
      }
    }
  }
</script>



