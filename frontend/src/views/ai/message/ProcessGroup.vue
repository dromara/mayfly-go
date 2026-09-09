<template>
    <div class="process-group">
        <!-- 活跃/待审批：内联渲染（跟随 part.active 流式高亮） -->
        <div v-if="isActive || isPendingApproval" class="process-group--inline">
            <ProcessGroupEvents
                :events="events"
                :follow-active="true"
                :turn-id="turnId"
                :pending-interrupts="pendingInterrupts"
                @interrupt-action="(a) => $emit('interrupt-action', a)"
            />
        </div>

        <!-- 单步已完成：直接内联 -->
        <div v-else-if="isSingleCompleted" class="process-group--single">
            <ProcessGroupEvents
                :events="events"
                :turn-id="turnId"
                :pending-interrupts="pendingInterrupts"
                @interrupt-action="(a) => $emit('interrupt-action', a)"
            />
        </div>

        <!-- 多步已完成：紧凑摘要行 + CollapsibleSection 详情 -->
        <CollapsibleSection v-else :title="summaryText" :summary="stepCountText">
            <ProcessGroupEvents
                :events="events"
                :turn-id="turnId"
                :pending-interrupts="pendingInterrupts"
                @interrupt-action="(a) => $emit('interrupt-action', a)"
            />
        </CollapsibleSection>
    </div>
</template>

<script setup lang="ts">
/**
 * ProcessGroup - 连续过程项渐进披露组件
 *
 * 三级展示：
 * 1. 活跃中/待审批：内联渲染
 * 2. 单步已完成：直接内联
 * 3. 多步已完成：紧凑摘要行 + 可折叠详情
 *
 * 摘要策略：思考作为背景活动最多显示一次，工具调用等离散动作逐个展示
 * 格式：思考 · 工具A · 工具B · +N
 */
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import type { MessagePart, ReasoningPart as ReasoningPartType, ToolCallPart as ToolCallPartType, InterruptEvent } from '../protocol/types';
import type { InterruptActionEvent } from '../interrupt/types';
import { isInterruptDecided } from '../registries/statuses';
import CollapsibleSection from './CollapsibleSection.vue';
import ProcessGroupEvents from './ProcessGroupEvents.vue';

const props = defineProps<{
    /** 连续过程 parts 列表 */
    events: MessagePart[];
    /** 组状态 */
    status: 'active' | 'completed' | 'interrupted';
    /** 待处理中断列表（传递给 ToolCallPart 进行内联渲染） */
    pendingInterrupts?: InterruptEvent[];
    /** 所属 turn ID */
    turnId?: string;
}>();

const emit = defineEmits<{
    (e: 'interrupt-action', action: InterruptActionEvent): void;
}>();

const { t } = useI18n();

/** 是否活跃中 */
const isActive = computed(() => props.status === 'active');

/** 是否待审批（检查是否有 pendingInterrupt 匹配当前组内的 tool_call；已决策的不算） */
const isPendingApproval = computed(() => {
    if (!props.pendingInterrupts?.length) return false;
    const tcParts = toolCallParts.value;
    // 检查是否有未决策中断匹配组内的某个 tool_call
    return props.pendingInterrupts.some((interrupt) =>
        !isInterruptDecided(interrupt.status)
        && tcParts.some((tc) => tc.toolCallId && interrupt.toolCallId === tc.toolCallId),
    );
});

/** 是否单步已完成 */
const isSingleCompleted = computed(
    () => props.status === 'completed' && props.events.length === 1,
);

/** 推理 parts */
const reasoningParts = computed(() =>
    props.events.filter((p): p is ReasoningPartType => p.type === 'reasoning'),
);

/** 工具调用 parts */
const toolCallParts = computed(() =>
    props.events.filter((p): p is ToolCallPartType => p.type === 'tool_call'),
);

/** 可见计数 */
const stepCountText = computed(() => `${props.events.length} ${t('ai.chat.steps')}`);

/**
 * 摘要文本
 * 策略：思考作为背景活动最多显示一次；连续同名工具 run-length 合并
 * （排查场景常连续调同一工具，逐个展示只会重复同一名字）
 * 格式：思考 · 工具A ×3 · 工具B · +N
 */
const summaryText = computed(() => {
    const parts: string[] = [];
    if (reasoningParts.value.length > 0) {
        parts.push(t('ai.chat.thinking'));
    }
    // 连续同名工具合并为 Name ×N
    const actions = toolCallParts.value.map((p) => p.toolName);
    const merged: { name: string; count: number }[] = [];
    for (const name of actions) {
        const last = merged[merged.length - 1];
        if (last && last.name === name) {
            last.count++;
        } else {
            merged.push({ name, count: 1 });
        }
    }
    const MAX_DISPLAY = 3;
    const remainingSlots = MAX_DISPLAY - parts.length;
    const truncated = merged.length > remainingSlots;
    const shown = merged.slice(0, remainingSlots);
    for (const { name, count } of shown) {
        parts.push(count > 1 ? `${name} ×${count}` : name);
    }
    if (truncated) {
        const hiddenSteps = actions.length - shown.reduce((sum, m) => sum + m.count, 0);
        parts.push(`+${hiddenSteps}`);
    }
    return parts.join(' · ');
});
</script>

<style scoped>
.process-group {
    font-size: 13px;
    margin: 2px 0;
}

/* 事件列表与摘要行样式分别收敛到 ProcessGroupEvents / CollapsibleSection */
</style>
