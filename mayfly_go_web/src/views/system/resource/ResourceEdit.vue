<template>
    <div class="system-menu-dialog-container layout-pd">
        <el-dialog :title="title" :destroy-on-close="true" v-model="dialogVisible" width="800px">
            <el-form :model="form" :inline="true" ref="menuForm" :rules="rules" label-width="auto">
                <el-row :gutter="35">
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" class="mb20">
                        <el-form-item class="w100" prop="type" label="类型" required>
                            <el-select class="w100" v-model="form.type" :disabled="typeDisabled" placeholder="请选择">
                                <el-option v-for="item in ResourceTypeEnum" :key="item.value" :label="item.label" :value="item.value"> </el-option>
                            </el-select>
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" class="mb20">
                        <el-form-item class="w100" prop="name" label="名称" required>
                            <el-input v-model.trim="form.name" placeholder="资源名[菜单名]" auto-complete="off"></el-input>
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" class="mb20">
                        <el-form-item class="w100" prop="code" label="path|code">
                            <template #label>
                                path|code
                                <el-tooltip
                                    effect="dark"
                                    content="菜单类型则为访问路径（若菜单路径不以'/'开头则访问地址会自动拼接父菜单路径）、否则为资源唯一编码"
                                    placement="top"
                                >
                                    <el-icon>
                                        <question-filled />
                                    </el-icon>
                                </el-tooltip>
                            </template>
                            <el-input v-model.trim="form.code" placeholder="菜单不以'/'开头则自动拼接父菜单路径"></el-input>
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" class="mb20" v-if="form.type === menuTypeValue">
                        <el-form-item class="w100" label="图标">
                            <icon-selector v-model="form.meta.icon" />
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" class="mb20" v-if="form.type === menuTypeValue">
                        <el-form-item class="w100">
                            <template #label>
                                路由名
                                <el-tooltip effect="dark" content="与vue的组件名一致才可使组件缓存生效，如ResourceList" placement="top">
                                    <el-icon>
                                        <question-filled />
                                    </el-icon>
                                </el-tooltip>
                            </template>
                            <el-input v-model.trim="form.meta.routeName" placeholder="请输入路由名称"></el-input>
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" class="mb20" v-if="form.type === menuTypeValue">
                        <el-form-item class="w100" prop="code">
                            <template #label>
                                组件路径
                                <el-tooltip effect="dark" content="访问的组件路径，如：`system/resource/ResourceList`，默认在`views`目录下" placement="top">
                                    <el-icon>
                                        <question-filled />
                                    </el-icon>
                                </el-tooltip>
                            </template>
                            <el-input v-model.trim="form.meta.component" placeholder="请输入组件路径"></el-input>
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" class="mb20" v-if="form.type === menuTypeValue">
                        <el-form-item class="w100" prop="isKeepAlive">
                            <template #label>
                                是否缓存
                                <el-tooltip
                                    effect="dark"
                                    content="选择是则会被`keep-alive`缓存(重新进入页面不会刷新页面及重新请求数据)，需要路由名与vue的组件名一致"
                                    placement="top"
                                >
                                    <el-icon>
                                        <question-filled />
                                    </el-icon>
                                </el-tooltip>
                            </template>
                            <el-select v-model="form.meta.isKeepAlive" placeholder="请选择" class="w100">
                                <el-option v-for="item in trueFalseOption" :key="item.value" :label="item.label" :value="item.value"> </el-option>
                            </el-select>
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" class="mb20" v-if="form.type === menuTypeValue">
                        <el-form-item class="w100">
                            <template #label>
                                是否隐藏
                                <el-tooltip effect="dark" content="选择隐藏则路由将不会出现在菜单栏中，但仍然可以访问。禁用则不可访问与操作" placement="top">
                                    <el-icon>
                                        <question-filled />
                                    </el-icon>
                                </el-tooltip>
                            </template>
                            <el-select v-model="form.meta.isHide" placeholder="请选择" class="w100">
                                <el-option v-for="item in trueFalseOption" :key="item.value" :label="item.label" :value="item.value"> </el-option>
                            </el-select>
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" class="mb20" v-if="form.type === menuTypeValue">
                        <el-form-item class="w100" prop="code" label="tag不可删除">
                            <el-select v-model="form.meta.isAffix" placeholder="请选择" class="w100">
                                <el-option v-for="item in trueFalseOption" :key="item.value" :label="item.label" :value="item.value"> </el-option>
                            </el-select>
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" class="mb20" v-if="form.type === menuTypeValue">
                        <el-form-item class="w100" prop="linkType">
                            <template #label>
                                外链
                                <el-tooltip effect="dark" content="内嵌: 以iframe展示、外链: 新标签打开" placement="top">
                                    <el-icon>
                                        <question-filled />
                                    </el-icon>
                                </el-tooltip>
                            </template>
                            <el-select class="w100" @change="changeLinkType" v-model="form.meta.linkType" placeholder="请选择">
                                <el-option :key="0" label="否" :value="0"> </el-option>
                                <el-option :key="1" label="内嵌" :value="1"> </el-option>
                                <el-option :key="2" label="外链" :value="2"> </el-option>
                            </el-select>
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" class="mb20" v-if="form.type === menuTypeValue && form.meta.linkType > 0">
                        <el-form-item prop="code" label="链接地址" class="w100">
                            <el-input v-model.trim="form.meta.link" placeholder="外链/内嵌的链接地址（http://xxx.com）"></el-input>
                        </el-form-item>
                    </el-col>
                </el-row>
            </el-form>

            <template #footer>
                <div>
                    <el-button @click="cancel()">取 消</el-button>
                    <el-button type="primary" :loading="saveBtnLoading" @click="btnOk">确 定</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { ref, toRefs, reactive, watch } from 'vue';
