<template>
    <div class="layout-navbars-tagsview" :class="{ 'layout-navbars-tagsview-shadow': themeConfig.layout === 'classic' }">
        <el-scrollbar ref="scrollbarRef" @wheel.prevent="onHandleScroll">
            <ul class="layout-navbars-tagsview-ul" ref="tagsUlRef">
                <li
                    v-for="(v, k) in tagsViews"
                    :key="k"
                    class="layout-navbars-tagsview-ul-li"
                    :data-name="v.name"
                    :class="{ 'is-active': isActive(v) }"
                    @contextmenu.prevent="onContextmenu(v, $event)"
                    @click="onTagsClick(v, k)"
                    :ref="
                        (el) => {
                            if (el) tagsRefs[k] = el as HTMLElement;
                        }
                    "
                >
                    <SvgIcon :name="v.icon" class="layout-navbars-tagsview-ul-li-iconfont" v-if="themeConfig.isTagsviewIcon" />
                    <span>{{ $t(v.title) }}</span>

                    <SvgIcon
                        v-if="isActive(v)"
                        name="RefreshRight"
                        class="text-[14px]! layout-navbars-tagsview-ul-li-icon layout-navbars-tagsview-ul-li-refresh"
                        @click.stop="refreshCurrentTagsView($route.fullPath)"
                    />
                    <span v-if="!v.isAffix" class="layout-navbars-tagsview-ul-li-close-wrap" @click.stop="closeCurrentTagsView(v.path)">
                        <SvgIcon name="Close" class="text-[14px]! layout-navbars-tagsview-ul-li-close-icon" />
                    </span>
                </li>
            </ul>
        </el-scrollbar>
        <Contextmenu :items="state.contextmenu.items" :dropdown="state.contextmenu.dropdown" ref="contextmenuRef" />
    </div>
</template>

<script lang="ts" setup name="layoutTagsView">
import { reactive, onMounted, ref, nextTick, onBeforeUpdate, getCurrentInstance, watch } from 'vue';
import type { ComponentPublicInstance } from 'vue';
import { useRoute, useRouter, onBeforeRouteUpdate, type RouteLocationNormalized } from 'vue-router';
import screenfull from 'screenfull';
import { storeToRefs } from 'pinia';
import { useThemeConfig } from '@/store/themeConfig';
import Sortable from 'sortablejs';
import { Contextmenu, ContextmenuItem } from '@/components/contextmenu';
import { getTagViews, setTagViews, removeTagViews } from '@/common/utils/storage';
import { useTagsViews } from '@/store/tagsViews';
import { useKeepALiveNames } from '@/store/keepAliveNames';

const { proxy } = getCurrentInstance() as { proxy: ComponentPublicInstance & { $refs: Record<string, any> } };
const tagsRefs = ref<HTMLElement[]>([]);
const scrollbarRef = ref();
const contextmenuRef = ref();
const tagsUlRef = ref();

const { themeConfig } = storeToRefs(useThemeConfig());
const { tagsViews } = storeToRefs(useTagsViews());

const keepAliveNamesStores = useKeepALiveNames();

const route = useRoute();
const router = useRouter();

const contextmenuItems = [
    new ContextmenuItem(0, 'layout.tagsView.refresh').withIcon('RefreshRight').withOnClick((data: unknown) => {
        const d = data as TagsView;
        // path为fullPath
        let { path } = d;
        let currentTag = tagsViews.value.find((v: TagsView) => v.path === path);
        refreshCurrentTagsView(path);
        router.push({ path, query: currentTag?.query });
    }),

    new ContextmenuItem(1, 'layout.tagsView.close').withIcon('Close').withOnClick((data: unknown) => closeCurrentTagsView((data as TagsView).path)),

    new ContextmenuItem(2, 'layout.tagsView.closeOther').withIcon('CircleClose').withOnClick((data: unknown) => {
        const d = data as TagsView;
        let { path } = d;
        let currentTag = tagsViews.value.find((v: TagsView) => v.path === path);
        router.push({ path, query: currentTag?.query });
        closeOtherTagsView(path);
    }),

    new ContextmenuItem(3, 'layout.tagsView.closeAll').withIcon('FolderDelete').withOnClick((data: unknown) => closeAllTagsView((data as TagsView).path)),

    new ContextmenuItem(4, 'layout.tagsView.fullscreen').withIcon('full-screen').withOnClick((data: unknown) => openCurrenFullscreen((data as TagsView).path)),
];

const state = reactive({
    routePath: route.fullPath,
    // dropdown: { x: '', y: '' },
    tagsRefsIndex: 0,
    sortable: null as Sortable | null,
    contextmenu: {
        items: contextmenuItems,
        dropdown: { x: 0, y: 0 },
    },
});

