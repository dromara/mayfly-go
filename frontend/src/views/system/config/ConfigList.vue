<template>
    <div class="h-full">
        <page-table
            ref="pageTableRef"
            :search-items="searchItems"
            :page-api="configApi.list"
            :columns="columns"
            v-model:query-form="query"
            :data-handler-fn="handleData"
        >
            <template #tableHeader>
                <el-button v-auth="perms.saveConfig" type="primary" icon="plus" @click="onEditConfig(false)">{{ $t('common.create') }}</el-button>
            </template>

            <template #status="{ data }">
                <el-tag v-if="data.status == 1" type="success">{{ $t('common.enable') }}</el-tag>
                <el-tag v-if="data.status == -1" type="danger">{{ $t('common.disable') }}</el-tag>
            </template>

            <template #action="{ data }">
                <el-button :disabled="data.status == -1" type="warning" @click="showSetConfigDialog(data)" link>{{ $t('system.sysconf.conf') }}</el-button>
                <el-button v-if="actionBtns[perms.saveConfig]" @click="onEditConfig(data)" type="primary" link>{{ $t('common.edit') }}</el-button>
            </template>
        </page-table>

        <el-dialog @close="onCloseSetConfigDialog" :title="$t('system.sysconf.confItemSetting')" v-model="paramsDialog.visible" width="700px">
            <auto-form
                ref="paramsFormRef"
                v-if="paramsDialog.schema"
                :schema="paramsDialog.schema"
                v-model="(paramsDialog.params as any)"
            />

            <!-- 无 schema 时降级为单值输入（params 为原始值，经 computed 代理为对象表单） -->
            <auto-form v-else ref="paramsFormRef" v-model="fallbackForm" :items="fallbackItems" label-width="auto" />

            <template #footer>
                <el-button @click="onCloseSetConfigDialog()">{{ $t('common.cancel') }}</el-button>
                <el-button v-auth="'config:save'" type="primary" @click="setConfig()">{{ $t('common.confirm') }}</el-button>
            </template>
        </el-dialog>

        <config-edit
            :title="$t(state.configEdit.title)"
            v-model:visible="state.configEdit.visible"
            :data="state.configEdit.config"
            @val-change="onConfigEditChange"
        />
    </div>
</template>

<script lang="ts" setup>
import { hasPerms } from '@/components/auth/auth';
import { AutoForm, type AutoFormItem } from '@/components/auto-form';
import { isJsonFormSchema, type AutoFormJsonSchema } from '@/components/auto-form/json';
import { TableColumn } from '@/components/page-table';
import PageTable from '@/components/page-table/PageTable.vue';
import { SearchItem } from '@/components/page-table/SearchForm';
import { Msg } from '@/hooks/useI18n';
import { computed, defineAsyncComponent, onMounted, reactive, ref, toRefs, useTemplateRef } from 'vue';
import { useI18n } from 'vue-i18n';
import { configApi } from '../api';
import type { SysConfig } from '../types';
import type { PageResult } from '@/types/common';

const ConfigEdit = defineAsyncComponent(() => import('./ConfigEdit.vue'));

const { t } = useI18n();

const perms = {
    saveConfig: 'config:save',
};

const searchItems = [SearchItem.input('key', 'system.sysconf.confKey')];

const columns = ref([
    TableColumn.new('i18nName', 'system.sysconf.confItem'),
    TableColumn.new('key', 'system.sysconf.confKey'),
    TableColumn.new('value', 'system.sysconf.confValue').canBeautify(),
    TableColumn.new('i18nRemark', 'common.remark'),
    TableColumn.new('modifier', 'common.modifier'),
    TableColumn.new('updateTime', 'common.updateTime').isTime(),
]);
const actionColumn = TableColumn.new('action', 'common.operation').isSlot().fixedRight().setMinWidth(130).noShowOverflowTooltip().alignCenter();
const actionBtns = hasPerms([perms.saveConfig]);

const pageTableRef = useTemplateRef<InstanceType<typeof PageTable>>('pageTableRef');
const paramsFormRef = ref<{ validate: () => Promise<unknown> } | null>(null);

