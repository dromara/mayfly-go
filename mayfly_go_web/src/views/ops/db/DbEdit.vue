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
                        :resource-code="form.code"
                        :resource-type="TagResourceTypeEnum.Db.value"
                        style="width: 100%"
                    />
                </el-form-item>

                <el-form-item prop="instanceId" label="数据库实例" required>
                    <el-select
                        :disabled="form.id !== undefined"
                        remote
                        :remote-method="getInstances"
                        @change="changeInstance"
                        v-model="form.instanceId"
                        placeholder="请输入实例名称搜索并选择实例"
                        filterable
                        clearable
                        class="w100"
                    >
                        <el-option v-for="item in state.instances" :key="item.id" :label="`${item.name}`" :value="item.id">
                            {{ item.name }}
                            <el-divider direction="vertical" border-style="dashed" />

                            {{ item.type }} / {{ item.host }}:{{ item.port }}
                            <el-divider direction="vertical" border-style="dashed" />
                            {{ item.username }}
                        </el-option>
                    </el-select>
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
import { TagResourceTypeEnum } from '@/common/commonEnum';
import type { CheckboxValueType } from 'element-plus';

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
    form: {
        id: null,
        tagId: [],
        name: null,
        code: '',
        database: '',
        remark: '',
        instanceId: null as any,
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

        // 将数据库名使用空格切割，获取所有数据库列表
        state.dbNamesSelected = newValue.db.database.split(' ');
    } else {
        state.form = {} as any;
        state.dbNamesSelected = [];
    }
});

const changeInstance = () => {
    state.dbNamesSelected = [];
    getAllDatabase();
};

const getAllDatabase = async () => {
    if (state.form.instanceId > 0) {
        state.allDatabases = await dbApi.getAllDatabase.request({ instanceId: state.form.instanceId });
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
    }
    await getAllDatabase();
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
