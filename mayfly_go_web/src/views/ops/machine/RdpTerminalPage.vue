<template>
    <div class="terminal-wrapper" ref="terminalWrapperRef">
        <machine-rdp ref="rdpRef" :machine-id="route.query.id" />
    </div>
</template>

<script lang="ts" setup>
import { useRoute } from 'vue-router';
import MachineRdp from '@/components/terminal-rdp/MachineRdp.vue';
import { onMounted, ref } from 'vue';
import { TerminalExpose } from '@/components/terminal-rdp';
const route = useRoute();

const rdpRef = ref({} as TerminalExpose);
const terminalWrapperRef = ref({} as any);

onMounted(() => {
    let width = terminalWrapperRef.value.clientWidth;
    let height = terminalWrapperRef.value.clientHeight;
    console.log(width, height);
    rdpRef.value?.init(width, height, false);
});
</script>
<style lang="scss">
.terminal-wrapper {
    height: calc(100vh);
}
</style>
