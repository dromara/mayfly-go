<template>
    <div class="mask-column-tag-edit">
        <auto-form-drawer
            v-model:visible="dialogVisible"
            :title="title"
            :items="items"
            :data="editData"
            :confirm-api="btnOk"
            @submitted="emit('val-change')"
            @cancel="emit('cancel')"
            @opened="onOpened"
        >
            <!-- 实例/库选择：与数据迁移一致，采用资源树选择（标签分组→实例→库），选中库后自动回填 instanceId 与 dbName -->
            <template #instanceId="{ form }">
                <db-select-tree
                    v-model:db-id="selectState.dbId"
                    v-model:code="selectState.code"
                    v-model:inst-name="selectState.instName"
                    v-model:instance-id="form.instanceId"
                    v-model:db-name="form.dbName"
                    @select-db="onSelectDb(form)"
                />
            </template>

            <!-- 表名/列名：选中库/表后联动加载选项，支持手输（留空匹配所有） -->
            <template #tableName="{ form }">
                <el-select
                    v-model="form.tableName"
                    filterable
                    clearable
                    allow-create
                    default-first-option
                    :loading="tableLoading"
                    :placeholder="$t('db.maskTableTips')"
                    @visible-change="(visible: boolean) => visible && loadTableOptions(form)"
                    @change="onTableChange(form)"
                >
                    <el-option v-for="tn in tableOptions" :key="tn" :label="tn" :value="tn" />
                </el-select>
            </template>
            <template #columnName="{ form }">
                <el-select
                    v-model="form.columnName"
                    filterable
                    clearable
                    allow-create
                    default-first-option
                    :loading="columnLoading"
                    :placeholder="$t('db.maskColumnTips')"
                    @visible-change="(visible: boolean) => visible && loadColumnOptions(form)"
                >
                    <el-option v-for="cn in columnOptions" :key="cn" :label="cn" :value="cn" />
                </el-select>
            </template>
        </auto-form-drawer>
    </div>
</template>

<script lang="ts" setup>
import { Rules } from '@/common/rule';
import { deepClone } from '@/common/utils/object';
import { AutoFormDrawer, type AutoFormData, type AutoFormItem } from '@/components/auto-form';
import { computed, reactive, ref, type PropType } from 'vue';
import { dbApi, dbMaskApi } from '../../api';
import DbSelectTree from '../DbSelectTree.vue';
import type { DbMaskColumn, DbMaskRule } from '../../types';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();

const props = defineProps({
    data: {
        type: Object as PropType<DbMaskColumn | null>,
        default: null,
    },
    title: {
        type: String,
    },
});

const emit = defineEmits(['update:visible', 'cancel', 'val-change']);

const dialogVisible = defineModel<boolean>('visible', { default: false });

const rules = ref<DbMaskRule[]>([]);

/** 资源树选择的临时状态：dbId/code/instName 仅用于选择器展示与回填，不随表单保存 */
const selectState = reactive({
    dbId: undefined as number | undefined,
    code: '',
    instName: '',
});

/** 表名/列名下拉选项（选中库/表后联动加载，均支持手输通配） */
const tableOptions = ref<string[]>([]);
const columnOptions = ref<string[]>([]);
const tableLoading = ref(false);
const columnLoading = ref(false);

const algorithmOptions = [
    { label: t('db.maskAlgoFull'), value: 'full' },
    { label: t('db.maskAlgoPartial'), value: 'partial' },
    { label: t('db.maskAlgoHash'), value: 'hash' },
    { label: t('db.maskAlgoRegexReplace'), value: 'regexReplace' },
    { label: t('db.maskAlgoPhone'), value: 'phone' },
    { label: t('db.maskAlgoEmail'), value: 'email' },
    { label: t('db.maskAlgoIdcard'), value: 'idcard' },
    { label: t('db.maskAlgoBankCard'), value: 'bankCard' },
];

