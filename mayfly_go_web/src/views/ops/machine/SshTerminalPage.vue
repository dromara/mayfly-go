<template>
    <div class="terminal-wrapper">
        <TerminalBody :socket-url="getMachineTerminalSocketUrl(route.query.ac)" />
    </div>
</template>

<script lang="ts" setup>
import { useRoute } from 'vue-router';
import TerminalBody from '@/components/terminal/TerminalBody.vue';
import { getMachineTerminalSocketUrl } from './api';

const route = useRoute();
</script>
<style lang="scss">
.terminal-wrapper {
    height: calc(100vh);
}
</style>
