<template>
    <div>
        <el-table :data="cmdConfs" stripe>
            <el-table-column prop="name" label="名称" show-overflow-tooltip min-width="100px"> </el-table-column>
            <el-table-column prop="cmds" label="过滤命令" min-width="320px" show-overflow-tooltip>
                <template #default="scope">
                    <el-tag class="ml2 mt2" v-for="cmd in scope.row.cmds" :key="cmd" type="danger">
                        {{ cmd }}
                    </el-tag>
                </template>
            </el-table-column>
            <el-table-column prop="codePaths" label="关联机器" min-width="250px" show-overflow-tooltip>
                <template #default="scope">
                    <TagCodePath :path="scope.row.tags.map((tag: any) => tag.codePath)" />
                </template>
            </el-table-column>
            <el-table-column prop="remark" label="备注" show-overflow-tooltip width="120px"> </el-table-column>
            <el-table-column prop="creator" label="创建者" show-overflow-tooltip width="100px"> </el-table-column>

            <el-table-column label="操作" min-wdith="100px">
                <template #header>
                    <el-text tag="b">操作</el-text>
                    <el-button v-auth="'cmdconf:save'" class="ml5" type="primary" circle size="small" icon="Plus" @click="openFormDialog(false)"> </el-button>
                </template>
                <template #default="scope">
                    <el-button v-auth="'cmdconf:save'" @click="openFormDialog(scope.row)" type="primary" link>编辑</el-button>
                    <el-button v-auth="'cmdconf:del'" @click="deleteCmdConf(scope.row)" type="danger" link>删除</el-button>
                </template>
            </el-table-column>
        </el-table>

        <el-drawer title="命令配置" v-model="dialogVisible" :show-close="false" width="600px" :destroy-on-close="true" :close-on-click-modal="false">
            <template #header>
                <DrawerHeader header="命令配置" :back="cancelEdit" />
            </template>

            <el-form ref="formRef" :model="state.form" :rules="rules" label-width="auto">
                <el-form-item prop="name" label="名称" required>
                    <el-input v-model="form.name" placeholder="名称"></el-input>
                </el-form-item>

                <el-form-item prop="cmds" label="过滤命令" required>
                    <el-row>
                        <el-tag
                            class="ml2 mt2"
                            v-for="tag in form.cmds"
                            :key="tag"
                            closable
                            :disable-transitions="false"
                            @close="handleCmdClose(tag)"
                            type="danger"
                        >
                            {{ tag }}
                        </el-tag>
                        <el-input
                            v-if="state.inputCmdVisible"
                            ref="cmdInputRef"
                            v-model="state.cmdInputValue"
                            class="mt3"
                            size="small"
                            @keyup.enter="handleCmdInputConfirm"
                            @blur="handleCmdInputConfirm"
                            placeholder="请输入命令正则表达式"
                        />
                        <el-button v-else class="ml2 mt2" size="small" @click="showCmdInput"> + 新建命令 </el-button>
                    </el-row>
                </el-form-item>

                <el-form-item label="备注">
                    <el-input v-model="form.remark" type="textarea" :rows="2"></el-input>
                </el-form-item>

                <el-form-item ref="tagSelectRef" prop="codePaths" label="关联机器">
                    <tag-tree-check height="calc(100vh - 430px)" :tag-type="TagResourceTypeEnum.MachineAuthCert.value" v-model="form.codePaths" />
                </el-form-item>
            </el-form>
            <template #footer>
                <div class="dialog-footer">
                    <el-button :loading="submiting" @click="cancelEdit">取 消</el-button>
                    <el-button v-auth="'cmdconf:save'" type="primary" :loading="submiting" @click="submitForm">确 定</el-button>
                </div>
            </template>
        </el-drawer>
    </div>
</template>

<script lang="ts" setup>
import { ref, toRefs, reactive, onMounted, nextTick } from 'vue';
import TagTreeCheck from '../../component/TagTreeCheck.vue';
import { TagResourceTypeEnum } from '@/common/commonEnum';
import { ElMessage, ElMessageBox } from 'element-plus';
import { cmdConfApi } from '../api';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';
import TagCodePath from '../../component/TagCodePath.vue';
import _ from 'lodash';

const rules = {
    tags: [
        {
            required: true,
            message: '请选择关联的机器',
            trigger: ['change'],
        },
    ],
    cmds: [
        {
            required: true,
            message: '请创建命令',
            trigger: ['change', 'blur'],
        },
    ],
    name: [
        {
            required: true,
            message: '请输入名称',
            trigger: ['change', 'blur'],
        },
    ],
};

const tagSelectRef: any = ref(null);
const formRef: any = ref(null);
const cmdInputRef: any = ref(null);

const DefaultForm = {
    id: 0,
    name: '',
    codePaths: [],
    cmds: [] as any,
    remark: '',
};

const state = reactive({
    cmdConfs: [],
    dialogVisible: false,
    form: DefaultForm,
    submiting: false,
    inputCmdVisible: false,
    cmdInputValue: '',
});

const { cmdConfs, dialogVisible, form, submiting } = toRefs(state);

onMounted(async () => {
    getCmdConfs();
});

const getCmdConfs = async () => {
    state.cmdConfs = await cmdConfApi.list.request();
};

const handleCmdClose = (tag: string) => {
    state.form.cmds.splice(state.form.cmds.indexOf(tag), 1);
};

const showCmdInput = () => {
    state.inputCmdVisible = true;
    nextTick(() => {
        cmdInputRef.value!.input!.focus();
    });
};

const handleCmdInputConfirm = () => {
    if (state.cmdInputValue) {
        state.form.cmds.push(state.cmdInputValue);
    }
    state.inputCmdVisible = false;
    state.cmdInputValue = '';
};

const openFormDialog = (data: any) => {
    if (!data) {
        state.form = { ...DefaultForm };
    } else {
        state.form = _.cloneDeep(data);
        state.form.codePaths = data.tags.map((tag: any) => tag.codePath);
    }
    state.dialogVisible = true;
};

const deleteCmdConf = async (data: any) => {
    await ElMessageBox.confirm(`确定删除该[${data.name}]命令配置?`, '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
    });

    await cmdConfApi.delete.request({ id: data.id });
    ElMessage.success('操作成功');
    getCmdConfs();
};

const cancelEdit = () => {
    state.dialogVisible = false;
    // 取消表单的校验
    setTimeout(() => {
        state.form = { ...DefaultForm };
        formRef.value.resetFields();
    }, 200);
};

const submitForm = () => {
    formRef.value.validate(async (valid: boolean) => {
        if (!valid) {
            ElMessage.error('请正确填写信息');
            return false;
        }

        try {
            state.submiting = true;
            await cmdConfApi.save.request(state.form);
            ElMessage.success('操作成功');

            cancelEdit();
            getCmdConfs();
        } finally {
            state.submiting = false;
        }
    });
};
</script>
<style></style>
