<template>
    <div class="role-list">
        <el-card>
            <el-button type="primary" icon="plus" @click="editConfig(false)">添加</el-button>
            <el-button :disabled="chooseId == null" @click="editConfig(chooseData)" type="primary" icon="edit">编辑</el-button>

            <el-table :data="configs" @current-change="choose" ref="table" style="width: 100%">
                <el-table-column label="选择" width="55px">
                    <template #default="scope">
                        <el-radio v-model="chooseId" :label="scope.row.id">
                            <i></i>
                        </el-radio>
                    </template>
                </el-table-column>
                <el-table-column prop="name" label="配置项"></el-table-column>
                <el-table-column prop="key" label="配置key"></el-table-column>
                <el-table-column prop="value" label="配置值" min-width="100px" show-overflow-tooltip></el-table-column>
                <el-table-column prop="remark" label="备注" min-width="100px" show-overflow-tooltip></el-table-column>
                <el-table-column prop="updateTime" label="更新时间">
                    <template #default="scope">
                        {{ $filters.dateFormat(scope.row.createTime) }}
                    </template>
                </el-table-column>
                <el-table-column prop="modifier" label="修改者" show-overflow-tooltip></el-table-column>
            </el-table>
            <el-row style="margin-top: 20px" type="flex" justify="end">
                <el-pagination
                    style="text-align: right"
                    @current-change="handlePageChange"
                    :total="total"
                    layout="prev, pager, next, total, jumper"
                    v-model:current-page="query.pageNum"
                    :page-size="query.pageSize"
                ></el-pagination>
            </el-row>
        </el-card>

        <config-edit :title="configEdit.title" v-model:visible="configEdit.visible" :data="configEdit.config" @val-change="configEditChange" />
    </div>
</template>

<script lang="ts">
import { toRefs, reactive, onMounted, defineComponent } from 'vue';
import ConfigEdit from './ConfigEdit.vue';
import { configApi } from '../api';
import { ElMessage, ElMessageBox } from 'element-plus';
export default defineComponent({
    name: 'ConfigList',
    components: {
        ConfigEdit,
    },
    setup() {
        const state = reactive({
            dialogFormVisible: false,
            currentEditPermissions: false,
            query: {
                pageNum: 1,
                pageSize: 10,
                name: null,
            },
            total: 0,
            configs: [],
            chooseId: null,
            chooseData: null,
            configEdit: {
                title: '配置修改',
                visible: false,
                config: {},
            },
        });

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

        const choose = (item: any) => {
            if (!item) {
                return;
            }
            state.chooseId = item.id;
            state.chooseData = item;
        };

        const configEditChange = () => {
            ElMessage.success('修改成功！');
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

        return {
            ...toRefs(state),
            search,
            handlePageChange,
            choose,
            configEditChange,
            editConfig,
        };
    },
});
</script>
<style lang="scss">
</style>
