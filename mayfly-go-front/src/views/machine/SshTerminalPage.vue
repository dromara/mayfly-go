<template>
  <div class="file-manage">
    <ssh-terminal ref="terminal" :machineId="machineId" :height="height+'px'" />
  </div>
</template>

<script lang="ts">
import { Component, Vue, Prop, Watch } from 'vue-property-decorator'
import SshTerminal from './SshTerminal.vue'

@Component({
  name: 'ServiceManage',
  components: {
    SshTerminal,
  },
})
export default class ServiceManage extends Vue {
  machineId = ''
  height = 700

  created() {
    this.height = window.innerHeight
    this.machineId = this.$route.params.id
  }
}
</script>
<style lang="less">
</style>
