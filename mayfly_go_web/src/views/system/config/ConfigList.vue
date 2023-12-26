<template>
    <div>
        <page-table ref="pageTableRef" :page-api="configApi.list" v-model:selection-data="selectionData" :columns="columns">
            <template #tableHeader>
                <el-button v-auth="perms.saveConfig" type="primary" icon="plus" @click="editConfig(false)">添加</el-button>
            </template>

            <template #status="{ data }">
                <el-tag v-if="data.status == 1" type="success">正常</el-tag>
                <el-tag v-if="data.status == -1" type="danger">禁用</el-tag>
            </template>

            <template #action="{ data }">
                <el-button :disabled="data.status == -1" type="warning" @click="showSetConfigDialog(data)" link>配置</el-button>
                <el-button v-if="actionBtns[perms.saveConfig]" @click="editConfig(data)" type="primary" link>编辑 </el-button>
            </template>
        </page-table>

        <el-dialog @close="closeSetConfigDialog" title="配置项设置" v-model="paramsDialog.visible" width="600px">
            <dynamic-form
                ref="paramsFormRef"
                v-if="paramsDialog.paramsFormItem.length > 0"
                :form-items="paramsDialog.paramsFormItem"
                v-model="paramsDialog.params"
            />

            <el-form v-else ref="paramsFormRef" label-width="auto">
                <el-form-item label="配置值" required>
                    <el-input v-model="paramsDialog.params" :placeholder="paramsDialog.config.remark" autocomplete="off" clearable></el-input>
                </el-form-item>
            </el-form>

            <template #footer>
                <span class="dialog-footer">
                    <el-button @click="closeSetConfigDialog()">取 消</el-button>
                    <el-button v-auth="'config:save'" type="primary" @click="setConfig()">确 定</el-button>
                </span>
            </template>
        </el-dialog>

        <config-edit :title="configEdit.title" v-model:visible="configEdit.visible" :data="configEdit.config" @val-change="configEditChange" />
    </div>
</template>

<script lang="ts" setup>
import { ref, toRefs, reactive, onMounted, Ref } from 'vue';
import ConfigEdit from './ConfigEdit.vue';
import { configApi } from '../api';
import { ElMessage } from 'element-plus';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { hasPerms } from '@/components/auth/auth';
import { DynamicForm } from '@/components/dynamic-form';

const perms = {
    saveConfig: 'config:save',
};
const columns = ref([
    TableColumn.new('name', '配置项'),
    TableColumn.new('key', '配置key'),
    TableColumn.new('value', '配置值').canBeautify(),
    TableColumn.new('remark', '备注'),
    TableColumn.new('modifier', '更新账号'),
    TableColumn.new('updateTime', '更新时间').isTime(),
]);
const actionColumn = TableColumn.new('action', '操作').isSlot().fixedRight().setMinWidth(130).noShowOverflowTooltip().alignCenter();
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
        title: '配置修改',
        visible: false,
        config: {},
    },
});

const { selectionData, paramsDialog, configEdit } = toRefs(state);

onMounted(() => {
    if (Object.keys(actionBtns).length > 0) {
        columns.value.push(actionColumn);
    }
});

const search = async () => {
    pageTableRef.value.search();
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

const closeSetConfigDialog = () => {
    state.paramsDialog.visible = false;
    setTimeout(() => {
        state.paramsDialog.config = {};
        state.paramsDialog.params = {};
        state.paramsDialog.paramsFormItem = [];
    }, 300);
};

const setConfig = async () => {
    let paramsValue = state.paramsDialog.params;
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
    ElMessage.success('保存成功');
    closeSetConfigDialog();
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

const configEditChange = () => {
    ElMessage.success('保存成功');
    search();
};

const editConfig = (data: any) => {
    if (data) {
        state.configEdit.config = data;
    } else {
        state.configEdit.config = false;
    }

    state.configEdit.visible = true;
};
</script>
<style lang="scss"></style>
