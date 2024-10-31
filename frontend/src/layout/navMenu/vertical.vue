<template>
    <el-menu
        router
        :default-active="state.defaultActive"
        background-color="transparent"
        :collapse="setIsCollapse"
        :unique-opened="themeConfig.isUniqueOpened"
        :collapse-transition="false"
    >
        <template v-for="val in menuLists">
            <el-sub-menu :index="val.path" v-if="val.children && val.children.length > 0" :key="val.path">
                <template #title>
                    <SvgIcon :name="val.meta.icon" />
                    <span>{{ val.meta.title }}</span>
                </template>
                <SubItem :chil="val.children" />
            </el-sub-menu>
            <el-menu-item :index="val.path" :key="val?.path" v-else>
                <SvgIcon :name="val.meta.icon" />
                <template #title v-if="!val.meta.link || (val.meta.link && val.meta.linkType == 1)">
                    <span>{{ val.meta.title }}</span>
                </template>
                <template #title v-else>
                    <a :href="val.meta.link" target="_blank">{{ val.meta.title }}</a></template
                >
            </el-menu-item>
        </template>
    </el-menu>
</template>

<script lang="ts" setup name="navMenuVertical">
import { reactive, computed } from 'vue';
import { storeToRefs } from 'pinia';
import { useThemeConfig } from '@/store/themeConfig';
import { useRoute, onBeforeRouteUpdate } from 'vue-router';
import SubItem from '@/layout/navMenu/subItem.vue';
import mittBus from '@/common/utils/mitt';

// 定义父组件传过来的值
const props = defineProps({
    // 菜单列表
    menuList: {
        type: Array<any>,
        default: () => [],
    },
});

const { themeConfig } = storeToRefs(useThemeConfig());
const route = useRoute();
const state = reactive({
    defaultActive: route.path,
});
// 获取父级菜单数据
const menuLists = computed(() => {
    return props.menuList;
});
// 设置菜单的收起/展开
const setIsCollapse = computed(() => {
    return document.body.clientWidth < 1000 ? false : themeConfig.value.isCollapse;
});
// 路由更新时
onBeforeRouteUpdate((to) => {
    state.defaultActive = to.path;
    mittBus.emit('onMenuClick');
    const clientWidth = document.body.clientWidth;
    if (clientWidth < 1000) themeConfig.value.isCollapse = false;
});
</script>
