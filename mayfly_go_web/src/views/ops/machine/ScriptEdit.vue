<template>
    <div class="mock-data-dialog">
        <el-dialog
            :title="title"
            v-model="dialogVisible"
            :close-on-click-modal="false"
            :before-close="cancel"
            :show-close="true"
            :destroy-on-close="true"
            width="800px"
        >
            <el-form :model="form" ref="mockDataForm" label-width="70px">
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

                <el-form-item prop="params" label="参数">
                    <el-input v-model.trim="form.params" placeholder="参数数组json，若无可不填"></el-input>
                </el-form-item>

                <el-form-item prop="script" label="内容" id="content">
                    <codemirror ref="cmEditor" v-model="form.script" language="shell" />
                </el-form-item>
            </el-form>

            <template #footer>
                <div class="dialog-footer">
                    <el-button
                        v-auth="'machine:script:save'"
                        type="primary"
                        :loading="btnLoading"
                        @click="btnOk"
                        size="small"
                        :disabled="submitDisabled"
                        >保 存</el-button
                    >
                    <el-button @click="cancel()" :disabled="submitDisabled" size="small">关 闭</el-button>
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
        const mockDataForm: any = ref(null);

        const state = reactive({
            dialogVisible: false,
            submitDisabled: false,
            form: {
                id: null,
                name: '',
                machineId: 0,
                description: '',
                script: '',
                params: null,
                type: null,
            },
            btnLoading: false,
        });

        watch(props, (newValue) => {
            if (newValue.data) {
                state.form = { ...newValue.data };
            } else {
                state.form = {} as any;
                state.form.script = '';
            }
            state.dialogVisible = newValue.visible;
        });

        const btnOk = () => {
            state.form.machineId = isCommon.value ? 9999999 : (machineId.value as any);
            console.log('machineid:', machineId);
            mockDataForm.value.validate((valid: any) => {
                if (valid) {
                    notEmpty(state.form.name, '名称不能为空');
                    notEmpty(state.form.description, '描述不能为空');
                    notEmpty(state.form.script, '内容不能为空');
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
        };

        return {
            ...toRefs(state),
            enums,
            mockDataForm,
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
