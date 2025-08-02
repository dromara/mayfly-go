import { getToken } from '@/common/utils/storage';

import { createWebSocket } from './request';
import { ElNotification } from 'element-plus';
import { MsgSubtypeEnum } from './commonEnum';
import EnumValue from './Enum';
import { h } from 'vue';
import { MessageRenderer } from '@/components/message/message';

class SysSocket {
    /**
     * socket连接
     */
    socket: WebSocket | null = null;

    /**
     * key -> 消息类别，value -> 消息对应的处理器函数
     */
    categoryHandlers: Map<string, any> = new Map();

    /**
     * 初始化全局系统消息websocket
     */
    async init() {
        // 存在则不需要重新建立连接
        if (this.socket) {
            return;
        }
        const token = getToken();
        if (!token) {
            return null;
        }
        console.log('init system ws');
        try {
            this.socket = await createWebSocket('/sysmsg');
            this.socket.onmessage = async (event: { data: string }) => {
                let message;
                try {
                    message = JSON.parse(event.data);
                } catch (e) {
                    console.error('解析ws消息失败', e);
                    return;
                }

                // 存在消息类别对应的处理器，则进行处理，否则进行默认通知处理
                const handler = this.categoryHandlers.get(message.category);
                if (handler) {
                    handler(message);
                    return;
                }

                const msgSubtype = EnumValue.getEnumByValue(MsgSubtypeEnum, message.subtype);
                if (!msgSubtype) {
                    console.log(`not found msg subtype: ${message.subtype}`);
                    return;
                }

                // 动态导入 i18n 或延迟获取 i18n 实例
                let title = '';
                try {
                    // 方式1: 动态导入
                    const { i18n } = await import('@/i18n');
                    title = i18n.global.t(msgSubtype?.label);
                } catch (e) {
                    console.warn('i18n not ready, using default title');
                }

                ElNotification({
                    duration: 0,
                    title,
                    message: h(MessageRenderer, { content: message.msg }),
                    type: msgSubtype?.extra.notifyType || 'info',
                });
            };
        } catch (e) {
            console.error('open system ws error', e);
        }
    }

    destory() {
        this.socket?.close();
        this.socket = null;
        this.categoryHandlers?.clear();
    }

    /**
     * 注册消息处理函数
     *
     * @param category 消息类别
     * @param handlerFunc 消息处理函数
     */
    async registerMsgHandler(category: any, handlerFunc: any) {
        if (this.categoryHandlers.has(category)) {
            console.log(`${category}该类别消息处理器已存在...`);
            return;
        }
        if (typeof handlerFunc != 'function') {
            throw new Error('message handler需为函数');
        }
        this.categoryHandlers.set(category, handlerFunc);
    }
}

// 全局系统消息websocket;
const sysSocket = new SysSocket();

export default sysSocket;
