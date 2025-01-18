<template>
    <div>
        <el-drawer @open="initSort" :title="title" v-model="visible" :before-close="cancel" :destroy-on-close="true" :close-on-click-modal="false" size="40%">
            <template #header>
                <DrawerHeader :header="title" :back="cancel" />
            </template>

            <el-form :model="form" ref="formRef" :rules="rules" label-width="auto">
                <el-form-item prop="name" :label="$t('common.name')">
                    <el-input v-model.trim="form.name" auto-complete="off" clearable></el-input>
                </el-form-item>
                <el-form-item prop="defKey" label="Key">
                    <el-input :disabled="form.id" v-model.trim="form.defKey" auto-complete="off" clearable></el-input>
                </el-form-item>
                <el-form-item prop="status" :label="$t('common.status')">
                    <EnumSelect :enums="ProcdefStatus" v-model="form.status" />
                </el-form-item>

                <FormItemTooltip prop="condition" :label="$t('flow.triggeringCondition')" :tooltip="$t('flow.triggeringConditionTips')">
                    <el-input
                        v-model="form.condition"
                        :rows="10"
                        type="textarea"
                        :placeholder="$t('flow.conditionPlaceholder')"
                        auto-complete="off"
                        clearable
                    ></el-input>
                </FormItemTooltip>

                <el-form-item prop="remark" :label="$t('common.remark')">
                    <el-input v-model.trim="form.remark" auto-complete="off" clearable></el-input>
                </el-form-item>

                <el-form-item ref="tagSelectRef" prop="codePaths" :label="$t('tag.relateTag')">
                    <tag-tree-check height="300px" v-model="form.codePaths" :tag-type="[TagResourceTypePath.Db, TagResourceTypeEnum.Redis.value]" />
                </el-form-item>

                <el-divider content-position="left">{{ $t('flow.approvalNode') }}</el-divider>

                <el-table ref="taskTableRef" :data="tasks" row-key="taskKey" stripe style="width: 100%">
                    <el-table-column prop="name" min-width="100px">
                        <template #header>
                            <el-button class="ml0" type="primary" circle size="small" icon="Plus" @click="addTask()"> </el-button>
                            <span class="ml10">{{ $t('flow.nodeName') }}<span class="ml5" style="color: red">*</span></span>
                            <el-tooltip :content="$t('flow.nodeNameTips')" placement="top">
                                <SvgIcon class="ml5" name="question-filled" />
                            </el-tooltip>
                        </template>
                        <template #default="scope">
                            <el-input v-model="scope.row.name"> </el-input>
                        </template>
                    </el-table-column>

                    <el-table-column prop="userId" min-width="150px" show-overflow-tooltip>
                        <template #header>
                            <span class="ml10">{{ $t('flow.auditor') }}<span class="ml5" style="color: red">*</span></span>
                        </template>

                        <template #default="scope">
                            <AccountSelectFormItem v-model="scope.row.userId" label="" />
                        </template>
                    </el-table-column>

                    <el-table-column :label="$t('common.operation')" width="110px">
                        <template #default="scope">
                            <el-link @click="deleteTask(scope.$index)" class="ml5" type="danger" icon="delete" plain></el-link>
                        </template>
                    </el-table-column>
                </el-table>
            </el-form>

            <template #footer>
                <div>
                    <el-button @click="cancel()">{{ $t('common.cancel') }}</el-button>
                    <el-button type="primary" :loading="saveBtnLoading" @click="btnOk">{{ $t('common.confirm') }}</el-button>
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
import TagTreeCheck from '../ops/component/TagTreeCheck.vue';
import { TagResourceTypeEnum, TagResourceTypePath } from '@/common/commonEnum';
import EnumSelect from '@/components/enumselect/EnumSelect.vue';
import { useI18nFormValidate, useI18nPleaseInput, useI18nSaveSuccessMsg } from '@/hooks/useI18n';
import { useI18n } from 'vue-i18n';
import FormItemTooltip from '@/components/form/FormItemTooltip.vue';

const { t } = useI18n();

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
            message: useI18nPleaseInput('common.name'),
            trigger: ['change', 'blur'],
        },
    ],
    defKey: [
        {
            required: true,
            message: useI18nPleaseInput('Key'),
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
        condition: '',
        remark: null,
        // 流程的审批节点任务
        tasks: '',
        codePaths: [],
    },
    sortable: '' as any,
});

const { form, tasks } = toRefs(state);

const { isFetching: saveBtnLoading, execute: saveFlowDefExec } = procdefApi.save.useApi(form);

watch(props, (newValue: any) => {
    if (newValue.data) {
        state.form = { ...newValue.data };
        state.form.codePaths = newValue.data.tags?.map((tag: any) => tag.codePath);
        const tasks = JSON.parse(state.form.tasks);
        tasks.forEach((t: any) => {
            t.userId = Number.parseInt(t.userId);
        });
        state.tasks = tasks;
    } else {
        state.form = { status: ProcdefStatus.Enable.value } as any;
        state.form.condition = t('flow.conditionDefault');
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
    await useI18nFormValidate(formRef);
    const checkRes = checkTasks();
    if (checkRes.err) {
        ElMessage.error(checkRes.err);
        return false;
    }

    state.form.tasks = JSON.stringify(checkRes.tasks);
    await saveFlowDefExec();
    useI18nSaveSuccessMsg();
    emit('val-change', state.form);
    //重置表单域
    formRef.value.resetFields();
    state.form = {} as any;
};

const checkTasks = () => {
    if (state.tasks?.length == 0) {
        return { err: t('flow.tasksNotEmpty') };
    }

    const tasks = [];
    for (let i = 0; i < state.tasks.length; i++) {
        const task = { ...state.tasks[i] };
        if (!task.name || !task.userId) {
            return { err: t('flow.tasksNoComplete', { index: i + 1 }) };
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
