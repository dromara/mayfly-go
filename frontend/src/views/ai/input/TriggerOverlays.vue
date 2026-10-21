<template>
    <!-- 触发器浮层：通用列表菜单（技能）/ 面板模式（资源树），由注册表的 panelComponent 决定 -->
    <TriggerMenu
        v-if="trigger.visible && !trigger.isPanelMode"
        ref="menuRef"
        :visible="trigger.visible"
        :items="trigger.items"
        :selected-index="trigger.selectedIndex"
        :menu-style="trigger.menuStyle"
        @select="trigger.selectItem"
    />
    <!-- 面板模式：组件与选中转换都由注册表提供（activeDef.panelComponent / onPanelSelect），
         新增面板型触发器只需注册一个组件，零模板改动 -->
    <component
        :is="trigger.activeDef?.panelComponent"
        v-else-if="trigger.visible && trigger.isPanelMode"
        ref="panelRef"
        :visible="trigger.visible"
        :menu-style="trigger.menuStyle"
        :query="trigger.query"
        @select="trigger.selectPanel"
        @close="trigger.dismiss"
    />
</template>

<script setup lang="ts">
/**
 * TriggerOverlays - 提示词编辑器的触发器浮层（技能菜单 / 资源树面板）
 *
 * 为什么独立成组件：这段浮层是编辑器能力的一部分，聊天输入框与表单里的提示词编辑器都要用。
 * 留在 ChatInput 模板里，第二个消费方就得复制一遍，而「新增触发类型时补哪儿」也会变成两处。
 *
 * 浮层根 DOM 直接写回 prompt.overlayEl（编辑器定位浮层时要同步读到它），因此这里传的是
 * usePromptEditor 的返回对象而不是拷贝出来的状态。
 */
import { ref, watch } from 'vue';
import { TriggerMenu } from './triggers';
import type { usePromptEditor } from './usePromptEditor';

const props = defineProps<{
    /** usePromptEditor 返回的编辑器句柄（含触发器状态与浮层元素槽位） */
    prompt: ReturnType<typeof usePromptEditor>;
}>();

const trigger = props.prompt.trigger;
const menuRef = ref<InstanceType<typeof TriggerMenu> | null>(null);

/**
 * 面板组件契约：只需 expose panelRef（浮层根 DOM）。
 *
 * 面板由注册表动态提供，这里不依赖任何具体面板组件的类型，否则「动态渲染」会被
 * 一个写死的 import 类型重新绑回资源树
 */
interface PanelHandle {
    panelRef?: HTMLElement | null;
}
const panelRef = ref<PanelHandle | null>(null);

// 列表菜单与面板互斥渲染，谁在场谁就是定位与「点击外部」判定的参照元素。
// flush:'post'：模板 ref 在组件挂载后填充，必须在 DOM 更新完成后写回，
// 否则 useTriggerState.open 的 nextTick 里 start() 会拿不到浮层元素
watch(
    [menuRef, panelRef],
    ([menu, panel]) => {
        props.prompt.overlayEl.value = menu?.menuRef ?? panel?.panelRef ?? null;
    },
    { flush: 'post' },
);
</script>
