<template>
    <el-dialog v-model="visible" :title="$t('db.sensitive.title')" width="820px" append-to-body destroy-on-close>
        <div class="flex flex-col gap-2">
            <div class="text-xs text-gray-400">{{ $t('db.sensitive.tip') }}</div>

            <div class="flex items-center gap-2">
                <el-switch v-model="unprotectedOnly" :active-text="$t('db.sensitive.unprotectedOnly')" />
                <span class="text-xs text-gray-400">
                    {{ $t('db.sensitive.scanned', { scanned: result.scannedTables, total: result.totalTables }) }}
                </span>
                <span v-if="result.truncated" class="text-xs text-orange-500">{{ $t('db.sensitive.truncated') }}</span>
            </div>

            <el-table v-loading="loading" :data="visibleFindings" size="small" border max-height="360" :empty-text="$t('db.sensitive.empty')">
                <el-table-column :label="$t('db.columnAcl.target')" min-width="180" show-overflow-tooltip>
                    <template #default="scope">{{ scope.row.tableName }}.{{ scope.row.columnName }}</template>
                </el-table-column>
                <el-table-column prop="columnType" :label="$t('db.sensitive.type')" min-width="120" show-overflow-tooltip />
                <el-table-column :label="$t('db.sensitive.category')" min-width="110">
                    <template #default="scope">{{ $t(`db.sensitive.categoryLabel.${scope.row.category}`) }}</template>
                </el-table-column>
                <el-table-column :label="$t('db.sensitive.level')" width="90">
                    <template #default="scope">
                        <el-tag :type="scope.row.level === 'high' ? 'danger' : 'warning'" size="small">
                            {{ $t(`db.sensitive.levelLabel.${scope.row.level}`) }}
                        </el-tag>
                    </template>
                </el-table-column>
                <el-table-column :label="$t('db.sensitive.status')" width="110">
                    <template #default="scope">
                        <el-tag v-if="scope.row.protected" type="success" size="small">{{ $t('db.sensitive.protected') }}</el-tag>
                        <el-tag v-else type="info" size="small">{{ $t('db.sensitive.unprotected') }}</el-tag>
                    </template>
                </el-table-column>
                <el-table-column :label="$t('common.operation')" width="130" fixed="right">
                    <template #default="scope">
                        <el-button v-if="!scope.row.protected" link type="primary" :loading="savingKey === keyOf(scope.row)" @click="protect(scope.row)">
                            {{ $t('db.sensitive.protect') }}
                        </el-button>
                        <span v-else class="text-xs text-gray-400">{{ $t('db.sensitive.alreadyProtected') }}</span>
                    </template>
                </el-table-column>
            </el-table>
        </div>
        <template #footer>
            <el-button @click="visible = false">{{ $t('common.close') }}</el-button>
            <el-button type="primary" :loading="loading" @click="load">{{ $t('common.refresh') }}</el-button>
        </template>
    </el-dialog>
</template>

<script lang="ts" setup>
import { computed, ref, watch } from 'vue';
import { Msg, useI18nConfirm } from '@/hooks/useI18n';
import { i18n } from '@/i18n';
import { dbApi, dbColumnAclApi } from '../api';
import type { SensitiveColumnFinding, SensitiveScanResult } from '../types';

const props = defineProps<{ dbId: number; db: string }>();
const visible = defineModel<boolean>('visible', { default: false });

const loading = ref(false);
const savingKey = ref('');
const unprotectedOnly = ref(true);
const result = ref<SensitiveScanResult>({ findings: [], scannedTables: 0, totalTables: 0, truncated: false });

const visibleFindings = computed(() =>
    unprotectedOnly.value ? result.value.findings.filter((item) => !item.protected) : result.value.findings,
);

const keyOf = (row: SensitiveColumnFinding) => `${row.tableName}.${row.columnName}`;

const load = async () => {
    loading.value = true;
    try {
        result.value = await dbApi.scanSensitiveColumns.request({ id: props.dbId, db: props.db });
    } finally {
        loading.value = false;
    }
};

/**
 * 一键纳入列保护：角色留空 = 对所有账号生效。
 *
 * 这里明确二次确认：扫描只是「看起来敏感」，真正挡住所有人是管理动作，
 * 误点一次就会让某个业务查询整体报错。
 */
const protect = async (row: SensitiveColumnFinding) => {
    // 用仓库统一的确认原语（返回 boolean）：裸 ElMessageBox 的「取消」是 reject，
    // 调用点一旦忘判就会在用户点取消后照样执行
    const confirmed = await useI18nConfirm('db.sensitive.protectConfirm', { target: keyOf(row) });
    if (!confirmed) {
        return;
    }
    savingKey.value = keyOf(row);
    try {
        await dbColumnAclApi.save.request({
            dbId: props.dbId,
            tableName: row.tableName,
            columnName: row.columnName,
            roleIds: [],
            remark: i18n.global.t('db.sensitive.autoRemark'),
        });
        Msg.saveSuccess();
        // 就地翻转状态，避免整库重扫（重扫要逐表取列元数据，代价远高于改一行）
        row.protected = true;
    } finally {
        savingKey.value = '';
    }
};

watch(visible, async (opened) => {
    if (opened) {
        await load();
    }
});
</script>
