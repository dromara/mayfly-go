<template>
    <template v-for="item in items" :key="String(item.clickId)">
        <template v-if="!item.affix && !item.isHide(payload)">
            <!-- 子菜单（递归渲染，支持任意层级） -->
            <!-- 子菜单（递归渲染，支持任意层级）；权限指令放在有真实元素根的 SubTrigger 上（Sub 根节点是 fragment，指令不生效） -->
            <ContextMenuSub v-if="hasVisibleChildren(item)">
                <ContextMenuSubTrigger v-auth="item.permission">
                    <SvgIcon v-if="item.icon" :name="item.icon" />
                    <span>{{ $t(item.txt) }}</span>
                </ContextMenuSubTrigger>
                <ContextMenuSubContent>
                    <ContextmenuItemNode :items="visibleChildren(item)" :payload="payload" @select="onChildSelect" />
                </ContextMenuSubContent>
            </ContextMenuSub>

            <!-- 叶子项（声明了子项但全部不可见时，整项隐藏） -->
            <ContextMenuItem v-else-if="!item.children?.length" v-auth="item.permission" @select="onSelect(item)">
                <SvgIcon v-if="item.icon" :name="item.icon" />
                <span>{{ $t(item.txt) }}</span>
            </ContextMenuItem>
        </template>
    </template>
</template>

<script setup lang="ts">
import { ContextMenuItem, ContextMenuSub, ContextMenuSubContent, ContextMenuSubTrigger } from '@/components/ui/context-menu';
import SvgIcon from '@/components/svg-icon/index.vue';
import { ContextmenuItem } from './item';

const props = defineProps<{
    /** 同级菜单项列表 */
    items: ContextmenuItem[];
    /** openContextmenu 传入的业务数据（isHide/onClickFunc 的入参） */
    payload: unknown;
}>();

const emit = defineEmits<{ select: [item: ContextmenuItem] }>();

/** 存在可见子项才渲染为子菜单，否则整项隐藏（避免空壳子菜单） */
const hasVisibleChildren = (item: ContextmenuItem): boolean => (item.children ?? []).some((child) => !child.affix && !child.isHide(props.payload));

const visibleChildren = (item: ContextmenuItem): ContextmenuItem[] => (item.children ?? []).filter((child) => !child.affix && !child.isHide(props.payload));

const onSelect = (item: ContextmenuItem) => {
    item.onClickFunc?.(props.payload);
    emit('select', item);
};

/** 子节点冒泡的选中事件：onClickFunc 已由子节点执行，这里只继续向上透传（否则每层递归都会重复执行一次） */
const onChildSelect = (item: ContextmenuItem) => {
    emit('select', item);
};
</script>
