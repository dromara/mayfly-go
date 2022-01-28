<template>
    <div class="layout-navbars-tagsview" :class="{ 'layout-navbars-tagsview-shadow': getThemeConfig.layout === 'classic' }">
        <el-scrollbar ref="scrollbarRef" @wheel.prevent="onHandleScroll">
            <ul class="layout-navbars-tagsview-ul" :class="setTagsStyle" ref="tagsUlRef">
                <li
                    v-for="(v, k) in tagsViewList"
                    :key="k"
                    class="layout-navbars-tagsview-ul-li"
                    :data-name="v.name"
                    :class="{ 'is-active': isActive(v) }"
                    @contextmenu.prevent="onContextmenu(v, $event)"
                    @click="onTagsClick(v, k)"
                    :ref="
                        (el) => {
                            if (el) tagsRefs[k] = el;
                        }
                    "
                >
                    <i class="iconfont icon-webicon318 layout-navbars-tagsview-ul-li-iconfont font14" v-if="isActive(v)"></i>
                    <SvgIcon
                        :name="v.meta.icon"
                        class="layout-navbars-tagsview-ul-li-iconfont"
                        v-if="!isActive(v) && getThemeConfig.isTagsviewIcon"
                    />
                    <span>{{ v.meta.title }}</span>
                    <template v-if="isActive(v)">
                        <SvgIcon
                            name="RefreshRight"
                            class="ml5 layout-navbars-tagsview-ul-li-refresh"
                            @click.stop="refreshCurrentTagsView($route.fullPath)"
                        />
                        <SvgIcon
                            name="Close"
                            class="layout-navbars-tagsview-ul-li-icon layout-icon-active"
                            v-if="!v.meta.isAffix"
                            @click.stop="closeCurrentTagsView(getThemeConfig.isShareTagsView ? v.path : v.path)"
                        />
                    </template>
                    <SvgIcon
                        name="Close"
                        class="layout-navbars-tagsview-ul-li-icon layout-icon-three"
                        v-if="!v.meta.isAffix"
                        @click.stop="closeCurrentTagsView(getThemeConfig.isShareTagsView ? v.path : v.path)"
                    />
                </li>
            </ul>
        </el-scrollbar>
        <Contextmenu :dropdown="dropdown" ref="contextmenuRef" @currentContextmenuClick="onCurrentContextmenuClick" />
    </div>
</template>

