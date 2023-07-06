<template>
    <div>
        <el-button @click="showEditDialog(null)" icon="plus" size="small" plain type="primary" class="mb10">添加新行</el-button>
        <el-table size="small" border :data="setDatas" height="450" min-height="300" stripe>
            <el-table-column type="index" :label="'ID (Total: ' + total + ')'" sortable width="100"> </el-table-column>
            <el-table-column resizable sortable prop="value" label="value" show-overflow-tooltip min-width="200"> </el-table-column>
            <el-table-column label="操作">
                <template #header>
                    <el-input
                        class="key-detail-filter-value"
                        v-model="state.filterValue"
                        @keyup.enter="sscanData(true, true)"
                        placeholder="输入关键词回车搜索"
                        clearable
                        size="small"
                    />
                </template>
                <template #default="scope">
                    <el-link @click="showEditDialog(scope.row)" :underline="false" type="primary" icon="edit" plain></el-link>
                    <el-popconfirm title="确定删除?" @confirm="srem(scope.row, scope.$index)">
                        <template #reference>
                            <el-link v-auth="'redis:data:del'" :underline="false" type="danger" icon="delete" size="small" plain class="ml5"></el-link>
                        </template>
                    </el-popconfirm>
                </template>
            </el-table-column>
        </el-table>
        <!-- load more content -->
        <div class="content-more-container">
            <el-button size="small" @click="sscanData(false)" :disabled="loadMoreDisable" class="content-more-btn"> 加载更多 </el-button>
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

    filterValue: '',
    scanParam: {
        count: 50,
        cursor: 0,
    },
    total: 0,
    setDatas: [] as any,
    loadMoreDisable: false,
    value: [{ value: '' }],
    editDialog: {
        visible: false,
        content: '',
        dataRow: null as any,
    },
});

const { total, setDatas, loadMoreDisable, editDialog } = toRefs(state);

onMounted(() => {
    state.redisId = props.redisId;
    state.db = props.db;
    state.key = props.keyInfo?.key;
    initData();
});

const initData = () => {
    state.filterValue = '';
    sscanData(true, true);
    getTotal();
};

const getScanMatch = () => {
    return state.filterValue ? `*${state.filterValue}*` : '*';
};

const sscanData = async (resetDatas = true, resetCursor = false) => {
    if (resetCursor) {
        state.scanParam.cursor = 0;
    }
    const res = await redisApi.sscan.request({
        ...getBaseReqParam(),
        match: getScanMatch(),
        ...state.scanParam,
    });

    if (resetDatas) {
        state.setDatas = [];
    }
    res.keys.forEach((x: any) => {
        state.setDatas.push({
            value: x,
        });
    });
    state.scanParam.cursor = res.cursor;
    state.loadMoreDisable = res.cursor == 0;
};

const getTotal = () => {
    redisApi.scard.request(getBaseReqParam()).then((res) => {
        state.total = res;
    });
};

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
        await redisApi.srem.request({
            member: state.editDialog.dataRow.value,
            ...param,
        });
    }

    // 获取set member内容并新增
    const member = formatViewerRef.value.getContent();
    await redisApi.sadd.request({
        member,
        ...param,
    });

    ElMessage.success('保存成功');
    if (dataRow) {
        state.editDialog.dataRow.value = member;
    } else {
        state.setDatas.unshift({ value: member });
        state.total++;
    }
    state.editDialog.visible = false;
    state.editDialog.dataRow = null;
};

const srem = async (row: any, index: any) => {
    await redisApi.srem.request({
        ...getBaseReqParam(),
        member: row.value,
    });
    ElMessage.success('删除成功');
    state.setDatas.splice(index, 1);
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
