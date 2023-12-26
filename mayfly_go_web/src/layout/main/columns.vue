<template>
    <el-container class="layout-container">
        <ColumnsAside />
        <div class="layout-columns-warp">
            <Aside />
            <el-container class="flex-center layout-backtop">
                <Header v-if="isFixedHeader" />
                <el-scrollbar>
                    <Header v-if="!isFixedHeader" />
                    <Main />
                </el-scrollbar>
            </el-container>
        </div>
        <el-backtop target=".layout-backtop .el-scrollbar__wrap"></el-backtop>
    </el-container>
</template>

<script lang="ts" setup name="layoutColumns">
import { computed } from 'vue';
import Aside from '@/layout/component/aside.vue';
import Header from '@/layout/component/header.vue';
import Main from '@/layout/component/main.vue';
import ColumnsAside from '@/layout/component/columnsAside.vue';
import { useThemeConfig } from '@/store/themeConfig';

const isFixedHeader = computed(() => {
    return useThemeConfig().themeConfig.isFixedHeader;
});
</script>