// 存储 tagsViewList 到浏览器临时缓存中，页面刷新时，保留记录
const addBrowserSetSession = (tagsViewList: Array<object>) => {
    setTagViews(tagsViewList);
};

// 获取  tagsViewRoutes 列表
const getTagsViewRoutes = () => {
    state.routePath = route.fullPath;
    tagsViews.value = [];
    if (!themeConfig.value.isCacheTagsView) {
        removeTagViews();
    }
    initTagsView();
};
// 获取路由信息：如果是设置了固定的（isAffix），进行初始化显示
const initTagsView = () => {
    const tagViews = getTagViews();
    if (tagViews && themeConfig.value.isCacheTagsView) {
        tagsViews.value = tagViews;
    } else {
        tagsViews.value?.map((v: TagsView) => {
            if (v.isAffix && !v.isHide) {
                tagsViews.value.push({ ...v });
                keepAliveNamesStores.setCacheKeepAlive(v);
            }
        });
        addTagsView(route.fullPath);
    }
    // 初始化当前元素(li)的下标
    setTagsRefsIndex(route.fullPath);
    // 添加初始化横向滚动条移动到对应位置
    tagsViewmoveToCurrentTag();
};

// 1、添加 tagsView：未设置隐藏（isHide）也添加到在 tagsView 中
// path为fullPath
const addTagsView = (path: string, to: RouteLocationNormalized | null = null, tagViewIndex: number = -1) => {
    nextTick(async () => {
        if (!to) {
            to = route;
        }

        for (let tv of tagsViews.value) {
            if (tv.path === path) {
                return false;
            }
        }

        const tagView: TagsView = {
            path: path,
            name: to.name as string,
            query: to.query as TagsView['query'],
            title: to.meta.title as string,
            icon: to.meta.icon as string,
            isAffix: to.meta.isAffix as boolean,
            isKeepAlive: to.meta.isKeepAlive as boolean,
        };

        if (tagViewIndex != -1) {
            tagsViews.value.splice(tagViewIndex + 1, 0, tagView);
        } else {
            tagsViews.value.push(tagView);
        }
        await keepAliveNamesStores.addCachedView(tagView);
        addBrowserSetSession(tagsViews.value);
    });
};

// 2、刷新当前 tagsView：
// path为fullPath
const refreshCurrentTagsView = async (path: string) => {
    const item = getTagsView(path);
    await keepAliveNamesStores.delCachedView(item);
    keepAliveNamesStores.addCachedView(item);
    useTagsViews().setCurrentRefreshPath(path);
};

const getTagsView = (path: string) => {
    return tagsViews.value.find((v: TagsView) => v.path === path);
};

// 3、关闭当前 tagsView：如果是设置了固定的（isAffix），不可以关闭
// path为fullPath
const closeCurrentTagsView = (path: string) => {
    tagsViews.value.map((v: TagsView, k: number, arr: TagsView[]) => {
        if (!v.isAffix) {
            if (v.path === path) {
                keepAliveNamesStores.delCachedView(v);
                tagsViews.value.splice(k, 1);
                setTimeout(() => {
                    if (state.routePath !== path) {
                        return;
                    }
                    let next: TagsView;
                    // 最后一个且高亮时
                    if (tagsViews.value.length === k) {
                        next = k !== arr.length ? arr[k] : arr[arr.length - 1];
                    } else {
                        next = arr[k];
                    }

                    if (next) {
                        router.push({ path: next.path, query: next.query });
                    } else {
                        router.push({ path: '/' });
                    }
                }, 0);
            }
        }
    });
    addBrowserSetSession(tagsViews.value);
};

// 4、关闭其它 tagsView：如果是设置了固定的（isAffix），不进行关闭
const closeOtherTagsView = (path: string) => {
    const oldTagViews = tagsViews.value;
    tagsViews.value = [];
    oldTagViews.map((v: TagsView) => {
        if (v.isAffix && !v.isHide) {
            keepAliveNamesStores.delOthersCachedViews(v);
            tagsViews.value.push({ ...v });
        }
    });
    addTagsView(path);
};

