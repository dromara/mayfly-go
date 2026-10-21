<template>
    <div class="run-transcript">
        <MessageScrollerProvider :auto-scroll="false" default-scroll-position="end">
            <MessageList
                :messages="messages"
                :loading="loading"
                :has-more="hasMore"
                :loading-more="loadingMore"
                :conversation-id="conversationId"
                :pending-interrupts="[]"
                mode="transcript"
                :empty-text="t('agentops.transcriptEmpty')"
                @load-more="loadMoreMessages"
            />
        </MessageScrollerProvider>
    </div>
</template>

<script setup lang="ts" name="RunTranscript">
/**
 * RunTranscript - 一次运维运行的对话轨迹（只读）
 *
 * 职责边界：只做「按 conversationId 拉历史 + 交给只读列表渲染」。没有输入框、不建 WebSocket、
 * 不订阅运行中 turn、不承载中断决策，也不显示会话侧栏——查看一份产物不该顺手获得改它的能力。
 *
 * 消息装配复用聊天页的同一套事件重放（useChatMessages → store.loadTurnGroups），因此历史回放
 * 与实时输出的形态天然一致，TurnItem 加字段不需要两处维护。
 *
 * 为什么不复用 ChatContainer：那是「可交互会话」的编排层（socket、attach、待发队列、自动命名），
 * 只读查看面继承它就得反向禁用一堆能力，等于把刚补上的写侧授权缺口又留在原地。
 */
import { computed, onBeforeUnmount, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { MessageScrollerProvider } from '@/components/ui/message-scroller';
import MessageList from '../message/MessageList.vue';
import { useChatMessages } from '../hooks/useChatMessages';
import { useConvSlice } from '../hooks/useConvSlice';
import { useChatStore } from '../stores/chatStore';

const props = defineProps<{
    /** 运行关联的会话 id（无痕运行为 0，父级据此决定是否渲染本组件） */
    conversationId: number;
}>();

const { t } = useI18n();
const store = useChatStore();

const convId = computed(() => props.conversationId);
const { messages, hasMore, loadingMore } = useConvSlice(convId);
const { loading, loadMoreMessages, reloadMessages } = useChatMessages(convId);

// 打开或切换运行时都重新拉取：运行可能仍在进行，复用切片缓存会停在旧快照。
// reloadMessages 先清旧消息再取最新一批，分页游标（oldestTurnId/hasMore）随之重置
watch(
    convId,
    (id) => {
        if (id) void reloadMessages();
    },
    { immediate: true }
);

// 关闭抽屉时释放本次轨迹的消息：一次运维运行可能有几十轮带工具明细的消息，
// 而聊天页的 LRU 只淘汰 active=false 的切片，不主动回收就会随浏览历史一直增长
onBeforeUnmount(() => {
    if (props.conversationId) store.resetSlice(props.conversationId);
});
</script>

<style scoped>
.run-transcript {
    display: flex;
    flex-direction: column;
    flex: 1;
    height: 100%;
    min-height: 0;
}
</style>
