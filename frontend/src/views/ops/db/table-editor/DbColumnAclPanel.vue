<template>
    <div class="flex flex-col gap-2">
        <div class="flex items-start justify-between gap-2">
            <div class="text-xs text-gray-400">{{ $t('db.columnAcl.tip') }}</div>
            <!-- 与表保护页签保持同一交互：列表 + 新增弹窗，不常驻表单 -->
            <el-button type="primary" plain size="small" icon="Plus" @click="openCreate">{{ $t('common.add') }}</el-button>
        </div>

        <el-table v-loading="loading" :data="rules" size="small" border max-height="300" :empty-text="$t('db.columnAcl.empty')">
            <el-table-column :label="$t('db.columnAcl.target')" min-width="200" show-overflow-tooltip>
                <template #default="scope">{{ scope.row.tableName }}.{{ scope.row.columnName }}</template>
            </el-table-column>
            <el-table-column :label="$t('db.tableAcl.roles')" min-width="200">
                <template #default="scope">
                    <span v-if="!scope.row.roleIds?.length" class="text-orange-500">{{ $t('db.tableAcl.allAccounts') }}</span>
                    <span v-else>{{ roleNames(scope.row.roleIds) }}</span>
                </template>
            </el-table-column>
            <el-table-column prop="remark" :label="$t('db.comment')" min-width="140" show-overflow-tooltip />
            <el-table-column :label="$t('common.operation')" width="120" fixed="right">
                <template #default="scope">
                    <el-button link type="primary" @click="edit(scope.row)">{{ $t('common.edit') }}</el-button>
                    <el-button link type="danger" @click="remove(scope.row)">{{ $t('common.delete') }}</el-button>
                </template>
            </el-table-column>
        </el-table>

        <el-dialog
            v-model="formVisible"
            :title="form.id ? $t('db.columnAcl.editTitle') : $t('db.columnAcl.createTitle')"
            width="520px"
            append-to-body
        >
            <el-form :model="form" label-width="96px">
                <el-form-item :label="$t('db.table')" required>
                    <el-input v-model="form.tableName" :placeholder="$t('db.tableAcl.tablePlaceholder')" clearable />
                </el-form-item>
                <el-form-item :label="$t('db.columnAcl.column')" required>
                    <el-input v-model="form.columnName" :placeholder="$t('db.columnAcl.columnPlaceholder')" clearable />
                </el-form-item>
                <el-form-item :label="$t('db.tableAcl.roles')">
                    <el-select v-model="form.roleIds" multiple filterable clearable class="w-full" :placeholder="$t('db.tableAcl.allAccounts')">
                        <el-option v-for="role in roles" :key="role.id" :label="`${role.name} [${role.code}]`" :value="role.id" />
                    </el-select>
                    <div class="text-xs text-gray-400 mt-1">{{ $t('db.tableAcl.rolesTip') }}</div>
                </el-form-item>
                <el-form-item :label="$t('db.comment')">
                    <el-input v-model="form.remark" :placeholder="$t('db.columnAcl.remarkPlaceholder')" clearable />
                </el-form-item>
            </el-form>
            <template #footer>
                <el-button @click="formVisible = false">{{ $t('common.cancel') }}</el-button>
                <el-button type="primary" :loading="saving" @click="save">{{ $t('common.save') }}</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { onMounted, reactive, ref } from 'vue';
import { Msg, useI18nConfirm } from '@/hooks/useI18n';
import { dbColumnAclApi } from '../api';
import type { DbColumnAcl } from '../types';
import { useAclRoles } from './useAclRoles';
import { DbInst } from '../db';

const props = defineProps<{ dbId: number; dbName: string }>();

const rules = ref<DbColumnAcl[]>([]);
const loading = ref(false);
const saving = ref(false);
const formVisible = ref(false);
const { roles, roleNames } = useAclRoles();

const form = reactive<{ id?: number; tableName: string; columnName: string; roleIds: number[]; remark: string }>({
    tableName: '',
    columnName: '',
    roleIds: [],
    remark: '',
});

const reset = () => {
    form.id = undefined;
    form.tableName = '';
    form.columnName = '';
    form.roleIds = [];
    form.remark = '';
};

/** 新增：清空表单后开弹窗（与编辑共用同一个弹窗） */
const openCreate = () => {
    reset();
    formVisible.value = true;
};

const load = async () => {
    loading.value = true;
    try {
        rules.value = await dbColumnAclApi.list.request({ dbId: props.dbId });
    } finally {
        loading.value = false;
    }
};

const edit = (row: DbColumnAcl) => {
    form.id = row.id;
    form.tableName = row.tableName;
    form.columnName = row.columnName;
    form.roleIds = [...(row.roleIds ?? [])];
    form.remark = row.remark ?? '';
    formVisible.value = true;
};

const remove = async (row: DbColumnAcl) => {
    // 删错等于把这一列的保护摘掉，确认框里带表.列名
    if (!(await useI18nConfirm('db.columnAcl.deleteConfirm', { target: `${row.tableName}.${row.columnName}` }))) {
        return;
    }
    await dbColumnAclApi.remove.request({ id: row.id });
    Msg.deleteSuccess();
    await load();
    DbInst.invalidateSchema(props.dbId, props.dbName);
};

const save = async () => {
    if (!form.tableName.trim() || !form.columnName.trim()) {
        Msg.warning('db.columnAcl.required');
        return;
    }
    saving.value = true;
    try {
        const payload = {
            id: form.id,
            dbId: props.dbId,
            tableName: form.tableName.trim(),
            columnName: form.columnName.trim(),
            roleIds: form.roleIds,
            remark: form.remark,
        };
        if (form.id) {
            await dbColumnAclApi.update.request(payload);
        } else {
            await dbColumnAclApi.save.request(payload);
        }
        Msg.saveSuccess();
        formVisible.value = false;
        reset();
        await load();
        // 规则变了，表清单/补全的本地缓存必须跟着失效，否则新保护的表还会被提示出来
        DbInst.invalidateSchema(props.dbId, props.dbName);
    } finally {
        saving.value = false;
    }
};

onMounted(load);
</script>
