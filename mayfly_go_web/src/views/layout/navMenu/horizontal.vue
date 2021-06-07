<template>
    <div class="el-menu-horizontal-warp">
        <el-scrollbar @wheel.prevent="onElMenuHorizontalScroll" ref="elMenuHorizontalScrollRef">
            <el-menu router :default-active="defaultActive" background-color="transparent" mode="horizontal" @select="onHorizontalSelect">
                <template v-for="val in menuLists">
                    <el-submenu :index="val.path" v-if="val.children && val.children.length > 0" :key="val.path">
                        <template #title>
                            <i :class="val.meta.icon ? val.meta.icon : ''"></i>
                            <span>{{ val.meta.title }}</span>
                        </template>
                        <SubItem :chil="val.children" />
                    </el-submenu>
                    <el-menu-item :index="val.path" :key="val.path" v-else>
                        <template #title v-if="!val.meta.link || (val.meta.link && val.meta.isIframe)">
                            <i :class="val.meta.icon ? val.meta.icon : ''"></i>
                            {{ val.meta.title }}
                        </template>
                        <template #title v-else>
                            <a :href="val.meta.link" target="_blank">
                                <i :class="val.meta.icon ? val.meta.icon : ''"></i>
                                {{ val.meta.title }}
                            </a>
                        </template>
                    </el-menu-item>
                </template>
            </el-menu>
        </el-scrollbar>
    </div>
</template>

<script lang="ts">
import { toRefs, reactive, computed, defineComponent, getCurrentInstance, onMounted, nextTick } from 'vue';
import { useRoute, onBeforeRouteUpdate } from 'vue-router';
import { useStore } from '@/store/index.ts';
import SubItem from '@/views/layout/navMenu/subItem.vue';
export default defineComponent({
    name: 'navMenuHorizontal',
    components: { SubItem },
    props: {
        menuList: {
            type: Array,
            default: () => [],
        },
    },
    setup(props) {
        const { proxy } = getCurrentInstance() as any;
        const route = useRoute();
        const store = useStore();
        const state: any = reactive({
            defaultActive: null,
        });
        // 获取父级菜单数据
        const menuLists = computed(() => {
            return props.menuList;
        });
        // 设置横向滚动条可以鼠标滚轮滚动
        const onElMenuHorizontalScroll = (e: any) => {
            const eventDelta = e.wheelDelta || -e.deltaY * 40;
            proxy.$refs.elMenuHorizontalScrollRef.$refs.wrap.scrollLeft =
                proxy.$refs.elMenuHorizontalScrollRef.$refs.wrap.scrollLeft + eventDelta / 4;
        };
        // 初始化数据，页面刷新时，滚动条滚动到对应位置
        const initElMenuOffsetLeft = () => {
            nextTick(() => {
                let els: any = document.querySelector('.el-menu.el-menu--horizontal li.is-active');
                if (!els) return false;
                proxy.$refs.elMenuHorizontalScrollRef.$refs.wrap.scrollLeft = els.offsetLeft;
            });
        };
        // 设置页面当前路由高亮
        const setCurrentRouterHighlight = (path: string) => {
            const currentPathSplit = path.split('/');
            if (store.state.themeConfig.themeConfig.layout === 'classic') {
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
            filterRoutesFun(store.state.routesList.routesList).map((v, k) => {
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
            proxy.mittBus.emit('setSendClassicChildren', setSendClassicChildren(path));
        };
        // 页面加载时
        onMounted(() => {
            initElMenuOffsetLeft();
            setCurrentRouterHighlight(route.path);
        });
        // 路由更新时
        onBeforeRouteUpdate((to) => {
            setCurrentRouterHighlight(to.path);
            proxy.mittBus.emit('onMenuClick');
        });
        return {
            menuLists,
            onElMenuHorizontalScroll,
            onHorizontalSelect,
            ...toRefs(state),
        };
    },
});
</script>

<style scoped lang="scss">
.el-menu-horizontal-warp {
    flex: 1;
    overflow: hidden;
    margin-right: 30px;
    ::v-deep(.el-scrollbar__bar.is-vertical) {
        display: none;
    }
    ::v-deep(a) {
        width: 100%;
    }
    .el-menu.el-menu--horizontal {
        display: flex;
        height: 100%;
        width: 100%;
        box-sizing: border-box;
    }
}
</style>
