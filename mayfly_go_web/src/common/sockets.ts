import Config from './config';
import { ElNotification, NotificationHandle } from 'element-plus';
import SocketBuilder from './SocketBuilder';
import { getToken } from '@/common/utils/storage';
import { createVNode, reactive } from "vue";
import { buildProgressProps } from "@/components/progress-notify/progress-notify";
import ProgressNotify from '/src/components/progress-notify/progress-notify.vue';

export default {
    /**
     * 全局系统消息websocket
     */
    sysMsgSocket() {
        const token = getToken();
        if (!token) {
            return null;
        }
        const messageTypes = {
            0: "error",
            1: "success",
            2: "info",
        }
        const notifyMap: Map<Number, any> = new Map()

        return SocketBuilder.builder(`${Config.baseWsUrl}/sysmsg?token=${token}`)
            .message((event: { data: string }) => {
                const message = JSON.parse(event.data);
                const type = messageTypes[message.type]
                switch (message.category) {
                    case "execSqlFileProgress":
                        const content = JSON.parse(message.msg)
                        const id = content.id
                        let progress = notifyMap.get(id)
                        if (content.terminated) {
                            if (progress != undefined) {
                                progress.notification?.close()
                                notifyMap.delete(id)
                                progress = undefined
                            }
                            return
                        }
                        if (progress == undefined) {
                            progress = {
                                props: reactive(buildProgressProps()),
                                notification: undefined,
                            }
                        }
                        progress.props.progress.sqlFileName = content.sqlFileName
                        progress.props.progress.executedStatements = content.executedStatements
                        if (!notifyMap.has(id)) {
                            const vNodeMessage = createVNode(
                                ProgressNotify,
                                progress.props,
                                null,
                            )
                            progress.notification = ElNotification({
                                duration: 0,
                                title: message.title,
                                message: vNodeMessage,
                                type: type,
                                showClose: false,
                            });
                            notifyMap.set(id, progress)
                        }
                        break;
                    default:
                        ElNotification({
                            duration: 0,
                            title: message.title,
                            message: message.msg,
                            type: type,
                        });
                        break;
                }
            })
            .open((event: any) => console.log(event))
            .build();
    },
};
