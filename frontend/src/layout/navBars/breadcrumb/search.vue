<template>
    <div class="layout-search-dialog">
        <el-dialog v-model="state.isShowSearch" width="300px" destroy-on-close :modal="false" fullscreen :show-close="false">
            <el-autocomplete
                v-model="state.menuQuery"
                :fetch-suggestions="menuSearch"
                placeholder="菜单搜索"
                prefix-icon="el-icon-search"
                ref="layoutMenuAutocompleteRef"
                @select="onHandleSelect"
                @blur="onSearchBlur"
            >
                <template #prefix>
                    <el-icon class="el-input__icon">
                        <search />
                    </el-icon>
                </template>
                <template #default="{ item }">
                    <div><SvgIcon :name="item.meta.icon" class="mr5" />{{ item.meta.title }}</div>
                </template>
            </el-autocomplete>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup name="layoutBreadcrumbSearch">
import { reactive, ref, nextTick } from 'vue';
import { useRouter } from 'vue-router';
import { useRoutesList } from '@/store/routesList';

const layoutMenuAutocompleteRef: any = ref(null);
const router = useRouter();
const state: any = reactive({
    isShowSearch: false,
    menuQuery: '',
    tagsViewList: [],
});
// 搜索弹窗打开
const openSearch = () => {
    state.menuQuery = '';
    state.isShowSearch = true;
    initTageView();
    nextTick(() => {
        setTimeout(() => {
            layoutMenuAutocompleteRef.value.focus();
        });
    });
};
// 搜索弹窗关闭
const closeSearch = () => {
    state.isShowSearch = false;
};
// 菜单搜索数据过滤
const menuSearch = (queryString: any, cb: any) => {
    let results = queryString ? state.tagsViewList.filter(createFilter(queryString)) : state.tagsViewList;
    cb(results);
};
// 菜单搜索过滤
const createFilter = (queryString: any) => {
    return (restaurant: any) => {
        return (
            restaurant.path.toLowerCase().indexOf(queryString.toLowerCase()) > -1 || restaurant.meta.title.toLowerCase().indexOf(queryString.toLowerCase()) > -1
        );
    };
};
// 初始化菜单数据
const initTageView = () => {
    if (state.tagsViewList.length > 0) return false;
    getRoutes(useRoutesList().routesList).map((v: any) => {
        if (!v.meta.isHide) {
            state.tagsViewList.push({ ...v });
        }
    });
};
// 获取所有根节点的route，即可访问的route
const getRoutes = (routes: any) => {
    const menu: any = [];
    for (let i = 0; i < routes.length; i++) {
        const item = { ...routes[i] };
        if (item.children) {
            getRoutes(item.children).forEach((r: any) => {
                menu.push(r);
            });
            continue;
        }
        menu.push(item);
    }
    return menu;
};

// 当前菜单选中时
const onHandleSelect = (item: any) => {
    let { path, redirect } = item;
    if (item.meta.link && item.meta.linkType == 2) window.open(item.meta.link);
    else if (redirect) router.push(redirect);
    else router.push(path);
    closeSearch();
};
// input 失去焦点时
const onSearchBlur = () => {
    closeSearch();
};

defineExpose({ openSearch });
</script>

<style scoped lang="scss">
.layout-search-dialog {
    ::v-deep(.el-dialog) {
        box-shadow: unset !important;
        border-radius: 0 !important;
        background: rgba(0, 0, 0, 0.5);
    }

    ::v-deep(.el-autocomplete) {
        width: 560px;
        position: absolute;
        top: 100px;
        left: 50%;
        transform: translateX(-50%);
    }
}
</style>
