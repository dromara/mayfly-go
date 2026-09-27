<template>
    <div class="db-transfer-edit">
        <auto-form-drawer
            v-model:visible="dialogVisible"
            :title="title"
            :items="items"
            :data="editData"
            size="45%"
            :confirm-api="btnOk"
            @submitted="emit('cancel')"
            @opened="onOpened"
            @cancel="emit('cancel')"
        >
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
                    <el-input v-model="state.filterSrcTableText" :placeholder="$t('db.transferTableFilter')" size="small" />
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
import { AutoFormDrawer, defineFormItems } from '@/components/auto-form';
import { useAutoFormModel } from '@/hooks/useAutoFormModel';
import CrontabInput from '@/components/crontab/CrontabInput.vue';
import SvgIcon from '@/components/svg-icon/index.vue';
import { dbApi } from '@/views/ops/db/api';
import DbSelectTree from '@/views/ops/db/widgets/DbSelectTree.vue';
import { getDbDialect, getDbDialectMap } from '@/views/ops/db/dialect';
import { dbTransferApi } from '@/views/ops/db/transfer/api';
import type { DbTransferTaskListVO, Db, DbNodeParams } from '@/views/ops/db/types';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();

const props = defineProps({
    data: {
        type: Object as PropType<DbTransferTaskListVO | null>,
        default: null,
    },
    title: {
        type: String,
    },
});

//定义事件
const emit = defineEmits<{
    /** 取消编辑，父级关闭弹窗 */
    cancel: [];
    /** 保存成功，回传表单，父级据此刷新任务列表 */
    'val-change': [form: DbTransferForm];
}>();

const dialogVisible = defineModel<boolean>('visible', { default: false });

const fileTypeOptions = [
    { label: '.zip', value: 'zip' },
    { label: '.sql', value: 'sql' },
];

/** 表单声明（defineFormItems<DbTransferForm>，渲染 + 校验唯一数据源；复杂控件走 custom 插槽承载） */
const items = defineFormItems<DbTransferForm>([
    { prop: 'taskName', label: 'db.taskName', required: true },
    {
        prop: 'status',
        label: 'common.status',
        type: 'switch',
        span: 12,
        props: { inlinePrompt: true, activeText: t('common.enable'), inactiveText: t('common.disable'), activeValue: 1, inactiveValue: -1 },
    },
    {
        prop: 'cronEnabled',
        label: 'db.cronEnabled',
        type: 'radio',
        span: 12,
        required: true,
        options: [
            { label: 'common.yes', value: 1 },
            { label: 'common.no', value: -1 },
        ],
    },
    { prop: 'cron', label: 'cron', type: 'custom', required: (f) => f.cronEnabled == 1 },
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
        prop: 'concurrency',
        label: 'db.concurrency',
        type: 'number',
        span: 12,
        when: (f) => f.mode === 1,
        min: 1,
        max: 16,
        props: { placeholder: t('db.concurrencyTips'), step: 1, stepStrictly: true },
    },
    {
        prop: 'nameCase',
        label: 'db.nameCase',
        tooltip: 'db.nameCaseTips',
        type: 'radio',
        span: 12,
        required: true,
        options: [
            { label: 'db.none', value: 1 },
            { label: 'db.upper', value: 2 },
            { label: 'db.lower', value: 3 },
        ],
    },
    {
        prop: 'deleteTable',
        label: 'db.deleteTable',
        type: 'radio',
        span: 12,
        required: true,
        tooltip: 'db.deleteTableTips',
        options: [
            { label: 'common.yes', value: 1 },
            { label: 'common.no', value: 2 },
        ],
    },
    { prop: 'dbObjDivider', label: 'db.dbObj', type: 'divider' },
    // 迁移表由左侧树勾选写入 checkedKeys，故校验直接读树的勾选结果
    { prop: 'checkedKeys', type: 'custom', validate: () => (getCheckedKeys().length > 0 ? true : 'db.noTransferTableMsg') },
]);

/**
 * 迁移任务编辑表单
 *
 * 枚举型字段（mode/strategy/nameCase/deleteTable/cronEnabled/runningState）取后端同一数值型，
 * 合法取值由 items 中对应的枚举选项限定，避免与行数据（实体同样使用 number）之间靠断言换算。
 */
