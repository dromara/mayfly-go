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
            <el-form :model="form" ref="scriptForm" label-width="70px" size="small">
                <el-form-item prop="method" label="名称">
                    <el-input v-model.trim="form.name" placeholder="请输入名称"></el-input>
                </el-form-item>

                <el-form-item prop="description" label="描述">
                    <el-input v-model.trim="form.description" placeholder="请输入描述"></el-input>
                </el-form-item>

                <el-form-item prop="type" label="类型">
                    <el-select v-model="form.type" default-first-option style="width: 100%" placeholder="请选择类型">
                        <el-option v-for="item in enums.scriptTypeEnum" :key="item.value" :label="item.label" :value="item.value"></el-option>
                    </el-select>
                </el-form-item>

                <el-row style="margin-left: 30px; margin-bottom: 5px">
                    <el-button @click="onAddParam" size="small" type="success">新增占位符参数</el-button>
                </el-row>
                <el-form-item :key="param" v-for="(param, index) in params" prop="params" :label="`参数${index + 1}`">
                    <el-row>
                        <el-col :span="5"><el-input v-model="param.model" placeholder="内容中用{{.model}}替换"></el-input></el-col>
                        <el-divider :span="1" direction="vertical" border-style="dashed" />
                        <el-col :span="4"><el-input v-model="param.name" placeholder="字段名"></el-input></el-col>
                        <el-divider :span="1" direction="vertical" border-style="dashed" />
                        <el-col :span="4"><el-input v-model="param.placeholder" placeholder="字段说明"></el-input></el-col>
                        <el-divider :span="1" direction="vertical" border-style="dashed" />
                        <el-col :span="4">
                            <el-input v-model="param.options" placeholder="可选值 ,分割"></el-input>
                        </el-col>
                        <el-divider :span="1" direction="vertical" border-style="dashed" />
                        <el-col :span="2"><el-button @click="onDeleteParam(index)" size="small" type="danger">删除</el-button></el-col>
                    </el-row>
                </el-form-item>

                <el-form-item prop="script" label="内容" id="content">
                    <codemirror ref="cmEditor" v-model="form.script" language="shell" width="700px" />
                </el-form-item>
            </el-form>

            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="cancel()" :disabled="submitDisabled">关 闭</el-button>
                    <el-button
                        v-auth="'machine:script:save'"
                        type="primary"
                        :loading="btnLoading"
                        @click="btnOk"
                        :disabled="submitDisabled"
                        >保 存</el-button
                    >
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts">
import { ref, toRefs, reactive, watch, defineComponent } from 'vue';
import { ElMessage } from 'element-plus';
import { machineApi } from './api';
import enums from './enums';
import { notEmpty } from '@/common/assert';

import { codemirror } from '@/components/codemirror';

export default defineComponent({
    name: 'ScriptEdit',
    components: {
        codemirror,
    },
    props: {
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
    },
    setup(props: any, { emit }) {
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

        watch(props, (newValue) => {
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
            state.form.machineId = isCommon.value ? 9999999 : (machineId.value as any);
            console.log('machineid:', machineId);
            scriptForm.value.validate((valid: any) => {
                if (valid) {
                    notEmpty(state.form.name, '名称不能为空');
                    notEmpty(state.form.description, '描述不能为空');
                    notEmpty(state.form.script, '内容不能为空');
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

        return {
            ...toRefs(state),
            enums,
            onAddParam,
            onDeleteParam,
            scriptForm,
            btnOk,
            cancel,
        };
    },
});
</script>
<style lang="scss">
// 	.m-dialog {
// 		.el-cascader {
// 			width: 100%;
// 		}
// 	}
#content {
    .CodeMirror {
        height: 300px !important;
    }
}
</style>
