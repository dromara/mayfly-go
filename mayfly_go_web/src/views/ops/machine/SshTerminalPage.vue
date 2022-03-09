<template>
    <div>
        <ssh-terminal ref="terminal" :machineId="machineId" :height="height + 'px'" />
    </div>
</template>

<script lang="ts">
import SshTerminal from './SshTerminal.vue';
import { reactive, toRefs, defineComponent, onMounted } from 'vue';
import { useRoute } from 'vue-router';

export default defineComponent({
    name: 'SshTerminalPage',
    components: {
        SshTerminal,
    },
    props: {
        machineId: { type: Number },
    },
    setup() {
        const route = useRoute();
        const state = reactive({
            machineId: 0,
            height: 700,
        });

        onMounted(() => {
            state.height = window.innerHeight + 5;
            state.machineId = Number.parseInt(route.query.id as string);
        });

        return {
            ...toRefs(state),
        };
    },
});
</script>
<style lang="scss">
</style>
