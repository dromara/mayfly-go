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
            <el-form :model="form" ref="dbForm" :rules="rules" label-width="auto">
                <el-form-item prop="name" :label="$t('common.name')" required>
                    <el-input v-model.trim="form.name" auto-complete="off"></el-input>
                </el-form-item>

                <el-form-item prop="authCertName" :label="$t('db.acName')" required>
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
                </el-form-item>

                <el-form-item prop="getDatabaseMode" :label="$t('db.getDbMode')" required>
                    <EnumSelect :enums="DbGetDbNamesMode" v-model="form.getDatabaseMode" @change="onChangeGetDatabaseMode" />
                </el-form-item>

                <el-form-item prop="database" label="DB">
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
                        @focus="getAllDatabase(form.authCertName)"
                        :loading="state.loadingDbNames"
                    >
                        <template #header>
                            <el-checkbox v-model="checkAllDbNames" :indeterminate="indeterminateDbNames" @change="onCheckAll">
                                {{ $t('db.allSelect') }}
                            </el-checkbox>
                        </template>
                        <el-option v-for="db in state.dbNamesFiltered" :key="db" :label="db" :value="db" />
                    </el-select>
                </el-form-item>

                <el-form-item prop="remark" :label="$t('common.remark')">
                    <el-input v-model.trim="form.remark" auto-complete="off" type="textarea"></el-input>
                </el-form-item>
            </el-form>

            <template #footer>
                <el-button @click="onCancel()">{{ $t('common.cancel') }}</el-button>
                <el-button type="primary" @click="onConfirm">{{ $t('common.confirm') }}</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, watch, ref } from 'vue';
import { dbApi } from './api';
import type { CheckboxValueType } from 'element-plus';
import { DbType } from '@/views/ops/db/dialect';

import EnumTag from '@/components/enumtag/EnumTag.vue';
import { AuthCertCiphertextTypeEnum } from '../tag/enums';
import { resourceAuthCertApi } from '../tag/api';
import { TagResourceTypeEnum } from '@/common/commonEnum';
import { DbGetDbNamesMode } from './enums';
import EnumSelect from '@/components/enumselect/EnumSelect.vue';
import { useI18nFormValidate } from '@/hooks/useI18n';
import { Rules } from '@/common/rule';

const props = defineProps({
    instance: {
        type: [Boolean, Object, null],
    },
    db: {
        type: [Boolean, Object],
    },
    title: {
        type: String,
    },
});

const dialogVisible = defineModel<boolean>('visible', { default: false });

//定义事件
const emit = defineEmits(['cancel', 'val-change', 'confirm']);

const rules = {
    instanceId: [Rules.requiredSelect('db.dbInst')],
    name: [Rules.requiredInput('common.name')],
    authCertName: [Rules.requiredSelect('db.acName')],
    getDatabaseMode: [Rules.requiredSelect('db.getDbMode')],
};

const checkAllDbNames = ref(false);
const indeterminateDbNames = ref(false);

const dbForm: any = ref(null);
// const tagSelectRef: any = ref(null);

const state = reactive({
    allDatabases: [] as any,
    dbNamesSelected: [] as any,
    dbNamesFiltered: [] as any,
    filterString: '',
    selectInstalce: {} as any,
    authCerts: [] as any,
    form: {
        id: null,
        name: null,
        code: '',
        getDatabaseMode: DbGetDbNamesMode.Auto.value,
        database: '',
        remark: '',
        instanceId: null as any,
        authCertName: '',
    },
    instances: [] as any,
    loadingDbNames: false,
});

const { allDatabases, form, dbNamesSelected } = toRefs(state);

watch(dialogVisible, () => {
    if (!dialogVisible.value) {
        return;
    }
    const db: any = props.db;
    if (db.code) {
        state.form = { ...db };
        if (db.getDatabaseMode == DbGetDbNamesMode.Assign.value) {
            // 将数据库名使用空格切割，获取所有数据库列表
            state.dbNamesSelected = db.database.split(' ');
        }
    } else {
        state.form = { getDatabaseMode: DbGetDbNamesMode.Auto.value } as any;
        state.dbNamesSelected = [];
    }
});

const onChangeGetDatabaseMode = (val: any) => {
    if (val == DbGetDbNamesMode.Auto.value) {
        state.dbNamesSelected = [];
    }
};

const getAuthCerts = async () => {
    const inst: any = props.instance;
    const res = await resourceAuthCertApi.listByQuery.request({
        resourceCode: inst.code,
        resourceType: TagResourceTypeEnum.DbInstance.value,
        pageSize: 100,
    });
    state.authCerts = res.list || [];
};

const getAllDatabase = async (authCertName: string) => {
    try {
        state.loadingDbNames = true;
        const req = { ...(props.instance as any) };
        req.authCert = state.authCerts?.find((x: any) => x.name == authCertName);
        let dbs = await dbApi.getAllDatabase.request(req);
        state.allDatabases = dbs;

        // 如果是oracle，且没查出数据库列表，则取实例sid
        let instance = state.instances.find((item: any) => item.id === state.form.instanceId);
        if (instance && instance.type === DbType.oracle && dbs.length === 0) {
            state.allDatabases = [instance.sid];
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
