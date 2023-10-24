<template>
    <div class="mock-data-dialog">
        <el-dialog
            :title="title"
            v-model="dialogVisible"
            :close-on-click-modal="false"
            :before-close="cancel"
            :show-close="true"
            :destroy-on-close="true"
            width="900px"
        >
            <el-form :model="form" :rules="rules" ref="scriptForm" label-width="auto">
                <el-form-item prop="name" label="名称" required>
                    <el-input v-model="form.name" placeholder="请输入名称"></el-input>
                </el-form-item>

                <el-form-item prop="description" label="描述" required>
                    <el-input v-model="form.description" placeholder="请输入描述"></el-input>
                </el-form-item>

                <el-form-item prop="type" label="类型" required>
                    <el-select v-model="form.type" default-first-option style="width: 100%" placeholder="请选择类型">
                        <el-option v-for="item in ScriptResultEnum" :key="item.value" :label="item.label" :value="item.value"></el-option>
                    </el-select>
                </el-form-item>

                <el-row style="margin-left: 30px; margin-bottom: 5px">
                    <el-button @click="onAddParam" type="success">新增占位符参数</el-button>
                </el-row>
                <el-form-item :key="param" v-for="(param, index) in params" prop="params" :label="`参数${index + 1}`">
                    <el-row>
                        <el-col :span="5">
                            <el-input v-model.trim="param.model" placeholder="内容中用{{.model}}替换"></el-input>
                        </el-col>
                        <span :span="1">
                            <el-divider direction="vertical" border-style="dashed" />
                        </span>
                        <el-col :span="4">
                            <el-input v-model.trim="param.name" placeholder="字段名"></el-input>
                        </el-col>
                        <span :span="1">
                            <el-divider direction="vertical" border-style="dashed" />
                        </span>
                        <el-col :span="4">
                            <el-input v-model="param.placeholder" placeholder="字段说明"></el-input>
                        </el-col>
                        <span :span="1">
                            <el-divider direction="vertical" border-style="dashed" />
                        </span>
                        <el-col :span="4">
                            <el-input v-model="param.options" placeholder="可选值 ,分割"></el-input>
                        </el-col>
                        <span :span="1">
                            <el-divider direction="vertical" border-style="dashed" />
                        </span>
                        <el-col :span="2">
                            <el-button @click="onDeleteParam(index)" type="danger">删除</el-button>
                        </el-col>
                    </el-row>
                </el-form-item>

                <el-form-item required prop="script" class="100w">
                    <div style="width: 100%">
                        <monaco-editor v-model="form.script" language="shell" height="300px" />
                    </div>
                </el-form-item>
            </el-form>

            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="cancel()" :disabled="submitDisabled">关 闭</el-button>
                    <el-button v-auth="'machine:script:save'" type="primary" :loading="btnLoading" @click="btnOk" :disabled="submitDisabled">保 存</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { ref, toRefs, reactive, watch } from 'vue';
import { ElMessage } from 'element-plus';
import { machineApi } from './api';
import { ScriptResultEnum } from './enums';
import MonacoEditor from '@/components/monaco/MonacoEditor.vue';

const props = defineProps({
    visible: {
        type: Boolean,
    },
    data: {
        type: Object,
    },
    title: {
        type: String,
    },
    machineId: {
        type: Number,
    },
    isCommon: {
        type: Boolean,
    },
});

const emit = defineEmits(['update:visible', 'cancel', 'submitSuccess']);

const rules = {
    name: [
        {
            required: true,
            message: '请输入名称',
            trigger: ['change', 'blur'],
        },
    ],
    description: [
        {
            required: true,
            message: '请输入描述',
            trigger: ['blur', 'change'],
        },
    ],
    type: [
        {
            required: true,
            message: '请选择类型',
            trigger: ['change', 'blur'],
        },
    ],
    script: [
        {
            required: true,
            message: '请输入脚本',
            trigger: ['blur', 'change'],
        },
    ],
};

const { isCommon, machineId } = toRefs(props);
const scriptForm: any = ref(null);

const state = reactive({
    dialogVisible: false,
    submitDisabled: false,
    params: [] as any,
    form: {
        id: null,
        name: '',
        machineId: 0,
        description: '',
        script: '',
        params: '',
        type: null,
    },
    btnLoading: false,
});

const { dialogVisible, submitDisabled, params, form, btnLoading } = toRefs(state);

watch(props, (newValue: any) => {
    state.dialogVisible = newValue.visible;
    if (!newValue.visible) {
        return;
    }
    if (newValue.data) {
        state.form = { ...newValue.data };
        if (state.form.params) {
            state.params = JSON.parse(state.form.params);
        }
    } else {
        state.form = {} as any;
        state.form.script = '';
    }
});

const onAddParam = () => {
    state.params.push({ name: '', model: '', placeholder: '' });
};

const onDeleteParam = (idx: number) => {
    state.params.splice(idx, 1);
};

const btnOk = () => {
    state.form.machineId = isCommon.value ? 9999999 : (machineId?.value as any);
    scriptForm.value.validate((valid: any) => {
        if (valid) {
            if (state.params) {
                state.form.params = JSON.stringify(state.params);
            }
            machineApi.saveScript.request(state.form).then(
                () => {
                    ElMessage.success('保存成功');
                    emit('submitSuccess');
                    state.submitDisabled = false;
                    cancel();
                },
                () => {
                    state.submitDisabled = false;
                }
            );
        } else {
            return false;
        }
    });
};

const cancel = () => {
    emit('update:visible', false);
    emit('cancel');
    state.params = [];
};
</script>
<style lang="scss"></style>