const state = reactive({
    query: {
        pageNum: 1,
        pageSize: 0,
        name: null,
    },
    selectionData: [],
    paramsDialog: {
        visible: false,
        config: {} as SysConfig,
        params: {} as Record<string, unknown> | string,
        /** 配置项表单定义（v1 JSON Schema，无定义时为 null） */
        schema: null as AutoFormJsonSchema | null,
    },
    configEdit: {
        title: 'common.edit',
        visible: false,
        config: false as SysConfig | false,
    },
});

const { query, paramsDialog } = toRefs(state);

/** 降级表单代理：params 为原始值，包装为 { value } 对象供 auto-form 使用 */
const fallbackForm = computed({
    get: () => ({ value: paramsDialog.value.params as unknown as string }),
    set: (v) => (state.paramsDialog.params = v.value),
});

/** 降级表单声明（placeholder 动态取配置项备注，经 props 透传覆盖） */
const fallbackItems = computed<AutoFormItem[]>(() => [
    { prop: 'value', label: 'system.sysconf.confValue', required: true, props: { placeholder: state.paramsDialog.config.remark } },
]);

onMounted(() => {
    if (Object.keys(actionBtns).length > 0) {
        columns.value.push(actionColumn);
    }
});

const search = async () => {
    pageTableRef.value?.search();
};

const handleData = (res: PageResult<SysConfig & { i18nName?: string; i18nRemark?: string }>) => {
    const dataList = res.list;
    // 内容国际化
    for (let x of dataList) {
        x.i18nName = t(x.name);
        x.i18nRemark = t(x.remark);
    }
    return res;
};

const showSetConfigDialog = (row: SysConfig) => {
    state.paramsDialog.config = row;
    // 存在配置项表单定义则弹窗提示输入对应的配置项
    if (row.params) {
        try {
            const parsed = JSON.parse(row.params);
            state.paramsDialog.schema = isJsonFormSchema(parsed) ? parsed : null;
        } catch {
            state.paramsDialog.schema = null;
        }
    } else {
        state.paramsDialog.schema = null;
    }
    if (state.paramsDialog.schema?.fields?.length && row.value) {
        state.paramsDialog.params = JSON.parse(row.value);
    } else if (!state.paramsDialog.schema?.fields?.length) {
        state.paramsDialog.schema = null;
        state.paramsDialog.params = row.value;
    }
    state.paramsDialog.visible = true;
};

const onCloseSetConfigDialog = () => {
    state.paramsDialog.visible = false;
    setTimeout(() => {
        state.paramsDialog.config = {} as SysConfig;
        state.paramsDialog.params = {};
        state.paramsDialog.schema = null;
    }, 300);
};

const setConfig = async () => {
    let paramsValue: Record<string, unknown> | string | null = state.paramsDialog.params;
    if (state.paramsDialog.schema) {
        // AutoForm暴露的validate为Promise风格（不接收回调参数），校验失败时reject
        try {
            await paramsFormRef.value?.validate();
        } catch (e) {
            // 校验失败，不保存
            return;
        }
        const paramsObj = state.paramsDialog.params as Record<string, unknown>;
        // 如果配置项删除，则需要将value中对应的字段移除
        for (let paramKey in paramsObj) {
            if (!hasParam(paramKey, state.paramsDialog.schema.fields)) {
                delete paramsObj[paramKey];
            }
        }
        paramsValue = JSON.stringify(paramsObj);
    }
    // 说明校验失败
    if (paramsValue == null) {
        return;
    }
    await configApi.save.request({
        id: state.paramsDialog.config.id,
        key: state.paramsDialog.config.key,
        name: state.paramsDialog.config.name,
        value: paramsValue,
    });
    Msg.saveSuccess();
    onCloseSetConfigDialog();
    search();
};

const hasParam = (paramKey: string, fields: { prop: string }[]) => {
    for (let field of fields) {
        if (field.prop == paramKey) {
            return true;
        }
    }
    return false;
};

const onConfigEditChange = () => {
    Msg.saveSuccess();
    search();
};

const onEditConfig = (data: SysConfig | false) => {
    if (data) {
        state.configEdit.title = 'common.edit';
        state.configEdit.config = data;
    } else {
        state.configEdit.title = 'common.create';
        state.configEdit.config = false;
    }

    state.configEdit.visible = true;
};
</script>
<style lang="scss"></style>
