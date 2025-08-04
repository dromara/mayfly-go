<template>
    <el-header class="layout-header">
        <NavBarsIndex />
    </el-header>
</template>

<script setup lang="ts" name="layoutHeader">
import { defineAsyncComponent } from 'vue';

const NavBarsIndex = defineAsyncComponent(() => import('@/layout/navBars/index.vue'));
</script>