// 5、关闭全部 tagsView：如果是设置了固定的（isAffix），不进行关闭
const closeAllTagsView = (path: string) => {
    keepAliveNamesStores.delAllCachedViews();
    const oldTagViews = tagsViews.value;
    tagsViews.value = [];
    oldTagViews.map((v: TagsView) => {
        if (v.isAffix && !v.isHide) {
            tagsViews.value.push({ ...v });
            if (tagsViews.value.some((v: TagsView) => v.path === path)) {
                router.push({ path, query: route.query });
            }
        }
    });
    if (tagsViews.value) {
        router.push({ path: '/' });
    }
    addBrowserSetSession(tagsViews.value);
};
// 6、开启当前页面全屏
const openCurrenFullscreen = (path: string) => {
    const item = tagsViews.value.find((v: TagsView) => v.path === path);
    nextTick(() => {
        router.push({ path, query: item?.query });
        const element = document.querySelector('.layout-main');
        const screenfulls = screenfull as typeof screenfull & { request: (el: Element) => Promise<void> };
        if (element) screenfulls.request(element);
    });
};

// 判断页面高亮
const isActive = (tagView: TagsView) => {
    return tagView.path === state.routePath;
};
// 右键点击时：传 x,y 坐标值到子组件中（props）
const onContextmenu = (v: TagsView, e: MouseEvent) => {
    const { clientX, clientY } = e;
    state.contextmenu.dropdown.x = clientX;
    state.contextmenu.dropdown.y = clientY;
    contextmenuRef.value?.openContextmenu(v);
};
// 当前的 tagsView 项点击时
const onTagsClick = (v: TagsView, k: number) => {
    state.routePath = decodeURI(v.path);
    state.tagsRefsIndex = k;
    try {
        router.push(v);
    } catch (e) {
        // skip
    }
};
// 更新滚动条显示
const updateScrollbar = () => {
    proxy.$refs.scrollbarRef.update();
};
// 鼠标滚轮滚动
const onHandleScroll = (e: WheelEvent) => {
    proxy.$refs.scrollbarRef.$refs.wrapRef.scrollLeft += (e as WheelEvent & { wheelDelta: number }).wheelDelta / 4;
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
        let liFirst: HTMLElement | undefined = tagsRefs.value[0];
        // 最后 li
        let liLast: HTMLElement | undefined = tagsRefs.value[tagsRefs.value.length - 1];
        // 当前滚动条的值
        let scrollRefs = proxy.$refs.scrollbarRef.$refs.wrapRef;
        // 当前滚动条滚动宽度
        let scrollS = scrollRefs.scrollWidth;
        // 当前滚动条偏移宽度
        let offsetW = scrollRefs.offsetWidth;
        // 当前滚动条偏移距离
        let scrollL = scrollRefs.scrollLeft;
        // 上一个 tags li dom
        let liPrevTag: HTMLElement | undefined = tagsRefs.value[state.tagsRefsIndex - 1];
        // 下一个 tags li dom
        let liNextTag: HTMLElement | undefined = tagsRefs.value[state.tagsRefsIndex + 1];
        // 上一个 tags li dom 的偏移距离
        let beforePrevL: number | string = '';
        // 下一个 tags li dom 的偏移距离
        let afterNextL: number | string = '';
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
const setTagsRefsIndex = (path: string) => {
    if (tagsViews.value.length > 0) {
        state.tagsRefsIndex = tagsViews.value.findIndex((item: TagsView) => item.path === path);
    }
};
// 设置 tagsView 可以进行拖拽
const initSortable = () => {
    const el = document.querySelector('.layout-navbars-tagsview-ul') as HTMLElement | null;
    if (!el) return false;
    if (!themeConfig.value.isSortableTagsView) state.sortable && state.sortable.destroy();
    if (themeConfig.value.isSortableTagsView) {
        state.sortable = Sortable.create(el, {
            animation: 300,
            dataIdAttr: 'data-name',
            onEnd: () => {
                if (!state.sortable) return;
                const sortEndList: TagsView[] = [];
                state.sortable.toArray().map((val: string) => {
                    tagsViews.value.map((v: TagsView) => {
                        if (v.name === val) sortEndList.push({ ...v });
                    });
                });
                addBrowserSetSession(sortEndList);
            },
        });
    }
};

watch(
    () => themeConfig.value.isSortableTagsView,
    (isSortableTagsView: boolean) => {
        if (isSortableTagsView) {
            initSortable();
        }
    }
);

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
    const path = decodeURI(to.fullPath);
    state.routePath = path;
    addTagsView(path, to, state.tagsRefsIndex);
    setTagsRefsIndex(path);
    tagsViewmoveToCurrentTag();
});
</script>