<script lang="ts">
import { toRefs, reactive, onMounted, computed, ref, nextTick, onBeforeUpdate, onBeforeMount, onUnmounted, getCurrentInstance, watch } from 'vue';
import { useRoute, useRouter, onBeforeRouteUpdate } from 'vue-router';
import screenfull from 'screenfull';
import { useStore } from '@/store/index.ts';
import { setSession, getSession, removeSession } from '@/common/utils/storage.ts';
import Sortable from 'sortablejs';
import Contextmenu from '@/views/layout/navBars/tagsView/contextmenu.vue';
export default {
    name: 'layoutTagsView',
    components: { Contextmenu },
    setup() {
        const { proxy } = getCurrentInstance() as any;
        const tagsRefs = ref([]);
        const scrollbarRef = ref();
        const contextmenuRef = ref();
        const tagsUlRef = ref();
        const store = useStore();
        const route = useRoute();
        const router = useRouter();
        const state: any = reactive({
            routePath: route.fullPath,
            dropdown: { x: '', y: '' },
            tagsRefsIndex: 0,
            tagsViewList: [],
            sortable: '',
        });
        // 动态设置 tagsView 风格样式
        const setTagsStyle = computed(() => {
            return store.state.themeConfig.themeConfig.tagsStyle;
        });
        // 获取布局配置信息
        const getThemeConfig = computed(() => {
            return store.state.themeConfig.themeConfig;
        });
        // 存储 tagsViewList 到浏览器临时缓存中，页面刷新时，保留记录
        const addBrowserSetSession = (tagsViewList: Array<object>) => {
            setSession('tagsViewList', tagsViewList);
        };
        // 获取 vuex 中的 tagsViewRoutes 列表
        const getTagsViewRoutes = () => {
            state.routePath = route.fullPath;
            state.tagsViewList = [];
            if (!store.state.themeConfig.themeConfig.isCacheTagsView) removeSession('tagsViewList');
            initTagsView();
        };
        // vuex 中获取路由信息：如果是设置了固定的（isAffix），进行初始化显示
        const initTagsView = () => {
            if (getSession('tagsViewList') && store.state.themeConfig.themeConfig.isCacheTagsView) {
                state.tagsViewList = getSession('tagsViewList');
            } else {
                // state.tagsViews.map((v: any) => {
                // 	if (v.meta.isAffix && !v.meta.isHide) state.tagsViewList.push({ ...v });
                // });
                addTagsView(route.fullPath);
            }
            // 初始化当前元素(li)的下标
            getTagsRefsIndex(route.fullPath);
            // 添加初始化横向滚动条移动到对应位置
            tagsViewmoveToCurrentTag();
        };

        // 1、添加 tagsView：未设置隐藏（isHide）也添加到在 tagsView 中
        // path为fullPath
        const addTagsView = (path: string, to: any = null) => {
            if (!to) {
                to = route;
            }
            path = decodeURI(path);
            for (let tv of state.tagsViewList) {
                if (tv.fullPath === path) {
                    return false;
                }
            }
            state.tagsViewList.push({ ...to });
            // addBrowserSetSession(state.tagsViewList);
        };

        // 2、刷新当前 tagsView：
        // path为fullPath
        const refreshCurrentTagsView = (path: string) => {
            proxy.mittBus.emit('onTagsViewRefreshRouterView', path);
        };

        // 3、关闭当前 tagsView：如果是设置了固定的（isAffix），不可以关闭
        // path为fullPath
        const closeCurrentTagsView = (path: string) => {
            console.log(path)
            state.tagsViewList.map((v: any, k: number, arr: any) => {
                if (!v.meta.isAffix) {
                    if (v.fullPath === path) {
                        state.tagsViewList.splice(k, 1);
                        setTimeout(() => {
                            // 最后一个
                            if (state.tagsViewList.length === k) router.push({ path: arr[arr.length - 1].path, query: arr[arr.length - 1].query });
                            // 否则，跳转到下一个
                            else router.push({ path: arr[k].path, query: arr[k].query });
                        }, 0);
                    }
                }
            });
            // addBrowserSetSession(state.tagsViewList);
        };

        // 4、关闭其它 tagsView：如果是设置了固定的（isAffix），不进行关闭
        const closeOtherTagsView = (path: string) => {
            const oldTagViews = state.tagsViewList;
            state.tagsViewList = [];
            oldTagViews.map((v: any) => {
                if (v.meta.isAffix && !v.meta.isHide) state.tagsViewList.push({ ...v });
            });
            addTagsView(path);
        };

        // 5、关闭全部 tagsView：如果是设置了固定的（isAffix），不进行关闭
        const closeAllTagsView = (path: string) => {
            const oldTagViews = state.tagsViewList;
            state.tagsViewList = [];
            oldTagViews.map((v: any) => {
                if (v.meta.isAffix && !v.meta.isHide) {
                    state.tagsViewList.push({ ...v });
                    if (state.tagsViewList.some((v: any) => v.path === path)) router.push({ path, query: route.query });
                    else router.push({ path: v.path, query: route.query });
                }
            });
            // addBrowserSetSession(state.tagsViewList);
        };
        // 6、开启当前页面全屏
        const openCurrenFullscreen = (path: string) => {
            const item = state.tagsViewList.find((v: any) => v.fullPath === path);
            nextTick(() => {
                router.push({ path, query: item.query });
                const element = document.querySelector('.layout-main');
                const screenfulls: any = screenfull;
                screenfulls.request(element);
            });
        };
        // 当前项右键菜单点击
        const onCurrentContextmenuClick = (data: any) => {
            // path为fullPath
            let { id, path } = data;
            let currentTag = state.tagsViewList.find((v: any) => v.fullPath === path);
            switch (id) {
                case 0:
                    refreshCurrentTagsView(path);
                    router.push({ path, query: currentTag.query });
                    break;
                case 1:
                    closeCurrentTagsView(path);
                    break;
                case 2:
                    router.push({ path, query: currentTag.query });
                    closeOtherTagsView(path);
                    break;
                case 3:
                    closeAllTagsView(path);
                    break;
                case 4:
                    openCurrenFullscreen(path);
                    break;
            }
        };
        // 判断页面高亮
        const isActive = (route: any) => {
            return route.fullPath === state.routePath;
        };
        // 右键点击时：传 x,y 坐标值到子组件中（props）
        const onContextmenu = (v: any, e: any) => {
            const { clientX, clientY } = e;
            state.dropdown.x = clientX;
            state.dropdown.y = clientY;
            contextmenuRef.value.openContextmenu(v);
        };
        // 当前的 tagsView 项点击时
        const onTagsClick = (v: any, k: number) => {
            state.routePath = decodeURI(v.fullPath);
            state.tagsRefsIndex = k;
            router.push(v);
        };
        // 更新滚动条显示
        const updateScrollbar = () => {
            proxy.$refs.scrollbarRef.update();
        };
        // 鼠标滚轮滚动
        const onHandleScroll = (e: any) => {
            proxy.$refs.scrollbarRef.$refs.wrap.scrollLeft += e.wheelDelta / 4;
        };
        // tagsView 横向滚动
        const tagsViewmoveToCurrentTag = () => {
            nextTick(() => {
                if (tagsRefs.value.length <= 0) return false;
                // 当前 li 元素
                let liDom = tagsRefs.value[state.tagsRefsIndex];
                // 当前 li 元素下标
                let liIndex = state.tagsRefsIndex;
                // 当前 ul 下 li 元素总长度
                let liLength = tagsRefs.value.length;
                // 最前 li
                let liFirst: any = tagsRefs.value[0];
                // 最后 li
                let liLast: any = tagsRefs.value[tagsRefs.value.length - 1];
                // 当前滚动条的值
                let scrollRefs = proxy.$refs.scrollbarRef.$refs.wrap$;
                // 当前滚动条滚动宽度
                let scrollS = scrollRefs.scrollWidth;
                // 当前滚动条偏移宽度
                let offsetW = scrollRefs.offsetWidth;
                // 当前滚动条偏移距离
                let scrollL = scrollRefs.scrollLeft;
                // 上一个 tags li dom
                let liPrevTag: any = tagsRefs.value[state.tagsRefsIndex - 1];
                // 下一个 tags li dom
                let liNextTag: any = tagsRefs.value[state.tagsRefsIndex + 1];
                // 上一个 tags li dom 的偏移距离
                let beforePrevL: any = '';
                // 下一个 tags li dom 的偏移距离
                let afterNextL: any = '';
                if (liDom === liFirst) {
                    // 头部
                    scrollRefs.scrollLeft = 0;
                } else if (liDom === liLast) {
                    // 尾部
                    scrollRefs.scrollLeft = scrollS - offsetW;
                } else {
                    // 非头/尾部
                    if (liIndex === 0) beforePrevL = liFirst.offsetLeft - 5;
                    else beforePrevL = liPrevTag?.offsetLeft - 5;
                    if (liIndex === liLength) afterNextL = liLast.offsetLeft + liLast.offsetWidth + 5;
                    else afterNextL = liNextTag.offsetLeft + liNextTag.offsetWidth + 5;
                    if (afterNextL > scrollL + offsetW) {
                        scrollRefs.scrollLeft = afterNextL - offsetW;
                    } else if (beforePrevL < scrollL) {
                        scrollRefs.scrollLeft = beforePrevL;
                    }
                }
                // 更新滚动条，防止不出现
                updateScrollbar();
            });
        };
        // 获取 tagsView 的下标：用于处理 tagsView 点击时的横向滚动
        const getTagsRefsIndex = (path: string) => {
            if (state.tagsViewList.length > 0) {
                state.tagsRefsIndex = state.tagsViewList.findIndex((item: any) => item.fullPath === path);
            }
        };
        // 设置 tagsView 可以进行拖拽
        const initSortable = () => {
            const el: any = document.querySelector('.layout-navbars-tagsview-ul');
            if (!el) return false;
            if (!getThemeConfig.value.isSortableTagsView) state.sortable && state.sortable.destroy();
            if (getThemeConfig.value.isSortableTagsView) {
                state.sortable = Sortable.create(el, {
                    animation: 300,
                    dataIdAttr: 'data-name',
                    onEnd: () => {
                        const sortEndList: any = [];
                        state.sortable.toArray().map((val: any) => {
                            state.tagsViewList.map((v: any) => {
                                if (v.name === val) sortEndList.push({ ...v });
                            });
                        });
                        // addBrowserSetSession(sortEndList);
                    },
                });
            }
        };
        // 监听路由的变化，动态赋值给 tagsView
        // watch(store.state, (val) => {
        // 	if (val.tagsViews.tagsViews.length === state.tagsViewList.length) return false;
        // 	getTagsViewRoutes();
        // });
        // 页面加载前
        onBeforeMount(() => {
            // 监听非本页面调用 0 刷新当前，1 关闭当前，2 关闭其它，3 关闭全部 4 当前页全屏
            proxy.mittBus.on('onCurrentContextmenuClick', (data: object) => {
                onCurrentContextmenuClick(data);
            });
            // 监听布局配置界面开启/关闭拖拽
            proxy.mittBus.on('openOrCloseSortable', () => {
                initSortable();
            });
        });
        // 页面卸载时
        onUnmounted(() => {
            // 取消非本页面调用监听
            proxy.mittBus.off('onCurrentContextmenuClick');
            // 取消监听布局配置界面开启/关闭拖拽
            proxy.mittBus.off('openOrCloseSortable');
        });
        // 页面更新时
        onBeforeUpdate(() => {
            tagsRefs.value = [];
        });
        // 页面加载时
        onMounted(() => {
            // 初始化 tagsViewRoutes 列表
            getTagsViewRoutes();
            initSortable();
        });
        // 路由更新时
        onBeforeRouteUpdate((to) => {
            state.routePath = decodeURI(to.fullPath);
            addTagsView(to.fullPath, to);
            getTagsRefsIndex(to.fullPath);
            tagsViewmoveToCurrentTag();
        });
        return {
            isActive,
            onContextmenu,
            getTagsViewRoutes,
            onTagsClick,
            tagsRefs,
            contextmenuRef,
            scrollbarRef,
            tagsUlRef,
            onHandleScroll,
            getThemeConfig,
            setTagsStyle,
            refreshCurrentTagsView,
            closeCurrentTagsView,
            onCurrentContextmenuClick,
            ...toRefs(state),
        };
    },
};
</script>

