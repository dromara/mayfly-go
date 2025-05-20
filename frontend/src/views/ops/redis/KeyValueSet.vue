<template>
    <div class="flex flex-col">
        <div>
            <el-button @click="showEditDialog(null)" icon="plus" size="small" plain type="primary" class="mb-2">{{ $t('redis.addNewLine') }}</el-button>
        </div>
        <div class="flex-1 overflow-auto">
            <el-table size="small" border :data="setDatas" height="100%" stripe>
                <el-table-column type="index" :label="'ID (Total: ' + total + ')'" sortable width="100"> </el-table-column>
                <el-table-column resizable sortable prop="value" label="value" show-overflow-tooltip min-width="200"> </el-table-column>
                <el-table-column :label="$t('common.operation')">
                    <template #header>
                        <el-input
                            v-model="state.filterValue"
                            @keyup.enter="sscanData(true, true)"
                            :placeholder="$t('redis.filterPlaceholder')"
                            clearable
                            size="small"
                        />
                    </template>
                    <template #default="scope">
                        <el-link @click="showEditDialog(scope.row)" underline="never" type="primary" icon="edit" plain></el-link>
                        <el-popconfirm :title="$t('redis.deleteConfirm')" @confirm="srem(scope.row, scope.$index)">
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
            <el-button size="small" @click="sscanData(false)" :disabled="loadMoreDisable" class="content-more-btn"> {{ $t('redis.loadMore') }} </el-button>
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
    // SSCAN key cursor [MATCH pattern] [COUNT count]
    // 响应[cursor, vals[]]
    const res = await props.redis.runCmd(['SSCAN', state.key, state.scanParam.cursor, 'MATCH', getScanMatch(), 'COUNT', state.scanParam.count]);
    if (resetDatas) {
        state.setDatas = [];
    }
    res[1].forEach((x: any) => {
        state.setDatas.push({
            value: x,
        });
    });
    state.scanParam.cursor = res[0];
    state.loadMoreDisable = state.scanParam.cursor == 0;
};

const getTotal = () => {
    // SCARD key
    props.redis.runCmd(['SCARD', state.key]).then((res) => {
        state.total = res;
    });
};

const showEditDialog = (row: any) => {
    state.editDialog.dataRow = row;
    state.editDialog.content = row ? row.value : '';
    state.editDialog.visible = true;
};

const confirmEditData = async () => {
    // 存在数据行，则说明为修改，则要先删除旧数据后新增
    const dataRow = state.editDialog.dataRow;
    if (dataRow) {
        await props.redis.runCmd(['SREM', state.key, state.editDialog.dataRow.value]);
    }

    // 获取set member内容并新增
    const member = formatViewerRef.value.getContent();
    // SADD key member [member ...]
    await props.redis.runCmd(['SADD', state.key, member]);

    useI18nSaveSuccessMsg();
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
    // SREM key member [member ...]
    await props.redis.runCmd(['SREM', state.key, row.value]);
    useI18nDeleteSuccessMsg();
    state.setDatas.splice(index, 1);
    state.total--;
};

defineExpose({ initData });
</script>
<style lang="scss"></style>
