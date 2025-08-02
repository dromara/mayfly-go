<template>
    <div class="el-menu-horizontal-warp">
        <el-menu
            router
            :default-active="state.defaultActive"
            background-color="transparent"
            mode="horizontal"
            @select="onHorizontalSelect"
            class="horizontal-menu"
        >
            <template v-for="val in menuLists">
                <el-sub-menu :index="val.path" v-if="val.children && val.children.length > 0" :key="val.path">
                    <template #title>
                        <SvgIcon :name="val.meta.icon" />
                        <span>{{ $t(val.meta.title) }}</span>
                    </template>
                    <SubItem :chil="val.children" />
                </el-sub-menu>
                <el-menu-item :index="val.path" :key="val?.path" v-else>
                    <template #title v-if="!val.meta.link || (val.meta.link && val.meta.linkType == 1)">
                        <SvgIcon :name="val.meta.icon" />
                        {{ $t(val.meta.title) }}
                    </template>
                    <template #title v-else>
                        <a class="w-full" :href="val.meta.link" target="_blank">
                            <SvgIcon :name="val.meta.icon" />
                            {{ $t(val.meta.title) }}
                        </a>
                    </template>
                </el-menu-item>
            </template>
        </el-menu>
    </div>
</template>

<script lang="ts" setup name="navMenuHorizontal">
import { reactive, computed, onMounted, inject } from 'vue';
import { useRoute, onBeforeRouteUpdate } from 'vue-router';
import SubItem from '@/layout/navMenu/subItem.vue';
import { useRoutesList } from '@/store/routesList';
import { useThemeConfig } from '@/store/themeConfig';

// 定义父组件传过来的值
const props = defineProps({
    // 菜单列表
    menuList: {
        type: Array<any>,
        default: () => [],
    },
});

const route = useRoute();
const state: any = reactive({
    defaultActive: null,
});
// 注入 classicMenuData
const classicMenuData: any = inject('classicMenuData', null);

// 获取父级菜单数据
const menuLists = computed(() => {
    return props.menuList;
});

// 设置页面当前路由高亮
const setCurrentRouterHighlight = (path: string) => {
    const currentPathSplit = path.split('/');
    if (useThemeConfig().themeConfig.layout === 'classic') {
        state.defaultActive = `/${currentPathSplit[1]}`;
    } else {
        state.defaultActive = path;
    }
};
// 路由过滤递归函数
const filterRoutesFun = (arr: Array<object>) => {
    return arr
        .filter((item: any) => !item.meta.isHide)
        .map((item: any) => {
            item = Object.assign({}, item);
            if (item.children) item.children = filterRoutesFun(item.children);
            return item;
        });
};
// 传送当前子级数据到菜单中
const setSendClassicChildren = (path: string) => {
    const currentPathSplit = path.split('/');
    let currentData: any = {};
    filterRoutesFun(useRoutesList().routesList).map((v, k) => {
        if (v.path === `/${currentPathSplit[1]}`) {
            v['k'] = k;
            currentData['item'] = [{ ...v }];
            currentData['children'] = [{ ...v }];
            if (v.children) currentData['children'] = v.children;
        }
    });
    return currentData;
};
// 菜单激活回调
const onHorizontalSelect = (path: string) => {
    if (classicMenuData) {
        classicMenuData.value = setSendClassicChildren(path);
    }
};
// 页面加载时
onMounted(() => {
    setCurrentRouterHighlight(route.path);
});
// 路由更新时
onBeforeRouteUpdate((to) => {
    setCurrentRouterHighlight(to.path);
});
</script>

<style scoped lang="scss">
.el-menu-horizontal-warp {
    flex: 1;
    overflow: hidden;
    margin-right: 30px;

    .horizontal-menu {
        border: none !important;
        height: 100%;
        width: 100%;
        box-sizing: border-box;

        ::v-deep(.el-menu-item) {
            height: 42px;
            line-height: 42px;
            padding: 0 15px !important;
            margin: 0 5px;
            border-radius: 6px;
            display: flex;
            align-items: center;
        }

        ::v-deep(.el-sub-menu__title) {
            height: 42px;
            line-height: 42px;
            padding: 0 25px 0 15px !important; /* 右边留出更多空间给箭头图标 */
            margin: 0 5px;
            border-radius: 6px;
            display: flex;
            align-items: center;
        }

        ::v-deep(.el-sub-menu__icon-arrow) {
            right: 5px !important;
            margin-top: -5px !important;
        }

        ::v-deep(.el-menu-item.is-active),
        ::v-deep(.el-sub-menu.is-active .el-sub-menu__title) {
            color: #409eff;
            background-color: rgba(64, 158, 255, 0.1);
        }
    }
}
</style>
