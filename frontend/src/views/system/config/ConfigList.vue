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
            <dynamic-form
                ref="paramsFormRef"
                v-if="paramsDialog.paramsFormItem.length > 0"
                :form-items="(paramsDialog.paramsFormItem as any)"
                v-model="(paramsDialog.params as any)"
            />

            <el-form v-else ref="paramsFormRef" label-width="auto">
                <el-form-item :label="$t('system.sysconf.confValue')" required>
                    <el-input v-model="paramsDialog.params" :placeholder="paramsDialog.config.remark" autocomplete="off" clearable></el-input>
                </el-form-item>
            </el-form>

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
import { DynamicForm } from '@/components/dynamic-form';
import { TableColumn } from '@/components/page-table';
import PageTable from '@/components/page-table/PageTable.vue';
import { SearchItem } from '@/components/page-table/SearchForm';
import { Msg } from '@/hooks/useI18n';
import { defineAsyncComponent, onMounted, reactive, ref, toRefs, useTemplateRef } from 'vue';
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
const paramsFormRef = ref<{ validate: (cb: (valid: boolean) => void) => void } | null>(null);

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
        paramsFormItem: [] as Record<string, unknown>[],
    },
    configEdit: {
        title: 'common.edit',
        visible: false,
        config: false as SysConfig | false,
    },
});

const { query, paramsDialog } = toRefs(state);

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
    // 存在配置项则弹窗提示输入对应的配置项
    if (row.params) {
        state.paramsDialog.paramsFormItem = JSON.parse(row.params);
        if (state.paramsDialog.paramsFormItem && state.paramsDialog.paramsFormItem.length > 0) {
            if (row.value) {
                state.paramsDialog.params = JSON.parse(row.value);
            }
        } else {
            state.paramsDialog.params = row.value;
        }
    } else {
        state.paramsDialog.params = row.value;
    }
    state.paramsDialog.visible = true;
};

const onCloseSetConfigDialog = () => {
    state.paramsDialog.visible = false;
    setTimeout(() => {
        state.paramsDialog.config = {} as SysConfig;
        state.paramsDialog.params = {};
        state.paramsDialog.paramsFormItem = [];
    }, 300);
};

const setConfig = async () => {
    let paramsValue: Record<string, unknown> | string | null = state.paramsDialog.params;
    if (state.paramsDialog.paramsFormItem.length > 0) {
        // DynamicForm暴露的validate为Promise风格（不接收回调参数），校验失败时reject
        try {
            await paramsFormRef.value?.validate();
        } catch (e) {
            // 校验失败，不保存
            return;
        }
        const paramsObj = state.paramsDialog.params as Record<string, unknown>;
        // 如果配置项删除，则需要将value中对应的字段移除
        for (let paramKey in paramsObj) {
            if (!hasParam(paramKey, state.paramsDialog.paramsFormItem)) {
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

const hasParam = (paramKey: string, paramItems: Record<string, unknown>[]) => {
    for (let paramItem of paramItems) {
        if (paramItem.model == paramKey) {
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
