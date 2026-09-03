<template>
    <!-- 按事件顺序逐项分发（对齐 tokhub renderEventList） -->
    <div class="process-group-events">
        <template v-for="part in events" :key="part.id">
            <ReasoningPart
                v-if="part.type === 'reasoning'"
                :content="part.text"
                :active="followActive ? part.active : false"
            />
            <ToolCallPart
                v-else-if="part.type === 'tool_call'"
                :tool-name="part.toolName"
                :arguments="part.arguments"
                :output="part.output"
                :status="part.status"
                :tool-call-id="part.toolCallId"
                :duration-ms="part.durationMs"
                :resume-type="part.resumeType"
                :turn-id="turnId"
                :pending-interrupts="pendingInterrupts"
                @interrupt-action="(a) => $emit('interrupt-action', a)"
            />
        </template>
    </div>
</template>

<script setup lang="ts">
/**
 * ProcessGroupEvents - 过程事件列表渲染
 * ProcessGroup 三种展示形态（内联 / 单步 / 折叠详情）共用的渲染块，
 * 差异仅在于是否跟随 part.active 流式高亮（followActive）
 */
import type { MessagePart, InterruptEvent } from '../protocol/types';
import type { InterruptActionEvent } from '../interrupt/types';
import ReasoningPart from './events/ReasoningPart.vue';
import ToolCallPart from './events/ToolCallPart.vue';

defineProps<{
    /** 连续过程 parts 列表 */
    events: MessagePart[];
    /** 是否跟随 part.active 流式高亮（仅活跃形态为 true） */
    followActive?: boolean;
    /** 待处理中断列表（传递给 ToolCallPart 进行内联渲染） */
    pendingInterrupts?: InterruptEvent[];
    /** 所属 turn ID */
    turnId?: string;
}>();

defineEmits<{
    (e: 'interrupt-action', action: InterruptActionEvent): void;
}>();
</script>

<style scoped>
.process-group-events {
    display: flex;
    flex-direction: column;
    gap: 4px;
}
</style>
