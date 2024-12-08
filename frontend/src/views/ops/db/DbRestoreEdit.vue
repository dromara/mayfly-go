<template>
    <div>
        <el-dialog :title="title" :model-value="visible" :before-close="cancel" :close-on-click-modal="false" width="38%">
            <el-form :model="state.form" ref="restoreForm" label-width="auto" :rules="rules">
                <el-form-item label="恢复方式">
                    <el-radio-group :disabled="state.editOrCreate" v-model="state.restoreMode">
                        <el-radio label="point-in-time">指定时间点</el-radio>
                        <el-radio label="backup-history">指定备份</el-radio>
                    </el-radio-group>
                </el-form-item>
                <el-form-item prop="dbName" label="数据库名称">
                    <el-select
                        :disabled="state.editOrCreate"
                        @change="changeDatabase"
                        v-model="state.form.dbName"
                        placeholder="数据库名称"
                        filterable
                        clearable
                        class="w100"
                    >
                        <el-option v-for="item in props.dbNames" :key="item" :label="`${item}`" :value="item"> </el-option>
                    </el-select>
                </el-form-item>
                <el-form-item v-if="state.restoreMode == 'point-in-time'" prop="pointInTime" label="恢复时间点">
                    <el-date-picker :disabled="state.editOrCreate" v-model="state.form.pointInTime" type="datetime" placeholder="恢复时间点" />
                </el-form-item>
                <el-form-item v-if="state.restoreMode == 'backup-history'" prop="dbBackupHistoryId" label="数据库备份">
                    <el-select
                        :disabled="state.editOrCreate"
                        @change="changeHistory"
                        v-model="state.history"
                        value-key="id"
                        placeholder="数据库备份"
                        filterable
                        clearable
                        class="w100"
                    >
                        <el-option
                            v-for="item in state.histories"
                            :key="item.id"
                            :label="item.name + (item.binlogFileName ? ' ' : ' 不') + '支持指定时间点恢复'"
                            :value="item"
                        >
                        </el-option>
                    </el-select>
                </el-form-item>
                <el-form-item prop="startTime" label="开始时间">
                    <el-date-picker :disabled="state.editOrCreate" v-model="state.form.startTime" type="datetime" placeholder="开始时间" />
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
import { onMounted, reactive, ref, watch } from 'vue';
import { dbApi } from './api';
import { ElMessage, ElMessageBox } from 'element-plus';

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
    dbNames: {
        type: Array,
        required: true,
    },
});

//定义事件
const emit = defineEmits(['cancel', 'val-change']);

const visible = defineModel<boolean>('visible', {
    default: false,
});

const validatePointInTime = (rule: any, value: any, callback: any) => {
    if (value > new Date()) {
        callback(new Error('恢复时间点晚于当前时间'));
        return;
    }
    if (!state.histories || state.histories.length == 0) {
        callback(new Error('数据库没有备份记录'));
        return;
    }
    let last = null;
    for (const history of state.histories) {
        if (!history.binlogFileName || history.binlogFileName.length === 0) {
            break;
        }
        if (new Date(history.createTime) < value) {
            callback();
            return;
        }
        last = history;
    }
    if (!last) {
        callback(new Error('现有数据库备份不支持指定时间恢复'));
        return;
    }
    callback(last.name + ' 之前的数据库备份不支持指定时间恢复');
};

