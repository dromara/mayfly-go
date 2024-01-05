<template>
    <div class="layout-navbars-tagsview" :class="{ 'layout-navbars-tagsview-shadow': themeConfig.layout === 'classic' }">
        <el-scrollbar ref="scrollbarRef" @wheel.prevent="onHandleScroll">
            <ul class="layout-navbars-tagsview-ul" :class="setTagsStyle" ref="tagsUlRef">
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
                            if (el) tagsRefs[k] = el;
                        }
                    "
                >
                    <SvgIcon name="iconfont icon-tag-view-active" class="layout-navbars-tagsview-ul-li-iconfont font14" v-if="isActive(v)" />
                    <SvgIcon :name="v.icon" class="layout-navbars-tagsview-ul-li-iconfont" v-if="!isActive(v) && themeConfig.isTagsviewIcon" />
                    <span>{{ v.title }}</span>
                    <template v-if="isActive(v)">
                        <SvgIcon
                            name="RefreshRight"
                            class="font14 ml5 layout-navbars-tagsview-ul-li-refresh"
                            @click.stop="refreshCurrentTagsView($route.fullPath)"
                        />
                        <SvgIcon
                            name="Close"
                            class="font14 layout-navbars-tagsview-ul-li-icon layout-icon-active"
                            v-if="!v.isAffix"
                            @click.stop="closeCurrentTagsView(themeConfig.isShareTagsView ? v.path : v.path)"
                        />
                    </template>

                    <SvgIcon
                        name="Close"
                        class="font14 layout-navbars-tagsview-ul-li-icon layout-icon-three"
                        v-if="!v.isAffix"
                        @click.stop="closeCurrentTagsView(themeConfig.isShareTagsView ? v.path : v.path)"
                    />
                </li>
            </ul>
        </el-scrollbar>
        <Contextmenu :items="state.contextmenu.items" :dropdown="state.contextmenu.dropdown" ref="contextmenuRef" />
    </div>
</template>

<script lang="ts" setup name="layoutTagsView">
import { reactive, onMounted, computed, ref, nextTick, onBeforeUpdate, onBeforeMount, onUnmounted, getCurrentInstance } from 'vue';
import { useRoute, useRouter, onBeforeRouteUpdate } from 'vue-router';
import screenfull from 'screenfull';
import { storeToRefs } from 'pinia';
import { useThemeConfig } from '@/store/themeConfig';
import mittBus from '@/common/utils/mitt';
import Sortable from 'sortablejs';
import { Contextmenu, ContextmenuItem } from '@/components/contextmenu';
import { getTagViews, setTagViews, removeTagViews } from '@/common/utils/storage';
import { useTagsViews } from '@/store/tagsViews';
import { useKeepALiveNames } from '@/store/keepAliveNames';

const { proxy } = getCurrentInstance() as any;
const tagsRefs = ref([]) as any;
const scrollbarRef = ref();
const contextmenuRef = ref();
const tagsUlRef = ref();

const { themeConfig } = storeToRefs(useThemeConfig());
const { tagsViews } = storeToRefs(useTagsViews());

const keepAliveNamesStores = useKeepALiveNames();

const route = useRoute();
const router = useRouter();

const contextmenuItems = [
    new ContextmenuItem(0, '刷新').withIcon('RefreshRight').withOnClick((data: any) => {
        // path为fullPath
        let { path } = data;
        let currentTag = tagsViews.value.find((v: any) => v.path === path);
        refreshCurrentTagsView(path);
        router.push({ path, query: currentTag?.query });
    }),

    new ContextmenuItem(1, '关闭').withIcon('Close').withOnClick((data: any) => closeCurrentTagsView(data.path)),

    new ContextmenuItem(2, '关闭其他').withIcon('CircleClose').withOnClick((data: any) => {
        let { path } = data;
        let currentTag = tagsViews.value.find((v: any) => v.path === path);
        router.push({ path, query: currentTag?.query });
        closeOtherTagsView(path);
    }),

    new ContextmenuItem(3, '关闭所有').withIcon('FolderDelete').withOnClick((data: any) => closeAllTagsView(data.path)),

    new ContextmenuItem(4, '当前页全屏').withIcon('full-screen').withOnClick((data: any) => openCurrenFullscreen(data.path)),
];

