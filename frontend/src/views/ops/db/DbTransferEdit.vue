<template>
    <div class="db-transfer-edit">
        <el-drawer :title="title" v-model="dialogVisible" :before-close="cancel" :destroy-on-close="true" :close-on-click-modal="false" size="40%">
            <template #header>
                <DrawerHeader :header="title" :back="cancel" />
            </template>

            <el-form :model="form" ref="dbForm" :rules="rules" label-width="auto">
                <el-divider content-position="left">基本信息</el-divider>

                <el-form-item prop="taskName" label="任务名" required>
                    <el-input v-model.trim="form.taskName" placeholder="请输入任务名" auto-complete="off" />
                </el-form-item>

                <el-form-item>
                    <el-row class="w100">
                        <el-col :span="12">
                            <el-form-item prop="status" label="启用状态">
                                <el-switch v-model="form.status" inline-prompt active-text="启用" inactive-text="禁用" :active-value="1" :inactive-value="-1" />
                            </el-form-item>
                        </el-col>

                        <el-col :span="12">
                            <el-form-item prop="cronAble" label="定时迁移" required>
                                <el-radio-group v-model="form.cronAble">
                                    <el-radio label="是" :value="1" />
                                    <el-radio label="否" :value="-1" />
                                </el-radio-group>
                            </el-form-item>
                        </el-col>
                    </el-row>
                </el-form-item>

                <el-form-item prop="cron" label="cron" :required="form.cronAble == 1">
                    <CrontabInput v-model="form.cron" />
                </el-form-item>

                <el-form-item prop="srcDbId" label="源数据库" class="w100" required>
                    <db-select-tree
                        placeholder="请选择源数据库"
                        v-model:db-id="form.srcDbId"
                        v-model:inst-name="form.srcInstName"
                        v-model:db-name="form.srcDbName"
                        v-model:tag-path="form.srcTagPath"
                        v-model:db-type="form.srcDbType"
                        @select-db="onSelectSrcDb"
                    />
                </el-form-item>

                <el-form-item prop="mode" label="迁移方式" required>
                    <el-radio-group v-model="form.mode">
                        <el-radio label="迁移到数据库" :value="1" />
                        <el-radio label="迁移到文件(自动命名)" :value="2" />
                    </el-radio-group>
                </el-form-item>

                <el-form-item v-if="form.mode === 2">
                    <el-row class="w100">
                        <el-col :span="12">
                            <el-form-item prop="targetFileDbType" label="文件数据库类型" :required="form.mode === 2">
                                <el-select v-model="form.targetFileDbType" placeholder="数据库类型" clearable filterable>
                                    <el-option
                                        v-for="(dbTypeAndDialect, key) in getDbDialectMap()"
                                        :key="key"
                                        :value="dbTypeAndDialect[0]"
                                        :label="dbTypeAndDialect[1].getInfo().name"
                                    >
                                        <SvgIcon :name="dbTypeAndDialect[1].getInfo().icon" :size="20" />
                                        {{ dbTypeAndDialect[1].getInfo().name }}
                                    </el-option>
                                    <template #prefix>
                                        <SvgIcon :name="getDbDialect(form.targetFileDbType!).getInfo().icon" :size="20" />
                                    </template>
                                </el-select>
                            </el-form-item>
                        </el-col>

                        <el-col :span="12">
                            <el-form-item label="文件保留天数">
                                <el-input-number v-model="form.fileSaveDays" :min="-1" :max="1000">
                                    <template #suffix>
                                        <span>天</span>
                                    </template>
                                </el-input-number>
                            </el-form-item>
                        </el-col>
                    </el-row>
                </el-form-item>

                <el-form-item prop="strategy" label="迁移策略" required>
                    <el-radio-group v-model="form.strategy">
                        <el-radio label="全量" :value="1" />
                        <el-radio label="增量（暂不可用）" :value="2" disabled />
                    </el-radio-group>
                </el-form-item>

                <el-form-item v-if="form.mode == 1" prop="targetDbId" label="目标数据库" class="w100" :required="form.mode === 1">
                    <db-select-tree
                        placeholder="请选择目标数据库"
                        v-model:db-id="form.targetDbId"
                        v-model:inst-name="form.targetInstName"
                        v-model:db-name="form.targetDbName"
                        v-model:tag-path="form.targetTagPath"
                        v-model:db-type="form.targetDbType"
                        @select-db="onSelectTargetDb"
                    />
                </el-form-item>

                <el-form-item prop="nameCase" label="转换表、字段名" required>
                    <el-radio-group v-model="form.nameCase">
                        <el-radio label="无" :value="1" />
                        <el-radio label="大写" :value="2" />
                        <el-radio label="小写" :value="3" />
                    </el-radio-group>
                </el-form-item>

                <el-divider content-position="left">数据库对象</el-divider>
                <el-form-item>
                    <el-input v-model="state.filterSrcTableText" placeholder="过滤表" size="small" />
                </el-form-item>
                <el-form-item class="w100">
                    <el-tree
                        ref="srcTreeRef"
                        class="w100"
                        style="max-height: 200px; overflow-y: auto"
                        default-expand-all
                        :expand-on-click-node="false"
                        :data="state.srcTableTree"
                        node-key="id"
                        show-checkbox
                        @check-change="handleSrcTableCheckChange"
                        :filter-node-method="filterSrcTableTreeNode"
                    />
                </el-form-item>
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
import { nextTick, reactive, ref, toRefs, watch } from 'vue';
import { dbApi } from './api';
import { ElMessage } from 'element-plus';
import DbSelectTree from '@/views/ops/db/component/DbSelectTree.vue';
import CrontabInput from '@/components/crontab/CrontabInput.vue';
import { getDbDialect, getDbDialectMap } from '@/views/ops/db/dialect';
import SvgIcon from '@/components/svgIcon/index.vue';
import _ from 'lodash';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';

