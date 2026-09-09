<template>
    <ContextMenu :modal="false" @update:open="menuOpen = $event">
        <!--
            虚拟触发：1px 固定定位的隐形 Trigger，openContextmenu(item) 时把内部派发点
            移到目标坐标并派发 contextmenu 事件交给官方组件打开；reka-ui 取事件的
            clientX/Y 作为锚点，定位/边界碰撞/子菜单/键盘导航全部由它处理
        -->
        <ContextMenuTrigger class="contextmenu-virtual-trigger">
            <span ref="dispatchRef" />
        </ContextMenuTrigger>
        <!--
            focus-outside.prevent：el-dialog 等容器的焦点管理会把焦点抢回弹窗，触发 reka 的
            focusOutside dismiss 导致菜单打开即关闭；外部点击由 pointerDownOutside 负责关闭，
            Escape 由 escapeKeyDown 负责，均不受此拦截影响
        -->
        <ContextMenuContent v-if="visibleItems.length" class="w-auto min-w-36 z-[2190]" @focus-outside.prevent>
            <ContextmenuItemNode :items="visibleItems" :payload="state.item" @select="onSelect" />
        </ContextMenuContent>
    </ContextMenu>
</template>

<script setup lang="ts" name="layoutTagsViewContextmenu">
import { computed, nextTick, reactive, ref } from 'vue';
import { ContextMenu, ContextMenuContent, ContextMenuTrigger } from '@/components/ui/context-menu';
import { ContextmenuItem, filterVisibleItems } from './item';
import ContextmenuItemNode from './ContextmenuItemNode.vue';

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

const state = reactive({
    // openContextmenu(item) 传入的业务数据，作为 isHide/onClickFunc 的入参
    item: {} as unknown,
});

const pos = reactive({ x: 0, y: 0 });
const dispatchRef = ref<HTMLElement>();
// 菜单是否展开（由 reka-ui 的 update:open 同步）：closeContextmenu 仅在展开时才派发 Escape
const menuOpen = ref(false);

// 过滤后的可见菜单项（含子菜单递归过滤）
const visibleItems = computed(() => filterVisibleItems(props.items, state.item));

/**
 * 打开右键菜单：等调用方刚重建的 items 同步到 props 后再校验/派发；
 * 派发点位于 Trigger 内部，事件带着目标坐标冒泡到 reka-ui 的监听元素打开菜单
 */
const openContextmenu = async (item: unknown) => {
    state.item = item;
    // 等父组件渲染后再读坐标与 items：props 随父组件渲染才更新，同步读会拿到上一次的坐标（首次为初始 0,0）
    await nextTick();
    pos.x = props.dropdown.x;
    pos.y = props.dropdown.y;
    if (!visibleItems.value.length || !Number.isFinite(pos.x) || !Number.isFinite(pos.y)) {
        return;
    }
    dispatchRef.value?.dispatchEvent(
        new MouseEvent('contextmenu', { bubbles: true, cancelable: true, view: window, clientX: pos.x, clientY: pos.y })
    );
};

// 关闭右键菜单：派发 Escape 键交给 reka-ui 的关闭逻辑。
// 仅在菜单确实展开时派发——该 Escape 冒泡到 document 会被 el-dialog/el-drawer 的
// 全局 Esc 处理器捕获并连带关闭宿主弹层（如 AI 助手抽屉内嵌资源树点击节点即误关抽屉）
const closeContextmenu = () => {
    if (!menuOpen.value) {
        return;
    }
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true, cancelable: true }));
};

// 菜单项选中：透传历史事件，保持对外 API 兼容
const onSelect = (item: ContextmenuItem) => {
    emit('currentContextmenuClick', { id: item.clickId, item: state.item });
};

// 暴露变量
defineExpose({
    openContextmenu,
    closeContextmenu,
});
</script>

<style scoped>
.contextmenu-virtual-trigger {
    display: none;
}
</style>
