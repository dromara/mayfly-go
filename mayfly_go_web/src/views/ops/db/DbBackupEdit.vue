<template>
    <div>
        <el-dialog :title="title" :model-value="visible" :before-close="cancel" :close-on-click-modal="false" :destroy-on-close="true" width="38%">
            <el-form :model="state.form" ref="backupForm" label-width="auto" :rules="rules">
                <el-form-item prop="dbNames" label="数据库名称">
                    <el-select
                        @change="changeDatabase"
                        v-model="state.selectedDbNames"
                        multiple
                        clearable
                        collapse-tags
                        collapse-tags-tooltip
                        filterable
                        :disabled="state.editOrCreate"
                        placeholder="数据库名称"
                        style="width: 100%"
                    >
                        <el-option v-for="db in state.dbNamesWithoutBackup" :key="db" :label="db" :value="db" />
                    </el-select>
                </el-form-item>

                <el-form-item prop="name" label="任务名称">
                    <el-input v-model.number="state.form.name" type="text" placeholder="任务名称"></el-input>
                </el-form-item>
                <el-form-item prop="startTime" label="开始时间">
                    <el-date-picker v-model="state.form.startTime" type="datetime" placeholder="开始时间" />
                </el-form-item>
                <el-form-item prop="intervalDay" label="备份周期">
                    <el-input v-model.number="state.form.intervalDay" type="number" placeholder="备份周期（单位：天）"></el-input>
                </el-form-item>
            </el-form>

            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="cancel()">取 消</el-button>
                    <el-button type="primary" :loading="state.btnLoading" @click="btnOk">确 定</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { reactive, ref, watch } from 'vue';
import { dbApi } from './api';
import { ElMessage } from 'element-plus';

const props = defineProps({
    visible: {
        type: Boolean,
    },
    data: {
        type: [Boolean, Object],
    },
    title: {
        type: String,
    },
    dbId: {
        type: [Number],
        required: true,
    },
});

//定义事件
const emit = defineEmits(['update:visible', 'cancel', 'val-change']);

const rules = {
    dbNames: [
        {
            required: true,
            message: '请选择需要备份的数据库',
            trigger: ['change', 'blur'],
        },
    ],
    intervalDay: [
        {
            required: true,
            pattern: /^[1-9]\d*$/,
            message: '请输入正整数',
            trigger: ['change', 'blur'],
        },
    ],
    startTime: [
        {
            required: true,
            message: '请选择开始时间',
            trigger: ['change', 'blur'],
        },
    ],
};

const backupForm: any = ref(null);

const state = reactive({
    form: {
        id: 0,
        dbId: 0,
        dbNames: String,
        name: null as any,
        intervalDay: 1,
        startTime: null as any,
        repeated: null as any,
    },
    btnLoading: false,
    selectedDbNames: [] as any,
    dbNamesWithoutBackup: [] as any,
    editOrCreate: false,
});

watch(props, (newValue: any) => {
    if (newValue.visible) {
        init(newValue.data);
    }
});

/**
 * 改变表单中的数据库字段，方便表单错误提示。如全部删光，可提示请添加数据库
 */
const changeDatabase = () => {
    state.form.dbNames = state.selectedDbNames.length == 0 ? '' : state.selectedDbNames.join(' ');
};

const init = (data: any) => {
    state.selectedDbNames = [];
    state.form.dbId = props.dbId;
    if (data) {
        state.editOrCreate = true;
        state.dbNamesWithoutBackup = [data.dbName];
        state.selectedDbNames = [data.dbName];
        state.form.id = data.id;
        state.form.dbNames = data.dbName;
        state.form.name = data.name;
        state.form.intervalDay = data.intervalDay;
        state.form.startTime = data.startTime;
    } else {
        state.editOrCreate = false;
        state.form.name = '';
        state.form.intervalDay = 1;
        const now = new Date();
        state.form.startTime = new Date(now.getFullYear(), now.getMonth(), now.getDate() + 1);
        getDbNamesWithoutBackup();
    }
};

const getDbNamesWithoutBackup = async () => {
    if (props.dbId > 0) {
        state.dbNamesWithoutBackup = await dbApi.getDbNamesWithoutBackup.request({ dbId: props.dbId });
    }
};

const btnOk = async () => {
    backupForm.value.validate(async (valid: boolean) => {
        if (valid) {
            state.form.repeated = true;
            const reqForm = { ...state.form };
            let api = dbApi.createDbBackup;
            if (props.data) {
                api = dbApi.saveDbBackup;
            }
            api.request(reqForm).then(() => {
                ElMessage.success('保存成功');
                emit('val-change', state.form);
                state.btnLoading = true;
                setTimeout(() => {
                    state.btnLoading = false;
                }, 1000);

                cancel();
            });
        } else {
            ElMessage.error('请正确填写信息');
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
