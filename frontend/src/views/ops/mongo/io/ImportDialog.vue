<template>
    <el-dialog v-model="visible" width="640px" :title="$t('mongo.importTitle', { ns })" destroy-on-close :close-on-click-modal="false" @closed="onClosed">
        <div class="flex flex-col gap-3">
            <div class="flex flex-wrap items-center gap-3">
                <el-upload :show-file-list="false" :auto-upload="false" :on-change="onFileChange" accept=".json,.ndjson,.jsonl">
                    <el-button type="primary" plain size="small" icon="Upload">{{ $t('mongo.importSelectFile') }}</el-button>
                </el-upload>
                <el-text size="small" type="info">{{ $t('mongo.importFormatsHint') }}</el-text>
            </div>

            <!-- 导入是往集合里加数据，条数与「哪些行没进去」必须先看见再点确认 -->
            <div v-if="ioState.fileName" class="flex flex-wrap items-center gap-3 text-[13px]">
                <span class="font-mono">{{ ioState.fileName }}</span>
                <el-tag size="small" type="success">{{ $t('mongo.importParsed', { count: ioState.docs.length }) }}</el-tag>
                <el-tag v-if="ioState.issues.length" size="small" type="warning">
                    {{ $t('mongo.importSkipped', { count: ioState.issues.length }) }}
                </el-tag>
            </div>

            <div v-if="ioState.issues.length" class="issue-box">
                <div v-for="issue in visibleIssues" :key="`${issue.at}-${issue.reason}`" class="issue-line">
                    <span class="text-gray-500">{{ $t('mongo.importLine', { line: issue.at }) }}</span>
                    <span class="ml-2">{{ $t(issueReasonKey(issue)) }}</span>
                </div>
                <div v-if="ioState.issues.length > MAX_ISSUE_ROWS" class="issue-line text-gray-400">
                    {{ $t('mongo.importMoreIssues', { count: ioState.issues.length - MAX_ISSUE_ROWS }) }}
                </div>
            </div>

            <el-progress v-if="ioState.importing" :percentage="progress" :stroke-width="10" />

            <el-alert
                v-if="ioState.error"
                :title="$t('mongo.importPartial', { imported: ioState.progress.done, total: ioState.docs.length })"
                type="warning"
                show-icon
                :closable="false"
            >
                <div class="mt-1 text-[12px]">{{ ioState.error }}</div>
            </el-alert>
        </div>

        <template #footer>
            <el-button @click="visible = false">{{ $t('common.cancel') }}</el-button>
            <el-button v-auth="perms.dataSave" type="primary" :loading="ioState.importing" :disabled="!ioState.docs.length || !target" @click="onImport">
                {{ $t('mongo.importDo', { count: ioState.docs.length }) }}
            </el-button>
        </template>
    </el-dialog>
</template>

<script lang="ts" setup>
import { computed } from 'vue';

import { Msg } from '@/hooks/useI18n';
import { perms } from '../perms';
import { useDocIO } from '../resource/composables/useDocIO';
import type { ImportIssue } from '../io/parse';
import type { CollectionParam } from '../types';

const props = defineProps<{ target: CollectionParam | null }>();

const visible = defineModel<boolean>('visible', { default: false });

const emit = defineEmits<{
    /** 有数据写入：父级重查当前条件以看见新文档 */
    success: [];
}>();

const { state: ioState, readImportFile, resetImport, importDocs } = useDocIO();

/** 一次最多列出多少条问题：坏行上千时列表本身会把对话框撑爆 */
const MAX_ISSUE_ROWS = 8;

const ns = computed(() => (props.target ? `${props.target.database}.${props.target.collection}` : ''));

const visibleIssues = computed(() => ioState.issues.slice(0, MAX_ISSUE_ROWS));

const progress = computed(() => {
    const total = ioState.progress.total;
    if (!total) {
        return 0;
    }
    return Math.round((ioState.progress.done / total) * 100);
});

function issueReasonKey(issue: ImportIssue): string {
    return issue.reason === 'invalidJson' ? 'mongo.importIssueInvalidJson' : 'mongo.importIssueNotObject';
}

async function onFileChange(uploadFile: { raw?: File }) {
    const raw = uploadFile.raw;
    if (!raw) {
        return;
    }
    const parsed = await readImportFile(raw);
    if (!parsed.docs.length) {
        Msg.warning('mongo.importNothing');
    }
}

async function onImport() {
    if (!props.target) {
        return;
    }
    const total = ioState.docs.length;
    const inserted = await importDocs(props.target);
    if (inserted >= total) {
        Msg.success('mongo.importSuccess', { count: inserted });
    } else {
        // 部分成功：说清已写入多少，否则用户会重复导入整份文件造成双份数据
        Msg.warning('mongo.importPartial', { imported: inserted, total });
    }
    if (inserted > 0) {
        emit('success');
    }
    visible.value = false;
}

function onClosed() {
    resetImport();
}
</script>

<style lang="scss" scoped>
.issue-box {
    max-height: 160px;
    padding: 8px 10px;
    overflow: auto;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 4px;
    background: var(--el-fill-color-light);
    font-family: var(--el-font-family-mono, monospace);
    font-size: 12px;
    line-height: 1.8;
}

.issue-line {
    color: var(--el-text-color-regular);
}
</style>
