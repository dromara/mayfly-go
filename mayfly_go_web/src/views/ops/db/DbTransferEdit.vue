<template>
    <div class="db-transfer-edit">
        <el-dialog
            :title="title"
            v-model="dialogVisible"
            :before-close="cancel"
            :close-on-click-modal="false"
            :close-on-press-escape="false"
            :destroy-on-close="true"
            width="850px"
        >
            <el-form :model="form" ref="dbForm" :rules="rules" label-width="auto">
                <el-tabs v-model="tabActiveName">
                    <el-tab-pane label="基本信息" :name="basicTab">
                        <el-form-item prop="taskName" label="任务名" required>
                            <el-input v-model.trim="form.taskName" placeholder="请输入任务名" auto-complete="off" />
                        </el-form-item>
                        <el-form-item prop="srcDbId" label="源数据库" required>
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

                        <el-form-item prop="targetDbId" label="目标数据库" required>
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

                        <el-form-item prop="strategy" label="迁移策略" required>
                            <el-select v-model="form.strategy" filterable placeholder="迁移策略">
                                <el-option label="全量" :value="1" />
                                <el-option label="增量（暂不可用）" disabled :value="2" />
                            </el-select>
                        </el-form-item>

                        <el-form-item prop="nameCase" label="转换表、字段名" required>
                            <el-select v-model="form.nameCase">
                                <el-option label="无" :value="1" />
                                <el-option label="大写" :value="2" />
                                <el-option label="小写" :value="3" />
                            </el-select>
                        </el-form-item>
                        <el-form-item prop="deleteTable" label="创建前删除表" required>
                            <el-select v-model="form.deleteTable">
                                <el-option label="是" :value="1" />
                                <el-option label="否" :value="2" />
                            </el-select>
                        </el-form-item>
                    </el-tab-pane>
                    <el-tab-pane label="数据库对象" :name="tableTab" :disabled="!baseFieldCompleted">
                        <el-form-item>
                            <el-input v-model="state.filterSrcTableText" style="width: 240px" placeholder="过滤表" />
                        </el-form-item>
                        <el-form-item>
                            <el-tree
                                ref="srcTreeRef"
                                style="width: 760px; max-height: 400px; overflow-y: auto"
                                default-expand-all
                                :expand-on-click-node="false"
                                :data="state.srcTableTree"
                                node-key="id"
                                show-checkbox
                                @check-change="handleSrcTableCheckChange"
                                :filter-node-method="filterSrcTableTreeNode"
                            />
                        </el-form-item>
                    </el-tab-pane>
                </el-tabs>
            </el-form>

            <template #footer>
                <div>
                    <el-button @click="cancel()">取 消</el-button>
                    <el-button type="primary" :loading="saveBtnLoading" @click="btnOk">确 定</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { computed, nextTick, reactive, ref, toRefs, watch } from 'vue';
import { dbApi } from './api';
import { ElMessage } from 'element-plus';
import DbSelectTree from '@/views/ops/db/component/DbSelectTree.vue';

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

const rules = {};

const dbForm: any = ref(null);

const basicTab = 'basic';
const tableTab = 'table';

type FormData = {
    id?: number;
    taskName: string;
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
    tabActiveName: 'basic',
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

const { tabActiveName, form, submitForm } = toRefs(state);

const { isFetching: saveBtnLoading, execute: saveExec } = dbApi.saveDbTransferTask.useApi(submitForm);

// 基础字段信息是否填写完整
const baseFieldCompleted = computed(() => {
    return state.form.srcDbId && state.form.targetDbId && state.form.targetDbName;
});

watch(dialogVisible, async (newValue: boolean) => {
    if (!newValue) {
        return;
    }
    state.tabActiveName = 'basic';
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

    state.form = props.data as FormData;
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
<style lang="scss">
.db-transfer-edit {
    .el-select {
        width: 100%;
    }
}
</style>
