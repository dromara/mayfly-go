<template>
    <div class="menu-dialog">
        <el-dialog :title="title" :destroy-on-close="true" v-model="dialogVisible" width="769px">
            <el-form :model="form" :inline="true" ref="menuForm" :rules="rules" label-width="95px">
                <el-row :gutter="10">
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" class="mb10">
                        <el-form-item prop="type" label="类型" required>
                            <el-select v-model="form.type" :disabled="typeDisabled" placeholder="请选择" >
                                <el-option v-for="item in enums.ResourceTypeEnum" :key="item.value" :label="item.label" :value="item.value">
                                </el-option>
                            </el-select>
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" class="mb10">
                        <el-form-item prop="name" label="名称" required>
                            <el-input v-model.trim="form.name" placeholder="资源名[菜单名]" auto-complete="off"></el-input>
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" class="mb10">
                        <el-form-item prop="code" label="path|code">
                            <el-input v-model.trim="form.code" placeholder="菜单不带/自动拼接父路径"></el-input>
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" class="mb10">
                        <el-form-item label="序号" prop="weight" required>
                            <el-input v-model.trim="form.weight" type="number" placeholder="请输入序号"></el-input>
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" class="mb10">
                        <el-form-item v-if="form.type === enums.ResourceTypeEnum.MENU.value" label="图标">
                            <icon-selector v-model="form.meta.icon" type="ele" />
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" class="mb10">
                        <el-form-item v-if="form.type === enums.ResourceTypeEnum.MENU.value" prop="code" label="路由名">
                            <el-input v-model.trim="form.meta.routeName" placeholder="请输入路由名称"></el-input>
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" class="mb10">
                        <el-form-item v-if="form.type === enums.ResourceTypeEnum.MENU.value" prop="code" label="组件">
                            <el-input v-model.trim="form.meta.component" placeholder="请输入组件名"></el-input>
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" class="mb10">
                        <el-form-item v-if="form.type === enums.ResourceTypeEnum.MENU.value" prop="code" label="是否缓存">
                            <el-select v-model="form.meta.isKeepAlive" placeholder="请选择" width="w100">
                                <el-option v-for="item in trueFalseOption" :key="item.value" :label="item.label" :value="item.value"> </el-option>
                            </el-select>
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" class="mb10">
                        <el-form-item v-if="form.type === enums.ResourceTypeEnum.MENU.value" prop="code" label="是否隐藏">
                            <el-select v-model="form.meta.isHide" placeholder="请选择" width="w100">
                                <el-option v-for="item in trueFalseOption" :key="item.value" :label="item.label" :value="item.value"> </el-option>
                            </el-select>
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" class="mb10">
                        <el-form-item v-if="form.type === enums.ResourceTypeEnum.MENU.value" prop="code" label="tag不可删除">
                            <el-select v-model="form.meta.isAffix" placeholder="请选择" width="w100">
                                <el-option v-for="item in trueFalseOption" :key="item.value" :label="item.label" :value="item.value"> </el-option>
                            </el-select>
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" class="mb10">
                        <el-form-item v-if="form.type === enums.ResourceTypeEnum.MENU.value" prop="code" label="是否iframe">
                            <el-select @change="changeIsIframe" v-model="form.meta.isIframe" placeholder="请选择" width="w100">
                                <el-option v-for="item in trueFalseOption" :key="item.value" :label="item.label" :value="item.value"> </el-option>
                            </el-select>
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" class="mb10">
                        <el-form-item
                            v-if="form.type === enums.ResourceTypeEnum.MENU.value && form.meta.isIframe"
                            prop="code"
                            label="iframe地址"
                            width="w100"
                        >
                            <el-input v-model.trim="form.meta.link" placeholder="请输入iframe url"></el-input>
                        </el-form-item>
                    </el-col>
                </el-row>
            </el-form>

            <template #footer>
                <div>
                    <el-button @click="cancel()">取 消</el-button>
                    <el-button type="primary" :loading="btnLoading" @click="btnOk">确 定</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts">
import { ref, toRefs, reactive, watch, defineComponent } from 'vue';
import { ElMessage } from 'element-plus';
import { resourceApi } from '../api';
import enums from '../enums';
import { notEmpty } from '@/common/assert';
import iconSelector from '@/components/iconSelector/index.vue';

