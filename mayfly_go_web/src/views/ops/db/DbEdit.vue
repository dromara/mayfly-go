<template>
    <div>
        <el-dialog
            :title="title"
            v-model="dialogVisible"
            @open="open"
            :before-close="cancel"
            :close-on-click-modal="false"
            :destroy-on-close="true"
            width="38%"
        >
            <el-form :model="form" ref="dbForm" :rules="rules" label-width="auto">
                <el-form-item ref="tagSelectRef" prop="tagId" label="标签" required>
                    <tag-tree-select
                        @change-tag="
                            (tagIds) => {
                                form.tagId = tagIds;
                                tagSelectRef.validate();
                            }
                        "
                        multiple
                        :select-tags="form.tagId"
                        style="width: 100%"
                    />
                </el-form-item>

                <el-form-item prop="instanceId" label="数据库实例" required>
                    <el-select
                        :disabled="form.id !== undefined"
                        remote
                        :remote-method="getInstances"
                        @change="changeInstance"
                        v-model="state.selectInstalce"
                        value-key="id"
                        placeholder="请输入实例名称搜索并选择实例"
                        filterable
                        clearable
                        class="w100"
                    >
                        <el-option v-for="item in state.instances" :key="item.id" :label="`${item.name}`" :value="item">
                            {{ item.name }}
                            <el-divider direction="vertical" border-style="dashed" />

                            {{ item.type }} / {{ item.host }}:{{ item.port }}
                            <el-divider direction="vertical" border-style="dashed" />
                            {{ item.username }}
                        </el-option>
                    </el-select>
                </el-form-item>

                <el-form-item prop="authCertName" label="授权凭证" required>
                    <el-select @focus="getAuthCerts" @change="changeAuthCert" v-model="form.authCertName" placeholder="请选择授权凭证" filterable>
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

                <el-form-item prop="code" label="编号" required>
                    <el-input :disabled="form.id" v-model.trim="form.code" placeholder="请输入编号 (数字字母下划线), 不可修改" auto-complete="off"></el-input>
                </el-form-item>
                <el-form-item prop="name" label="别名" required>
                    <el-input v-model.trim="form.name" placeholder="请输入数据库别名" auto-complete="off"></el-input>
                </el-form-item>

                <el-form-item prop="database" label="数据库名">
                    <el-select
                        v-model="dbNamesSelected"
                        multiple
                        clearable
                        collapse-tags
                        collapse-tags-tooltip
                        filterable
                        :filter-method="filterDbNames"
                        allow-create
                        placeholder="请确保数据库实例信息填写完整后获取库名"
                        style="width: 100%"
                    >
                        <template #header>
                            <el-checkbox v-model="checkAllDbNames" :indeterminate="indeterminateDbNames" @change="handleCheckAll"> 全选 </el-checkbox>
                        </template>
                        <el-option v-for="db in state.dbNamesFiltered" :key="db" :label="db" :value="db" />
                    </el-select>
                </el-form-item>

                <el-form-item prop="remark" label="备注">
                    <el-input v-model.trim="form.remark" auto-complete="off" type="textarea"></el-input>
                </el-form-item>

                <procdef-select-form-item v-model="form.flowProcdefKey" />
            </el-form>

            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="cancel()">取 消</el-button>
                    <el-button type="primary" :loading="saveBtnLoading" @click="btnOk">确 定</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, watch, ref } from 'vue';
import { dbApi } from './api';
import { ElMessage } from 'element-plus';
import TagTreeSelect from '../component/TagTreeSelect.vue';
import type { CheckboxValueType } from 'element-plus';
import ProcdefSelectFormItem from '@/views/flow/components/ProcdefSelectFormItem.vue';
import { DbType } from '@/views/ops/db/dialect';
import { ResourceCodePattern } from '@/common/pattern';
import { resourceAuthCertApi } from '../tag/api';
import { TagResourceTypeEnum } from '@/common/commonEnum';
import EnumTag from '@/components/enumtag/EnumTag.vue';
import { AuthCertCiphertextTypeEnum } from '../tag/enums';

const props = defineProps({
    visible: {
        type: Boolean,
    },
    db: {
        type: [Boolean, Object],
    },
    title: {
        type: String,
    },
});

