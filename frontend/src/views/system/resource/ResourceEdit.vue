<template>
    <div class="system-menu-dialog-container layout-pd">
        <el-dialog :title="title" :destroy-on-close="true" v-model="dialogVisible" width="800px">
            <el-form :model="form" :inline="true" ref="menuForm" :rules="rules" label-width="auto">
                <el-row :gutter="35">
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12">
                        <el-form-item class="w100" prop="type" :label="$t('common.type')" required>
                            <enum-select :enums="ResourceTypeEnum" v-model="form.type" :disabled="typeDisabled" />
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12">
                        <el-form-item class="w100" prop="name" :label="$t('common.name')" required>
                            <el-input v-model.trim="form.name" auto-complete="off"></el-input>
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12">
                        <el-form-item class="w100" prop="code" label="path|code">
                            <template #label>
                                path|code
                                <el-tooltip :content="$t('system.menu.menuCodeTips')" placement="top">
                                    <el-icon>
                                        <question-filled />
                                    </el-icon>
                                </el-tooltip>
                            </template>
                            <el-input v-model.trim="form.code" :placeholder="$t('system.menu.menuCodePlaceholder')" auto-complete="off"></el-input>
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" v-if="form.type === menuTypeValue">
                        <el-form-item class="w100" :label="$t('system.menu.icon')">
                            <icon-selector v-model="form.meta.icon" />
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" v-if="form.type === menuTypeValue">
                        <el-form-item class="w100">
                            <template #label>
                                {{ $t('system.menu.routerName') }}
                                <el-tooltip :content="$t('system.menu.routerNameTips')" placement="top">
                                    <el-icon>
                                        <question-filled />
                                    </el-icon>
                                </el-tooltip>
                            </template>
                            <el-input v-model.trim="form.meta.routeName"></el-input>
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" v-if="form.type === menuTypeValue">
                        <el-form-item class="w100" prop="code">
                            <template #label>
                                {{ $t('system.menu.componentPath') }}
                                <el-tooltip :content="$t('system.menu.componentPathTips')" placement="top">
                                    <el-icon>
                                        <question-filled />
                                    </el-icon>
                                </el-tooltip>
                            </template>
                            <el-input v-model.trim="form.meta.component"></el-input>
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" v-if="form.type === menuTypeValue">
                        <el-form-item class="w100" prop="isKeepAlive">
                            <template #label>
                                {{ $t('system.menu.isCache') }}
                                <el-tooltip :content="$t('system.menu.isCacheTips')" placement="top">
                                    <el-icon>
                                        <question-filled />
                                    </el-icon>
                                </el-tooltip>
                            </template>
                            <el-select v-model="form.meta.isKeepAlive" class="w100">
                                <el-option v-for="item in trueFalseOption" :key="item.value" :label="item.label" :value="item.value"> </el-option>
                            </el-select>
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" v-if="form.type === menuTypeValue">
                        <el-form-item class="w100">
                            <template #label>
                                {{ $t('system.menu.isHide') }}
                                <el-tooltip :content="$t('system.menu.isHideTips')" placement="top">
                                    <el-icon>
                                        <question-filled />
                                    </el-icon>
                                </el-tooltip>
                            </template>
                            <el-select v-model="form.meta.isHide" class="w100">
                                <el-option v-for="item in trueFalseOption" :key="item.value" :label="item.label" :value="item.value"> </el-option>
                            </el-select>
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" v-if="form.type === menuTypeValue">
                        <el-form-item class="w100" prop="code" :label="$t('system.menu.tagIsDelete')">
                            <el-select v-model="form.meta.isAffix" class="w100">
                                <el-option v-for="item in trueFalseOption" :key="item.value" :label="item.label" :value="item.value"> </el-option>
                            </el-select>
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" v-if="form.type === menuTypeValue">
                        <el-form-item class="w100" prop="linkType">
                            <template #label>
                                {{ $t('system.menu.externalLink') }}
                                <el-tooltip content="" placement="top">
                                    <el-icon>
                                        <question-filled />
                                    </el-icon>
                                </el-tooltip>
                            </template>
                            <el-select class="w100" @change="changeLinkType" v-model="form.meta.linkType">
                                <el-option :key="0" :label="$t('system.menu.no')" :value="0"> </el-option>
                                <el-option :key="1" :label="$t('system.menu.inline')" :value="1"> </el-option>
                                <el-option :key="2" :label="$t('system.menu.externalLink')" :value="2"> </el-option>
                            </el-select>
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" v-if="form.type === menuTypeValue && form.meta.linkType > 0">
                        <el-form-item prop="code" :label="$t('system.menu.linkAddress')" class="w100">
                            <el-input v-model.trim="form.meta.link" :placeholder="$t('system.menu.linkPlaceholder')"></el-input>
                        </el-form-item>
                    </el-col>
                </el-row>
            </el-form>

            <template #footer>
                <div>
                    <el-button @click="cancel()">{{ $t('common.cancel') }}</el-button>
                    <el-button type="primary" :loading="saveBtnLoading" @click="btnOk">{{ $t('common.confirm') }}</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { ref, toRefs, reactive, watchEffect } from 'vue';
import { ElMessage } from 'element-plus';
import { resourceApi } from '../api';
import { ResourceTypeEnum } from '../enums';
import { notEmpty } from '@/common/assert';
import iconSelector from '@/components/iconSelector/index.vue';
import { useI18n } from 'vue-i18n';
import EnumSelect from '@/components/enumselect/EnumSelect.vue';

const { t } = useI18n();

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
            message: t('system.menu.menuNameRuleMsg'),
            trigger: ['change', 'blur'],
        },
    ],
};

const trueFalseOption = [
    {
        label: t('system.menu.yes'),
        value: true,
    },
    {
        label: t('system.menu.no'),
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

watchEffect(() => {
    state.dialogVisible = props.visible;
    if (props.data) {
        state.form = { ...(props.data as any) };
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

const btnOk = async () => {
    try {
        await menuForm.value.validate();
    } catch (e: any) {
        ElMessage.error(t('common.formValidationError'));
        return false;
    }

    const submitForm = { ...state.form };
    if (submitForm.type == 1) {
        // 如果是菜单，则解析meta，如果值为false或者''则去除该值
        submitForm.meta = parseMenuMeta(submitForm.meta);
    } else {
        submitForm.meta = null as any;
    }

    state.submitForm = submitForm;
    await saveResouceExec();

    emit('val-change', submitForm);
    ElMessage.success(t('common.saveSuccess'));
    cancel();
};

const parseMenuMeta = (meta: any) => {
    let metaForm: any = {};
    // 如果是菜单，则校验meta
    notEmpty(meta.routeName, t('system.menu.routeNameNotEmpty'));
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
<style lang="scss"></style>
