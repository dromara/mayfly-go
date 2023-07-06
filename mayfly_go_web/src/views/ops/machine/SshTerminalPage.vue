<template>
    <div>
        <ssh-terminal ref="terminal" :machineId="machineId" :height="height + 'px'" />
    </div>
</template>

<script lang="ts" setup>
import SshTerminal from './SshTerminal.vue';
import { reactive, toRefs, onMounted } from 'vue';
import { useRoute } from 'vue-router';

const route = useRoute();
const state = reactive({
    machineId: 0,
    height: 0,
});

const { machineId, height } = toRefs(state);

onMounted(() => {
    state.height = window.innerHeight + 5;
    state.machineId = Number.parseInt(route.query.id as string);
});
</script>
<style lang="scss"></style>
