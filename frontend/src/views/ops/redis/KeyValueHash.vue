<template>
    <div class="flex flex-col">
        <div>
            <el-button @click="showEditDialog(null)" icon="plus" size="small" plain type="primary" class="mb-2">{{ $t('redis.addNewLine') }}</el-button>
        </div>
        <div class="flex-1 overflow-auto">
            <el-table size="small" border :data="hashValues" height="100%" stripe>
                <el-table-column type="index" :label="'ID (Total: ' + total + ')'" sortable width="100"> </el-table-column>
                <el-table-column resizable sortable prop="field" label="field" show-overflow-tooltip min-width="100"> </el-table-column>
                <el-table-column resizable sortable prop="value" label="value" show-overflow-tooltip min-width="200"> </el-table-column>
                <el-table-column :label="$t('common.operation')">
                    <template #header>
                        <el-input
                            v-model="state.filterValue"
                            @keyup.enter="hscan(true, true)"
                            :placeholder="$t('redis.filterPlaceholder')"
                            clearable
                            size="small"
                        />
                    </template>
                    <template #default="scope">
                        <el-link @click="showEditDialog(scope.row)" underline="never" type="primary" icon="edit" plain></el-link>
                        <el-popconfirm :title="$t('redis.deleteConfirm')" @confirm="hdel(scope.row.field, scope.$index)">
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
            <el-button size="small" @click="hscan()" :disabled="loadMoreDisable" class="content-more-btn"> {{ $t('redis.loadMore') }} </el-button>
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
                    <el-input v-model="editDialog.field" placeholder="field" />
                </el-form-item>
                <el-form-item>
                    <format-viewer class="!w-full" ref="formatViewerRef" :content="editDialog.value"></format-viewer>
                </el-form-item>
            </el-form>

            <template #footer>
                <div>
                    <el-button @click="editDialog.visible = false">{{ $t('common.cancel') }}</el-button>
                    <el-button v-auth="'redis:data:save'" type="primary" @click="confirmEditData">{{ $t('common.confirm') }}</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>
<script lang="ts" setup>
import { ref, onMounted, reactive, toRefs } from 'vue';
import { notBlank } from '@/common/assert';
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

    props.redis.runCmd(['HLEN', state.key]).then((res) => (state.total = res));

    // HSCAN key cursor [MATCH pattern] [COUNT count]
    // 返回值 [coursor, keys:[]]
    let scanRes = await props.redis.runCmd(['HSCAN', state.key, state.scanParam.cursor, 'MATCH', getScanMatch(), 'COUNT', state.scanParam.count]);
    state.scanParam.cursor = scanRes[0];
    state.loadMoreDisable = state.scanParam.cursor == 0;
    const keys = scanRes[1];

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
    await props.redis.runCmd(['HDEL', state.key, field]);

    useI18nDeleteSuccessMsg();
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
    const field = state.editDialog.field;
    notBlank(field, 'field not empty');

    // 获取hash value内容并新增
    const value = formatViewerRef.value.getContent();

    const res = await props.redis.runCmd(['HSET', state.key, field, value]);
    useI18nSaveSuccessMsg();
    // 响应0则为被覆盖，则重新scan
    if (res == 0) {
        hscan(true, true);
    } else {
        state.hashValues.unshift({ value, field });
        state.total++;
    }
    state.editDialog.visible = false;
    state.editDialog.dataRow = null;
};

defineExpose({ initData });
</script>
<style lang="scss"></style>