const state = reactive({
    routePath: route.fullPath,
    // dropdown: { x: '', y: '' },
    tagsRefsIndex: 0,
    sortable: '' as any,
    contextmenu: {
        items: contextmenuItems,
        dropdown: { x: '', y: '' },
    },
});

// 动态设置 tagsView 风格样式
const setTagsStyle = computed(() => {
    return themeConfig.value.tagsStyle;
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
        tagsViews.value?.map((v: any) => {
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
const addTagsView = (path: string, to: any = null, tagViewIndex: number = -1) => {
    nextTick(async () => {
        if (!to) {
            to = route;
        }

        for (let tv of tagsViews.value) {
            if (tv.path === path) {
                return false;
            }
        }

        const tagView = {
            path: path,
            name: to.name,
            query: to.query,
            title: to.meta.title,
            icon: to.meta.icon,
            isAffix: to.meta.isAffix,
            isKeepAlive: to.meta.isKeepAlive,
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
    mittBus.emit('onTagsViewRefreshRouterView', path);
};

const getTagsView = (path: string) => {
    return tagsViews.value.find((v: any) => v.path === path);
};

// 3、关闭当前 tagsView：如果是设置了固定的（isAffix），不可以关闭
// path为fullPath
const closeCurrentTagsView = (path: string) => {
    tagsViews.value.map((v: TagsView, k: number, arr: any) => {
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
    oldTagViews.map((v: any) => {
        if (v.isAffix && !v.isHide) {
            tagsViews.value.push({ ...v });
            if (tagsViews.value.some((v: any) => v.path === path)) {
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
    const item = tagsViews.value.find((v: any) => v.path === path);
    nextTick(() => {
        router.push({ path, query: item?.query });
        const element = document.querySelector('.layout-main');
        const screenfulls: any = screenfull;
        screenfulls.request(element);
    });
};

// 判断页面高亮
const isActive = (tagView: TagsView) => {
    return tagView.path === state.routePath;
};
// 右键点击时：传 x,y 坐标值到子组件中（props）
const onContextmenu = (v: any, e: any) => {
    const { clientX, clientY } = e;
    state.contextmenu.dropdown.x = clientX;
    state.contextmenu.dropdown.y = clientY;
    contextmenuRef.value.openContextmenu(v);
};
// 当前的 tagsView 项点击时
const onTagsClick = (v: any, k: number) => {
    state.routePath = decodeURI(v.path);
    state.tagsRefsIndex = k;
    try {
        router.push(v);
    } catch (e) {}
};
// 更新滚动条显示
const updateScrollbar = () => {
    proxy.$refs.scrollbarRef.update();
};
// 鼠标滚轮滚动
const onHandleScroll = (e: any) => {
    proxy.$refs.scrollbarRef.$refs.wrapRef.scrollLeft += e.wheelDelta / 4;
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
        let scrollRefs = proxy.$refs.scrollbarRef.$refs.wrapRef;
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
const setTagsRefsIndex = (path: string) => {
    if (tagsViews.value.length > 0) {
        state.tagsRefsIndex = tagsViews.value.findIndex((item: any) => item.path === path);
    }
};
// 设置 tagsView 可以进行拖拽
const initSortable = () => {
    const el: any = document.querySelector('.layout-navbars-tagsview-ul');
    if (!el) return false;
    if (!themeConfig.value.isSortableTagsView) state.sortable && state.sortable.destroy();
    if (themeConfig.value.isSortableTagsView) {
        state.sortable = Sortable.create(el, {
            animation: 300,
            dataIdAttr: 'data-name',
            onEnd: () => {
                const sortEndList: any = [];
                state.sortable.toArray().map((val: any) => {
                    tagsViews.value.map((v: any) => {
                        if (v.name === val) sortEndList.push({ ...v });
                    });
                });
                addBrowserSetSession(sortEndList);
            },
        });
    }
};

// 页面加载前
onBeforeMount(() => {
    // 监听布局配置界面开启/关闭拖拽
    mittBus.on('openOrCloseSortable', () => {
        initSortable();
    });
});
// 页面卸载时
onUnmounted(() => {
    // 取消监听布局配置界面开启/关闭拖拽
    mittBus.off('openOrCloseSortable');
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
    const path = decodeURI(to.fullPath);
    state.routePath = path;
    addTagsView(path, to, state.tagsRefsIndex);
    setTagsRefsIndex(path);
    tagsViewmoveToCurrentTag();
});
</script>

<style scoped lang="scss">
.layout-navbars-tagsview {
    background-color: var(--bg-main-color);
    border-bottom: 1px solid var(--el-border-color-light, #ebeef5);
    position: relative;
    z-index: 4;

    :deep(.el-scrollbar__wrap) {
        overflow-x: auto !important;
    }

    &-ul {
        list-style: none;
        margin: 0;
        padding: 0;
        height: 34px;
        display: flex;
        align-items: center;
        color: var(--el-text-color-regular);
        font-size: 12px;
        white-space: nowrap;
        padding: 0 15px;

        &-li {
            height: 26px;
            line-height: 26px;
            display: flex;
            align-items: center;
            border: 1px solid var(--el-border-color-lighter);
            padding: 0 15px;
            margin-right: 5px;
            border-radius: 2px;
            position: relative;
            z-index: 0;
            cursor: pointer;
            justify-content: space-between;

            &:hover {
                background-color: var(--el-color-primary-light-9);
                color: var(--el-color-primary);
                border-color: var(--el-color-primary-light-5);
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
                    color: var(--el-color-white);
                    background-color: var(--el-color-primary-light-3);
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
            color: var(--el-color-white);
            background: var(--el-color-primary);
            border-color: var(--el-color-primary);
            transition: border-color 3s ease;
        }
    }

    // 风格2
    .tags-style-two {
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
            color: var(--el-color-primary) !important;
        }
    }

    // 风格3
    .tags-style-three {
        align-items: flex-end;

        .tgs-style-three-svg {
            -webkit-mask-image: url('data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iNzAiIGhlaWdodD0iNzAiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyIgZmlsbD0ibm9uZSI+CgogPGc+CiAgPHRpdGxlPkxheWVyIDE8L3RpdGxlPgogIDxwYXRoIHRyYW5zZm9ybT0icm90YXRlKC0wLjEzMzUwNiA1MC4xMTkyIDUwKSIgaWQ9InN2Z18xIiBkPSJtMTAwLjExOTE5LDEwMGMtNTUuMjI4LDAgLTEwMCwtNDQuNzcyIC0xMDAsLTEwMGwwLDEwMGwxMDAsMHoiIG9wYWNpdHk9InVuZGVmaW5lZCIgc3Ryb2tlPSJudWxsIiBmaWxsPSIjRjhFQUU3Ii8+CiAgPHBhdGggZD0ibS0wLjYzNzY2LDcuMzEyMjhjMC4xMTkxOSwwIDAuMjE3MzcsMC4wNTc5NiAwLjQ3Njc2LDAuMTE5MTljMC4yMzIsMC4wNTQ3NyAwLjI3MzI5LDAuMDM0OTEgMC4zNTc1NywwLjExOTE5YzAuMDg0MjgsMC4wODQyOCAwLjM1NzU3LDAgMC40NzY3NiwwbDAuMTE5MTksMGwwLjIzODM4LDAiIGlkPSJzdmdfMiIgc3Ryb2tlPSJudWxsIiBmaWxsPSJub25lIi8+CiAgPHBhdGggZD0ibTI4LjkyMTM0LDY5LjA1MjQ0YzAsMC4xMTkxOSAwLDAuMjM4MzggMCwwLjM1NzU3bDAsMC4xMTkxOWwwLDAuMTE5MTkiIGlkPSJzdmdfMyIgc3Ryb2tlPSJudWxsIiBmaWxsPSJub25lIi8+CiAgPHJlY3QgaWQ9InN2Z180IiBoZWlnaHQ9IjAiIHdpZHRoPSIxLjMxMTA4IiB5PSI2LjgzNTUyIiB4PSItMC4wNDE3MSIgc3Ryb2tlPSJudWxsIiBmaWxsPSJub25lIi8+CiAgPHJlY3QgaWQ9InN2Z181IiBoZWlnaHQ9IjEuNzg3ODQiIHdpZHRoPSIwLjExOTE5IiB5PSI2OC40NTY1IiB4PSIyOC45MjEzNCIgc3Ryb2tlPSJudWxsIiBmaWxsPSJub25lIi8+CiAgPHJlY3QgaWQ9InN2Z182IiBoZWlnaHQ9IjQuODg2NzciIHdpZHRoPSIxOS4wNzAzMiIgeT0iNTEuMjkzMjEiIHg9IjM2LjY2ODY2IiBzdHJva2U9Im51bGwiIGZpbGw9Im5vbmUiLz4KIDwvZz4KPC9zdmc+'),
                url('data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iNzAiIGhlaWdodD0iNzAiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyIgZmlsbD0ibm9uZSI+CiA8Zz4KICA8dGl0bGU+TGF5ZXIgMTwvdGl0bGU+CiAgPHBhdGggdHJhbnNmb3JtPSJyb3RhdGUoLTg5Ljc2MjQgNy4zMzAxNCA1NS4xMjUyKSIgc3Ryb2tlPSJudWxsIiBpZD0ic3ZnXzEiIGZpbGw9IiNGOEVBRTciIGQ9Im02Mi41NzQ0OSwxMTcuNTIwODZjLTU1LjIyOCwwIC0xMDAsLTQ0Ljc3MiAtMTAwLC0xMDBsMCwxMDBsMTAwLDB6IiBjbGlwLXJ1bGU9ImV2ZW5vZGQiIGZpbGwtcnVsZT0iZXZlbm9kZCIvPgogIDxwYXRoIGQ9Im0tMC42Mzc2Niw3LjMxMjI4YzAuMTE5MTksMCAwLjIxNzM3LDAuMDU3OTYgMC40NzY3NiwwLjExOTE5YzAuMjMyLDAuMDU0NzcgMC4yNzMyOSwwLjAzNDkxIDAuMzU3NTcsMC4xMTkxOWMwLjA4NDI4LDAuMDg0MjggMC4zNTc1NywwIDAuNDc2NzYsMGwwLjExOTE5LDBsMC4yMzgzOCwwIiBpZD0ic3ZnXzIiIHN0cm9rZT0ibnVsbCIgZmlsbD0ibm9uZSIvPgogIDxwYXRoIGQ9Im0yOC45MjEzNCw2OS4wNTI0NGMwLDAuMTE5MTkgMCwwLjIzODM4IDAsMC4zNTc1N2wwLDAuMTE5MTlsMCwwLjExOTE5IiBpZD0ic3ZnXzMiIHN0cm9rZT0ibnVsbCIgZmlsbD0ibm9uZSIvPgogIDxyZWN0IGlkPSJzdmdfNCIgaGVpZ2h0PSIwIiB3aWR0aD0iMS4zMTEwOCIgeT0iNi44MzU1MiIgeD0iLTAuMDQxNzEiIHN0cm9rZT0ibnVsbCIgZmlsbD0ibm9uZSIvPgogIDxyZWN0IGlkPSJzdmdfNSIgaGVpZ2h0PSIxLjc4Nzg0IiB3aWR0aD0iMC4xMTkxOSIgeT0iNjguNDU2NSIgeD0iMjguOTIxMzQiIHN0cm9rZT0ibnVsbCIgZmlsbD0ibm9uZSIvPgogIDxyZWN0IGlkPSJzdmdfNiIgaGVpZ2h0PSI0Ljg4Njc3IiB3aWR0aD0iMTkuMDcwMzIiIHk9IjUxLjI5MzIxIiB4PSIzNi42Njg2NiIgc3Ryb2tlPSJudWxsIiBmaWxsPSJub25lIi8+CiA8L2c+Cjwvc3ZnPg=='),
                url("data:image/svg+xml,<svg xmlns='http://www.w3.org/2000/svg'><rect rx='8' width='100%' height='100%' fill='%23F8EAE7'/></svg>");
            -webkit-mask-size:
                18px 30px,
                20px 30px,
                calc(100% - 30px) calc(100% + 17px);
            -webkit-mask-position:
                right bottom,
                left bottom,
                center top;
            -webkit-mask-repeat: no-repeat;
        }

        .layout-navbars-tagsview-ul-li {
            padding: 0 5px;
            border-width: 15px 27px 15px;
            border-style: solid;
            border-color: transparent;
            margin: 0 -15px;

            .layout-icon-active {
                display: none;
            }

            .layout-icon-three {
                display: block;
            }

            &:hover {
                @extend .tgs-style-three-svg;
                background: var(--tagsview3-active-background-color);
                color: unset;
            }
        }

        .is-active {
            @extend .tgs-style-three-svg;
            background: var(--tagsview3-active-background-color) !important;
            color: var(--el-color-primary) !important;
            z-index: 1;
        }
    }
}

.layout-navbars-tagsview-shadow {
    box-shadow: rgb(0 21 41 / 4%) 0px 1px 4px;
}
</style>
