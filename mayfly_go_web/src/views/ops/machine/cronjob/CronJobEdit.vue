<template>
    <div class="mock-data-dialog">
        <el-dialog
            :title="title"
            v-model="dialogVisible"
            :close-on-click-modal="false"
            :before-close="cancel"
            :show-close="true"
            :destroy-on-close="true"
            width="900px"
        >
            <el-form :model="form" ref="formRef" :rules="rules" label-width="auto">
                <el-form-item prop="name" label="名称">
                    <el-input v-model="form.name" placeholder="请输入名称"></el-input>
                </el-form-item>

                <el-form-item prop="cron" label="cron表达式">
                    <CrontabInput v-model="form.cron" />
                </el-form-item>

                <el-form-item prop="status" label="状态">
                    <el-select v-model="form.status" default-first-option style="width: 100%" placeholder="请选择状态">
                        <el-option v-for="item in CronJobStatusEnum" :key="item.value" :label="item.label" :value="item.value"></el-option>
                    </el-select>
                </el-form-item>

                <el-form-item prop="saveExecResType" label="记录类型">
                    <el-select v-model="form.saveExecResType" default-first-option style="width: 100%" placeholder="请选择记录类型">
                        <el-option v-for="item in CronJobSaveExecResTypeEnum" :key="item.value" :label="item.label" :value="item.value"></el-option>
                    </el-select>
                </el-form-item>

                <el-form-item prop="remark" label="备注">
                    <el-input v-model="form.remark" placeholder="请输入备注"></el-input>
                </el-form-item>

                <el-form-item prop="machineIds" label="关联机器">
                    <el-select multiple v-model="form.machineIds" filterable placeholder="请选关联机器" style="width: 100%">
                        <el-option v-for="ac in state.machines" :key="ac.id" :value="ac.id" :label="ac.ip">
                            {{ ac.ip }}
                            <el-divider direction="vertical" border-style="dashed" />
                            {{ ac.tagPath }}{{ ac.name }}
                        </el-option>
                    </el-select>
                </el-form-item>

                <el-form-item prop="script" label="执行脚本" required>
                    <monaco-editor style="width: 100%" v-model="form.script" language="shell" height="300px"
                /></el-form-item>
            </el-form>

            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="cancel()" :disabled="submitDisabled">关 闭</el-button>
                    <el-button v-auth="'machine:script:save'" type="primary" :loading="btnLoading" @click="btnOk" :disabled="submitDisabled">保 存</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { ref, toRefs, reactive, watch, onMounted } from 'vue';
import { ElMessage } from 'element-plus';
import { cronJobApi, machineApi } from '../api';
import { CronJobStatusEnum, CronJobSaveExecResTypeEnum } from '../enums';
import { notEmpty } from '@/common/assert';
import MonacoEditor from '@/components/monaco/MonacoEditor.vue';
import CrontabInput from '@/components/crontab/CrontabInput.vue';

const props = defineProps({
    visible: {
        type: Boolean,
    },
    data: {
        type: Object,
    },
    title: {
        type: String,
    },
});

const emit = defineEmits(['update:visible', 'cancel', 'submitSuccess']);

const formRef: any = ref(null);

const rules = {
    name: [
        {
            required: true,
            message: '请输入名称',
            trigger: ['change', 'blur'],
        },
    ],
    cron: [
        {
            required: true,
            message: '请输入cron表达式',
            trigger: ['change', 'blur'],
        },
    ],
    status: [
        {
            required: true,
            message: '请选择状态',
            trigger: ['change', 'blur'],
        },
    ],
    saveExecResType: [
        {
            required: true,
            message: '请选择执行记录类型',
            trigger: ['change', 'blur'],
        },
    ],
    script: [
        {
            required: true,
            message: '请输入执行脚本',
            trigger: ['change', 'blur'],
        },
    ],
};

const state = reactive({
    dialogVisible: false,
    submitDisabled: false,
    chooseMachines: [],
    form: {
        id: null,
        name: '',
        cron: '',
        machineIds: [],
        remark: '',
        script: '',
        status: 1,
        saveExecResType: -1,
    },
    machines: [] as any,
    btnLoading: false,
});

const { dialogVisible, submitDisabled, form, btnLoading } = toRefs(state);

onMounted(async () => {
    const res = await machineApi.list.request({ pageNum: 1, pageSize: 100 });
    state.machines = res.list;
});

watch(props, async (newValue: any) => {
    state.dialogVisible = newValue.visible;
    if (!newValue.visible) {
        return;
    }
    if (newValue.data) {
        state.form = { ...newValue.data };
        state.form.machineIds = await cronJobApi.relateMachineIds.request({ cronJobId: state.form.id });
    } else {
        state.form = { script: '', status: 1 } as any;
        state.chooseMachines = [];
    }
});

const btnOk = () => {
    formRef.value.validate((valid: any) => {
        if (valid) {
            notEmpty(state.form.name, '名称不能为空');
            notEmpty(state.form.script, '脚本内容不能为空');
            cronJobApi.save.request(state.form).then(
                () => {
                    ElMessage.success('保存成功');
                    emit('submitSuccess');
                    state.submitDisabled = false;
                    cancel();
                },
                () => {
                    state.submitDisabled = false;
                }
            );
        } else {
            return false;
        }
    });
};

const cancel = () => {
    emit('update:visible', false);
    emit('cancel');
};
</script>
<style lang="scss"></style>
