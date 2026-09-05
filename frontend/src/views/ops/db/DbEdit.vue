<template>
    <div>
        <el-dialog
            :title="title"
            v-model="dialogVisible"
            @open="open"
            :before-close="onCancel"
            :close-on-click-modal="false"
            :destroy-on-close="true"
            width="38%"
        >
            <auto-form ref="dbForm" v-model="form" :items="items" label-width="auto">
                <!-- 认证名选择（选项含用户名/加密类型/备注富渲染） -->
                <template #authCertName>
                    <el-select v-model="form.authCertName" filterable>
                        <el-option v-for="item in state.authCerts" :key="item.id" :label="`${item.name}`" :value="item.name">
                            {{ item.name }}

                            <el-divider direction="vertical" border-style="dashed" />
                            {{ item.username }}

                            <el-divider direction="vertical" border-style="dashed" />
                            <EnumTag :value="item.ciphertextType" :enums="AuthCertCiphertextTypeEnum" />

                            <el-divider direction="vertical" border-style="dashed" />
                            {{ item.remark }}
                        </el-option>
                    </el-select>
                </template>

                <!-- 指定 DB 多选（全选/自定义过滤/允许创建） -->
                <template #database>
                    <el-select
                        :disabled="form.getDatabaseMode == DbGetDbNamesMode.Auto.value || !form.authCertName"
                        v-model="dbNamesSelected"
                        multiple
                        clearable
                        collapse-tags
                        collapse-tags-tooltip
                        filterable
                        :filter-method="filterDbNames"
                        allow-create
                        :placeholder="$t('db.selectDbPlacehoder')"
                        @focus="getAllDatabase(form.authCertName ?? '')"
                        :loading="state.loadingDbNames"
                    >
                        <template #header>
                            <el-checkbox v-model="checkAllDbNames" :indeterminate="indeterminateDbNames" @change="onCheckAll">
                                {{ $t('db.allSelect') }}
                            </el-checkbox>
                        </template>
                        <el-option v-for="db in state.dbNamesFiltered" :key="db" :label="db" :value="db" />
                    </el-select>
                </template>
            </auto-form>

            <template #footer>
                <el-button @click="onCancel()">{{ $t('common.cancel') }}</el-button>
                <el-button type="primary" @click="onConfirm">{{ $t('common.confirm') }}</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, watch, ref, useTemplateRef, type PropType } from 'vue';
import { dbApi } from './api';
import type { CheckboxValueType } from 'element-plus';
import { DbType } from '@/views/ops/db/dialect';

import EnumTag from '@/components/enum-tag/EnumTag.vue';
import { AuthCertCiphertextTypeEnum } from '../tag/enums';
import { resourceAuthCertApi } from '../tag/api';
import { TagResourceTypeEnum } from '@/common/commonEnum';
import { DbGetDbNamesMode } from './enums';
import { useI18nFormValidate } from '@/hooks/useI18n';
import { AutoForm, type AutoFormItem } from '@/components/auto-form';
import type { Db, DbInstance } from './types';
import type { ResourceAuthCert } from '@/types/common';

/** Db 编辑表单 (id/name/instanceId 允许 null 表示未选择) */
type DbForm = Partial<Omit<Db, 'id' | 'name' | 'instanceId'>> & {
    id?: number | null;
    name?: string | null;
    instanceId?: number | null;
};

const props = defineProps({
    instance: {
        type: [Boolean, Object, null],
    },
    db: {
        type: Object as PropType<Partial<Db> | null>,
        default: null,
    },
    title: {
        type: String,
    },
});

const dialogVisible = defineModel<boolean>('visible', { default: false });

//定义事件
const emit = defineEmits(['cancel', 'val-change', 'confirm']);

const checkAllDbNames = ref(false);
const indeterminateDbNames = ref(false);

const dbForm = useTemplateRef<{ validate: (...args: unknown[]) => unknown }>('dbForm');

const state = reactive({
    allDatabases: [] as string[],
    dbNamesSelected: [] as string[],
    dbNamesFiltered: [] as string[],
    filterString: '',
    selectInstalce: {} as Record<string, unknown>,
    authCerts: [] as ResourceAuthCert[],
    form: {
        id: null,
        name: null,
        code: '',
        getDatabaseMode: DbGetDbNamesMode.Auto.value,
        database: '',
        remark: '',
        instanceId: null as number | null,
        authCertName: '',
    } as DbForm,
    instances: [] as DbInstance[],
    loadingDbNames: false,
});

const { allDatabases, form, dbNamesSelected } = toRefs(state);