//定义事件
const emit = defineEmits(['update:visible', 'cancel', 'val-change']);

const rules = {
    tagId: [
        {
            required: true,
            message: '请选择标签',
            trigger: ['change', 'blur'],
        },
    ],

    instanceId: [
        {
            required: true,
            message: '请选择数据库实例',
            trigger: ['change', 'blur'],
        },
    ],
    code: [
        {
            required: true,
            message: '请输入编码',
            trigger: ['change', 'blur'],
        },
        {
            pattern: ResourceCodePattern.pattern,
            message: ResourceCodePattern.message,
            trigger: ['blur'],
        },
    ],
    name: [
        {
            required: true,
            message: '请输入别名',
            trigger: ['change', 'blur'],
        },
    ],
    database: [
        {
            required: true,
            message: '请添加数据库',
            trigger: ['change', 'blur'],
        },
    ],
};

const checkAllDbNames = ref(false);
const indeterminateDbNames = ref(false);

const dbForm: any = ref(null);
const tagSelectRef: any = ref(null);

const state = reactive({
    dialogVisible: false,
    allDatabases: [] as any,
    dbNamesSelected: [] as any,
    dbNamesFiltered: [] as any,
    filterString: '',
    selectInstalce: {} as any,
    authCerts: [] as any,
    form: {
        id: null,
        tagId: [],
        name: null,
        code: '',
        database: '',
        remark: '',
        instanceId: null as any,
        authCertName: '',
        flowProcdefKey: '',
    },
    instances: [] as any,
});

const { dialogVisible, allDatabases, form, dbNamesSelected } = toRefs(state);

const { isFetching: saveBtnLoading, execute: saveDbExec } = dbApi.saveDb.useApi(form);

watch(props, async (newValue: any) => {
    state.dialogVisible = newValue.visible;
    if (!state.dialogVisible) {
        return;
    }
    if (newValue.db) {
        state.form = { ...newValue.db };
        state.form.tagId = newValue.db.tags.map((t: any) => t.tagId);
        // 将数据库名使用空格切割，获取所有数据库列表
        state.dbNamesSelected = newValue.db.database.split(' ');
    } else {
        state.form = {} as any;
        state.dbNamesSelected = [];
    }
});

const changeInstance = async () => {
    state.dbNamesSelected = [];
    state.form.instanceId = state.selectInstalce.id;
};

const getAuthCerts = async () => {
    const res = await resourceAuthCertApi.listByQuery.request({
        resourceCode: state.selectInstalce.code,
        resourceType: TagResourceTypeEnum.Db.value,
        pageSize: 100,
    });
    state.authCerts = res.list || [];
};

const changeAuthCert = (val: string) => {
    getAllDatabase(val);
};

const getAllDatabase = async (authCertName: string) => {
    if (state.form.instanceId > 0) {
        let dbs = await dbApi.getAllDatabase.request({ instanceId: state.form.instanceId, authCertName });
        state.allDatabases = dbs;

        // 如果是oracle，且没查出数据库列表，则取实例sid
        let instance = state.instances.find((item: any) => item.id === state.form.instanceId);
        if (instance && instance.type === DbType.oracle && dbs.length === 0) {
            state.allDatabases = [instance.sid];
        }
    }
};

const getInstances = async (instanceName: string = '', id = 0) => {
    if (!id && !instanceName) {
        state.instances = [];
        return;
    }
    const data = await dbApi.instances.request({ id, name: instanceName });
    if (data) {
        state.instances = data.list;
    }
};

const open = async () => {
    if (state.form.instanceId) {
        // 根据id获取，因为需要回显实例名称
        await getInstances('', state.form.instanceId);
        state.selectInstalce = state.instances[0];
        await getAllDatabase(state.form.authCertName);
    }
};

const btnOk = async () => {
    dbForm.value.validate(async (valid: boolean) => {
        if (!valid) {
            ElMessage.error('请正确填写信息');
            return false;
        }

        await saveDbExec();
        ElMessage.success('保存成功');
        emit('val-change', state.form);
        cancel();
    });
};

const resetInputDb = () => {
    state.dbNamesSelected = [];
    state.allDatabases = [];
    state.instances = [];
};

const cancel = () => {
    emit('update:visible', false);
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

const handleCheckAll = (val: CheckboxValueType) => {
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