type DbTransferForm = {
    id?: number;
    taskName: string;
    status: number;
    cronEnabled: number;
    cron: string;
    mode: number;
    targetFileDbType?: string;
    fileSaveDays?: number;
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
    strategy: number;
    /** 迁移并发度（1~16），0/空表示用后端默认值4 */
    concurrency?: number;
    nameCase: number;
    deleteTable?: number;
    checkedKeys: string;
    runningState: number;
    /** 导出附加参数；表单经嵌套路径 'extra.fileType' 写入，形状由后端决定，故与实体一致保持开放记录 */
    extra?: Record<string, unknown>;
};

/** 新建态默认值（taskName/cron 由用户输入，此处为空串） */
const basicFormData: DbTransferForm = {
    taskName: '',
    mode: 1,
    status: 1,
    cron: '',
    cronEnabled: -1,
    strategy: 1,
    nameCase: 1,
    deleteTable: 1,
    concurrency: 4,
    checkedKeys: '',
    runningState: 1,
    extra: { fileType: fileTypeOptions[0].value },
};

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

/** 抽屉打开后接管的内部表单（源表勾选写 checkedKeys、提交时读取） */
const { openedWith, requireForm } = useAutoFormModel<DbTransferForm>();

const { execute: saveExec } = dbTransferApi.saveDbTransferTask.useApi();

/** 传给 AutoFormDrawer 的回填数据（新建态用默认值，编辑态深拷贝行数据并补齐缺省项） */
const editData = computed<DbTransferForm | null>(() => {
    if (props.data?.id) {
        const row = deepClone(props.data);
        return {
            ...basicFormData,
            ...row,
            cronEnabled: row.cronEnabled || -1,
            mode: row.mode || 1,
            deleteTable: row.deleteTable || 1,
            extra: row.extra || { fileType: fileTypeOptions[0].value },
        };
    }
    return { ...basicFormData };
});

const onOpened = openedWith(async (form) => {
    const propsData = props.data;
    if (!propsData?.id) {
        await nextTick(() => {
            srcTreeRef.value?.setCheckedKeys([]);
        });
        return;
    }

    const { srcDbId, targetDbId } = form;

    //  初始化src数据源
    if (srcDbId) {
        // 通过tagPath查询实例列表
        const dbInfoRes = await dbApi.dbs.request({ id: srcDbId });
        const db = dbInfoRes.list[0] as Db & { databases?: string[] };
        // 初始化实例
        db.databases = db.database?.split(' ').sort() || [];

        if (srcDbId && form.srcDbName) {
            await loadDbTables(srcDbId, form.srcDbName);
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
    srcTreeRef.value?.setCheckedKeys(form.checkedKeys ? form.checkedKeys.split(',') : []);
});

watch(
    () => state.filterSrcTableText,
    (val) => {
        srcTreeRef.value!.filter(val);
    }
);

const onSelectSrcDb = async (params: DbNodeParams) => {
    //  初始化数据源
    params.databases = params.dbs; // 数据源里需要这个值
    await loadDbTables(params.id, params.db);
};

const onSelectTargetDb = async (_params: DbNodeParams) => {
    // Target db selected
};

/** 读取源表树的勾选结果（树未挂载时为空） */
const getCheckedKeys = () => {
    const tree = srcTreeRef.value;
    if (!tree) {
        return [];
    }
    let checks = tree.getCheckedKeys(false);
    if (checks.indexOf('all') >= 0) {
        return ['all'];
    }
    return checks.filter((item: string) => !defaultKeys.includes(item));
};

const loadDbTables = async (dbId: number, db: string) => {
    // 加载db下的表
    srcTableList.value = await dbApi.tableInfos.request({ id: dbId, db });
    handleLoadSrcTableTree();
};

const handleSrcTableCheckChange = (data: { id: string; name: string }, checked: boolean) => {
    if (data.id === 'all') {
        srcTableListDisabled.value = checked;
        const form = requireForm();
        if (checked) {
            form.checkedKeys = 'all';
        } else {
            form.checkedKeys = '';
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
            // 存入 reactive 后 Ref 会被自动解包为 boolean，保留响应式禁用状态（解包发生在代理层，静态不可推知）
            disabled: srcTableListDisabled as unknown as boolean,
        };
    });
};

const srcTreeRef = ref();

// confirmApi 提交动作：把树勾选结果写回 checkedKeys（“至少选一张表”已由 checkedKeys 字段的 validate 声明）
const btnOk = async () => {
    const reqForm = { ...requireForm() };

    const checkedKeys = getCheckedKeys();
    if (checkedKeys.length > 0) {
        reqForm.checkedKeys = checkedKeys.join(',');
    }

    await saveExec(reqForm);
    emit('val-change', reqForm);
};
</script>
<style lang="scss"></style>
