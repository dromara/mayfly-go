<template>
    <el-steps align-center :active="stepActive">
        <el-step v-for="task in tasksArr" :status="getStepStatus(task)" :title="task.name" :key="task.taskKey">
            <template #description>
                <div>{{ `${task.accountUsername}(${task.accountName})` }}</div>
                <div v-if="task.completeTime">{{ `${dateFormat(task.completeTime)}` }}</div>
                <div v-if="task.remark">{{ task.remark }}</div>
            </template>
        </el-step>
    </el-steps>
</template>

<script lang="ts" setup>
import { toRefs, reactive, watch, onMounted } from 'vue';
import { accountApi } from '../../system/api';
import { ProcinstTaskStatus } from '../enums';
import { dateFormat } from '@/common/utils/date';
import { procdefApi } from '../api';
import { ElSteps, ElStep } from 'element-plus';

const props = defineProps({
    // 流程定义任务
    tasks: {
        type: [String, Object],
    },
    procdefKey: {
        type: String,
    },
    // 流程实例任务列表
    procinstTasks: {
        type: [Array],
    },
});

const state = reactive({
    tasksArr: [] as any,
    stepActive: 0,
});

const { tasksArr, stepActive } = toRefs(state);

watch(
    () => props.tasks,
    (newValue: any) => {
        parseTasks(newValue);
    }
);

watch(
    () => props.procinstTasks,
    () => {
        parseTasks(props.tasks);
    }
);

watch(
    () => props.procdefKey,
    async (newValue: any) => {
        if (newValue) {
            parseTasksByKey(newValue);
        }
    }
);

onMounted(() => {
    if (props.procdefKey) {
        parseTasksByKey(props.procdefKey);
        return;
    }
    parseTasks(props.tasks);
});

const parseTasksByKey = async (key: string) => {
    const procdef = await procdefApi.getByKey.request({ key });
    parseTasks(procdef.tasks);
};

const parseTasks = async (tasksStr: any) => {
    if (!tasksStr) return;
    const tasks = JSON.parse(tasksStr);
    const userIds = tasks.map((x: any) => x.userId);
    const usersRes = await accountApi.querySimple.request({ ids: [...new Set(userIds)].join(','), pageSize: 50 });
    const users = usersRes.list;
    // 将数组转换为 Map 结构，以 id 为 key
    const userMap = users.reduce((acc: any, obj: any) => {
        acc.set(obj.id, obj);
        return acc;
    }, new Map());

    // 流程实例任务（用于显示完成时间，完成到哪一步等）
    let instTasksMap: any;
    if (props.procinstTasks) {
        state.stepActive = props.procinstTasks.length - 1;
        instTasksMap = props.procinstTasks.reduce((acc: any, obj: any) => {
            acc.set(obj.taskKey, obj);
            return acc;
        }, new Map());
    }

    for (let task of tasks) {
        const user = userMap.get(Number.parseInt(task.userId));
        task.accountUsername = user.username;
        task.accountName = user.name;

        // 存在实例任务，则赋值实例任务对应的完成时间和备注
        const instTask = instTasksMap?.get(task.taskKey);
        if (instTask) {
            task.status = instTask.status;
            task.completeTime = instTask.endTime;
            task.remark = instTask.remark;
        }
    }

    state.tasksArr = tasks;
};

const getStepStatus = (task: any): any => {
    const taskStatus = task.status;
    if (!taskStatus) {
        return 'wait';
    }

    if (taskStatus == ProcinstTaskStatus.Pass.value) {
        return 'success';
    }
    if (taskStatus == ProcinstTaskStatus.Process.value) {
        return 'proccess';
    }
    if (taskStatus == ProcinstTaskStatus.Back.value || taskStatus == ProcinstTaskStatus.Reject.value) {
        return 'error';
    }

    return 'wait';
};
</script>
<style lang="scss"></style>
