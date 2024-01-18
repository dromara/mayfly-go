<template>
    <div>
        <el-button @click="showEditDialog(null)" icon="plus" size="small" plain type="primary" class="mb10">添加新行</el-button>
        <el-table size="small" border :data="hashValues" height="500" min-height="300" stripe>
            <el-table-column type="index" :label="'ID (Total: ' + total + ')'" sortable width="100"> </el-table-column>
            <el-table-column resizable sortable prop="field" label="field" show-overflow-tooltip min-width="100"> </el-table-column>
            <el-table-column resizable sortable prop="value" label="value" show-overflow-tooltip min-width="200"> </el-table-column>
            <el-table-column label="操作">
                <template #header>
                    <el-input
                        class="key-detail-filter-value"
                        v-model="state.filterValue"
                        @keyup.enter="hscan(true, true)"
                        placeholder="关键词回车搜索"
                        clearable
                        size="small"
                    />
                </template>
                <template #default="scope">
                    <el-link @click="showEditDialog(scope.row)" :underline="false" type="primary" icon="edit" plain></el-link>
                    <el-popconfirm title="确定删除?" @confirm="hdel(scope.row.field, scope.$index)">
                        <template #reference>
                            <el-link v-auth="'redis:data:del'" :underline="false" type="danger" icon="delete" size="small" plain class="ml5"></el-link>
                        </template>
                    </el-popconfirm>
                </template>
            </el-table-column>
        </el-table>
        <!-- load more content -->
        <div class="content-more-container">
            <el-button size="small" @click="hscan()" :disabled="loadMoreDisable" class="content-more-btn"> 加载更多 </el-button>
        </div>

        <el-dialog title="添加新行" v-model="editDialog.visible" width="600px" :destroy-on-close="true" :close-on-click-modal="false">
            <el-form>
                <el-form-item>
                    <el-input v-model="editDialog.field" placeholder="field" />
                </el-form-item>
                <el-form-item>
                    <format-viewer class="w100" ref="formatViewerRef" :content="editDialog.value"></format-viewer>
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
import { ref, onMounted, reactive, toRefs } from 'vue';
import { redisApi } from './api';
import { ElMessage } from 'element-plus';
import { notBlank } from '@/common/assert';
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
    scanParam: {
        cursor: 0,
        count: 50,
    },
    filterValue: '',
    hashValues: [] as any,
    total: 0,
    loadMoreDisable: false,
    editDialog: {
        visible: false,
        field: '',
        value: '',
        dataRow: null as any,
    },
});

const { hashValues, total, loadMoreDisable, editDialog } = toRefs(state);

onMounted(() => {
    state.redisId = props.redisId;
    state.db = props.db;
    state.key = props.keyInfo?.key;
    initData();
});

const initData = () => {
    state.filterValue = '';
    hscan(true, true);
};

const getScanMatch = () => {
    return state.filterValue ? `*${state.filterValue}*` : '*';
};

const hscan = async (resetTableData = false, resetCursor = false) => {
    if (resetCursor) {
        state.scanParam.cursor = 0;
    }

    const scanRes = await redisApi.hscan.request({
        ...getBaseReqParam(),
        match: getScanMatch(),
        ...state.scanParam,
    });
    state.scanParam.cursor = scanRes.cursor;
    state.loadMoreDisable = scanRes.cursor == 0;
    state.total = scanRes.keySize;

    const keys = scanRes.keys;
    const hashValue = [];
    const fieldCount = keys.length / 2;
    let nextFieldIndex = 0;
    for (let i = 0; i < fieldCount; i++) {
        hashValue.push({ field: keys[nextFieldIndex++], value: keys[nextFieldIndex++] });
    }

    if (resetTableData) {
        state.hashValues = hashValue;
    } else {
        state.hashValues.push(...hashValue);
    }
};

const hdel = async (field: any, index: any) => {
    await redisApi.hdel.request({
        ...getBaseReqParam(),
        field,
    });

    ElMessage.success('删除成功');
    state.hashValues.splice(index, 1);
    state.total--;
};

const showEditDialog = (row: any) => {
    state.editDialog.dataRow = row;
    state.editDialog.field = row ? row.field : '';
    state.editDialog.value = row ? row.value : '';
    state.editDialog.visible = true;
};

const confirmEditData = async () => {
    const param = getBaseReqParam();

    const field = state.editDialog.field;
    notBlank(field, 'field不能为空');

    // 存在数据行，则说明为修改，则要先删除旧数据后新增
    const dataRow = state.editDialog.dataRow;
    if (dataRow) {
        await redisApi.hdel.request({
            ...param,
            field: dataRow.field,
        });
    }

    // 获取hash value内容并新增
    const value = formatViewerRef.value.getContent();
    const res = await redisApi.hset.request({
        ...param,
        value: [
            {
                field,
                value: value,
            },
        ],
    });

    ElMessage.success('保存成功');
    if (dataRow) {
        state.editDialog.dataRow.value = value;
        state.editDialog.dataRow.field = field;
    } else {
        // 响应0则为被覆盖，则重新scan
        if (res == 0) {
            hscan(true, true);
        } else {
            state.hashValues.unshift({ value, field });
            state.total++;
        }
    }
    state.editDialog.visible = false;
    state.editDialog.dataRow = null;
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
<style lang="scss">
#string-value-text {
    flex-grow: 1;
    display: flex;
    position: relative;

    .text-type-select {
        position: absolute;
        z-index: 2;
        right: 10px;
        top: 10px;
        max-width: 70px;
    }
}
</style>
