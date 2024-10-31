<template>
    <el-container class="layout-container">
        <Aside />
        <el-container class="flex-center layout-backtop">
            <Header v-if="isFixedHeader" />
            <el-scrollbar ref="layoutDefaultsScrollbarRef">
                <Header v-if="!isFixedHeader" />
                <Main />
            </el-scrollbar>
        </el-container>
        <el-backtop target=".layout-backtop .el-scrollbar__wrap"></el-backtop>
    </el-container>
</template>

<script lang="ts" setup name="layoutDefaults">
import { computed, getCurrentInstance, watch } from 'vue';
import { useRoute } from 'vue-router';
import Aside from '@/layout/component/aside.vue';
import Header from '@/layout/component/header.vue';
import Main from '@/layout/component/main.vue';
import { useThemeConfig } from '@/store/themeConfig';

const { proxy } = getCurrentInstance() as any;
const route = useRoute();
const isFixedHeader = computed(() => {
    return useThemeConfig().themeConfig.isFixedHeader;
});
// 监听路由的变化
watch(
    () => route.path,
    () => {
        try {
            proxy.$refs.layoutScrollbarRef.wrapRef.scrollTop = 0;
        } catch (e) {}
    }
);
</script>
