<template>
    <el-container class="layout-container layout-backtop !flex-col">
        <Header />
        <Main />
    </el-container>
</template>

<script lang="ts" setup name="layoutTransverse">
import { defineAsyncComponent } from 'vue';

const Main = defineAsyncComponent(() => import('@/layout/component/main.vue'));
const Header = defineAsyncComponent(() => import('@/layout/component/header.vue'));
</script>
