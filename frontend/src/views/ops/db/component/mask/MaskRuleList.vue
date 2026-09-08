<template>
    <div class="mask-rule-list h-full p-3">
        <el-tabs v-model="state.activeTab">
            <el-tab-pane :label="$t('db.maskRule')" name="rule">
                <page-table
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
                            v-model="data.status"
                            inline-prompt
                            :active-text="$t('common.enable')"
                            :inactive-text="$t('common.disable')"
                            :active-value="1"
                            :inactive-value="0"
                            @click="changeRuleStatus(data)"
                        />
                    </template>

                    <template #action="{ data }">
                        <el-button @click="editRule(data)" type="primary" link>{{ $t('common.edit') }}</el-button>
                        <el-button type="danger" link @click="delRule(data)">{{ $t('common.delete') }}</el-button>
                    </template>
                </page-table>
            </el-tab-pane>

            <el-tab-pane :label="$t('db.maskColumnTag')" name="tag">
                <page-table
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

                    <template #action1="{ data }">
                        <el-button @click="editTag(data)" type="primary" link>{{ $t('common.edit') }}</el-button>
                        <el-button type="danger" link @click="delTag(data)">{{ $t('common.delete') }}</el-button>
                    </template>
                </page-table>
            </el-tab-pane>
        </el-tabs>

        <mask-rule-edit v-model:visible="ruleEditDialog.visible" :title="ruleEditDialog.title" v-model:data="ruleEditDialog.data" @val-change="searchRules" />
        <mask-column-tag-edit v-model:visible="tagEditDialog.visible" :title="tagEditDialog.title" v-model:data="tagEditDialog.data" @val-change="searchTags" />
    </div>
</template>

<script lang="ts" setup>
import PageTable from '@/components/page-table/PageTable.vue';
import { TableColumn } from '@/components/page-table';
import { SearchItem } from '@/components/page-table/SearchForm';
import { Msg, useI18nDeleteConfirm } from '@/hooks/useI18n';
import { useI18n } from 'vue-i18n';
import { defineAsyncComponent, reactive, ref, toRefs, useTemplateRef } from 'vue';
import { dbApi, dbMaskApi } from '../../api';
import { DbMaskMatchTypeEnum, DbMaskTagActionEnum } from '../../enums';
import type { DbInstance, DbMaskColumn, DbMaskRule } from '../../types';

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
    TableColumn.new('status', 'common.status').isSlot().alignCenter(),
    TableColumn.new('remark', 'common.remark'),
    TableColumn.new('action', 'common.operation').isSlot().setMinWidth(110).fixedRight().alignCenter(),
]);

const tagColumns = ref([
    TableColumn.new('instanceId', 'db.maskInstance').isSlot().setMinWidth(140),
    TableColumn.new('scope', 'db.maskTagScope').isSlot().setMinWidth(180),
    TableColumn.new('action', 'db.maskAction').typeTag(DbMaskTagActionEnum).alignCenter(),
    TableColumn.new('algorithm', 'db.maskAlgorithm').isSlot().setMinWidth(130),
    TableColumn.new('remark', 'common.remark'),
    TableColumn.new('action1', 'common.operation').isSlot().setMinWidth(110).fixedRight().alignCenter(),
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
    const parts = [
        `${t('db.db')}: ${data.dbName || '*'}`,
        `${t('db.table')}: ${data.tableName || '*'}`,
        `${t('db.columnName')}: ${data.columnName || '*'}`,
    ];
    return parts.join(' / ');
};

const ruleTableRef = useTemplateRef<InstanceType<typeof PageTable>>('ruleTableRef');
const tagTableRef = useTemplateRef<InstanceType<typeof PageTable>>('tagTableRef');

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

const changeRuleStatus = async (rule: DbMaskRule) => {
    try {
        await dbMaskApi.updateMaskRule.request({ ...rule });
        Msg.operateSuccess();
    } catch (e) {
        rule.status = rule.status === 1 ? 0 : 1;
    }
};

const delRule = async (rule: DbMaskRule) => {
    try {
        await useI18nDeleteConfirm(rule.name);
        await dbMaskApi.deleteMaskRule.request({ id: rule.id });
        Msg.deleteSuccess();
        searchRules();
    } catch (e) {
        //
    }
};

const delTag = async (tag: DbMaskColumn) => {
    try {
        await useI18nDeleteConfirm(tag.columnName || tag.tableName || tag.dbName || String(tag.id));
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