/** 表单声明（AutoFormItem[]），绑定规则/算法仅在动作=绑定时展示；库名由资源树选择带出，不再单独表单项 */
const items: AutoFormItem[] = [
    { prop: 'instanceId', label: 'db.maskInstance', type: 'custom', required: true, rules: [Rules.requiredSelect('db.maskInstance')] },
    {
        prop: 'tableName',
        label: 'db.tableName',
        type: 'custom',
        span: 12,
        tooltip: 'db.maskTableTips',
    },
    {
        prop: 'columnName',
        label: 'db.columnName',
        type: 'custom',
        span: 12,
        tooltip: 'db.maskColumnTips',
    },
    {
        prop: 'action',
        label: 'db.maskAction',
        type: 'radio',
        required: true,
        options: [
            { label: 'db.maskActionBind', value: 1 },
            { label: 'db.maskActionExempt', value: 2 },
        ],
    },
    {
        prop: 'ruleId',
        label: 'db.maskBindRule',
        type: 'select',
        span: 12,
        when: (f) => f.action === 1,
        required: (f) => f.action === 1 && !f.algorithm,
        rules: [Rules.requiredSelect('db.maskBindRule')],
        // 选项异步加载脱敏规则库
        options: async () => {
            if (rules.value.length === 0) {
                try {
                    const res = await dbMaskApi.maskRules.request({ pageNum: 1, pageSize: 200 });
                    rules.value = res.list;
                } catch (e) {
                    //
                }
            }
            return rules.value.map((r) => ({ label: r.name, value: r.id }));
        },
    },
    {
        prop: 'algorithm',
        label: 'db.maskAlgorithm',
        type: 'select',
        span: 12,
        when: (f) => f.action === 1,
        tooltip: 'db.maskBindRuleTips',
        options: algorithmOptions,
    },
    {
        prop: 'params',
        label: 'db.maskParams',
        type: 'textarea',
        placeholder: 'db.maskParamsPlaceholder',
        tooltip: 'db.maskParamsPlaceholder',
    },
    { prop: 'remark', label: 'common.remark', type: 'textarea' },
];

type FormData = {
    id?: number;
    instanceId: number;
    dbName?: string;
    tableName?: string;
    columnName?: string;
    action: number;
    ruleId?: number;
    algorithm?: string;
    params?: string;
    remark?: string;
};

const basicFormData = {
    instanceId: undefined,
    dbName: '',
    tableName: '',
    columnName: '',
    action: 1,
} as unknown as FormData;

/** 新建态用默认值，编辑态深拷贝行数据回填 */
const editData = computed<AutoFormData | null>(() => {
    if (props.data?.id) {
        return deepClone(props.data) as unknown as AutoFormData;
    }
    return { ...basicFormData } as unknown as AutoFormData;
});

/** 加载当前库下表名选项 */
const loadTableOptions = async (rawForm: AutoFormData) => {
    const form = rawForm as unknown as FormData;
    if (!selectState.dbId || !form.dbName) {
        return;
    }
    tableLoading.value = true;
    try {
        const res = await dbApi.tableInfos.request({ id: selectState.dbId, db: form.dbName });
        tableOptions.value = res.map((x) => x.tableName);
    } catch (e) {
        //
    } finally {
        tableLoading.value = false;
    }
};

/** 加载当前表下列名选项 */
const loadColumnOptions = async (rawForm: AutoFormData) => {
    const form = rawForm as unknown as FormData;
    if (!selectState.dbId || !form.dbName || !form.tableName) {
        return;
    }
    columnLoading.value = true;
    try {
        const res = await dbApi.columnMetadata.request({ id: selectState.dbId, db: form.dbName, tableName: form.tableName });
        columnOptions.value = res.map((x) => x.columnName);
    } catch (e) {
        //
    } finally {
        columnLoading.value = false;
    }
};

/** 切换库后重置表/列及其选项，并预加载表名选项 */
const onSelectDb = (rawForm: AutoFormData) => {
    const form = rawForm as unknown as FormData;
    form.tableName = '';
    form.columnName = '';
    tableOptions.value = [];
    columnOptions.value = [];
    loadTableOptions(form);
};

/** 切换表后重置列并预加载列名选项 */
const onTableChange = (rawForm: AutoFormData) => {
    const form = rawForm as unknown as FormData;
    form.columnName = '';
    columnOptions.value = [];
    loadColumnOptions(form);
};

const onOpened = async () => {
    // 新增态：重置选择器临时状态，避免残留上一次编辑的回显信息
    if (!props.data?.id) {
        selectState.dbId = undefined;
        selectState.code = '';
        selectState.instName = '';
        tableOptions.value = [];
        columnOptions.value = [];
        return;
    }

    const form = editData.value as unknown as FormData;

    // 编辑态：仅有 instanceId，按实例反查 db 记录回填 dbId（用于加载表/列选项与路径回显）
    if (props.data.instanceId) {
        try {
            const res = await dbApi.dbs.request({ instanceId: props.data.instanceId });
            const db = res.list?.[0];
            if (db) {
                selectState.dbId = db.id;
                selectState.code = db.code;
                selectState.instName = db.name;
            }
        } catch (e) {
            //
        }
    }

    // 已选库/表时预加载表/列选项
    await loadTableOptions(form);
    await loadColumnOptions(form);
};

const btnOk = async (rawForm: AutoFormData) => {
    const reqForm = { ...(rawForm as unknown as FormData) };
    // 豁免动作不携带规则/算法
    if (reqForm.action !== 1) {
        reqForm.ruleId = 0;
        reqForm.algorithm = '';
    }
    if (reqForm.id) {
        await dbMaskApi.updateMaskColumn.request(reqForm);
    } else {
        await dbMaskApi.saveMaskColumn.request(reqForm);
    }
    emit('val-change', reqForm);
};
</script>
<style lang="scss" scoped></style>
