<template>
    <div>
        <page-table
            :show-selection="true"
            v-model:selection-data="selectionData"
            :data="configs"
            :columns="columns"
            :total="total"
            v-model:page-size="query.pageSize"
            v-model:page-num="query.pageNum"
            @pageChange="search()"
        >
            <template #queryRight>
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

        <el-dialog :before-close="closeSetConfigDialog" title="配置项设置" v-model="paramsDialog.visible" width="600px">
            <el-form v-if="paramsDialog.paramsFormItem.length > 0" ref="paramsFormRef" :model="paramsDialog.params" label-width="auto">
                <el-form-item v-for="item in paramsDialog.paramsFormItem" :key="item.name" :prop="item.model" :label="item.name" required>
                    <el-input
                        v-if="!item.options && !item.type"
                        v-model="paramsDialog.params[item.model]"
                        :placeholder="item.placeholder"
                        autocomplete="off"
                        clearable
                    ></el-input>
                    <el-checkbox
                        v-else-if="item.type == 'checkbox'"
                        v-model="paramsDialog.params[item.model]"
                        autocomplete="off"
                        :label="item.placeholder"
                        clearable
                    />
                    <el-select
                        v-else
                        v-model="paramsDialog.params[item.model]"
                        :placeholder="item.placeholder"
                        filterable
                        autocomplete="off"
                        clearable
                        style="width: 100%"
                    >
                        <el-option v-for="option in item.options.split(',')" :key="option" :label="option" :value="option" />
                    </el-select>
                </el-form-item>
            </el-form>
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
import { ref, toRefs, reactive, onMounted } from 'vue';
import ConfigEdit from './ConfigEdit.vue';
import { configApi } from '../api';
import { ElMessage } from 'element-plus';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { hasPerms } from '@/components/auth/auth';

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

const paramsFormRef: any = ref(null);
const state = reactive({
    query: {
        pageNum: 1,
        pageSize: 10,
        name: null,
    },
    total: 0,
    configs: [],
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

const { query, total, configs, selectionData, paramsDialog, configEdit } = toRefs(state);

onMounted(() => {
    if (Object.keys(actionBtns).length > 0) {
        columns.value.push(actionColumn);
    }
    search();
});

const search = async () => {
    let res = await configApi.list.request(state.query);
    state.configs = res.list;
    state.total = res.total;
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
        await paramsFormRef.value.validate(async (valid: boolean) => {
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
