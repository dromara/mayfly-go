import { computed, unref, type MaybeRef } from 'vue';
import { useChatStore } from '../stores/chatStore';

/**
 * useConvSlice - 按会话读取「消息切片」的只读视图
 *
 * 存在理由：消息列表容器（聊天页 ChatContainer、运维轨迹 RunTranscript）都需要同一组切片派生
 * （messages / hasMore / loadingMore / pendingInterrupts）。散在各容器里各写一份时，切片字段改名
 * 或补一个派生态就要改多处，且容易出现某个容器少读一项的静默不一致。
 *
 * 刻意只暴露读：写操作仍走 store 的显式 action（loadTurnGroups / addMessage 等），
 * 避免这个便捷层变成可以随手改切片的后门
 */
export function useConvSlice(convId: MaybeRef<number>) {
    const store = useChatStore();
    const slice = computed(() => store.getSlice(unref(convId)));

    return {
        /** 会话消息（未创建切片时为空列表，模板侧无需判空） */
        messages: computed(() => slice.value?.messages ?? []),
        /** 仍有更早的历史可加载（顶部翻页触发条件） */
        hasMore: computed(() => slice.value?.hasMoreMessages ?? false),
        /** 正在加载更早的历史（翻页 spinner 与防重复） */
        loadingMore: computed(() => slice.value?.isLoadingMoreMessages ?? false),
        /** 待处理中断（只读消费方应恒为空：查看面不承载决策） */
        pendingInterrupts: computed(() => slice.value?.pendingInterrupts ?? []),
    };
}
