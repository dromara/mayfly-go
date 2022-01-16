<template>
    <div>
        <el-dialog :title="title" v-model="dialogVisible" :show-close="false" :before-close="cancel" width="65%">
            <el-form :model="form" ref="apiForm" :rules="rules" label-width="85px" size="small">
                <el-form-item prop="serviceId" label="服务:" required>
                    <el-select style="width: 100%" v-model="form.serviceId" placeholder="请选择服务" @change="changeService" filterable>
                        <el-option v-for="item in services" :key="item.id" :label="`${item.name}`" :value="item.id"> </el-option>
                    </el-select>
                </el-form-item>

                <el-form-item prop="name" label="名称:" required>
                    <el-input v-model.trim="form.name" placeholder="请输入api名称" auto-complete="off"></el-input>
                </el-form-item>
                <el-form-item prop="code" label="code:" required>
                    <el-input :disabled="form.id" v-model.trim="form.code" placeholder="请输入code" auto-complete="off"></el-input>
                </el-form-item>
                <el-form-item prop="method" label="method:" required>
                    <el-select style="width: 100%" v-model="form.method" placeholder="请选择请求方法" @change="changeService" filterable>
                        <el-option key="get" label="GET" value="GET"> </el-option>
                        <el-option key="post" label="POST" value="POST"> </el-option>
                        <el-option key="put" label="PUT" value="PUT"> </el-option>
                        <el-option key="delete" label="DELETE" value="DELETE"> </el-option>
                    </el-select>
                </el-form-item>
                <el-form-item prop="uri" label="uri:" required>
                    <el-input v-model.trim="form.uri" placeholder="请输入method:uri格式"></el-input>
                </el-form-item>
                <el-form-item label="schema:">
                    <!-- <vue3-json-editor v-model="jsonschema" @json-change="schemaChange" :show-btns="false" :expandedOnStart="true" /> -->
                </el-form-item>
            </el-form>

            <template #footer>
                <div class="dialog-footer">
                    <el-button type="primary" :loading="btnLoading" @click="btnOk" size="mini">确 定</el-button>
                    <el-button @click="cancel()" size="mini">取 消</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts">
import { toRefs, reactive, watch, defineComponent, ref } from 'vue';
import { serviceApi } from '../api';
import { ElMessage } from 'element-plus';
// import { Vue3JsonEditor } from 'vue3-json-editor';

export default defineComponent({
    name: 'RedisEdit',
    components: {
        // Vue3JsonEditor,
    },
    props: {
        visible: {
            type: Boolean,
        },
        services: {
            type: Array,
        },
        api: {
            type: [Boolean, Object],
        },
        title: {
            type: String,
        },
    },
    setup(props: any, { emit }) {
        const apiForm: any = ref(null);
        const state = reactive({
            dialogVisible: false,
            services: [],
            form: {
                id: null,
                name: null,
                code: null,
                uri: null,
                serviceName: null,
                serviceId: null,
                schema: null,
            },
            jsonschema: {},
            btnLoading: false,
            rules: {
                serviceId: [
                    {
                        required: true,
                        message: '请选择服务',
                        trigger: ['change', 'blur'],
                    },
                ],
                name: [
                    {
                        required: true,
                        message: '请输入api名称',
                        trigger: ['change', 'blur'],
                    },
                ],
                method: [
                    {
                        required: true,
                        message: '请输入请求method',
                        trigger: ['change', 'blur'],
                    },
                ],
                uri: [
                    {
                        required: true,
                        message: '请输入请求uri',
                        trigger: ['change', 'blur'],
                    },
                ],
                code: [
                    {
                        required: true,
                        message: '请输入code',
                        trigger: ['change', 'blur'],
                    },
                ],
            },
        });

        watch(props, async (newValue) => {
            state.services = newValue.services;
            if (newValue.api) {
                state.form = { ...newValue.api };
                console.log(state.form)
                if (state.form.schema) {
                    state.jsonschema = JSON.parse(state.form.schema as any);
                }
            } else {
                state.form = { } as any;
            }
            state.dialogVisible = newValue.visible;
        });

        const changeService = (serviceId: number) => {
            for (let p of state.services as any) {
                if (p.id == serviceId) {
                    state.form.serviceName = p.name;
                }
            }
        };

        const schemaChange = (jsonValue: any) => {
            state.form.schema = JSON.stringify(jsonValue) as any;
        };

        const btnOk = async () => {
            apiForm.value.validate((valid: boolean) => {
                if (valid) {
                    serviceApi.saveServiceApi.request(state.form).then(() => {
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

        const cancel = () => {
            emit('update:visible', false);
            emit('cancel');
            setTimeout(() => {
                apiForm.value.resetFields();
                //  重置对象属性为null
                state.form = {} as any;
                state.jsonschema = {};
            }, 200);
        };

        return {
            ...toRefs(state),
            schemaChange,
            apiForm,
            changeService,
            btnOk,
            cancel,
        };
    },
});
</script>
<style lang="scss">
</style>
