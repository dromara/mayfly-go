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
                :form-items="paramsDialog.paramsFormItem"
                v-model="paramsDialog.params"
            />

            <el-form v-else ref="paramsFormRef" label-width="auto">
                <el-form-item :label="$t('system.sysconf.confValue')" required>
                    <el-input v-model="paramsDialog.params" :placeholder="paramsDialog.config.remark" autocomplete="off" clearable></el-input>
                </el-form-item>
            </el-form>

            <template #footer>
                <span class="dialog-footer">
                    <el-button @click="onCloseSetConfigDialog()">{{ $t('common.cancel') }}</el-button>
                    <el-button v-auth="'config:save'" type="primary" @click="setConfig()">{{ $t('common.confirm') }}</el-button>
                </span>
            </template>
        </el-dialog>

        <config-edit :title="$t(configEdit.title)" v-model:visible="configEdit.visible" :data="configEdit.config" @val-change="onConfigEditChange" />
    </div>
</template>

<script lang="ts" setup>
import { ref, toRefs, reactive, onMounted, Ref } from 'vue';
import ConfigEdit from './ConfigEdit.vue';
import { configApi } from '../api';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { hasPerms } from '@/components/auth/auth';
import { DynamicForm } from '@/components/dynamic-form';
import { SearchItem } from '@/components/SearchForm';
import { useI18n } from 'vue-i18n';
import { useI18nSaveSuccessMsg } from '@/hooks/useI18n';

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

const pageTableRef: Ref<any> = ref(null);
const paramsFormRef: any = ref(null);

const state = reactive({
    query: {
        pageNum: 1,
        pageSize: 0,
        name: null,
    },
    selectionData: [],
    paramsDialog: {
        visible: false,
        config: null as any,
        params: {},
        paramsFormItem: [] as any,
    },
    configEdit: {
        title: 'common.edit',
        visible: false,
        config: {},
    },
});

const { query, paramsDialog, configEdit } = toRefs(state);

onMounted(() => {
    if (Object.keys(actionBtns).length > 0) {
        columns.value.push(actionColumn);
    }
});

const search = async () => {
    pageTableRef.value.search();
};

const handleData = (res: any) => {
    const dataList = res.list;
    // 内容国际化
    for (let x of dataList) {
        x.i18nName = t(x.name);
        x.i18nRemark = t(x.remark);
    }
    return res;
};

const showSetConfigDialog = (row: any) => {
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
        state.paramsDialog.config = {};
        state.paramsDialog.params = {};
        state.paramsDialog.paramsFormItem = [];
    }, 300);
};

const setConfig = async () => {
    let paramsValue: any = state.paramsDialog.params;
    if (state.paramsDialog.paramsFormItem.length > 0) {
        await paramsFormRef.value.validate((valid: boolean) => {
            if (!valid) {
                paramsValue = null as any;
                return false;
            }
            if (state.paramsDialog.paramsFormItem.length > 0) {
                // 如果配置项删除，则需要将value中对应的字段移除
                for (let paramKey in paramsValue) {
                    if (!hasParam(paramKey, state.paramsDialog.paramsFormItem)) {
                        delete paramsValue[paramKey];
                    }
                }
                paramsValue = JSON.stringify(paramsValue);
            }
        });
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
    useI18nSaveSuccessMsg();
    onCloseSetConfigDialog();
    search();
};

const hasParam = (paramKey: string, paramItems: any) => {
    for (let paramItem of paramItems) {
        if (paramItem.model == paramKey) {
            return true;
        }
    }
    return false;
};

const onConfigEditChange = () => {
    useI18nSaveSuccessMsg();
    search();
};

const onEditConfig = (data: any) => {
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