export default defineComponent({
    name: 'ResourceEdit',
    components: {
        iconSelector,
    },
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
        typeDisabled: {
            type: Boolean,
        },
    },
    setup(props: any, { emit }) {
        const menuForm: any = ref(null);

        const defaultMeta = {
            routeName: '',
            icon: 'Menu',
            redirect: '',
            component: '',
            isKeepAlive: true,
            isHide: false,
            isAffix: false,
            isIframe: false,
        };

        const state = reactive({
            trueFalseOption: [
                {
                    label: '是',
                    value: true,
                },
                {
                    label: '否',
                    value: false,
                },
            ],
            dialogVisible: false,
            //弹出框对象
            dialogForm: {
                title: '',
                visible: false,
                data: {},
            },
            props: {
                value: 'id',
                label: 'name',
                children: 'children',
            },
            form: {
                id: null,
                name: null,
                pid: null,
                code: null,
                type: null,
                weight: 0,
                meta: {
                    routeName: '',
                    icon: '',
                    redirect: '',
                    component: '',
                    isKeepAlive: true,
                    isHide: false,
                    isAffix: false,
                    isIframe: false,
                },
            },
            // 资源类型选择是否禁用
            // typeDisabled: false,
            btnLoading: false,
            rules: {
                name: [
                    {
                        required: true,
                        message: '请输入资源名称',
                        trigger: ['change', 'blur'],
                    },
                ],
                weight: [
                    {
                        required: true,
                        message: '请输入序号',
                        trigger: ['change', 'blur'],
                    },
                ],
            },
        });

        watch(props, (newValue) => {
            state.dialogVisible = newValue.visible;
            if (newValue.data) {
                state.form = { ...newValue.data };
            } else {
                state.form = {} as any;
            }

            if (!state.form.meta) {
                state.form.meta = defaultMeta;
            }

            // 不存在或false，都为false
            const meta: any = state.form.meta;
            state.form.meta.isKeepAlive = meta.isKeepAlive ? true : false;
            state.form.meta.isHide = meta.isHide ? true : false;
            state.form.meta.isAffix = meta.isAffix ? true : false;
            state.form.meta.isIframe = meta.isIframe ? true : false;
        });

        // 改变iframe字段，如果为是，则设置默认的组件
        const changeIsIframe = (value: boolean) => {
            if (value) {
                state.form.meta.component = 'RouterParent';
            }
        };

        const btnOk = () => {
            const submitForm = { ...state.form };
            if (submitForm.type == 1) {
                // 如果是菜单，则解析meta，如果值为false或者''则去除该值
                submitForm.meta = parseMenuMeta(submitForm.meta);
            } else {
                submitForm.meta = null as any;
            }
            submitForm.weight = parseInt(submitForm.weight as any);
            menuForm.value.validate((valid: any) => {
                if (valid) {
                    resourceApi.save.request(submitForm).then(() => {
                        emit('val-change', submitForm);
                        state.btnLoading = true;
                        ElMessage.success('保存成功');
                        setTimeout(() => {
                            state.btnLoading = false;
                        }, 1000);

                        cancel();
                    });
                } else {
                    return false;
                }
            });
        };

        const parseMenuMeta = (meta: any) => {
            let metaForm: any = {};
            // 如果是菜单，则校验meta
            notEmpty(meta.routeName, '路由名不能为空');
            metaForm.routeName = meta.routeName;
            if (meta.isKeepAlive) {
                metaForm.isKeepAlive = true;
            }
            if (meta.isHide) {
                metaForm.isHide = true;
            }
            if (meta.isAffix) {
                metaForm.isAffix = true;
            }
            if (meta.isIframe) {
                metaForm.isIframe = true;
            }
            if (meta.link) {
                metaForm.link = meta.link;
            }
            if (meta.redirect) {
                metaForm.redirect = meta.redirect;
            }
            if (meta.component) {
                metaForm.component = meta.component;
            }
            if (meta.icon) {
                metaForm.icon = meta.icon;
            }
            return metaForm;
        };

        const cancel = () => {
            emit('update:visible', false);
            emit('cancel');
        };

        return {
            ...toRefs(state),
            enums,
            changeIsIframe,
            menuForm,
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
</style>
