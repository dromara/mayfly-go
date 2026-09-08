<template>
    <div class="terminal-wrapper">
        <TerminalPanes :socket-url="getMachineTerminalSocketUrl((route.query.ac ?? '') as string)" />
    </div>
</template>

<script lang="ts" setup>
import { useRoute } from 'vue-router';
import TerminalPanes from '@/components/terminal/TerminalPanes.vue';
import { getMachineTerminalSocketUrl } from './api';

const route = useRoute();
</script>
<style lang="scss">
.terminal-wrapper {
    height: calc(100vh);
}
</style>