import { ElMessage } from 'element-plus';
import { resourceApi } from '../api';
import { ResourceTypeEnum } from '../enums';
import { notEmpty } from '@/common/assert';
import iconSelector from '@/components/iconSelector/index.vue';

const props = defineProps({
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
});

//定义事件
const emit = defineEmits(['update:visible', 'cancel', 'val-change']);

const menuForm: any = ref(null);

const menuTypeValue = ResourceTypeEnum.Menu.value;

const defaultMeta = {
    routeName: '',
    icon: 'Menu',
    redirect: '',
    component: '',
    isKeepAlive: true,
    isHide: false,
    isAffix: false,
    linkType: 0,
    link: '',
};

const rules = {
    name: [
        {
            required: true,
            message: '请输入资源名称',
            trigger: ['change', 'blur'],
        },
    ],
};

const trueFalseOption = [
    {
        label: '是',
        value: true,
    },
    {
        label: '否',
        value: false,
    },
];

const state = reactive({
    dialogVisible: false,
    form: {
        id: null,
        name: null,
        pid: null,
        code: null,
        type: null,
        meta: {
            routeName: '',
            icon: '',
            redirect: '',
            component: '',
            isKeepAlive: true,
            isHide: false,
            isAffix: false,
            linkType: 0,
            link: '',
        },
    },
    submitForm: {},
});

const { dialogVisible, form, submitForm } = toRefs(state);

const { isFetching: saveBtnLoading, execute: saveResouceExec } = resourceApi.save.useApi(submitForm);

watch(props, (newValue: any) => {
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
    state.form.meta.linkType = meta.linkType;
});

// 改变外链类型
const changeLinkType = () => {
    state.form.meta.component = '';
};

const btnOk = () => {
    const submitForm = { ...state.form };
    if (submitForm.type == 1) {
        // 如果是菜单，则解析meta，如果值为false或者''则去除该值
        submitForm.meta = parseMenuMeta(submitForm.meta);
    } else {
        submitForm.meta = null as any;
    }

    menuForm.value.validate(async (valid: any) => {
        if (valid) {
            state.submitForm = submitForm;
            await saveResouceExec();

            emit('val-change', submitForm);
            ElMessage.success('保存成功');
            cancel();
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
    if (meta.linkType) {
        metaForm.linkType = meta.linkType;
    }
    if (meta.link) {
        metaForm.link = meta.link;
    } else {
        delete metaForm['link'];
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
</script>
<style lang="scss">
// 	.m-dialog {
// 		.el-cascader {
// 			width: 100%;
// 		}
// 	}
</style>
