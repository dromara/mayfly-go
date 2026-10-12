<template>
    <!-- flex 行内对齐：按钮与「更多」触发器同基线（不靠逐个控件的 margin/line-height 碰运气） -->
    <div class="row-actions" :class="{ 'is-open': dropdownOpen }" :style="{ '--row-actions-gap': ACTION_BUTTON_GAP + 'px' }">
        <RowActionEntry v-for="item in inlineItems" :key="item.key" :item="item" :data="data" :options="options" />

        <!-- 超出内联上限的次要动作折进「更多」（悬停展开，与表格行内菜单的常规交互一致）：
             折叠项一律带文字，避免「⋯ 里塞一个动作」等于把功能藏起来 -->
        <el-dropdown v-if="foldedItems.length" @visible-change="dropdownOpen = $event" @command="onFolded">
            <el-button link type="primary" :size="options?.size" :aria-label="$t(ACTION_MORE_LABEL)">
                {{ $t(ACTION_MORE_LABEL) }}
                <el-icon class="el-icon--right"><arrow-down /></el-icon>
            </el-button>
            <template #dropdown>
                <el-dropdown-menu>
                    <el-dropdown-item
                        v-for="item in foldedItems"
                        :key="item.key"
                        :command="item.key"
                        :icon="item.icon"
                        :disabled="isActionBlocked(item, data)"
                    >
                        <span :class="item.type === 'danger' ? 'text-red-500' : ''">{{ actionLabel(item, data) }}</span>
                    </el-dropdown-item>
                </el-dropdown-menu>
            </template>
        </el-dropdown>
    </div>
</template>

<script lang="ts" setup>
import { computed, ref } from 'vue';
import { ArrowDown } from '@element-plus/icons-vue';
import RowActionEntry from './RowActionEntry';
import { runRowAction, actionLabel, isActionBlocked } from './rowAction';
import { planRowActions, ACTION_BUTTON_GAP, ACTION_MORE_LABEL, type ActionItem, type ActionOptions } from './index';

const props = defineProps<{
    /** 该列声明的行内操作项（含账号权限与行级显隐条件） */
    items: ActionItem[];
    /** 当前行数据 */
    data: Record<string, unknown>;
    /** 展示选项：与列宽计宽读同一个对象（由 TableColumn.setActions 存下并下发） */
    options?: ActionOptions;
}>();

/**
 * 折叠按「本行可见的那组按钮」算：先过行级 show，再按账号权限过滤与折叠。
 * 不能先折叠再过滤：否则「运行」这类主动作会被行状态挤进「更多」里（本行只该看到三个按钮时，
 * 第四个不该藏在菜单后面）。列宽按并集预留，本行最多也不会超出。
 */
const plan = computed(() =>
    planRowActions(props.items.filter((item) => !item.show || item.show(props.data)), props.options?.maxInline)
);

const inlineItems = computed(() => plan.value.inline);
const foldedItems = computed(() => plan.value.folded);

/** 菜单是否展开：用于区分「键盘 Tab 到触发器」与「展开时 EP 给触发器编程式 focus」 */
const dropdownOpen = ref(false);

// 菜单里的项没有就地气泡的位置，带 confirm 的项统一退回确认框
const onFolded = (key: string) => {
    const item = foldedItems.value.find((it) => it.key === key);
    if (item) {
        runRowAction(item, props.data, { folded: true });
    }
};
</script>

<style scoped>
.row-actions {
    display: inline-flex;
    /* 必须 nowrap：换行后内容不再横向溢出，“只量溢出”的 DOM 实测回填就看不见欠宽，
       列会永久停在算窄的宽度上静默折行；nowrap 下真放不下会横向溢出并被回填补宽 */
    flex-wrap: nowrap;
    align-items: center;
    /* 间距由脚本里的 ACTION_BUTTON_GAP 注入（同一个真源），改常量即同时改到计宽与渲染 */
    gap: var(--row-actions-gap);
}

/* 改用 flex gap 统一间距，否则与 Element Plus 相邻按钮的 margin-left 叠加成 24px */
.row-actions :deep(.el-button + .el-button) {
    margin-left: 0;
}

/*
 * 展开菜单时 EP 会把焦点给到触发按钮，Chrome 据此命中 :focus-visible 而画出一圈描边（悬停展开也一样），
 * 看起来像多了一个边框。展开本身就是反馈，所以只在展开态去掉描边；
 * 键盘 Tab 停在未展开的触发器上时描边仍在，焦点可见性不丢。
 */
.row-actions.is-open :deep(.el-dropdown .el-button:focus-visible) {
    outline: none;
}
</style>
