import { createWebSocket } from '@/common/request';
import { Msg } from '@/hooks/useI18n';
import { onBeforeUnmount, ref, type Ref } from 'vue';

/**
 * AI Chat WebSocket 连接管理 Hook
 * 负责 WebSocket 连接、重连、消息收发等
 */
export function useAiChatWebSocket(onMessage: (data: Record<string, unknown>) => void, currentSessionId: Ref<string>, isNewSession: Ref<boolean>) {
    const socket = ref<WebSocket | null>(null);
    const reconnectTimer = ref<ReturnType<typeof setTimeout> | null>(null);
    const reconnectAttempts = ref(0);
    const MAX_RECONNECT_ATTEMPTS = 5;
    const RECONNECT_DELAY = 3000;

    /**
     * 初始化 WebSocket 连接
     */
    const initSocket = async () => {
        try {
            const ws = await createWebSocket(`/ai/chat`);
            socket.value = ws;

            ws.onmessage = (e) => {
                const data = JSON.parse(e.data);

                // 会话隔离：只处理属于当前激活会话的消息
                if (data.sessionId && data.sessionId !== currentSessionId.value) {
                    // 新会话首次收到后端返回的真实 sessionId，更新并通知父组件
                    if (isNewSession.value) {
                        currentSessionId.value = data.sessionId;
                    } else {
                        return;
                    }
                }

                onMessage(data);
            };

            ws.onclose = (event) => {
                if (!event.wasClean) {
                    attemptReconnect();
                }
            };

            ws.onerror = () => {
                // WebSocket 错误由浏览器自动触发 onclose，重连逻辑已在 onclose 中处理
            };

            // 连接成功，重置重连计数
            reconnectAttempts.value = 0;
        } catch (e) {
            // 直接显示错误提示，不传递到消息处理器
            Msg.error('ai.chat.connectionFailed');
            attemptReconnect();
        }
    };

    /**
     * 尝试重连
     */
    const attemptReconnect = () => {
        if (reconnectAttempts.value >= MAX_RECONNECT_ATTEMPTS) {
            console.warn('达到最大重连次数，停止重连');
            Msg.error('ai.chat.connectionDisconnected');
            return;
        }

        reconnectAttempts.value++;

        if (reconnectTimer.value) {
            clearTimeout(reconnectTimer.value);
        }

        reconnectTimer.value = setTimeout(() => {
            initSocket();
        }, RECONNECT_DELAY);
    };

    /**
     * 清理 WebSocket 连接
     */
    const cleanupSocket = () => {
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
        reconnectAttempts.value = 0;
    };

    /**
     * 发送消息
     */
    const sendMessage = (type: 'text' | 'interruptResume', content: string) => {
        // 检查 WebSocket 连接状态
        if (!socket.value || socket.value.readyState === WebSocket.CLOSED || socket.value.readyState === WebSocket.CLOSING) {
            console.warn('WebSocket 连接已关闭，尝试重连...');

            // 如果正在重连中，等待重连完成
            if (reconnectAttempts.value > 0 && reconnectAttempts.value < MAX_RECONNECT_ATTEMPTS) {
                Msg.warning('ai.chat.reconnecting');
                attemptReconnect();
                return;
            }

            // 立即尝试重连
            attemptReconnect();
            Msg.error('ai.chat.connectionLost');
            return;
        }

        socket.value.send(
            JSON.stringify({
                type,
                sessionId: currentSessionId.value,
                content,
            })
        );
    };

    /**
     * 获取当前连接状态
     */
    const isConnected = () => {
        return socket.value && socket.value.readyState === WebSocket.OPEN;
    };

    // 组件卸载时清理连接
    onBeforeUnmount(() => {
        cleanupSocket();
    });

    return {
        initSocket,
        sendMessage,
        reconnectAttempts,
        MAX_RECONNECT_ATTEMPTS,
    };
}
