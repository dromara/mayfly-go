<template>
    <div>
        <el-button @click="showEditDialog(null)" icon="plus" size="small" plain type="primary" class="mb10">添加新行</el-button>
        <el-table size="small" border :data="values" height="450" min-height="300" stripe>
            <el-table-column type="index" :label="'ID (Total: ' + total + ')'" sortable width="100"> </el-table-column>
            <el-table-column resizable sortable prop="value" label="value" show-overflow-tooltip min-width="200"> </el-table-column>
            <el-table-column label="操作">
                <template #default="scope">
                    <el-link @click="showEditDialog(scope.row)" :underline="false" type="primary" icon="edit" plain></el-link>
                    <el-popconfirm title="确定删除?" @confirm="lrem(scope.row, scope.$index)">
                        <template #reference>
                            <el-link v-auth="'redis:data:del'" :underline="false" type="danger" icon="delete" size="small" plain class="ml5"></el-link>
                        </template>
                    </el-popconfirm>
                </template>
            </el-table-column>
        </el-table>
        <!-- load more content -->
        <div class="content-more-container">
            <el-button size="small" @click="getListValue(false)" :disabled="loadMoreDisable" class="content-more-btn"> 加载更多 </el-button>
        </div>

        <el-dialog title="添加新行" v-model="editDialog.visible" width="600px" :destroy-on-close="true" :close-on-click-modal="false">
            <el-form>
                <el-form-item>
                    <format-viewer class="w100" ref="formatViewerRef" :content="editDialog.content"></format-viewer>
                </el-form-item>
            </el-form>

            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="editDialog.visible = false">取 消</el-button>
                    <el-button v-auth="'redis:data:save'" type="primary" @click="confirmEditData">确 定</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>
<script lang="ts" setup>
import { ref, reactive, toRefs, onMounted } from 'vue';
import { redisApi } from './api';
import { ElMessage } from 'element-plus';
import FormatViewer from './FormatViewer.vue';

const props = defineProps({
    redisId: {
        type: [Number],
        require: true,
        default: 0,
    },
    db: {
        type: [Number],
        require: true,
        default: 0,
    },
    keyInfo: {
        type: [Object],
    },
});

const formatViewerRef = ref(null) as any;

const state = reactive({
    redisId: 0,
    db: 0,
    key: '',
    pageNum: 1,
    pageSize: 50,
    total: 0,
    values: [] as any,
    loadMoreDisable: false,
    editDialog: {
        visible: false,
        content: '',
        dataRow: null as any,
    },
});

const { total, values, loadMoreDisable, editDialog } = toRefs(state);

onMounted(() => {
    state.redisId = props.redisId;
    state.db = props.db;
    state.key = props.keyInfo?.key;
    initData();
});

const initData = () => {
    state.pageNum = 1;
    getListValue(true);
};

const getListValue = async (resetTableData = false) => {
    const pageNum = state.pageNum;
    const pageSize = state.pageSize;
    const res = await redisApi.getListValue.request({
        ...getBaseReqParam(),
        start: (pageNum - 1) * pageSize,
        stop: pageNum * pageSize - 1,
    });
    state.total = res.len;

    const datas = res.list.map((x: any) => {
        return {
            value: x,
        };
    });
    if (resetTableData) {
        state.values = datas;
    } else {
        state.values.push(...datas);
    }
    state.pageNum++;
    state.loadMoreDisable = state.values.length === state.total;
};

// const lset = async (row: any, rowIndex: number) => {
//     await redisApi.setListValue.request({
//         ...getBaseReqParam(),
//         index: (state.pageNum - 1) * state.pageSize + rowIndex,
//         value: row.value,
//     });
//     ElMessage.success('数据保存成功');
// };

const showEditDialog = (row: any) => {
    state.editDialog.dataRow = row;
    state.editDialog.content = row ? row.value : '';
    state.editDialog.visible = true;
};

const confirmEditData = async () => {
    const param = getBaseReqParam();

    // 存在数据行，则说明为修改，则要先删除旧数据后新增
    const dataRow = state.editDialog.dataRow;
    if (dataRow) {
        await redisApi.lrem.request({
            member: state.editDialog.dataRow.value,
            count: 1,
            ...param,
        });
    }

    // 获取list member内容并新增
    const member = formatViewerRef.value.getContent();
    await redisApi.saveListValue.request({
        value: [member],
        ...param,
    });

    ElMessage.success('保存成功');
    if (dataRow) {
        state.editDialog.dataRow.value = member;
    } else {
        state.values.push({ value: member });
        state.total++;
    }
    state.editDialog.visible = false;
    state.editDialog.dataRow = null;
};

const lrem = async (row: any, index: any) => {
    await redisApi.lrem.request({
        ...getBaseReqParam(),
        member: row.value,
        count: 1,
    });
    ElMessage.success('删除成功');
    state.values.splice(index, 1);
    state.total--;
};

const getBaseReqParam = () => {
    return {
        id: state.redisId,
        db: state.db,
        key: state.key,
    };
};

defineExpose({ initData });
</script>
<style lang="scss"></style>