const props = defineProps({
    data: {
        type: [Boolean, Object],
    },
    title: {
        type: String,
    },
});

//定义事件
const emit = defineEmits(['update:visible', 'cancel', 'val-change']);

const dialogVisible = defineModel<boolean>('visible', { default: false });

const rules = {
    taskName: [
        {
            required: true,
            message: '请输入任务名',
            trigger: ['change', 'blur'],
        },
    ],
    srcDbId: [
        {
            required: true,
            message: '请选择源库',
            trigger: ['change', 'blur'],
        },
    ],
    targetDbId: [
        {
            required: true,
            message: '请选择目标库',
            trigger: ['change', 'blur'],
        },
    ],
    targetFileDbType: [
        {
            required: true,
            message: '请选择目标文件语言类型',
            trigger: ['change', 'blur'],
        },
    ],
    cron: [
        {
            required: true,
            message: '请选择cron表达式',
            trigger: ['change', 'blur'],
        },
    ],
};

const dbForm: any = ref(null);

type FormData = {
    id?: number;
    taskName: string;
    status: number;
    cronAble: 1 | -1;
    cron: string;
    mode: 1 | 2;
    targetFileDbType?: string;
    fileSaveDays?: number;
    dbType: 1 | 2;
    srcDbId?: number;
    srcDbName?: string;
    srcDbType?: string;
    srcInstName?: string;
    srcTagPath?: string;
    srcTableNames?: string;
    targetDbId?: number;
    targetInstName?: string;
    targetDbName?: string;
    targetTagPath?: string;
    targetDbType?: string;
    strategy: 1 | 2;
    nameCase: 1 | 2 | 3;
    deleteTable?: 1 | 2;
    checkedKeys: string;
    runningState: 1 | 2;
};

const basicFormData = {
    mode: 1,
    status: 1,
    cronAble: -1,
    strategy: 1,
    nameCase: 1,
    deleteTable: 1,
    checkedKeys: '',
    runningState: 1,
} as FormData;

const srcTableList = ref<{ tableName: string; tableComment: string }[]>([]);
const srcTableListDisabled = ref(false);

const defaultKeys = ['tab-check', 'all', 'table-list'];

const state = reactive({
    form: basicFormData,
    submitForm: {} as any,
    srcTableFields: [] as string[],
    targetColumnList: [] as any[],
    filterSrcTableText: '',
    srcTableTree: [
        {
            id: 'tab-check',
            label: '表',
            children: [
                { id: 'all', label: '全部表（*）' },
                {
                    id: 'table-list',
                    label: '自定义',
                    disabled: srcTableListDisabled,
                    children: [] as any[],
                },
            ],
        },
    ],
});

