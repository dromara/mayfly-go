<template>
    <div>
        <el-drawer @open="initSort" :title="title" v-model="visible" :before-close="cancel" :destroy-on-close="true" :close-on-click-modal="false">
            <template #header>
                <DrawerHeader :header="title" :back="cancel" />
            </template>

            <el-form :model="form" ref="formRef" :rules="rules" label-width="auto">
                <el-form-item prop="name" label="名称">
                    <el-input v-model.trim="form.name" placeholder="请输入流程名称" auto-complete="off" clearable></el-input>
                </el-form-item>
                <el-form-item prop="defKey" label="key">
                    <el-input :disabled="form.id" v-model.trim="form.defKey" placeholder="请输入流程key" auto-complete="off" clearable></el-input>
                </el-form-item>
                <el-form-item prop="status" label="状态">
                    <el-select v-model="form.status" placeholder="请选择状态">
                        <el-option v-for="item in ProcdefStatus" :key="item.value" :label="item.label" :value="item.value"> </el-option>
                    </el-select>
                </el-form-item>
                <el-form-item prop="remark" label="备注">
                    <el-input v-model.trim="form.remark" placeholder="备注" auto-complete="off" clearable></el-input>
                </el-form-item>

                <el-divider content-position="left">审批节点</el-divider>

                <el-table ref="taskTableRef" :data="tasks" row-key="taskKey" stripe style="width: 100%">
                    <el-table-column prop="name" label="名称" min-width="100px">
                        <template #header>
                            <el-button class="ml0" type="primary" circle size="small" icon="Plus" @click="addTask()"> </el-button>
                            <span class="ml10">节点名称</span>
                            <el-tooltip content="点击指定节点可进行拖拽排序" placement="top">
                                <el-icon class="ml5">
                                    <question-filled />
                                </el-icon>
                            </el-tooltip>
                        </template>
                        <template #default="scope">
                            <el-input v-model="scope.row.name"> </el-input>
                        </template>
                    </el-table-column>
                    <el-table-column prop="userId" label="审核人员" min-width="150px" show-overflow-tooltip>
                        <template #default="scope">
                            <AccountSelectFormItem v-model="scope.row.userId" label="" />
                        </template>
                    </el-table-column>
                    <el-table-column label="操作" width="60px">
                        <template #default="scope">
                            <el-link @click="deleteTask(scope.$index)" class="ml5" type="danger" icon="delete" plain></el-link>
                        </template>
                    </el-table-column>
                </el-table>
            </el-form>

            <template #footer>
                <div>
                    <el-button @click="cancel()">取 消</el-button>
                    <el-button type="primary" :loading="saveBtnLoading" @click="btnOk">确 定</el-button>
                </div>
            </template>
        </el-drawer>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, watch, ref, nextTick } from 'vue';
import { procdefApi } from './api';
import { ElMessage } from 'element-plus';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';
import AccountSelectFormItem from '@/views/system/account/components/AccountSelectFormItem.vue';
import Sortable from 'sortablejs';
import { randomUuid } from '../../common/utils/string';
import { ProcdefStatus } from './enums';

const props = defineProps({
    data: {
        type: [Boolean, Object],
    },
    title: {
        type: String,
    },
});

const visible = defineModel<boolean>('visible', { default: false });

//定义事件
const emit = defineEmits(['cancel', 'val-change']);

const formRef: any = ref(null);
const taskTableRef: any = ref(null);

const rules = {
    name: [
        {
            required: true,
            message: '请输入流程名称',
            trigger: ['change', 'blur'],
        },
    ],
    defKey: [
        {
            required: true,
            message: '请输入流程key',
            trigger: ['change', 'blur'],
        },
    ],
};

const state = reactive({
    tasks: [] as any,
    form: {
        id: null,
        name: null,
        defKey: null,
        status: null,
        remark: null,
        // 流程的审批节点任务
        tasks: '',
    },
    sortable: '' as any,
});

const { form, tasks } = toRefs(state);

const { isFetching: saveBtnLoading, execute: saveFlowDefExec } = procdefApi.save.useApi(form);

watch(props, (newValue: any) => {
    if (newValue.data) {
        state.form = { ...newValue.data };
        const tasks = JSON.parse(state.form.tasks);
        tasks.forEach((t: any) => {
            t.userId = Number.parseInt(t.userId);
        });
        state.tasks = tasks;
    } else {
        state.form = { status: ProcdefStatus.Enable.value } as any;
        state.tasks = [];
    }
});

const initSort = () => {
    nextTick(() => {
        const table = taskTableRef.value.$el.querySelector('table > tbody') as any;
        state.sortable = Sortable.create(table, {
            animation: 200,
            //拖拽结束事件
            onEnd: (evt) => {
                const curRow = state.tasks.splice(evt.oldIndex, 1)[0];
                state.tasks.splice(evt.newIndex, 0, curRow);
            },
        });
    });
};

const addTask = () => {
    state.tasks.push({ taskKey: randomUuid() });
};

const deleteTask = (idx: any) => {
    state.tasks.splice(idx, 1);
};

const btnOk = async () => {
    formRef.value.validate(async (valid: boolean) => {
        if (!valid) {
            ElMessage.error('表单填写有误');
            return false;
        }
        const checkRes = checkTasks();
        if (checkRes.err) {
            ElMessage.error(checkRes.err);
            return false;
        }

        state.form.tasks = JSON.stringify(checkRes.tasks);
        await saveFlowDefExec();
        ElMessage.success('操作成功');
        emit('val-change', state.form);
        //重置表单域
        formRef.value.resetFields();
        state.form = {} as any;
    });
};

const checkTasks = () => {
    if (state.tasks?.length == 0) {
        return { err: '请完善审批节点任务' };
    }

    const tasks = [];
    for (let i = 0; i < state.tasks.length; i++) {
        const task = { ...state.tasks[i] };
        if (!task.name || !task.userId) {
            return { err: `请完善第${i + 1}个审批节点任务信息` };
        }
        // 转为字符串(方便后续万一需要调整啥的)
        task.userId = `${task.userId}`;
        if (!task.taskKey) {
            task.taskKey = randomUuid();
        }
        tasks.push(task);
    }

    return { tasks: tasks };
};

const cancel = () => {
    visible.value = false;
    emit('cancel');
};
</script>
<style lang="scss"></style>
