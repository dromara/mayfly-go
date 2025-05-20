<template>
    <div class="flex flex-col">
        <div>
            <el-button @click="showEditDialog(null, -1)" icon="plus" size="small" plain type="primary" class="mb-2">{{ $t('redis.addNewLine') }}</el-button>
        </div>
        <div class="flex-1 overflow-auto">
            <el-table size="small" border :data="values" height="100%" stripe>
                <el-table-column type="index" :label="'ID (Total: ' + total + ')'" sortable width="100"> </el-table-column>
                <el-table-column resizable sortable prop="value" label="value" show-overflow-tooltip min-width="200"> </el-table-column>
                <el-table-column :label="$t('common.operation')">
                    <template #default="scope">
                        <el-link @click="showEditDialog(scope.row, scope.$index)" underline="never" type="primary" icon="edit" plain></el-link>
                        <el-popconfirm :title="$t('redis.deleteConfirm')" @confirm="lrem(scope.row, scope.$index)">
                            <template #reference>
                                <el-link v-auth="'redis:data:del'" underline="never" type="danger" icon="delete" size="small" plain class="ml-1"></el-link>
                            </template>
                        </el-popconfirm>
                    </template>
                </el-table-column>
            </el-table>
        </div>
        <!-- load more content -->
        <div class="content-more-container">
            <el-button size="small" @click="getListValue(false)" :disabled="loadMoreDisable" class="content-more-btn"> {{ $t('redis.loadMore') }} </el-button>
        </div>

        <el-dialog
            :title="$t('redis.addNewLine')"
            v-model="editDialog.visible"
            width="600px"
            :destroy-on-close="true"
            :close-on-click-modal="false"
            body-class="p-1"
        >
            <el-form>
                <el-form-item>
                    <format-viewer class="!w-full" ref="formatViewerRef" :content="editDialog.content"></format-viewer>
                </el-form-item>
            </el-form>

            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="editDialog.visible = false">{{ $t('common.cancel') }}</el-button>
                    <el-button v-auth="'redis:data:save'" type="primary" @click="confirmEditData">{{ $t('common.confirm') }}</el-button>
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
import { useI18nDeleteSuccessMsg, useI18nSaveSuccessMsg } from '@/hooks/useI18n';

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
    pageNum: 1,
    pageSize: 50,
    total: 0,
    values: [] as any,
    loadMoreDisable: false,
    editDialog: {
        index: -1,
        visible: false,
        content: '',
    },
});

const { total, values, loadMoreDisable, editDialog } = toRefs(state);

onMounted(() => {
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

    props.redis.runCmd(['LLEN', state.key]).then((res) => (state.total = res));

    // LRANGE key start stop
    const res = await props.redis.runCmd(['LRANGE', state.key, (pageNum - 1) * pageSize, pageNum * pageSize - 1]);
    const datas = res.map((x: any) => {
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

const showEditDialog = (row: any, index = -1) => {
    state.editDialog.index = index;
    state.editDialog.content = row ? row.value : '';
    state.editDialog.visible = true;
};

const confirmEditData = async () => {
    const index = state.editDialog.index;
    // 获取list member内容并新增
    const member = formatViewerRef.value.getContent();
    try {
        // 索引=-1 说明是新增
        if (index == -1) {
            // RPUSH key element [element ...]
            await props.redis.runCmd(['RPUSH', state.key, member]);
        } else {
            // LSET key index element
            await props.redis.runCmd(['LSET', state.key, index, member]);
        }

        useI18nSaveSuccessMsg();
        initData();
    } finally {
        state.editDialog.visible = false;
    }
};

const lrem = async (row: any, index: any) => {
    // LREM key count element
    await props.redis.runCmd(['LREM', state.key, 1, row.value]);
    useI18nDeleteSuccessMsg();
    state.values.splice(index, 1);
    state.total--;
};

defineExpose({ initData });
</script>
<style lang="scss"></style>