<style scoped lang="scss">
.layout-navbars-tagsview {
    flex: 1;
    background-color: #ffffff;
    border-bottom: 1px solid #f1f2f3;
    ::v-deep(.el-scrollbar__wrap) {
        overflow-x: auto !important;
    }
    &-ul {
        list-style: none;
        margin: 0;
        padding: 0;
        height: 34px;
        display: flex;
        align-items: center;
        color: #606266;
        font-size: 12px;
        white-space: nowrap;
        padding: 0 15px;
        &-li {
            height: 26px;
            line-height: 26px;
            display: flex;
            align-items: center;
            border: 1px solid #e6e6e6;
            padding: 0 15px;
            margin-right: 5px;
            border-radius: 2px;
            position: relative;
            z-index: 0;
            cursor: pointer;
            justify-content: space-between;
            &:hover {
                background-color: var(--color-primary-light-9);
                color: var(--color-primary);
                border-color: var(--color-primary-light-6);
            }
            &-iconfont {
                position: relative;
                left: -5px;
                font-size: 12px;
            }
            &-icon {
                border-radius: 100%;
                position: relative;
                height: 14px;
                width: 14px;
                text-align: center;
                line-height: 14px;
                right: -5px;
                &:hover {
                    color: #fff;
                    background-color: var(--color-primary-light-3);
                }
            }
            .layout-icon-active {
                display: block;
            }
            .layout-icon-three {
                display: none;
            }
        }
        .is-active {
            color: #ffffff;
            background: var(--color-primary);
            border-color: var(--color-primary);
        }
    }
    // 风格2
    .tags-style-two {
        .layout-navbars-tagsview-ul-li {
            height: 34px !important;
            line-height: 34px !important;
            border: none !important;
            .layout-navbars-tagsview-ul-li-iconfont {
                display: none;
            }
            .layout-icon-active {
                display: none;
            }
            .layout-icon-three {
                display: block;
            }
        }
        .is-active {
            background: none !important;
            color: var(--color-primary) !important;
            border-bottom: 2px solid !important;
            border-color: var(--color-primary) !important;
            border-radius: 0 !important;
        }
    }
    // 风格3
    .tags-style-three {
        .layout-navbars-tagsview-ul-li {
            height: 34px !important;
            line-height: 34px !important;
            border-right: 1px solid #f6f6f6 !important;
            border-top: none !important;
            border-bottom: none !important;
            border-left: none !important;
            border-radius: 0 !important;
            margin-right: 0 !important;
            &:first-of-type {
                border-left: 1px solid #f6f6f6 !important;
            }
            .layout-icon-active {
                display: none;
            }
            .layout-icon-three {
                display: block;
            }
        }
        .is-active {
            background: white !important;
            color: var(--color-primary) !important;
            border-top: 1px solid !important;
            border-top-color: var(--color-primary) !important;
        }
    }
    // 风格4
    .tags-style-four {
        .layout-navbars-tagsview-ul-li {
            margin-right: 0 !important;
            border: none !important;
            position: relative;
            border-radius: 3px !important;
            .layout-icon-active {
                display: none;
            }
            .layout-icon-three {
                display: block;
            }
            &:hover {
                background: none !important;
            }
        }
        .is-active {
            background: none !important;
            color: var(--color-primary) !important;
        }
    }
}
.layout-navbars-tagsview-shadow {
    box-shadow: rgb(0 21 41 / 4%) 0px 1px 4px;
}
</style>
