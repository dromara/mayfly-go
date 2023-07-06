import Config from './config';
import { ElNotification } from 'element-plus';
import SocketBuilder from './SocketBuilder';
import { getSession } from '@/common/utils/storage';

export default {
    /**
     * 全局系统消息websocket
     */
    sysMsgSocket() {
        const token = getSession('token');
        if (!token) {
            return null;
        }
        return SocketBuilder.builder(`${Config.baseWsUrl}/sysmsg?token=${token}`)
            .message((event: { data: string }) => {
                const message = JSON.parse(event.data);
                let mtype: string;
                switch (message.type) {
                    case 0:
                        mtype = 'error';
                        break;
                    case 2:
                        mtype = 'info';
                        break;
                    case 1:
                        mtype = 'success';
                        break;
                    default:
                        mtype = 'info';
                }
                if (mtype == undefined) {
                    return;
                }
                ElNotification({
                    duration: 0,
                    title: message.title,
                    message: message.msg,
                    type: mtype as any,
                });
            })
            .open((event: any) => console.log(event))
            .build();
    },
};
