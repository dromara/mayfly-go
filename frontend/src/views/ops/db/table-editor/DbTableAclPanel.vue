<template>
    <div class="flex flex-col gap-2">
        <div class="flex items-start justify-between gap-2">
            <div class="text-xs text-gray-400">{{ $t('db.tableAcl.tip') }}</div>
            <!-- 新增走弹窗：表单常驻会让空态也顶着四个输入框，看起来像没配完的表单 -->
            <el-button type="primary" plain size="small" icon="Plus" @click="openCreate">{{ $t('common.add') }}</el-button>
        </div>

        <el-table v-loading="loading" :data="rules" size="small" border max-height="300" :empty-text="$t('db.tableAcl.empty')">
            <el-table-column prop="tableName" :label="$t('db.table')" min-width="160" show-overflow-tooltip />
            <el-table-column :label="$t('db.tableAcl.effect')" width="110">
                <template #default="scope">
                    <el-tag :type="scope.row.effect === 2 ? 'success' : 'danger'" size="small">
                        {{ scope.row.effect === 2 ? $t('db.tableAcl.effectAllow') : $t('db.tableAcl.effectDeny') }}
                    </el-tag>
                </template>
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
            :title="form.id ? $t('db.tableAcl.editTitle') : $t('db.tableAcl.createTitle')"
            width="520px"
            append-to-body
        >
            <el-form :model="form" label-width="96px">
                <el-form-item :label="$t('db.table')" required>
                    <el-input v-model="form.tableName" :placeholder="$t('db.tableAcl.tablePlaceholder')" clearable />
                </el-form-item>
                <el-form-item :label="$t('db.tableAcl.effect')">
                    <el-radio-group v-model="form.effect">
                        <el-radio :value="1">{{ $t('db.tableAcl.effectDeny') }}</el-radio>
                        <el-radio :value="2">{{ $t('db.tableAcl.effectAllow') }}</el-radio>
                    </el-radio-group>
                    <!-- 白名单是全局模式：配了任意一条 allow，该库其余表对所有账号不可见，必须当场讲清 -->
                    <div v-if="form.effect === 2" class="text-xs text-orange-500 mt-1">{{ $t('db.tableAcl.whitelistTip') }}</div>
                </el-form-item>
                <el-form-item :label="$t('db.tableAcl.roles')">
                    <el-select v-model="form.roleIds" multiple filterable clearable class="w-full" :placeholder="$t('db.tableAcl.allAccounts')">
                        <el-option v-for="role in roles" :key="role.id" :label="`${role.name} [${role.code}]`" :value="role.id" />
                    </el-select>
                    <div class="text-xs text-gray-400 mt-1">{{ $t('db.tableAcl.rolesTip') }}</div>
                </el-form-item>
                <el-form-item :label="$t('db.comment')">
                    <el-input v-model="form.remark" :placeholder="$t('db.tableAcl.remarkPlaceholder')" clearable />
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
import { dbTableAclApi } from '../api';
import type { DbTableAcl } from '../types';
import { useAclRoles } from './useAclRoles';
import { DbInst } from '../db';

const props = defineProps<{ dbId: number; dbName: string }>();

const rules = ref<DbTableAcl[]>([]);
const loading = ref(false);
const saving = ref(false);
const formVisible = ref(false);
const { roles, roleNames } = useAclRoles();

const form = reactive<{ id?: number; tableName: string; effect: number; roleIds: number[]; remark: string }>({
    tableName: '',
    // 默认「禁止」：白名单是全局模式切换，不能是新建规则时的默认值
    effect: 1,
    roleIds: [],
    remark: '',
});

const reset = () => {
    form.id = undefined;
    form.tableName = '';
    form.effect = 1;
    form.roleIds = [];
    form.remark = '';
};

/** 新增：清空表单后开弹窗（与编辑共用同一个弹窗，避免两套录入界面） */
const openCreate = () => {
    reset();
    formVisible.value = true;
};

const load = async () => {
    loading.value = true;
    try {
        rules.value = await dbTableAclApi.list.request({ dbId: props.dbId });
    } finally {
        loading.value = false;
    }
};

const edit = (row: DbTableAcl) => {
    form.id = row.id;
    form.tableName = row.tableName;
    form.effect = row.effect === 2 ? 2 : 1;
    form.roleIds = [...(row.roleIds ?? [])];
    form.remark = row.remark ?? '';
    formVisible.value = true;
};

const remove = async (row: DbTableAcl) => {
    // 删错等于把这张表的保护摘掉，必须确认并说清删的是哪张表
    if (!(await useI18nConfirm('db.tableAcl.deleteConfirm', { table: row.tableName }))) {
        return;
    }
    await dbTableAclApi.remove.request({ id: row.id });
    Msg.deleteSuccess();
    await load();
    DbInst.invalidateSchema(props.dbId, props.dbName);
};

const save = async () => {
    if (!form.tableName.trim()) {
        Msg.warning('db.tableAcl.tableRequired');
        return;
    }
    saving.value = true;
    try {
        const payload = {
            id: form.id,
            dbId: props.dbId,
            tableName: form.tableName.trim(),
            effect: form.effect,
            roleIds: form.roleIds,
            remark: form.remark,
        };
        if (form.id) {
            await dbTableAclApi.update.request(payload);
        } else {
            await dbTableAclApi.save.request(payload);
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
