<template>
    <div class="mask-rule-list h-full p-3">
        <el-tabs v-model="state.activeTab">
            <el-tab-pane :label="$t('db.maskRule')" name="rule">
                <data-table
                    ref="ruleTableRef"
                    :page-api="dbMaskApi.maskRules"
                    :search-items="ruleSearchItems"
                    v-model:query-form="ruleQuery"
                    :columns="ruleColumns"
                >
                    <template #tableHeader>
                        <el-button type="primary" icon="plus" @click="editRule(false)">{{ $t('common.create') }}</el-button>
                    </template>

                    <template #algorithm="{ data }">
                        <span>{{ getAlgoLabel(data.algorithm) }}</span>
                    </template>

                    <template #status="{ data }">
                        <el-switch
                            :model-value="
                                data.status === DbMaskRuleStatusEnum.Enabled.value ? DbMaskRuleStatusEnum.Enabled.value : DbMaskRuleStatusEnum.Disabled.value
                            "
                            @change="(val: number) => changeRuleStatus(data, val)"
                            inline-prompt
                            :active-text="$t('common.enable')"
                            :inactive-text="$t('common.disable')"
                            :active-value="DbMaskRuleStatusEnum.Enabled.value"
                            :inactive-value="DbMaskRuleStatusEnum.Disabled.value"
                        />
                    </template>
                </data-table>
            </el-tab-pane>

            <el-tab-pane :label="$t('db.maskColumnTag')" name="tag">
                <data-table
                    ref="tagTableRef"
                    :page-api="dbMaskApi.maskColumns"
                    :search-items="tagSearchItems"
                    v-model:query-form="tagQuery"
                    :columns="tagColumns"
                >
                    <template #tableHeader>
                        <el-button type="primary" icon="plus" @click="editTag(false)">{{ $t('common.create') }}</el-button>
                    </template>

                    <template #scope="{ data }">
                        <span>{{ formatTagScope(data) }}</span>
                    </template>

                    <template #instanceId="{ data }">
                        <span>{{ formatInstance(data.instanceId) }}</span>
                    </template>

                    <!-- 直接指定的算法（优先于绑定规则） -->
                    <template #algorithm="{ data }">
                        <span>{{ data.algorithm ? getAlgoLabel(data.algorithm) : '-' }}</span>
                    </template>
                </data-table>
            </el-tab-pane>
        </el-tabs>

        <mask-rule-edit v-model:visible="ruleEditDialog.visible" :title="ruleEditDialog.title" v-model:data="ruleEditDialog.data" @val-change="searchRules" />
        <mask-column-tag-edit v-model:visible="tagEditDialog.visible" :title="tagEditDialog.title" v-model:data="tagEditDialog.data" @val-change="searchTags" />
    </div>
</template>

<script lang="ts" setup>
import DataTable from '@/components/data-table/DataTable.vue';
import { TableColumn } from '@/components/data-table';
import { SearchItem } from '@/components/data-table/SearchForm';
import { Msg, useI18nDeleteConfirm } from '@/hooks/useI18n';
import { useI18n } from 'vue-i18n';
import { defineAsyncComponent, reactive, ref, toRefs, useTemplateRef } from 'vue';
import { dbApi, dbMaskApi } from '../api';
import { DbMaskMatchTypeEnum, DbMaskRuleStatusEnum, DbMaskTagActionEnum } from '../enums';
import type { DbInstance, DbMaskColumn, DbMaskRule } from '../types';

const MaskRuleEdit = defineAsyncComponent(() => import('./MaskRuleEdit.vue'));
const MaskColumnTagEdit = defineAsyncComponent(() => import('./MaskColumnTagEdit.vue'));

const { t } = useI18n();

const ruleSearchItems = [SearchItem.input('name', 'common.name'), SearchItem.input('pattern', 'db.maskPattern')];

const tagSearchItems = [
    SearchItem.input('dbName', 'db.dbName'),
    SearchItem.input('tableName', 'db.tableName'),
    SearchItem.input('columnName', 'db.columnName'),
];

const ruleColumns = ref([
    TableColumn.new('name', 'db.maskRuleName').setMinWidth(110),
    TableColumn.new('matchType', 'db.maskMatchType').typeTag(DbMaskMatchTypeEnum).alignCenter(),
    TableColumn.new('pattern', 'db.maskPattern').setMinWidth(150),
    TableColumn.new('algorithm', 'db.maskAlgorithm').isSlot().setMinWidth(130),
    TableColumn.new('weight', 'db.maskWeight').alignCenter(),
    // 状态列插槽渲染定宽 el-switch（圆钮不在字段文本里，离线估算与只量溢出回填都量不到）：+20 预留圆钮占位
    TableColumn.new('status', 'common.status').isSlot().setAddWidth(20).alignCenter(),
    TableColumn.new('remark', 'common.remark'),
    TableColumn.new('action', 'common.operation')
        .setActions<DbMaskRule>([
            { key: 'edit', label: 'common.edit', type: 'primary', onClick: (row) => editRule(row) },
            { key: 'del', label: 'common.delete', type: 'danger', onClick: (row) => delRule(row) },
        ])
        .fixedRight()
        .alignCenter(),
]);

