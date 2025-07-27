import { buildProgressProps } from '@/components/progress-notify/progress-notify';
import syssocket from './syssocket';
import { h, reactive } from 'vue';
import { ElNotification } from 'element-plus';
import ProgressNotify from '@/components/progress-notify/progress-notify.vue';

export async function initSysMsgs() {
    await registerDbSqlExecProgress();
}

const sqlExecNotifyMap: Map<string, any> = new Map();

async function registerDbSqlExecProgress() {
    await syssocket.registerMsgHandler('sqlScriptRunProgress', function (message: any) {
        const content = message.params;
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
                type: 'info',
                showClose: false,
            });
            sqlExecNotifyMap.set(id, progress);
        }
    });
}
