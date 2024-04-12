<template>
    <div class="terminal-wrapper" ref="terminalWrapperRef">
        <machine-rdp ref="rdpRef" :auth-cert="state.authCert" :machine-id="state.machineId" />
    </div>
</template>

<script lang="ts" setup>
import { useRoute } from 'vue-router';
import MachineRdp from '@/components/terminal-rdp/MachineRdp.vue';
import { computed, onMounted, ref } from 'vue';
import { TerminalExpose } from '@/components/terminal-rdp';
const route = useRoute();

const rdpRef = ref({} as TerminalExpose);
const terminalWrapperRef = ref({} as any);

const state = computed(() => {
    return {
        authCert: route.query.ac as string,
        machineId: Number(route.query.machineId),
    };
});

onMounted(() => {
    let width = terminalWrapperRef.value.clientWidth;
    let height = terminalWrapperRef.value.clientHeight;
    rdpRef.value?.init(width, height, false);
});
</script>
<style lang="scss">
.terminal-wrapper {
    height: calc(100vh);
}
</style>
