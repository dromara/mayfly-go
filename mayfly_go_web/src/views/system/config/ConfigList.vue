<template>
    <div class="role-list">
        <el-card>
            <el-button type="primary" icon="plus" @click="editConfig(false)">添加</el-button>
            <el-button :disabled="chooseId == null" @click="editConfig(chooseData)" type="primary" icon="edit">编辑
            </el-button>

            <el-table :data="configs" @current-change="choose" ref="table" style="width: 100%">
                <el-table-column label="选择" width="55px">
                    <template #default="scope">
                        <el-radio v-model="chooseId" :label="scope.row.id">
                            <i></i>
                        </el-radio>
                    </template>
                </el-table-column>
                <el-table-column prop="name" label="配置项" min-width="100px" show-overflow-tooltip></el-table-column>
                <el-table-column prop="key" label="配置key" min-width="100px"></el-table-column>
                <el-table-column prop="value" label="配置值" show-overflow-tooltip></el-table-column>
                <el-table-column prop="remark" label="备注" min-width="100px" show-overflow-tooltip></el-table-column>
                <el-table-column prop="updateTime" label="更新时间" min-width="100px">
                    <template #default="scope">
                        {{ dateFormat(scope.row.updateTime) }}
                    </template>
                </el-table-column>
                <el-table-column prop="modifier" label="修改者" min-width="60px" show-overflow-tooltip></el-table-column>
                <el-table-column label="操作" min-width="50" fixed="right">
                    <template #default="scope">
                        <el-link :disabled="scope.row.status == -1" type="warning"
                            @click="showSetConfigDialog(scope.row)" plain size="small" :underline="false">配置</el-link>
                    </template>
                </el-table-column>
            </el-table>
            <el-row style="margin-top: 20px" type="flex" justify="end">
                <el-pagination style="text-align: right" @current-change="handlePageChange" :total="total"
                    layout="prev, pager, next, total, jumper" v-model:current-page="query.pageNum"
                    :page-size="query.pageSize"></el-pagination>
            </el-row>
        </el-card>

        <el-dialog :before-close="closeSetConfigDialog" title="配置项设置" v-model="paramsDialog.visible" width="500px">
            <el-form v-if="paramsDialog.paramsFormItem.length > 0" ref="paramsForm" :model="paramsDialog.params"
                label-width="90px">
                <el-form-item v-for="item in paramsDialog.paramsFormItem" :key="item.name" :prop="item.model"
                    :label="item.name" required>
                    <el-input v-if="!item.options" v-model="paramsDialog.params[item.model]"
                        :placeholder="item.placeholder" autocomplete="off" clearable></el-input>
                    <el-select v-else v-model="paramsDialog.params[item.model]" :placeholder="item.placeholder"
                        filterable autocomplete="off" clearable style="width: 100%">
                        <el-option v-for="option in item.options.split(',')" :key="option" :label="option"
                            :value="option" />
                    </el-select>
                </el-form-item>
            </el-form>
            <el-form v-else ref="paramsForm" label-width="90px">
                <el-form-item label="配置值" required>
                    <el-input v-model="paramsDialog.params" :placeholder="paramsDialog.config.remark" autocomplete="off"
                        clearable></el-input>
                </el-form-item>
            </el-form>
            <template #footer>
                <span class="dialog-footer">
                    <el-button @click="closeSetConfigDialog()">取 消</el-button>
                    <el-button type="primary" @click="setConfig()">确 定</el-button>
                </span>
            </template>
        </el-dialog>

        <config-edit :title="configEdit.title" v-model:visible="configEdit.visible" :data="configEdit.config"
            @val-change="configEditChange" />
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, onMounted } from 'vue';
import ConfigEdit from './ConfigEdit.vue';
import { configApi } from '../api';
import { ElMessage } from 'element-plus';
import { dateFormat } from '@/common/utils/date';

const state = reactive({
    query: {
        pageNum: 1,
        pageSize: 10,
        name: null,
    },
    total: 0,
    configs: [],
    chooseId: null,
    chooseData: null,
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

const {
    query,
    total,
    configs,
    chooseId,
    chooseData,
    paramsDialog,
    configEdit,
} = toRefs(state)

onMounted(() => {
    search();
});

const search = async () => {
    let res = await configApi.list.request(state.query);
    state.configs = res.list;
    state.total = res.total;
};

const handlePageChange = (curPage: number) => {
    state.query.pageNum = curPage;
    search();
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
        // 如果配置项删除，则需要将value中对应的字段移除
        for (let paramKey in paramsValue) {
            if (!hasParam(paramKey, state.paramsDialog.paramsFormItem)) {
                delete paramsValue[paramKey];
            }
        }
        paramsValue = JSON.stringify(paramsValue);
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

const choose = (item: any) => {
    if (!item) {
        return;
    }
    state.chooseId = item.id;
    state.chooseData = item;
};

const configEditChange = () => {
    ElMessage.success('保存成功');
    state.chooseId = null;
    state.chooseData = null;
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
<style lang="scss">

</style>
