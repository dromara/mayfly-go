import { buildProgressProps } from '@/components/progress-notify/progress-notify';
import syssocket from './syssocket';
import { h, reactive } from 'vue';
import { ElNotification } from 'element-plus';
import ProgressNotify from '@/components/progress-notify/progress-notify.vue';

export function initSysMsgs() {
    registerDbSqlExecProgress();
}

const sqlExecNotifyMap: Map<string, any> = new Map();

function registerDbSqlExecProgress() {
    syssocket.registerMsgHandler('execSqlFileProgress', function (message: any) {
        const content = JSON.parse(message.msg);
        const id = content.id;
        let progress = sqlExecNotifyMap.get(id);
        if (content.terminated) {
            if (progress != undefined) {
                progress.notification?.close();
                sqlExecNotifyMap.delete(id);
                progress = undefined;
            }
            return;
        }

        if (progress == undefined) {
            progress = {
                props: reactive(buildProgressProps()),
                notification: undefined,
            };
        }

        progress.props.progress.title = content.title;
        progress.props.progress.executedStatements = content.executedStatements;
        if (!sqlExecNotifyMap.has(id)) {
            progress.notification = ElNotification({
                duration: 0,
                title: message.title,
                message: h(ProgressNotify, progress.props),
                type: syssocket.getMsgType(message.type),
                showClose: false,
            });
            sqlExecNotifyMap.set(id, progress);
        }
    });
}