const { form, submitForm } = toRefs(state);

const { isFetching: saveBtnLoading, execute: saveExec } = dbApi.saveDbTransferTask.useApi(submitForm);

watch(dialogVisible, async (newValue: boolean) => {
    if (!newValue) {
        return;
    }
    const propsData = props.data as any;
    if (!propsData?.id) {
        let d = {} as FormData;
        Object.assign(d, basicFormData);
        state.form = d;
        await nextTick(() => {
            srcTreeRef.value.setCheckedKeys([]);
        });
        return;
    }
    state.form = _.cloneDeep(props.data) as FormData;
    let { srcDbId, targetDbId } = state.form;

    //  初始化src数据源
    if (srcDbId) {
        // 通过tagPath查询实例列表
        const dbInfoRes = await dbApi.dbs.request({ id: srcDbId });
        const db = dbInfoRes.list[0];
        // 初始化实例
        db.databases = db.database?.split(' ').sort() || [];

        if (srcDbId && state.form.srcDbName) {
            await loadDbTables(srcDbId, state.form.srcDbName);
        }
    }

    //  初始化target数据源
    if (targetDbId) {
        // 通过tagPath查询实例列表
        const dbInfoRes = await dbApi.dbs.request({ id: targetDbId });
        const db = dbInfoRes.list[0];
        // 初始化实例
        db.databases = db.database?.split(' ').sort() || [];
    }

    // 初始化勾选迁移表
    srcTreeRef.value.setCheckedKeys(state.form.checkedKeys.split(','));

    // 初始化默认值
    state.form.cronAble = state.form.cronAble || 0;
    state.form.mode = state.form.mode || 1;
});

watch(
    () => state.filterSrcTableText,
    (val) => {
        srcTreeRef.value!.filter(val);
    }
);

const onSelectSrcDb = async (params: any) => {
    //  初始化数据源
    params.databases = params.dbs; // 数据源里需要这个值
    await loadDbTables(params.id, params.db);
};

const onSelectTargetDb = async (params: any) => {
    console.log(params);
};

const loadDbTables = async (dbId: number, db: string) => {
    // 加载db下的表
    srcTableList.value = await dbApi.tableInfos.request({ id: dbId, db });
    handleLoadSrcTableTree();
};

const handleSrcTableCheckChange = (data: { id: string; name: string }, checked: boolean) => {
    if (data.id === 'all') {
        srcTableListDisabled.value = checked;
        if (checked) {
            state.form.checkedKeys = 'all';
        } else {
            state.form.checkedKeys = '';
        }
    }
    if (data.id && (data.id + '').startsWith('list-item')) {
        //
    }
};

const filterSrcTableTreeNode = (value: string, data: any) => {
    if (!value) return true;
    return data.label.includes(value);
};

const handleLoadSrcTableTree = () => {
    state.srcTableTree[0].children[1].children = srcTableList.value.map((item) => {
        return {
            id: item.tableName,
            label: item.tableName + (item.tableComment && '-' + item.tableComment),
            disabled: srcTableListDisabled,
        };
    });
};

const getReqForm = async () => {
    return { ...state.form };
};

const srcTreeRef = ref();

const getCheckedKeys = () => {
    let checks = srcTreeRef.value!.getCheckedKeys(false);
    if (checks.indexOf('all') >= 0) {
        return ['all'];
    }
    return checks.filter((item: any) => !defaultKeys.includes(item));
};

const btnOk = async () => {
    dbForm.value.validate(async (valid: boolean) => {
        if (!valid) {
            ElMessage.error('请正确填写信息');
            return false;
        }

        state.submitForm = await getReqForm();

        let checkedKeys = getCheckedKeys();
        if (checkedKeys.length > 0) {
            state.submitForm.checkedKeys = checkedKeys.join(',');
        }

        if (!state.submitForm.checkedKeys) {
            ElMessage.error('请选择需要迁移的表');
            return false;
        }

        await saveExec();
        ElMessage.success('保存成功');
        emit('val-change', state.form);
        cancel();
    });
};

const cancel = () => {
    dialogVisible.value = false;
    emit('cancel');
};
</script>
<style lang="scss"></style>
