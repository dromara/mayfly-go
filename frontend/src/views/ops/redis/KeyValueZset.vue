<template>
    <div>
        <el-button @click="showEditDialog(null)" icon="plus" size="small" plain type="primary" class="mb10">添加新行</el-button>
        <el-table size="small" border :data="values" height="450" min-height="300" stripe>
            <el-table-column type="index" :label="'ID (Total: ' + total + ')'" sortable width="100"> </el-table-column>
            <el-table-column resizable sortable prop="score" label="score" show-overflow-tooltip min-width="100"> </el-table-column>
            <el-table-column resizable sortable prop="value" label="value" show-overflow-tooltip min-width="200"> </el-table-column>
            <el-table-column label="操作">
                <template #header>
                    <el-input
                        class="key-detail-filter-value"
                        v-model="state.filterValue"
                        @keyup.enter="zscanData(true)"
                        placeholder="输入关键词回车搜索"
                        clearable
                        size="small"
                    />
                </template>
                <template #default="scope">
                    <el-link @click="showEditDialog(scope.row)" :underline="false" type="primary" icon="edit" plain></el-link>
                    <el-popconfirm title="确定删除?" @confirm="zrem(scope.row, scope.$index)">
                        <template #reference>
                            <el-link v-auth="'redis:data:del'" :underline="false" type="danger" icon="delete" size="small" plain class="ml5"></el-link>
                        </template>
                    </el-popconfirm>
                </template>
            </el-table-column>
        </el-table>
        <!-- load more content -->
        <div class="content-more-container">
            <el-button size="small" @click="loadDatas()" :disabled="loadMoreDisable" class="content-more-btn"> 加载更多 </el-button>
        </div>

        <el-dialog title="添加新行" v-model="editDialog.visible" width="600px" :destroy-on-close="true" :close-on-click-modal="false">
            <el-form>
                <el-form-item>
                    <el-input type="number" v-model.number="editDialog.score" placeholder="score" />
                </el-form-item>
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
import { ElMessage } from 'element-plus';
import FormatViewer from './FormatViewer.vue';
import { RedisInst } from './redis';

const props = defineProps({
    redis: {
        type: RedisInst,
        required: true,
    },
    keyInfo: {
        type: [Object],
    },
});

const formatViewerRef = ref(null) as any;

const state = reactive({
    key: '',
    filterValue: '',
    scanCursor: 0,
    pageNum: 1,
    pageSize: 50,
    total: 0,
    values: [] as any,
    loadMoreDisable: false,
    editDialog: {
        visible: false,
        score: 0,
        content: '',
        dataRow: null as any,
    },
});

const { total, values, loadMoreDisable, editDialog } = toRefs(state);

onMounted(() => {
    state.key = props.keyInfo?.key;
    initData();
});

const initData = async () => {
    state.pageNum = 1;
    state.filterValue = '';
    getTotal();
    await zrevrange(true);
};

const loadDatas = (resetTableData = false) => {
    if (state.filterValue) {
        zscanData(resetTableData);
        return;
    }
    zrevrange(resetTableData);
};

const zrevrange = async (resetTableData = false) => {
    const pageNum = state.pageNum;
    const pageSize = state.pageSize;
    // ZREVRANGE key start stop [WITHSCORES]
    const res = await props.redis.runCmd(['ZREVRANGE', state.key, (pageNum - 1) * pageSize, pageNum * pageSize - 1, 'WITHSCORES']);

    const vs = [];
    for (let member of res) {
        vs.push({
            value: member[0],
            score: member[1],
        });
    }
    if (resetTableData) {
        state.values = vs;
    } else {
        state.values.push(...vs);
    }
    state.pageNum++;
    state.loadMoreDisable = state.total <= state.values.length;
};

const getScanMatch = () => {
    return state.filterValue ? `*${state.filterValue}*` : '*';
};

const zscanData = async (resetTableData = true, resetCursor = false) => {
    if (resetCursor) {
        state.scanCursor = 0;
    }
    // ZSCAN key cursor [MATCH pattern] [COUNT count]
    // 响应[coursor, vals[]]
    const res = await props.redis.runCmd(['ZSCAN', state.key, state.scanCursor, 'MATCH', getScanMatch(), 'COUNT', state.pageSize]);

    const keys = res[1];
    const vs = [];
    const memCount = keys.length / 2;
    let nextMemndex = 0;
    for (let i = 0; i < memCount; i++) {
        vs.push({ value: keys[nextMemndex++], score: keys[nextMemndex++] });
    }

    if (resetTableData) {
        state.values = vs;
    } else {
        state.values.push(...vs);
    }

    state.scanCursor = res[0];
    state.loadMoreDisable = state.scanCursor == 0;
};

const getTotal = () => {
    // ZCARD key
    props.redis.runCmd(['ZCARD', state.key]).then((res) => {
        state.total = res;
    });
};

const showEditDialog = (row: any) => {
    state.editDialog.dataRow = row;
    state.editDialog.content = row ? row.value : '';
    state.editDialog.score = row ? row.score : null;
    state.editDialog.visible = true;
};

const confirmEditData = async () => {
    // 存在数据行，则说明为修改，则要先删除旧数据后新增
    const dataRow = state.editDialog.dataRow;
    if (dataRow) {
        // ZREM key member [member ...]
        await props.redis.runCmd(['ZREM', state.key, state.editDialog.dataRow.value]);
    }

    const score = state.editDialog.score;
    // 获取zset member内容并新增
    const member = formatViewerRef.value.getContent();
    // ZADD key [NX | XX] [GT | LT] [CH] [INCR] score member [score member...]
    await props.redis.runCmd(['ZADD', state.key, score, member]);

    ElMessage.success('保存成功');
    if (dataRow) {
        state.editDialog.dataRow.value = member;
        state.editDialog.dataRow.score = score;
    } else {
        state.values.unshift({ value: member, score });
        state.total++;
    }
    state.editDialog.visible = false;
    state.editDialog.dataRow = null;
};

const zrem = async (row: any, index: any) => {
    await props.redis.runCmd(['ZREM', state.key, row.value]);
    ElMessage.success('删除成功');
    state.values.splice(index, 1);
    state.total--;
};

defineExpose({ initData });
</script>
<style lang="scss"></style>
