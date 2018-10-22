<template>
  <v-snackbar v-model="show"
              :color="color">
    {{ text }}
    <v-btn dark
           flat
           @click="show = false">
      Close
    </v-btn>
  </v-snackbar>
</template>
<script>
export default {
  data () {
    return {
      show: false,
      color: '',
      text: ''
    }
  }
}
</script>
