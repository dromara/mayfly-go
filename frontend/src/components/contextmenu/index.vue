<template>
    <transition @enter="onEnter" name="el-zoom-in-center">
        <div
            :aria-hidden="state.isShow ? 'false' : 'true'"
            class="el-dropdown__popper el-popper is-light is-pure custom-contextmenu"
            role="tooltip"
            data-popper-placement="bottom"
            :style="`top: ${state.dropdown.y + 5}px;left: ${state.dropdown.x}px;`"
            :key="state.menuKey"
            v-show="state.isShow && !allHide"
            @contextmenu="headerContextmenuClick"
        >
            <ul class="el-dropdown-menu">
                <template v-for="(v, k) in state.dropdownList">
                    <li
                        :id="String(v.clickId)"
                        v-auth="v.permission"
                        class="el-dropdown-menu__item"
                        aria-disabled="false"
                        tabindex="-1"
                        :key="k"
                        v-if="!v.affix && !v.isHide(state.item)"
                        @click="onCurrentContextmenuClick(v)"
                    >
                        <SvgIcon :name="v.icon" />
                        <span>{{ $t(v.txt) }}</span>
                    </li>
                </template>
            </ul>
            <div v-if="state.arrowLeft > 0" class="el-popper__arrow" :style="{ left: `${state.arrowLeft}px` }"></div>
        </div>
    </transition>
</template>

<script setup lang="ts" name="layoutTagsViewContextmenu">
import { computed, reactive, onMounted, onUnmounted, watch } from 'vue';
import { ContextmenuItem } from './index';
import SvgIcon from '@/components/svg-icon/index.vue';
import { useWindowSize } from '@vueuse/core';

// 定义父组件传过来的值
interface DropdownPosition {
    x: number;
    y: number;
}

const props = withDefaults(
    defineProps<{
        dropdown?: DropdownPosition;
        items?: ContextmenuItem[];
    }>(),
    { dropdown: () => ({ x: 0, y: 0 }), items: () => [] }
);

// 定义子组件向父组件传值/事件
const emit = defineEmits(['currentContextmenuClick']);

const { width: vw, height: vh } = useWindowSize();

// 定义变量内容
const state = reactive({
    isShow: false,
    dropdownList: [] as ContextmenuItem[],
    item: {} as unknown,
    arrowLeft: 10,
    menuKey: 0,
    dropdown: {
        x: 0,
        y: 0,
    },
});

// 下拉菜单宽高
let contextmenuWidth = 117;
let contextmenuHeight = 117;
// 下拉菜单元素
let ele: HTMLElement | null = null;

const onEnter = (el: Element) => {
    if (ele || (el as HTMLElement).offsetHeight == 0) {
        return;
    }

    ele = el as HTMLElement;
    contextmenuHeight = (el as HTMLElement).offsetHeight;
    contextmenuWidth = (el as HTMLElement).offsetWidth;
    setDropdowns(props.dropdown);
};

const setDropdowns = (dropdown: DropdownPosition) => {
    let { x, y } = dropdown;

    state.arrowLeft = 10;

    //  `Dropdown 下拉菜单` 的宽度
    if (x + contextmenuWidth > vw.value) {
        state.arrowLeft = contextmenuWidth - (vw.value - x);
        x = vw.value - contextmenuWidth - 5;
    }
    if (y + contextmenuHeight > vh.value) {
        y = vh.value - contextmenuHeight - 5;
        state.arrowLeft = 0;
    }

    state.dropdown.x = x;
    state.dropdown.y = y;
};

const allHide = computed(() => {
    for (let item of state.dropdownList) {
        if (!item.isHide(state.item)) {
            return false;
        }
    }
    return true;
});

// 当前项菜单点击
const onCurrentContextmenuClick = (ci: ContextmenuItem) => {
    // 存在点击事件，则触发该事件函数
    if (ci.onClickFunc) {
        ci.onClickFunc(state.item);
    }
    emit('currentContextmenuClick', { id: ci.clickId, item: state.item });
};

const headerContextmenuClick = (event: MouseEvent) => {
    event.preventDefault(); // 阻止默认的右击菜单行为
};

// 打开右键菜单：判断是否固定，固定则不显示关闭按钮
const openContextmenu = (item: unknown) => {
    state.item = item;
    closeContextmenu();
    state.menuKey++;
    setTimeout(() => {
        // 先设置位置再显示，避免首次渲染时从左上角 (0,0) 滑下的动画
        state.dropdown = { ...props.dropdown };
        state.isShow = true;
    }, 10);
};

// 关闭右键菜单
const closeContextmenu = () => {
    state.isShow = false;
};

// 监听页面监听进行右键菜单的关闭
onMounted(() => {
    document.body.addEventListener('click', closeContextmenu);
    state.dropdownList = props.items;
});

// 页面卸载时，移除右键菜单监听事件
onUnmounted(() => {
    document.body.removeEventListener('click', closeContextmenu);
});

watch(
    () => props.dropdown,
    () => {
        // 元素置为空，重新在onEnter赋值元素，否则会造成堆栈溢出
        ele = null;
    },
    {
        deep: true,
    }
);

watch(
    () => props.items,
    (x: ContextmenuItem[]) => {
        state.dropdownList = x;
    },
    {
        deep: true,
    }
);

// 暴露变量
defineExpose({
    openContextmenu,
    closeContextmenu,
});
</script>

<style scoped lang="scss">
.custom-contextmenu {
    transform-origin: center top;
    z-index: 2190;
    position: fixed;

    .el-dropdown-menu__item {
        padding: 5px 12px;
        font-size: 12px !important;
        white-space: nowrap;

        i {
            font-size: 12px !important;
        }
    }
}
</style>
