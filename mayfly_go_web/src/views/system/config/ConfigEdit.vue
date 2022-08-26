<template>
    <div>
        <el-dialog :title="title" v-model="dvisible" :show-close="false" :before-close="cancel" width="500px" :destroy-on-close="true">
            <el-form ref="configForm" :model="form" label-width="90px">
                <el-form-item prop="name" label="配置项:" required>
                    <el-input v-model="form.name"></el-input>
                </el-form-item>
                <el-form-item prop="key" label="配置key:" required>
                    <el-input :disabled="form.id != null" v-model="form.key"></el-input>
                </el-form-item>
                <el-form-item prop="value" label="配置值:" required>
                    <el-input v-model="form.value"></el-input>
                </el-form-item>
                <el-form-item label="备注:">
                    <el-input v-model="form.remark" type="textarea" :rows="2"></el-input>
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

<script lang="ts">
import { ref, toRefs, reactive, watch, defineComponent } from 'vue';
import { configApi } from '../api';

export default defineComponent({
    name: 'ConfigEdit',
    props: {
        visible: {
            type: Boolean,
        },
        data: {
            type: [Boolean, Object],
        },
        title: {
            type: String,
        },
    },
    setup(props: any, { emit }) {
        const configForm: any = ref(null);
        const state = reactive({
            dvisible: false,
            form: {
                id: null,
                name: '',
                key: '',
                value: '',
                remark: '',
            },
            btnLoading: false,
        });

        watch(props, (newValue) => {
            state.dvisible = newValue.visible;
            if (newValue.data) {
                state.form = { ...newValue.data };
            } else {
                state.form = {} as any;
            }
        });

        const cancel = () => {
            // 更新父组件visible prop对应的值为false
            emit('update:visible', false);
            // 若父组件有取消事件，则调用
            emit('cancel');
        };

        const btnOk = async () => {
            configForm.value.validate(async (valid: boolean) => {
                if (valid) {
                    await configApi.save.request(state.form);
                    emit('val-change', state.form);
                    cancel();
                    state.btnLoading = true;
                    setTimeout(() => {
                        state.btnLoading = false;
                    }, 1000);
                }
            });
        };

        return {
            ...toRefs(state),
            configForm,
            btnOk,
            cancel,
        };
    },
});
</script>
<style lang="scss">
</style>