const tagColumns = ref([
    TableColumn.new('instanceId', 'db.maskInstance').isSlot().setMinWidth(140),
    TableColumn.new('scope', 'db.maskTagScope').isSlot().setMinWidth(180),
    TableColumn.new('action', 'db.maskAction').typeTag(DbMaskTagActionEnum).alignCenter(),
    TableColumn.new('algorithm', 'db.maskAlgorithm').isSlot().setMinWidth(130),
    TableColumn.new('remark', 'common.remark'),
    TableColumn.new('action1', 'common.operation')
        .setActions<DbMaskColumn>([
            { key: 'edit', label: 'common.edit', type: 'primary', onClick: (row) => editTag(row) },
            { key: 'del', label: 'common.delete', type: 'danger', onClick: (row) => delTag(row) },
        ])
        .fixedRight()
        .alignCenter(),
]);

/** 算法标识 -> i18n key 尾段 */
const algoLabelKeys: Record<string, string> = {
    full: 'db.maskAlgoFull',
    partial: 'db.maskAlgoPartial',
    hash: 'db.maskAlgoHash',
    regexReplace: 'db.maskAlgoRegexReplace',
    phone: 'db.maskAlgoPhone',
    email: 'db.maskAlgoEmail',
    idcard: 'db.maskAlgoIdcard',
    bankCard: 'db.maskAlgoBankCard',
};

const getAlgoLabel = (algorithm: string) => {
    const key = algoLabelKeys[algorithm];
    return key ? t(key) : algorithm;
};

/** 实例列表（id -> name(host) 展示） */
const instances = ref<DbInstance[]>([]);

const loadInstances = async () => {
    if (instances.value.length > 0) {
        return;
    }
    try {
        const res = await dbApi.instances.request({ pageNum: 1, pageSize: 500 });
        instances.value = res.list;
    } catch (e) {
        //
    }
};
loadInstances();

const formatInstance = (id: number) => {
    const inst = instances.value.find((i) => i.id === id);
    return inst ? `${inst.name}(${inst.host})` : String(id);
};

/** 列标签生效范围文案：库名/表名/列名，留空为全部 */
const formatTagScope = (data: DbMaskColumn) => {
    const parts = [`${t('db.db')}: ${data.dbName || '*'}`, `${t('db.table')}: ${data.tableName || '*'}`, `${t('db.columnName')}: ${data.columnName || '*'}`];
    return parts.join(' / ');
};

const ruleTableRef = useTemplateRef<InstanceType<typeof DataTable>>('ruleTableRef');
const tagTableRef = useTemplateRef<InstanceType<typeof DataTable>>('tagTableRef');

const state = reactive({
    activeTab: 'rule',
    ruleQuery: {
        name: '',
        pattern: '',
        pageNum: 1,
        pageSize: 10,
    },
    tagQuery: {
        dbName: '',
        tableName: '',
        columnName: '',
        pageNum: 1,
        pageSize: 10,
    },
    ruleEditDialog: {
        visible: false,
        data: null as DbMaskRule | null,
        title: '',
    },
    tagEditDialog: {
        visible: false,
        data: null as DbMaskColumn | null,
        title: '',
    },
});

const { ruleQuery, tagQuery, ruleEditDialog, tagEditDialog } = toRefs(state);

const searchRules = () => {
    ruleTableRef.value?.search();
};

const searchTags = () => {
    tagTableRef.value?.search();
};

const editRule = (data: DbMaskRule | false) => {
    if (!data) {
        state.ruleEditDialog.data = null;
        state.ruleEditDialog.title = t('db.maskCreateRule');
    } else {
        state.ruleEditDialog.data = data;
        state.ruleEditDialog.title = t('db.maskEditRule');
    }
    state.ruleEditDialog.visible = true;
};

const editTag = (data: DbMaskColumn | false) => {
    if (!data) {
        state.tagEditDialog.data = null;
        state.tagEditDialog.title = t('db.maskCreateTag');
    } else {
        state.tagEditDialog.data = data;
        state.tagEditDialog.title = t('db.maskEditTag');
    }
    state.tagEditDialog.visible = true;
};

const changeRuleStatus = async (rule: DbMaskRule, status: number) => {
    const prev = rule.status;
    rule.status = status;
    try {
        await dbMaskApi.updateMaskRule.request({ ...rule });
        Msg.operateSuccess();
    } catch (e) {
        // 保存失败则回滚开关状态
        rule.status = prev;
    }
};

const delRule = async (rule: DbMaskRule) => {
    try {
        if (!(await useI18nDeleteConfirm(rule.name))) {
            return;
        }
        await dbMaskApi.deleteMaskRule.request({ id: rule.id });
        Msg.deleteSuccess();
        searchRules();
    } catch (e) {
        //
    }
};

const delTag = async (tag: DbMaskColumn) => {
    try {
        if (!(await useI18nDeleteConfirm(tag.columnName || tag.tableName || tag.dbName || String(tag.id)))) {
            return;
        }
        await dbMaskApi.deleteMaskColumn.request({ id: tag.id });
        Msg.deleteSuccess();
        searchTags();
    } catch (e) {
        //
    }
};
</script>
<style lang="scss" scoped>
.mask-rule-list {
    :deep(.el-tabs__content) {
        height: calc(100% - 55px);
    }
}
</style>
