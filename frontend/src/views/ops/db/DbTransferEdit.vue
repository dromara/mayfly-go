<template>
    <div class="db-transfer-edit">
        <el-drawer :title="title" v-model="dialogVisible" :before-close="cancel" :destroy-on-close="true" :close-on-click-modal="false" size="45%">
            <template #header>
                <DrawerHeader :header="title" :back="cancel" />
            </template>

            <el-form :model="form" ref="dbForm" :rules="rules" label-position="top" label-width="auto">
                <el-divider content-position="left">{{ $t('common.basic') }}</el-divider>

                <el-form-item prop="taskName" :label="$t('db.taskName')" required>
                    <el-input v-model.trim="form.taskName" auto-complete="off" />
                </el-form-item>

                <el-row class="!w-full">
                    <el-col :span="12">
                        <el-form-item prop="status" :label="$t('common.status')" label-position="left">
                            <el-switch
                                v-model="form.status"
                                inline-prompt
                                :active-text="$t('common.enable')"
                                :inactive-text="$t('common.disable')"
                                :active-value="1"
                                :inactive-value="-1"
                            />
                        </el-form-item>
                    </el-col>

                    <el-col :span="12">
                        <el-form-item prop="cronAble" :label="$t('db.cronAble')" required label-position="left">
                            <el-radio-group v-model="form.cronAble">
                                <el-radio :label="$t('common.yes')" :value="1" />
                                <el-radio :label="$t('common.no')" :value="-1" />
                            </el-radio-group>
                        </el-form-item>
                    </el-col>
                </el-row>

                <el-form-item prop="cron" label="cron" :required="form.cronAble == 1">
                    <CrontabInput v-model="form.cron" />
                </el-form-item>

                <el-form-item prop="srcDbId" :label="$t('db.srcDb')" class="!w-full" required>
                    <db-select-tree
                        v-model:db-id="form.srcDbId"
                        v-model:inst-name="form.srcInstName"
                        v-model:db-name="form.srcDbName"
                        v-model:tag-path="form.srcTagPath"
                        v-model:db-type="form.srcDbType"
                        @select-db="onSelectSrcDb"
                    />
                </el-form-item>

                <el-form-item prop="mode" :label="$t('db.transferMode')" required>
                    <el-radio-group v-model="form.mode">
                        <el-radio :label="$t('db.transfer2Db')" :value="1" />
                        <el-radio :label="$t('db.transfer2File')" :value="2" />
                    </el-radio-group>
                </el-form-item>

                <el-form-item v-if="form.mode === 2">
                    <el-row class="!w-full">
                        <el-col :span="12">
                            <el-form-item prop="targetFileDbType" :label="$t('db.dbFileType')" :required="form.mode === 2">
                                <el-select v-model="form.targetFileDbType" clearable filterable>
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
                            <el-form-item :label="$t('db.fileSaveDays')">
                                <el-input-number v-model="form.fileSaveDays" :min="-1" :max="1000">
                                    <template #suffix>
                                        <span>{{ $t('db.day') }}</span>
                                    </template>
                                </el-input-number>
                            </el-form-item>
                        </el-col>
                    </el-row>
                </el-form-item>

                <el-form-item prop="strategy" :label="$t('db.transferStrategy')" required>
                    <el-radio-group v-model="form.strategy">
                        <el-radio :label="$t('db.transferFull')" :value="1" />
                        <el-radio :label="$t('db.transferIncrement')" :value="2" disabled />
                    </el-radio-group>
                </el-form-item>

                <el-form-item v-if="form.mode == 1" prop="targetDbId" :label="$t('db.targetDb')" class="!w-full" :required="form.mode === 1">
                    <db-select-tree
                        v-model:db-id="form.targetDbId"
                        v-model:inst-name="form.targetInstName"
                        v-model:db-name="form.targetDbName"
                        v-model:tag-path="form.targetTagPath"
                        v-model:db-type="form.targetDbType"
                        @select-db="onSelectTargetDb"
                    />
                </el-form-item>

                <el-form-item prop="nameCase" :label="$t('db.nameCase')" required>
                    <el-radio-group v-model="form.nameCase">
                        <el-radio :label="$t('db.none')" :value="1" />
                        <el-radio :label="$t('db.upper')" :value="2" />
                        <el-radio :label="$t('db.lower')" :value="3" />
                    </el-radio-group>
                </el-form-item>

                <el-divider content-position="left">{{ $t('db.dbObj') }}</el-divider>
                <el-form-item>
                    <el-input v-model="state.filterSrcTableText" placeholder="filter table" size="small" />
                </el-form-item>
                <el-form-item class="!w-full">
                    <el-tree
                        ref="srcTreeRef"
                        class="!w-full"
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
                    <el-button @click="cancel()">{{ $t('common.cancel') }}</el-button>
                    <el-button type="primary" :loading="saveBtnLoading" @click="btnOk">{{ $t('common.confirm') }}</el-button>
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
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';
import { useI18nFormValidate, useI18nSaveSuccessMsg } from '@/hooks/useI18n';
import { useI18n } from 'vue-i18n';
import { Rules } from '@/common/rule';
import { deepClone } from '@/common/utils/object';

const { t } = useI18n();

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
    taskName: [Rules.requiredInput('db.taskName')],
    srcDbId: [Rules.requiredSelect('db.srcDb')],
    targetDbId: [Rules.requiredSelect('db.targetDb')],
    targetFileDbType: [Rules.requiredSelect('db.dbFileType')],
    cron: [Rules.requiredSelect('cron')],
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
            label: t('db.table'),
            children: [
                { id: 'all', label: `${t('db.allTable')}（*）` },
                {
                    id: 'table-list',
                    label: t('db.custom'),
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

    state.form = deepClone(props.data) as FormData;
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
    await useI18nFormValidate(dbForm);
    state.submitForm = await getReqForm();

    let checkedKeys = getCheckedKeys();
    if (checkedKeys.length > 0) {
        state.submitForm.checkedKeys = checkedKeys.join(',');
    }

    if (!state.submitForm.checkedKeys) {
        ElMessage.error(t('db.noTransferTableMsg'));
        return false;
    }

    await saveExec();
    useI18nSaveSuccessMsg();
    emit('val-change', state.form);
    cancel();
};

const cancel = () => {
    dialogVisible.value = false;
    emit('cancel');
};
</script>
<style lang="scss"></style>
