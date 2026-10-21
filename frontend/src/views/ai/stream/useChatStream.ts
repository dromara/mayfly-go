/**
 * useChatStream - WebSocket 流式处理
 */
import { createWebSocket } from '@/common/request';
import { Msg } from '@/hooks/useI18n';
import { onBeforeUnmount, ref } from 'vue';
import type { EventMsg, StreamCallbacks } from '../protocol/types';

export function useChatStream() {
    const socket = ref<WebSocket | null>(null);
    const reconnectTimer = ref<ReturnType<typeof setTimeout> | null>(null);
    const reconnectAttempts = ref(0);
    /**
     * 重连次数已耗尽的终态哨兵。
     *
     * 没这个哨兵时：耗尽分支既不置 reconnectTimer 也不标记终态，而 ensureOpen 的轮询条件是
     * `!reconnectTimer.value`，于是每 150ms 再调一次 attemptReconnect —— 10s 超时前会刷出
     * 约 66 条全局错误弹层，并重复调用 onError/onEnd（onError 每次都往 turn 追加一条 notice part）。
     * 任何一次断网态下的 sendStop/sendAttach/sendResume 都会踩到
     */
    const reconnectExhausted = ref(false);

    /** 复位重连状态（连接建立/主动停止/卸载三处共用）；不重置哨兵下次断网就永远不再提示 */
    const resetReconnectState = () => {
        reconnectAttempts.value = 0;
        reconnectExhausted.value = false;
    };
    const MAX_RECONNECT_ATTEMPTS = 5;
    const RECONNECT_DELAY = 3000;
    const currentCallbacks = ref<StreamCallbacks | null>(null);

    const initSocket = async (callbacks: StreamCallbacks) => {
        try {
            currentCallbacks.value = callbacks;
            const ws = await createWebSocket('/ai/chat/ws');
            socket.value = ws;

            ws.onmessage = (e: MessageEvent) => {
                try {
                    const msg: EventMsg = JSON.parse(e.data);
                    dispatchEvent(msg, callbacks);
                } catch (err) {
                    console.error('Failed to parse EventMsg:', err);
                }
            };

            ws.onclose = (event) => {
                if (!event.wasClean) {
                    attemptReconnect(callbacks);
                }
            };

            ws.onerror = () => {};

            // 连接就绪（首次连接/重连成功）：供调用方重新 attach 运行中 turn
            callbacks.onSocketOpen?.();

            resetReconnectState();
        } catch {
            Msg.error('ai.chat.connectionFailed');
            attemptReconnect(callbacks);
        }
    };

    const dispatchEvent = (msg: EventMsg, callbacks: StreamCallbacks) => {
        switch (msg.type) {
            case 'conversation_created':
                callbacks.onConversationCreated?.(msg.conversationId!);
                break;
            case 'turn_started':
                callbacks.onTurnStart?.(msg.turnId!, msg.conversationId);
                break;
            case 'item_started':
                if (msg.item) callbacks.onItemStarted?.(msg.item, msg.turnId!);
                break;
            case 'item_updated':
                if (msg.itemId && msg.delta) callbacks.onItemUpdated?.(msg.itemId, msg.delta, msg.turnId!);
                break;
            case 'item_completed':
                if (msg.item) callbacks.onItemCompleted?.(msg.item, msg.turnId!);
                break;
            case 'interrupted':
                if (msg.interrupt) callbacks.onInterrupted?.(msg.interrupt, msg.turnId!);
                break;
            case 'turn_completed':
                callbacks.onTurnCompleted?.(msg.turnId!, msg.status, msg.usage);
                break;
            case 'error':
                callbacks.onError?.(msg.error || 'Unknown error', msg.errSource);
                break;
            case 'end':
                callbacks.onEnd?.();
                break;
            // turn 运行时订阅协议
            case 'turn_attached':
                callbacks.onTurnAttached?.(msg.turnId!, msg.conversationId);
                break;
            case 'turn_not_running':
                callbacks.onTurnNotRunning?.(msg.conversationId);
                break;
        }
    };

    const attemptReconnect = (callbacks?: StreamCallbacks) => {
        const cb = callbacks || currentCallbacks.value;
        if (!cb) return;

        if (reconnectAttempts.value >= MAX_RECONNECT_ATTEMPTS) {
            // 只报一次并终态化：后续重入直接静默返回（UI 在首次已恢复）
            if (reconnectExhausted.value) return;
            reconnectExhausted.value = true;
            Msg.error('ai.chat.connectionDisconnected');
            // 通知调用方终态化本轮，恢复 UI（停止按钮/输入框），避免卡死在生成中状态
            cb.onError?.('ai.chat.connectionDisconnected');
            cb.onEnd?.();
            return;
        }

        reconnectAttempts.value++;
        if (reconnectTimer.value) clearTimeout(reconnectTimer.value);

        reconnectTimer.value = setTimeout(() => {
            reconnectTimer.value = null;
            initSocket(cb);
        }, RECONNECT_DELAY);
    };

    /**
     * 等待 socket 就绪（attach/stop 等控制消息的静默发送前置）：
     * 已就绪立即返回；连接中/重连调度中等待完成；无连接且未调度重连则触发一次重连。
     * 超时静默返回 false：attach 是后台续流动作，连接不可用时不弹全局提示打扰用户
     */
    const ensureOpen = (timeoutMs = 10000): Promise<boolean> => {
        return new Promise((resolve) => {
            const started = Date.now();
            const check = () => {
                const ws = socket.value;
                if (ws && ws.readyState === WebSocket.OPEN) {
                    resolve(true);
                    return;
                }
                if (Date.now() - started >= timeoutMs) {
                    resolve(false);
                    return;
                }
                // 无连接且不在重连等待中：触发一次重连（次数耗尽走终态化回调）
                if ((!ws || ws.readyState === WebSocket.CLOSED) && !reconnectTimer.value) {
                    attemptReconnect();
                }
                setTimeout(check, 150);
            };
            check();
        });
    };

    /** 发送消息；返回是否成功投递（socket 未就绪时返回 false，供调用方回滚） */
    const sendMessage = (data: Record<string, unknown>): boolean => {
        if (!socket.value || socket.value.readyState !== WebSocket.OPEN) {
            Msg.warning('ai.chat.connectionLost');
            attemptReconnect();
            return false;
        }
        socket.value.send(JSON.stringify(data));
        return true;
    };

    /**
     * 显式停止运行中 turn：发送 stop 协议消息（服务端取消 turn ctx，
     * 真正中断 LLM 请求与工具执行）。不断开连接，终态由
     * turn_completed(status=stopped) 事件驱动。先静默等待连接就绪
     * （停止不能因连接刚断而丢失），失败返回 false
     */
    const sendStop = async (conversationId: number): Promise<boolean> => {
        if (!(await ensureOpen())) return false;
        socket.value!.send(
            JSON.stringify({
                conversationId,
                type: 'stop',
                content: '',
            })
        );
        return true;
    };

    /**
     * 订阅运行中 turn：发送 attach 协议消息（服务端先回放缓存快照再推实时事件）。
     * 刷新页面/断线重连后调用，续上流式输出。静默等待连接就绪，失败返回 false
     */
    const sendAttach = async (conversationId: number): Promise<boolean> => {
        if (!(await ensureOpen())) return false;
        socket.value!.send(
            JSON.stringify({
                conversationId,
                type: 'attach',
                content: '',
            })
        );
        return true;
    };

    /**
     * 关闭连接（仅清理场景使用：组件卸载等；用户停止改用 sendStop 协议消息）
     */
    const stopStream = (): boolean => {
        if (reconnectTimer.value) {
            clearTimeout(reconnectTimer.value);
            reconnectTimer.value = null;
        }
        const ws = socket.value;
        if (!ws) return false;
        // 先摘掉 onclose：主动停止是 clean close，不应进入重连逻辑
        ws.onclose = null;
        ws.onerror = null;
        ws.onmessage = null;
        if (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING) {
            ws.close(1000, 'client stop');
        }
        socket.value = null;
        resetReconnectState();
        return true;
    };

    const cleanup = () => {
        if (reconnectTimer.value) {
            clearTimeout(reconnectTimer.value);
            reconnectTimer.value = null;
        }
        if (socket.value) {
            socket.value.onclose = null;
            socket.value.onerror = null;
            socket.value.onmessage = null;
            if (socket.value.readyState === WebSocket.OPEN || socket.value.readyState === WebSocket.CONNECTING) {
                socket.value.close();
            }
            socket.value = null;
        }
        resetReconnectState();
    };

    onBeforeUnmount(() => {
        cleanup();
    });

    return {
        initSocket,
        sendMessage,
        sendStop,
        sendAttach,
        stopStream,
        cleanup,
        reconnectAttempts,
    };
}
