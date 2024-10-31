<template>
    <div>
        <el-dialog :title="title" :model-value="visible" :before-close="cancel" :close-on-click-modal="false" :destroy-on-close="true" width="38%">
            <el-form :model="state.form" ref="backupForm" label-width="auto" :rules="rules">
                <el-form-item prop="dbNames" label="数据库名称">
                    <el-select
                        v-model="state.dbNamesSelected"
                        multiple
                        clearable
                        collapse-tags
                        collapse-tags-tooltip
                        filterable
                        :disabled="state.editOrCreate"
                        :filter-method="filterDbNames"
                        placeholder="数据库名称"
                        style="width: 100%"
                    >
                        <template #header>
                            <el-checkbox v-model="checkAllDbNames" :indeterminate="indeterminateDbNames" @change="handleCheckAll"> 全选 </el-checkbox>
                        </template>
                        <el-option v-for="db in state.dbNamesFiltered" :key="db" :label="db" :value="db" />
                    </el-select>
                </el-form-item>

                <el-form-item prop="name" label="任务名称">
                    <el-input v-model="state.form.name" type="text" placeholder="任务名称"></el-input>
                </el-form-item>
                <el-form-item prop="startTime" label="开始时间">
                    <el-date-picker v-model="state.form.startTime" type="datetime" placeholder="开始时间" />
                </el-form-item>
                <el-form-item prop="intervalDay" label="备份周期（天）">
                    <el-input v-model.number="state.form.intervalDay" type="number" placeholder="单位：天"></el-input>
                </el-form-item>
                <el-form-item prop="maxSaveDays" label="备份历史保留天数">
                    <el-input v-model.number="state.form.maxSaveDays" type="number" placeholder="0: 永久保留"></el-input>
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
import { reactive, ref, toRefs, watch } from 'vue';
import { dbApi } from './api';
import { ElMessage } from 'element-plus';
import type { CheckboxValueType } from 'element-plus';

const props = defineProps({
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

const visible = defineModel<boolean>('visible', {
    default: false,
});

//定义事件
const emit = defineEmits(['cancel', 'val-change']);

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
    maxSaveDays: [
        {
            required: true,
            pattern: /^[0-9]\d*$/,
            message: '请输入非负整数',
            trigger: ['change', 'blur'],
        },
    ],
};

const backupForm: any = ref(null);

const state = reactive({
    form: {
        id: 0,
        dbId: 0,
        dbNames: '',
        name: '',
        intervalDay: 1,
        startTime: null as any,
        repeated: true,
        maxSaveDays: 0,
    },
    btnLoading: false,
    dbNamesSelected: [] as any,
    dbNamesWithoutBackup: [] as any,
    dbNamesFiltered: [] as any,
    filterString: '',
    editOrCreate: false,
});

const { dbNamesSelected, dbNamesWithoutBackup } = toRefs(state);

const checkAllDbNames = ref(false);
const indeterminateDbNames = ref(false);

watch(visible, (newValue: any) => {
    if (newValue) {
        init(props.data);
    }
});

const init = (data: any) => {
    state.dbNamesSelected = [];
    state.form.dbId = props.dbId;
    if (data) {
        state.editOrCreate = true;
        state.dbNamesWithoutBackup = [data.dbName];
        state.dbNamesSelected = [data.dbName];
        state.form.id = data.id;
        state.form.dbNames = data.dbName;
        state.form.name = data.name;
        state.form.intervalDay = data.intervalDay;
        state.form.startTime = data.startTime;
        state.form.maxSaveDays = data.maxSaveDays;
    } else {
        state.editOrCreate = false;
        state.form.name = '';
        state.form.intervalDay = 1;
        const now = new Date();
        state.form.startTime = new Date(now.getFullYear(), now.getMonth(), now.getDate() + 1);
        state.form.maxSaveDays = 0;
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
        if (!valid) {
            ElMessage.error('请正确填写信息');
            return false;
        }

        state.form.repeated = true;
        const reqForm = { ...state.form };
        let api = dbApi.createDbBackup;
        if (props.data) {
            api = dbApi.saveDbBackup;
        }

        try {
            state.btnLoading = true;
            await api.request(reqForm);
            ElMessage.success('保存成功');
            emit('val-change', state.form);
            cancel();
        } finally {
            state.btnLoading = false;
        }
    });
};

const cancel = () => {
    visible.value = false;
    emit('cancel');
};

const checkDbSelect = (val: string[]) => {
    const selected = val.filter((dbName: string) => {
        return dbName.includes(state.filterString);
    });
    if (selected.length === 0) {
        checkAllDbNames.value = false;
        indeterminateDbNames.value = false;
        return;
    }
    if (selected.length === state.dbNamesFiltered.length) {
        checkAllDbNames.value = true;
        indeterminateDbNames.value = false;
        return;
    }
    indeterminateDbNames.value = true;
};

watch(dbNamesSelected, (val: string[]) => {
    checkDbSelect(val);
    state.form.dbNames = val.join(' ');
});

watch(dbNamesWithoutBackup, (val: string[]) => {
    state.dbNamesFiltered = val.map((dbName: string) => dbName);
});

const handleCheckAll = (val: CheckboxValueType) => {
    const selected = state.dbNamesSelected.filter((dbName: string) => {
        return !state.dbNamesFiltered.includes(dbName);
    });
    if (val) {
        state.dbNamesSelected = selected.concat(state.dbNamesFiltered);
    } else {
        state.dbNamesSelected = selected;
    }
};

const filterDbNames = (filterString: string) => {
    state.dbNamesFiltered = state.dbNamesWithoutBackup.filter((dbName: string) => {
        return dbName.includes(filterString);
    });
    state.filterString = filterString;
    checkDbSelect(state.dbNamesSelected);
};
</script>
<style lang="scss"></style>
