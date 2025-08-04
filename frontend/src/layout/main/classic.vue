<template>
    <el-container class="layout-container flex-center">
        <Header />
        <el-container class="flex-1 overflow-auto">
            <Aside />
            <div class="flex-center layout-backtop">
                <TagsView v-if="themeConfig.isTagsview" />
                <Main />
            </div>
        </el-container>
    </el-container>
</template>

<script lang="ts" setup name="layoutClassic">
import { storeToRefs } from 'pinia';
import { useThemeConfig } from '@/store/themeConfig';
import { defineAsyncComponent, provide, ref } from 'vue';

const Aside = defineAsyncComponent(() => import('@/layout/component/aside.vue'));
const Header = defineAsyncComponent(() => import('@/layout/component/header.vue'));
const Main = defineAsyncComponent(() => import('@/layout/component/main.vue'));
const TagsView = defineAsyncComponent(() => import('@/layout/navBars/tagsView/tagsView.vue'));

const { themeConfig } = storeToRefs(useThemeConfig());

// 提供 classic 布局的菜单数据
const classicMenuData = ref<any>(null);
provide('classicMenuData', classicMenuData);
</script>