<style scoped lang="css">
.layout-navbars-tagsview {
    background-color: var(--bg-main-color);
    border-bottom: 1px solid var(--el-border-color-light, #ebeef5);
    position: relative;
    z-index: 4;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
}

.layout-navbars-tagsview :deep(.el-scrollbar__wrap) {
    overflow-x: auto !important;
}

.layout-navbars-tagsview-ul {
    list-style: none;
    margin: 0;
    height: 38px;
    display: flex;
    align-items: center;
    color: var(--el-text-color-regular);
    font-size: 13px;
    white-space: nowrap;
    padding: 0 15px;
}

.layout-navbars-tagsview-ul-li {
    height: 30px;
    line-height: 30px;
    display: flex;
    align-items: center;
    border-radius: 6px;
    padding: 0 12px;
    margin-right: 6px;
    position: relative;
    z-index: 0;
    cursor: pointer;
    justify-content: space-between;
    transition: all 0.3s ease;
    border: 1px solid var(--el-border-color, #dcdfe6);
    box-sizing: border-box;
    background-color: var(--el-bg-color, #fafafa);
    color: var(--el-text-color-regular, #606266);
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
}

.layout-navbars-tagsview-ul-li:not(.is-active):hover {
    background-color: var(--el-fill-color-blank, #f5f7fa);
    color: var(--el-text-color-primary, #303133);
    border-color: var(--el-color-primary-light-7, #c6e2ff);
}

.layout-navbars-tagsview-ul-li-iconfont {
    position: relative;
    left: -3px;
    font-size: 12px;
    margin-right: 4px;
}

.layout-navbars-tagsview-ul-li-icon {
    border-radius: 4px;
    position: relative;
    height: 18px;
    width: 18px;
    right: -3px;
    margin-left: 4px;
    transition: all 0.25s ease;
    color: var(--el-text-color-secondary, #909399);
    display: flex;
    align-items: center;
    justify-content: center;
}

.layout-navbars-tagsview-ul-li-icon:hover {
    background-color: var(--el-color-info-light-7);
}

/* 关闭按钮动画容器：默认宽度0不占位，hover/active 时平滑展开 */
.layout-navbars-tagsview-ul-li-close-wrap {
    position: relative;
    right: -3px;
    height: 18px;
    width: 0;
    min-width: 0;
    margin-left: 0;
    overflow: hidden;
    opacity: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    cursor: pointer;
    pointer-events: none;
    transition:
        width 0.6s cubic-bezier(0.34, 1.56, 0.64, 1),
        min-width 0.6s cubic-bezier(0.34, 1.56, 0.64, 1),
        margin-left 0.6s cubic-bezier(0.34, 1.56, 0.64, 1),
        opacity 0.5s ease;
}

/* 鼠标悬浮 li 或激活状态时展开 */
.layout-navbars-tagsview-ul-li:hover .layout-navbars-tagsview-ul-li-close-wrap,
.layout-navbars-tagsview-ul .is-active .layout-navbars-tagsview-ul-li-close-wrap {
    width: 18px;
    min-width: 18px;
    opacity: 1;
    pointer-events: auto;
}

/* 关闭按钮图标：与刷新按钮尺寸和视觉完全一致 */
.layout-navbars-tagsview-ul-li-close-icon {
    border-radius: 4px;
    display: flex !important;
    align-items: center;
    justify-content: center;
    color: var(--el-text-color-secondary, #909399);
    transition:
        background-color 0.2s ease,
        color 0.2s ease;
    flex-shrink: 0;
}

/* 非激活 tag：关闭图标 hover 变红 */
.layout-navbars-tagsview-ul-li:not(.is-active):hover .layout-navbars-tagsview-ul-li-close-icon:hover {
    background-color: var(--el-color-danger);
    color: var(--el-color-white);
}

.layout-navbars-tagsview-ul .is-active {
    color: var(--el-color-primary, #409eff);
    background: var(--el-color-primary-light-9, #ecf5ff);
    border-color: var(--el-color-primary-light-5, #409eff);
    box-shadow: 0 2px 4px rgba(64, 158, 255, 0.2);
}

.layout-navbars-tagsview-ul .is-active .layout-navbars-tagsview-ul-li-icon {
    color: var(--el-color-primary, #409eff);
}

.layout-navbars-tagsview-ul .is-active .layout-navbars-tagsview-ul-li-icon:hover {
    background-color: var(--el-color-primary);
    color: var(--el-color-white);
}

/* 激活 tag 的关闭图标使用主色调 */
.layout-navbars-tagsview-ul .is-active .layout-navbars-tagsview-ul-li-close-icon {
    color: var(--el-color-primary, #409eff);
}

/* 激活 tag 的关闭图标 hover 变红 */
.layout-navbars-tagsview-ul .is-active .layout-navbars-tagsview-ul-li-close-icon:hover {
    background-color: var(--el-color-danger);
    color: var(--el-color-white);
}

.layout-navbars-tagsview-shadow {
    box-shadow: rgb(0 21 41 / 4%) 0px 1px 4px;
}
</style>