watch(dialogVisible, () => {
    if (!dialogVisible.value) {
        return;
    }
    const db = props.db;
    if (db?.code) {
        state.form = { ...db };
        if (db.getDatabaseMode == DbGetDbNamesMode.Assign.value) {
            // 将数据库名使用空格切割，获取所有数据库列表
            state.dbNamesSelected = db.database?.split(' ') ?? [];
        }
    } else {
        state.form = { getDatabaseMode: DbGetDbNamesMode.Auto.value, id: null, name: null, code: '', database: '', remark: '', instanceId: null, authCertName: '' };
        state.dbNamesSelected = [];
    }
});

const onChangeGetDatabaseMode = (val: number) => {
    if (val == DbGetDbNamesMode.Auto.value) {
        state.dbNamesSelected = [];
    }
};

/** 表单声明（AutoFormItem[]，渲染 + 校验唯一数据源；authCertName/database 选项渲染复杂走插槽） */
const items: AutoFormItem[] = [
    { prop: 'name', label: 'common.name', required: true },
    { prop: 'authCertName', label: 'db.acName', required: true },
    { prop: 'getDatabaseMode', label: 'db.getDbMode', type: 'enum', enums: DbGetDbNamesMode, required: true, onChange: (val: unknown) => onChangeGetDatabaseMode(val as number) },
    { prop: 'database', label: 'DB' },
    { prop: 'remark', label: 'common.remark', type: 'textarea' },
];

const getAuthCerts = async () => {
    const inst = props.instance as Partial<DbInstance> | false | null;
    const res = await resourceAuthCertApi.listByQuery.request({
        resourceCode: (inst && inst.code) || '',
        resourceType: TagResourceTypeEnum.DbInstance.value,
        pageSize: 100,
    });
    state.authCerts = res.list || [];
};

const getAllDatabase = async (authCertName: string) => {
    try {
        state.loadingDbNames = true;
        const req: Record<string, unknown> = { ...(props.instance as Record<string, unknown>) };
        req.authCert = state.authCerts?.find((x: ResourceAuthCert) => x.name == authCertName);
        let dbs = await dbApi.getAllDatabase.request(req);
        state.allDatabases = dbs;

        // 如果是oracle，且没查出数据库列表，则取实例sid
        let instance = state.instances.find((item) => item.id === state.form.instanceId) as (DbInstance & { sid?: string }) | undefined;
        if (instance && instance.type === DbType.oracle && dbs.length === 0) {
            state.allDatabases = [instance.sid ?? ''];
        }
    } finally {
        state.loadingDbNames = false;
    }
};

const open = async () => {
    await getAuthCerts();
    if (state.form.authCertName) {
        await getAllDatabase(state.form.authCertName);
    }
};

const onConfirm = async () => {
    await useI18nFormValidate(dbForm);
    emit('confirm', state.form);
};

const resetInputDb = () => {
    state.dbNamesSelected = [];
    state.allDatabases = [];
    state.instances = [];
};

const onCancel = () => {
    dialogVisible.value = false;
    emit('cancel');
    setTimeout(() => {
        resetInputDb();
    }, 500);
};

const checkDbSelect = (val: string[]) => {
    const selected = val.filter((dbName: string) => {
        return dbName.includes(state.filterString);
    });
    if (selected.length === 0) {
        checkAllDbNames.value = false;
        indeterminateDbNames.value = false;
        return;
    }
    if (selected.length === state.dbNamesFiltered.length) {
        checkAllDbNames.value = true;
        indeterminateDbNames.value = false;
        return;
    }
    indeterminateDbNames.value = true;
};

watch(dbNamesSelected, (val: string[]) => {
    checkDbSelect(val);
    state.form.database = val.join(' ');
});

watch(allDatabases, (val: string[]) => {
    state.dbNamesFiltered = val.map((dbName: string) => dbName);
});

const onCheckAll = (val: CheckboxValueType) => {
    const otherSelected = state.dbNamesSelected.filter((dbName: string) => {
        return !state.dbNamesFiltered.includes(dbName);
    });
    if (val) {
        state.dbNamesSelected = otherSelected.concat(state.dbNamesFiltered);
    } else {
        state.dbNamesSelected = otherSelected;
    }
};

const filterDbNames = (filterString: string) => {
    const dbNamesCreated = state.dbNamesSelected.filter((dbName: string) => {
        return !state.allDatabases.includes(dbName);
    });
    if (filterString.length === 0) {
        state.dbNamesFiltered = dbNamesCreated.concat(state.allDatabases);
        checkDbSelect(state.dbNamesSelected);
        return;
    }
    state.dbNamesFiltered = dbNamesCreated.concat(state.allDatabases).filter((dbName: string) => {
        if (dbName == filterString) {
            return false;
        }
        return dbName.includes(filterString);
    });
    state.dbNamesFiltered.unshift(filterString);
    state.filterString = filterString;
    checkDbSelect(state.dbNamesSelected);
};
</script>
<style lang="scss"></style>
