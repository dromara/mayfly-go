import Config from './config';
import { ElNotification } from 'element-plus';
import SocketBuilder from './SocketBuilder';
import { getToken } from '@/common/utils/storage';

import { joinClientParams } from './request';

class SysSocket {
    /**
     * socket连接
     */
    socket: any;

    /**
     * key -> 消息类别，value -> 消息对应的处理器函数
     */
    categoryHandlers: Map<string, any> = new Map();

    /**
     * 消息类型
     */
    messageTypes = {
        0: 'error',
        1: 'success',
        2: 'info',
    };

    /**
     * 初始化全局系统消息websocket
     */
    init() {
        // 存在则不需要重新建立连接
        if (this.socket) {
            return;
        }
        const token = getToken();
        if (!token) {
            return null;
        }
        console.log('init system ws');
        const sysMsgUrl = `${Config.baseWsUrl}/sysmsg?${joinClientParams()}`;
        this.socket = SocketBuilder.builder(sysMsgUrl)
            .message((event: { data: string }) => {
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

                const type = this.getMsgType(message.type);
                ElNotification({
                    duration: 0,
                    title: message.title,
                    message: message.msg,
                    type: type,
                });
            })
            .open((event: any) => console.log(event))
            .close(() => {
                console.log('close sys socket');
                this.socket = null;
            })
            .build();
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
    registerMsgHandler(category: any, handlerFunc: any) {
        this.init();
        if (this.categoryHandlers.has(category)) {
            console.log(`${category}该类别消息处理器已存在...`);
            return;
        }
        if (typeof handlerFunc != 'function') {
            throw new Error('message handler需为函数');
        }
        this.categoryHandlers.set(category, handlerFunc);
    }

    getMsgType(msgType: any) {
        return this.messageTypes[msgType];
    }
}

// 全局系统消息websocket;
const sysSocket = new SysSocket();

export default sysSocket;
