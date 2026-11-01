<template>
    <div>
        <el-descriptions :column="3" border label-width="110">
            <el-descriptions-item :span="1" :label="$t('flow.exportDb')">{{ form.dbName || '-' }}</el-descriptions-item>
            <el-descriptions-item :span="1" :label="$t('flow.exportContent')">{{ dumpTypeLabel }}</el-descriptions-item>
            <el-descriptions-item :span="1" :label="$t('flow.exportTableCount')">{{ form.tables?.length ?? 0 }}</el-descriptions-item>
            <el-descriptions-item :span="3" :label="$t('flow.exportTables')">
                <div class="max-h-24 overflow-auto text-xs">{{ (form.tables ?? []).join(', ') || '-' }}</div>
            </el-descriptions-item>
            <el-descriptions-item :span="3" :label="$t('flow.exportReason')">{{ form.reason || '-' }}</el-descriptions-item>
        </el-descriptions>

        <div class="mt-2 flex items-center gap-2">
            <!-- 只有审批通过的工单才给下载入口；其余状态说明原因，而不是给一个点了会报错的按钮 -->
            <el-button v-if="approved" type="primary" v-auth="'db:data:export'" @click="download">
                {{ $t('flow.exportDownload') }}
            </el-button>
            <span v-else class="text-xs text-gray-400">{{ $t('flow.exportNotApproved') }}</span>
            <span v-if="approved" class="text-xs text-gray-400">{{ $t('flow.exportDownloadHint') }}</span>
        </div>
    </div>
</template>

<script lang="ts" setup>
import { computed, reactive } from 'vue';
import { useI18n } from 'vue-i18n';
import config from '@/common/config';
import { joinClientParams } from '@/common/request';

const props = defineProps({
    procinst: {
        type: [Object],
        default: () => {},
    },
});

const { t, te } = useI18n();

const form = reactive<{ dbId: number; dbName: string; tables: string[]; dumpType: string; reason: string }>({
    dbId: 0,
    dbName: '',
    tables: [],
    dumpType: '3',
    reason: '',
});

// bizForm 是前端自持的 JSON blob：解析失败只说明这张工单不是本组件认识的形态，
// 按空表单渲染即可，不能在详情页抛错
try {
    const parsed = JSON.parse(props.procinst?.bizForm || '{}');
    form.dbId = Number(parsed.dbId ?? 0);
    form.dbName = String(parsed.dbName ?? '');
    form.tables = Array.isArray(parsed.tables) ? parsed.tables : [];
    form.dumpType = String(parsed.dumpType ?? '3');
    form.reason = String(parsed.reason ?? '');
} catch {
    // 保持空表单
}

const approved = computed(() => Number(props.procinst?.status) === 2 && form.dbId > 0);

const dumpTypeLabel = computed(() => {
    const key = `flow.exportDumpType${form.dumpType}`;
    return te(key) ? t(key) : '-';
});

/**
 * 下载走同源 iframe/新窗口：接口是流式响应（NoRes），axios 拿不到 blob，
 * 且 token 由查询参数携带——与即席导出同一套下载方式
 */
function download() {
    const url = `${config.baseApiUrl}/dbs/${form.dbId}/export-tickets/${props.procinst.id}/download?db=${encodeURIComponent(
        form.dbName,
    )}&${joinClientParams()}`;
    window.open(url, '_blank');
}
</script>
