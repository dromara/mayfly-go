<template>
    <div>
        <el-button @click="showEditDialog(null)" icon="plus" size="small" plain type="primary" class="mb10">添加新行</el-button>
        <el-table size="small" border :data="values" height="450" min-height=300 stripe>
            <el-table-column type="index" :label="'ID (Total: ' + total + ')'" sortable width="100">
            </el-table-column>
            <el-table-column resizable sortable prop="score" label="score" show-overflow-tooltip min-width="100">
            </el-table-column>
            <el-table-column resizable sortable prop="value" label="value" show-overflow-tooltip min-width="200">
            </el-table-column>
            <el-table-column label="操作">
                <template #header>
                    <el-input class="key-detail-filter-value" v-model="state.filterValue" @keyup.enter='zscanData(true)'
                        placeholder="输入关键词回车搜索" clearable size="small" />
                </template>
                <template #default="scope">
                    <el-link @click="showEditDialog(scope.row)" :underline="false" type="primary" icon="edit"
                        plain></el-link>
                    <el-popconfirm title="确定删除?" @confirm="zrem(scope.row, scope.$index)">
                        <template #reference>
                            <el-link v-auth="'redis:data:del'" :underline="false" type="danger" icon="delete" size="small"
                                plain class="ml5"></el-link>
                        </template>
                    </el-popconfirm>

                </template>
            </el-table-column>
        </el-table>
        <!-- load more content -->
        <div class='content-more-container'>
            <el-button size='small' @click='loadDatas()' :disabled='loadMoreDisable' class='content-more-btn'>
                加载更多
            </el-button>
        </div>

        <el-dialog title="添加新行" v-model="editDialog.visible" width="600px" :destroy-on-close="true"
            :close-on-click-modal="false">
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
        default: 0
    },
    keyInfo: {
        type: [Object],
    },
})

const formatViewerRef = ref(null) as any;

const state = reactive({
    redisId: 0,
    db: 0,
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
    }
});

const {
    total,
    values,
    loadMoreDisable,
    editDialog,
} = toRefs(state)

onMounted(() => {
    state.redisId = props.redisId;
    state.db = props.db;
    state.key = props.keyInfo?.key;
    initData();
})

const initData = async () => {
    state.pageNum = 1;
    state.filterValue = '';
    await getTotal();
    await zrevrange(true);
}

const loadDatas = (resetTableData = false) => {
    if (state.filterValue) {
        zscanData(resetTableData);
        return;
    }
    zrevrange(resetTableData);
}

const zrevrange = async (resetTableData = false) => {
    const pageNum = state.pageNum;
    const pageSize = state.pageSize;
    const res = await redisApi.zrevrange.request({
        ...getBaseReqParam(),
        start: (pageNum - 1) * pageSize,
        stop: pageNum * pageSize - 1,
    })

    const vs = [];
    for (let member of res) {
        vs.push({
            score: member.Score,
            value: member.Member
        })
    }
    if (resetTableData) {
        state.values = vs;
    } else {
        state.values.push(...vs);
    }
    state.pageNum++;
    state.loadMoreDisable = state.total <= state.values.length
}

const getScanMatch = () => {
    return state.filterValue ? `*${state.filterValue}*` : '*';
}

const zscanData = async (resetTableData = true, resetCursor = false) => {
    if (resetCursor) {
        state.scanCursor = 0;
    }
    const res = await redisApi.zscan.request({
        ...getBaseReqParam(),
        match: getScanMatch(),
        cursor: state.scanCursor,
        count: state.pageSize
    });

    const keys = res.keys;
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

    state.scanCursor = res.cursor;
    state.loadMoreDisable = res.cursor == 0
};

const getTotal = () => {
    redisApi.zcard.request({
        id: state.redisId,
        db: state.db,
        key: state.key
    }).then((res) => {
        state.total = res;
    });
}

const showEditDialog = (row: any) => {
    state.editDialog.dataRow = row;
    state.editDialog.content = row ? row.value : '';
    state.editDialog.score = row ? row.score : null;
    state.editDialog.visible = true;
}

const confirmEditData = async () => {
    const param = getBaseReqParam();

    // 存在数据行，则说明为修改，则要先删除旧数据后新增
    const dataRow = state.editDialog.dataRow
    if (dataRow) {
        await redisApi.zrem.request({
            member: state.editDialog.dataRow.value,
            ...param
        });
    }

    const score = state.editDialog.score
    // 获取zset member内容并新增
    const member = formatViewerRef.value.getContent()
    await redisApi.zadd.request({
        score,
        member,
        ...param
    });

    ElMessage.success("保存成功");
    if (dataRow) {
        state.editDialog.dataRow.value = member;
        state.editDialog.dataRow.score = score;
    } else {
        state.values.unshift({ value: member, score });
        state.total++;
    }
    state.editDialog.visible = false;
    state.editDialog.dataRow = null;
}

const zrem = async (row: any, index: any) => {
    await redisApi.zrem.request({
        ...getBaseReqParam(),
        member: row.value,
    })
    ElMessage.success("删除成功");
    state.values.splice(index, 1)
    state.total--;
}

const getBaseReqParam = () => {
    return {
        id: state.redisId,
        db: state.db,
        key: state.key
    }
}

defineExpose({ initData })
</script>
<style lang="scss"></style>