const rules = {
    dbName: [
        {
            required: true,
            message: '请选择需要恢复的数据库',
            trigger: ['change', 'blur'],
        },
    ],
    pointInTime: [
        {
            required: true,
            validator: validatePointInTime,
            trigger: ['change', 'blur'],
        },
    ],
    dbBackupHistoryId: [
        {
            required: true,
            message: '请选择数据库备份',
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

const restoreForm: any = ref(null);

const state = reactive({
    form: {
        id: 0,
        dbId: 0,
        dbName: null as any,
        intervalDay: 0,
        startTime: null as any,
        repeated: null as any,
        dbBackupId: null as any,
        dbBackupHistoryId: null as any,
        dbBackupHistoryName: null as any,
        pointInTime: null as any,
    },
    btnLoading: false,
    dbNamesSelected: [] as any,
    dbNamesWithoutRestore: [] as any,
    editOrCreate: false,
    histories: [] as any,
    history: null as any,
    restoreMode: null as any,
});

onMounted(async () => {
    await init(props.data);
});

watch(visible, (newValue: any) => {
    if (newValue) {
        init(props.data);
    }
});

/**
 * 改变表单中的数据库字段，方便表单错误提示。如全部删光，可提示请添加数据库
 */
const changeDatabase = async () => {
    await getBackupHistories(props.dbId, state.form.dbName);
};

const changeHistory = async () => {
    if (state.history) {
        state.form.dbBackupId = state.history.dbBackupId;
        state.form.dbBackupHistoryId = state.history.id;
        state.form.dbBackupHistoryName = state.history.name;
    }
};

const init = async (data: any) => {
    state.dbNamesSelected = [];
    state.form.dbId = props.dbId;
    if (data) {
        state.editOrCreate = true;
        state.dbNamesWithoutRestore = [data.dbName];
        state.dbNamesSelected = [data.dbName];
        state.form.id = data.id;
        state.form.dbName = data.dbName;
        state.form.intervalDay = data.intervalDay;
        state.form.pointInTime = data.pointInTime;
        state.form.startTime = data.startTime;
        state.form.dbBackupId = data.dbBackupId;
        state.form.dbBackupHistoryId = data.dbBackupHistoryId;
        state.form.dbBackupHistoryName = data.dbBackupHistoryName;
        if (data.pointInTime) {
            state.restoreMode = 'point-in-time';
        } else {
            state.restoreMode = 'backup-history';
        }
        state.history = {
            dbBackupId: data.dbBackupId,
            id: data.dbBackupHistoryId,
            name: data.dbBackupHistoryName,
            createTime: data.createTime,
        };
        await getBackupHistories(props.dbId, data.dbName);
    } else {
        state.form.dbName = '';
        state.editOrCreate = false;
        state.form.intervalDay = 0;
        state.form.repeated = false;
        state.form.pointInTime = new Date();
        state.form.startTime = new Date();
        state.histories = [];
        state.history = null;
        state.restoreMode = 'point-in-time';
        await getDbNamesWithoutRestore();
    }
};

const getDbNamesWithoutRestore = async () => {
    if (props.dbId > 0) {
        state.dbNamesWithoutRestore = await dbApi.getDbNamesWithoutRestore.request({ dbId: props.dbId });
    }
};

const btnOk = async () => {
    restoreForm.value.validate(async (valid: any) => {
        if (valid) {
            await ElMessageBox.confirm(`确定恢复数据库吗？`, '提示', {
                confirmButtonText: '确定',
                cancelButtonText: '取消',
                type: 'warning',
            });

            if (state.restoreMode == 'point-in-time') {
                state.form.dbBackupId = 0;
                state.form.dbBackupHistoryId = 0;
                state.form.dbBackupHistoryName = '';
            } else {
                state.form.pointInTime = null;
            }
            state.form.repeated = false;
            state.form.intervalDay = 0;
            const reqForm = { ...state.form };
            let api = dbApi.createDbRestore;
            if (props.data) {
                api = dbApi.saveDbRestore;
            }
            api.request(reqForm).then(() => {
                ElMessage.success('成功创建数据库恢复任务');
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
    visible.value = false;
    emit('cancel');
};

const getBackupHistories = async (dbId: Number, dbName: String) => {
    if (!dbId || !dbName) {
        state.histories = [];
        return;
    }
    const data = await dbApi.getDbBackupHistories.request({ dbId, dbName });
    if (!data || !data.list) {
        ElMessage.error('该数据库没有备份记录，无法创建数据库恢复任务');
        state.histories = [];
        return;
    }
    state.histories = data.list;
};
</script>
<style lang="scss"></style>
