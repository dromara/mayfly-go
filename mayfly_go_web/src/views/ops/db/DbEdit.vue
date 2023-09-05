<template>
    <div>
        <el-dialog :title="title" v-model="dialogVisible" @open="open" :before-close="cancel" :close-on-click-modal="false" :destroy-on-close="true" width="38%">
            <el-form :model="form" ref="dbForm" :rules="rules" label-width="auto">
                <el-form-item prop="tagId" label="标签:" required>
                    <tag-select v-model="form.tagId" v-model:tag-path="form.tagPath" style="width: 100%" />
                </el-form-item>

                <el-form-item prop="instanceId" label="数据库实例:" required>
                    <el-select :disabled="form.id !== undefined" @change="getAllDatabase" v-model="form.instanceId" placeholder="请选择实例" filterable clearable style="width: 200px">
                      <el-option v-for="item in state.instances" :key="item.id" :label="item.name" :value="item.id"> </el-option>
                    </el-select>
                </el-form-item>

                <el-form-item prop="name" label="别名:" required>
                    <el-input v-model.trim="form.name" placeholder="请输入数据库别名" auto-complete="off"></el-input>
                </el-form-item>

                <el-form-item prop="database" label="数据库名:" required>
                      <el-select
                          @change="changeDatabase"
                          v-model="databaseList"
                          multiple
                          clearable
                          collapse-tags
                          collapse-tags-tooltip
                          filterable
                          allow-create
                          placeholder="请确保数据库实例信息填写完整后获取库名"
                          style="width: 100%"
                      >
                          <el-option v-for="db in allDatabases" :key="db" :label="db" :value="db" />
                      </el-select>
                </el-form-item>

                <el-form-item prop="remark" label="备注:">
                    <el-input v-model.trim="form.remark" auto-complete="off" type="textarea"></el-input>
                </el-form-item>
            </el-form>

            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="cancel()">取 消</el-button>
                    <el-button type="primary" :loading="btnLoading" @click="btnOk">确 定</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, watch, ref } from 'vue';
import { dbApi } from './api';
import { ElMessage } from 'element-plus';
import TagSelect from '../component/TagSelect.vue';

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

const dbForm: any = ref(null);

const state = reactive({
    dialogVisible: false,
    allDatabases: [] as any,
    databaseList: [] as any,
    form: {
        id: null,
        tagId: null as any,
        tagPath: null as any,
        name: null,
        database: '',
        remark: '',
        instanceId: null as any,
    },
    btnLoading: false,
    instances: [] as any,
});

const { dialogVisible, allDatabases, databaseList, form, btnLoading } = toRefs(state);

watch(props, (newValue: any) => {
    state.dialogVisible = newValue.visible;
    if (!state.dialogVisible) {
        return;
    }
    if (newValue.db) {
        state.form = { ...newValue.db };
        // 将数据库名使用空格切割，获取所有数据库列表
        state.databaseList = newValue.db.database.split(' ');
    } else {
        state.form = {} as any;
        state.databaseList = [];
    }
});

/**
 * 改变表单中的数据库字段，方便表单错误提示。如全部删光，可提示请添加数据库
 */
const changeDatabase = () => {
    state.form.database = state.databaseList.length == 0 ? '' : state.databaseList.join(' ');
};

const getAllDatabase = async () => {
    const reqForm = { ...state.form };
    if (state.form.instanceId > 0) {
        state.allDatabases = await dbApi.getAllDatabase.request(reqForm);
    }
};

const getAllInstances = async () => {
    const data = await dbApi.instances.request(null);
    if (data) {
        state.instances = data.list;
    }
}
const open = async () => {
    await getAllInstances()
    await getAllDatabase()
};

const btnOk = async () => {
    dbForm.value.validate(async (valid: boolean) => {
        if (valid) {
            const reqForm = { ...state.form };
            dbApi.saveDb.request(reqForm).then(() => {
                ElMessage.success('保存成功');
                emit('val-change', state.form);
                state.btnLoading = true;
                setTimeout(() => {
                    state.btnLoading = false;
                }, 1000);

                cancel();
            });
        } else {
            ElMessage.error('请正确填写信息');
            return false;
        }
    });
};

const resetInputDb = () => {
    state.databaseList = [];
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
</script>
<style lang="scss"></style>
