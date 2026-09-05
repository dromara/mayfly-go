<template>
    <div class="db-transfer-edit">
        <auto-form-drawer v-model:visible="dialogVisible" :title="title" :items="items" :data="editData" size="45%" :confirm-api="btnOk" @submitted="emit('cancel')" @opened="onOpened" @cancel="emit('cancel')">
            <template #cron="{ form }">
                <CrontabInput v-model="form.cron" />
            </template>

            <!-- 源库选择 -->
            <template #srcDbId="{ form }">
                <db-select-tree
                    v-model:db-id="form.srcDbId"
                    v-model:inst-name="form.srcInstName"
                    v-model:db-name="form.srcDbName"
                    v-model:tag-path="form.srcTagPath"
                    v-model:db-type="form.srcDbType"
                    @select-db="onSelectSrcDb"
                />
            </template>

            <!-- 文件库类型（自定义 option 图标渲染） -->
            <template #targetFileDbType="{ form }">
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
                        <SvgIcon v-if="form.targetFileDbType" :name="getDbDialect(form.targetFileDbType).getInfo().icon" :size="20" />
                    </template>
                </el-select>
            </template>

            <template #fileSaveDays="{ form }">
                <el-input-number v-model="form.fileSaveDays" :min="-1" :max="1000">
                    <template #suffix>
                        <span>{{ $t('db.day') }}</span>
                    </template>
                </el-input-number>
            </template>

            <!-- 目标库选择 -->
            <template #targetDbId="{ form }">
                <db-select-tree
                    v-model:db-id="form.targetDbId"
                    v-model:inst-name="form.targetInstName"
                    v-model:db-name="form.targetDbName"
                    v-model:tag-path="form.targetTagPath"
                    v-model:db-type="form.targetDbType"
                    @select-db="onSelectTargetDb"
                />
            </template>

            <!-- 迁移表选择（过滤输入 + 树勾选） -->
            <template #checkedKeys>
                <div class="w-full">
                    <el-input v-model="state.filterSrcTableText" placeholder="filter table" size="small" />
                    <el-tree
                        ref="srcTreeRef"
                        class="w-full! overflow-y-auto"
                        style="max-height: 200px"
                        default-expand-all
                        :expand-on-click-node="false"
                        :data="state.srcTableTree"
                        node-key="id"
                        show-checkbox
                        @check-change="handleSrcTableCheckChange"
                        :filter-node-method="filterSrcTableTreeNode"
                    />
                </div>
            </template>
        </auto-form-drawer>
    </div>
</template>

<script lang="ts" setup>
import { computed, nextTick, reactive, ref, useTemplateRef, watch, type PropType } from 'vue';

import { Rules } from '@/common/rule';
import { deepClone } from '@/common/utils/object';
import { AutoFormDrawer, type AutoFormData, type AutoFormItem } from '@/components/auto-form';
import CrontabInput from '@/components/crontab/CrontabInput.vue';
import SvgIcon from '@/components/svg-icon/index.vue';
import { Msg } from '@/hooks/useI18n';
import { dbApi } from '@/views/ops/db/api';
import DbSelectTree from '@/views/ops/db/component/DbSelectTree.vue';
import { getDbDialect, getDbDialectMap } from '@/views/ops/db/dialect';
import { dbTransferApi } from '@/views/ops/db/transfer/api';
import type { DbTransferTask, Db } from '@/views/ops/db/types';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();

const props = defineProps({
    data: {
        type: Object as PropType<DbTransferTask | null>,
        default: null,
    },
    title: {
        type: String,
    },
});

//定义事件
const emit = defineEmits(['update:visible', 'cancel', 'val-change']);

const dialogVisible = defineModel<boolean>('visible', { default: false });

const fileTypeOptions = [
    { label: '.zip', value: 'zip' },
    { label: '.sql', value: 'sql' },
];

/** 表单声明（AutoFormItem[]，渲染 + 校验唯一数据源；复杂控件走 custom 插槽承载） */
const items: AutoFormItem[] = [
    { prop: 'taskName', label: 'db.taskName', required: true },
    {
        prop: 'status',
        label: 'common.status',
        type: 'switch',
        span: 12,
        props: { inlinePrompt: true, activeText: t('common.enable'), inactiveText: t('common.disable'), activeValue: 1, inactiveValue: -1 },
    },
    {
        prop: 'cronAble',
        label: 'db.cronAble',
        type: 'radio',
        span: 12,
        required: true,
        options: [
            { label: 'common.yes', value: 1 },
            { label: 'common.no', value: -1 },
        ],
    },
    { prop: 'cron', label: 'cron', type: 'custom', required: (f) => f.cronAble == 1 },
    { prop: 'srcDbId', label: 'db.srcDb', type: 'custom', rules: [Rules.requiredSelect('db.srcDb')] },
    {
        prop: 'mode',
        label: 'db.transferMode',
        type: 'radio',
        required: true,
        options: [
            { label: 'db.transfer2Db', value: 1 },
            { label: 'db.transfer2File', value: 2 },
        ],
    },
    { prop: 'targetFileDbType', label: 'db.dbFileType', type: 'custom', span: 10, when: (f) => f.mode === 2, rules: [Rules.requiredSelect('db.dbFileType')] },
    { prop: 'extra.fileType', label: 'db.fileType', type: 'select', span: 6, when: (f) => f.mode === 2, options: fileTypeOptions },
    { prop: 'fileSaveDays', label: 'db.fileSaveDays', type: 'custom', span: 8, when: (f) => f.mode === 2 },
    {
        prop: 'strategy',
        label: 'db.transferStrategy',
        type: 'radio',
        required: true,
        options: [
            { label: 'db.transferFull', value: 1 },
            // 增量迁移暂未实现，选项禁用
            { label: 'db.transferIncrement', value: 2, disabled: true },
        ],
    },
    { prop: 'targetDbId', label: 'db.targetDb', type: 'custom', when: (f) => f.mode === 1, rules: [Rules.requiredSelect('db.targetDb')] },
    {
        prop: 'nameCase',
        label: 'db.nameCase',
        type: 'radio',
        required: true,
        options: [
            { label: 'db.none', value: 1 },
            { label: 'db.upper', value: 2 },
            { label: 'db.lower', value: 3 },
        ],
    },
    { prop: 'dbObjDivider', label: 'db.dbObj', type: 'divider' },
    { prop: 'checkedKeys', type: 'custom' },
];

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
    extra: { fileType: string };
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
    extra: { fileType: fileTypeOptions[0].value },
} as FormData;

