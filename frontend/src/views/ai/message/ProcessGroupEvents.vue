<template>
    <!-- 按事件顺序逐项分发（renderEventList） -->
    <div class="process-group-events">
        <template v-for="part in events" :key="part.id">
            <ReasoningPart v-if="part.type === 'reasoning'" :content="part.text" :active="followActive ? part.active : false" />
            <ToolCallPart
                v-else-if="part.type === 'tool_call'"
                :tool-name="part.toolName"
                :arguments="part.arguments"
                :output="part.output"
                :status="part.status"
                :tool-call-id="part.toolCallId"
                :duration-ms="part.durationMs"
                :resume-type="part.resumeType"
                :interrupt-kind="part.interruptKind"
                :turn-id="turnId"
                :pending-interrupts="pendingInterrupts"
                @interrupt-action="(a) => $emit('interrupt-action', a)"
            />
            <!-- 未登记的段类型必须可见：此前 v-if 链之外没有任何分支，新增一种段会在过程区
                 区凭空消失且无任何报错（最坏的内容丢失是静默丢失） -->
            <div v-else class="process-group__unknown">
                {{ $t('ai.chat.unknownPartType', { type: part.type }) }}
            </div>
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
.process-group__unknown {
    padding: 4px 0;
    font-size: 12px;
    color: var(--el-text-color-secondary);
}
.process-group-events {
    display: flex;
    flex-direction: column;
    gap: 4px;
}
</style>
