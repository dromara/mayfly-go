<template>
    <el-dialog v-model="visible" width="680px" :title="$t('mongo.batchTitle', { ns })" :close-on-click-modal="false">
        <div class="flex flex-col gap-3">
            <el-radio-group v-model="batchState.mode">
                <el-radio-button value="update">{{ $t('mongo.batchUpdate') }}</el-radio-button>
                <el-radio-button value="delete">{{ $t('mongo.batchDelete') }}</el-radio-button>
            </el-radio-group>

            <div>
                <div class="field-label">{{ $t('mongo.filter') }} ({{ $t('mongo.filterHint') }})</div>
                <el-input v-model="filterDraft" type="textarea" :rows="4" class="mono" placeholder="{}" @input="onFilterChange" />
            </div>

            <!-- 影响面必须先被数字看见：一个写错的 filter 会静默改掉整个集合且无法回退 -->
            <div class="flex flex-wrap items-center gap-3">
                <el-button :loading="batchState.previewing" :disabled="!target" @click="onPreview">{{ $t('mongo.batchPreview') }}</el-button>
                <span v-if="batchState.previewed !== null" class="text-[13px]">
                    {{ $t('mongo.batchHit', { count: batchState.previewed }) }}
                </span>
                <span v-else class="text-[13px] text-gray-500">{{ $t('mongo.batchNotPreviewed') }}</span>
            </div>

            <div v-if="batchState.mode === 'update'">
                <div class="field-label">{{ $t('mongo.batchUpdateSpec') }} ({{ $t('mongo.batchUpdateSpecHint') }})</div>
                <el-input v-model="batchState.updateText" type="textarea" :rows="5" class="mono" />
                <el-checkbox v-model="batchState.upsert">{{ $t('mongo.batchUpsert') }}</el-checkbox>
            </div>

            <el-alert v-if="batchState.error" :title="batchState.error" type="error" show-icon :closable="false" />
            <el-alert v-else-if="batchState.result" :title="resultText" type="success" show-icon :closable="false" />
        </div>

        <template #footer>
            <el-button @click="visible = false">{{ $t('common.cancel') }}</el-button>
            <el-button
                v-auth="batchState.mode === 'update' ? perms.dataSave : perms.dataDel"
                :type="batchState.mode === 'update' ? 'primary' : 'danger'"
                :loading="batchState.submitting"
                :disabled="!target || batchState.previewed === null || batchState.previewed === 0"
                @click="onSubmit"
            >
                {{ batchState.mode === 'update' ? $t('mongo.batchDoUpdate') : $t('mongo.batchDoDelete') }}
            </el-button>
        </template>
    </el-dialog>
</template>

<script lang="ts" setup>
import { computed, ref, watch } from 'vue';

import { useI18n } from 'vue-i18n';
import { Msg, useI18nConfirm } from '@/hooks/useI18n';
import { perms } from '../perms';
import { useBatchOps, type BatchMode } from '../resource/composables/useBatchOps';
import type { CollectionParam } from '../types';

const props = defineProps<{
    target: CollectionParam | null;
    /** 打开时带入的查询条件：用户通常就是想改刚查出来的那一批 */
    filterText: string;
    /** 入口意图，进面板后仍可切换 */
    mode: BatchMode;
}>();

const visible = defineModel<boolean>('visible', { default: false });

const emit = defineEmits<{
    /** 已产生写入：父级重查以显示新内容 */
    done: [];
}>();

const { t } = useI18n();

const { state: batchState, open, onFilterChange, preview, validateSubmit, submit } = useBatchOps();

/** 本地草稿：条件文本一改就通过 onFilterChange 失效旧预览值 */
const filterDraft = ref('');

const ns = computed(() => (props.target ? `${props.target.database}.${props.target.collection}` : ''));

watch(visible, (opened) => {
    if (!opened) {
        return;
    }
    open(props.mode, props.filterText);
    filterDraft.value = batchState.filterText;
});

/**
 * 结果读数。
 *
 * 命中数与实际改动数不是一回事：值本来就相同的文档不会被计进 modifiedCount，
 * 说成「改了 N 条」会让人以为条件生效范围出了错。
 */
const resultText = computed(() => {
    const res = batchState.result;
    if (!res) {
        return '';
    }
    if (batchState.mode === 'delete') {
        return t('mongo.batchDeleted', { count: res.deletedCount });
    }
    return t('mongo.batchUpdated', { matched: res.matchedCount, modified: res.modifiedCount, upserted: res.upsertedCount });
});

async function onPreview() {
    if (!props.target) {
        return;
    }
    await preview(props.target);
}

async function onSubmit() {
    if (!props.target) {
        return;
    }
    // 先校验再弹确认：条件/更新内容不合法时不该让用户白做一次决定
    if (!validateSubmit()) {
        return;
    }

    const hit = batchState.previewed ?? 0;
    if (!(await useI18nConfirm(batchState.mode === 'update' ? 'mongo.batchUpdateConfirm' : 'mongo.batchDeleteConfirm', { count: hit }))) {
        // 取消或关掉弹窗：不继续后续操作
        return;
    }
    if (await submit(props.target)) {
        Msg.operateSuccess();
        emit('done');
    }
}
</script>

<style lang="scss" scoped>
.field-label {
    margin-bottom: 4px;
    font-size: 13px;
    color: var(--el-text-color-secondary);
}

.mono :deep(.el-textarea__inner) {
    font-family: var(--el-font-family-mono, monospace);
    font-size: 12px;
    line-height: 1.7;
}
</style>