const srcTableList = ref<{ tableName: string; tableComment: string }[]>([]);
const srcTableListDisabled = ref(false);

const defaultKeys = ['tab-check', 'all', 'table-list'];

const state = reactive({
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
                    children: [] as { id: string; label: string; disabled?: boolean }[],
                },
            ],
        },
    ],
});

/** 传给 AutoFormDrawer 的回填数据（新建态用默认值，编辑态深拷贝行数据并补齐缺省项） */
const editData = computed<AutoFormData | null>(() => {
    if (props.data?.id) {
        const form = deepClone(props.data) as unknown as FormData;
        form.cronAble = form.cronAble || -1;
        form.mode = form.mode || 1;
        form.extra = form.extra || { fileType: fileTypeOptions[0].value };
        return form as unknown as AutoFormData;
    }
    return { ...basicFormData } as unknown as AutoFormData;
});

/** 抽屉打开后暂存的内部表单引用（源表勾选写 checkedKeys、提交时读取） */
const internalForm = ref<AutoFormData>({});

const { execute: saveExec } = dbTransferApi.saveDbTransferTask.useApi();

const onOpened = async (form: AutoFormData) => {
    internalForm.value = form;
    const propsData = props.data;
    if (!propsData?.id) {
        await nextTick(() => {
            srcTreeRef.value?.setCheckedKeys([]);
        });
        return;
    }

    const formData = form as unknown as FormData;
    const { srcDbId, targetDbId } = formData;

    //  初始化src数据源
    if (srcDbId) {
        // 通过tagPath查询实例列表
        const dbInfoRes = await dbApi.dbs.request({ id: srcDbId });
        const db = dbInfoRes.list[0] as Db & { databases?: string[] };
        // 初始化实例
        db.databases = db.database?.split(' ').sort() || [];

        if (srcDbId && formData.srcDbName) {
            await loadDbTables(srcDbId, formData.srcDbName);
        }
    }

    //  初始化target数据源
    if (targetDbId) {
        // 通过tagPath查询实例列表
        const dbInfoRes = await dbApi.dbs.request({ id: targetDbId });
        const db = dbInfoRes.list[0] as Db & { databases?: string[] };
        // 初始化实例
        db.databases = db.database?.split(' ').sort() || [];
    }

    // 初始化勾选迁移表
    srcTreeRef.value?.setCheckedKeys(formData.checkedKeys.split(','));
};

watch(
    () => state.filterSrcTableText,
    (val) => {
        srcTreeRef.value!.filter(val);
    }
);

interface DbSelectParams {
    id: number;
    db: string;
    dbs: string[];
    type: string;
    databases?: string[];
    name?: string;
}

const onSelectSrcDb = async (params: DbSelectParams) => {
    //  初始化数据源
    params.databases = params.dbs; // 数据源里需要这个值
    await loadDbTables(params.id, params.db);
};

const onSelectTargetDb = async (_params: DbSelectParams) => {
    // Target db selected
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
            internalForm.value.checkedKeys = 'all';
        } else {
            internalForm.value.checkedKeys = '';
        }
    }
    if (data.id && (data.id + '').startsWith('list-item')) {
        //
    }
};

const filterSrcTableTreeNode = (value: string, data: { label: string }) => {
    if (!value) return true;
    return data.label.includes(value);
};

const handleLoadSrcTableTree = () => {
    state.srcTableTree[0].children[1].children = srcTableList.value.map((item) => {
        return {
            id: item.tableName,
            label: item.tableName + (item.tableComment && '-' + item.tableComment),
            // 存入 reactive 后 Ref 会被自动解包为 boolean，保留响应式禁用状态
            disabled: srcTableListDisabled as unknown as boolean,
        };
    });
};

const srcTreeRef = ref();

const getCheckedKeys = () => {
    let checks = srcTreeRef.value!.getCheckedKeys(false);
    if (checks.indexOf('all') >= 0) {
        return ['all'];
    }
    return checks.filter((item: string) => !defaultKeys.includes(item));
};

// confirmApi 提交动作：组装勾选表并前置校验（失败抛错中止，组件保持抽屉打开）；成功提示与关闭抽屉由组件内置逻辑处理
const btnOk = async (rawForm: AutoFormData) => {
    const reqForm = { ...(rawForm as unknown as FormData) };

    let checkedKeys = getCheckedKeys();
    if (checkedKeys.length > 0) {
        reqForm.checkedKeys = checkedKeys.join(',');
    }

    if (!reqForm.checkedKeys) {
        Msg.error('db.noTransferTableMsg');
        throw new Error('no transfer tables checked');
    }

    await saveExec(reqForm);
    emit('val-change', rawForm);
};
</script>
<style lang="scss"></style>
