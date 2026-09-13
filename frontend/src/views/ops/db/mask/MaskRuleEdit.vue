<template>
    <div class="mask-rule-edit">
        <auto-form-drawer
            v-model:visible="dialogVisible"
            :title="title"
            :items="items"
            :data="editData"
            :confirm-api="btnOk"
            @submitted="emit('val-change')"
            @cancel="emit('cancel')"
        />
    </div>
</template>

<script lang="ts" setup>
import { Rules } from '@/common/rule';
import { deepClone } from '@/common/utils/object';
import { AutoFormDrawer, type AutoFormData, type AutoFormItem } from '@/components/auto-form';
import { computed, type PropType } from 'vue';
import { dbMaskApi } from '../api';
import { DbMaskRuleStatusEnum } from '../enums';
import type { DbMaskRule } from '../types';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();

const props = defineProps({
    data: {
        type: Object as PropType<DbMaskRule | null>,
        default: null,
    },
    title: {
        type: String,
    },
});

// visible 由下方 defineModel 声明并自动派发 update:visible，此处不再重复登记
const emit = defineEmits<{
    /** 取消编辑，父级关闭弹窗 */
    cancel: [];
    /** 数据变更需刷新列表；由 auto-form 的 submitted 转发时不带表单，故 payload 可选 */
    'val-change': [form?: FormData];
}>();

const dialogVisible = defineModel<boolean>('visible', { default: false });

// 算法选项 label 传 i18n key：AutoForm 内部对 option.label 做 $t()，此处预先 t() 会被二次翻译（控制台报缺 key，且切换语言后文案不随之更新）
const algorithmOptions = [
    { label: 'db.maskAlgoFull', value: 'full' },
    { label: 'db.maskAlgoPartial', value: 'partial' },
    { label: 'db.maskAlgoHash', value: 'hash' },
    { label: 'db.maskAlgoRegexReplace', value: 'regexReplace' },
    { label: 'db.maskAlgoPhone', value: 'phone' },
    { label: 'db.maskAlgoEmail', value: 'email' },
    { label: 'db.maskAlgoIdcard', value: 'idcard' },
    { label: 'db.maskAlgoBankCard', value: 'bankCard' },
];

/** 表单声明（AutoFormItem[]） */
const items: AutoFormItem[] = [
    { prop: 'name', label: 'db.maskRuleName', required: true },
    {
        prop: 'matchType',
        label: 'db.maskMatchType',
        type: 'radio',
        required: true,
        span: 12,
        options: [
            { label: 'db.maskMatchTypeRegex', value: 1 },
            { label: 'db.maskMatchTypeExact', value: 2 },
            { label: 'db.maskMatchTypePrefix', value: 3 },
        ],
    },
    {
        prop: 'weight',
        label: 'db.maskWeight',
        type: 'number',
        span: 12,
        tooltip: 'db.maskWeightTips',
    },
    {
        prop: 'pattern',
        label: 'db.maskPattern',
        required: true,
        placeholder: 'db.maskPatternPlaceholder',
        rules: [Rules.requiredInput('db.maskPattern')],
    },
    { prop: 'algorithm', label: 'db.maskAlgorithm', type: 'select', required: true, options: algorithmOptions, rules: [Rules.requiredSelect('db.maskAlgorithm')] },
    {
        prop: 'params',
        label: 'db.maskParams',
        type: 'textarea',
        placeholder: 'db.maskParamsPlaceholder',
        tooltip: 'db.maskParamsPlaceholder',
    },
    {
        prop: 'status',
        label: 'common.status',
        type: 'switch',
        span: 12,
        props: {
            inlinePrompt: true,
            activeText: t('common.enable'),
            inactiveText: t('common.disable'),
            activeValue: DbMaskRuleStatusEnum.Enabled.value,
            inactiveValue: DbMaskRuleStatusEnum.Disabled.value,
        },
    },
    { prop: 'remark', label: 'common.remark', type: 'textarea' },
];

type FormData = {
    id?: number;
    name: string;
    matchType: number;
    pattern: string;
    algorithm: string;
    params?: string;
    status: number;
    weight?: number;
    remark?: string;
};

const basicFormData = {
    name: '',
    matchType: 1,
    pattern: '',
    algorithm: 'full',
    status: DbMaskRuleStatusEnum.Enabled.value,
    weight: 100,
} as FormData;

/** 新建态用默认值，编辑态深拷贝行数据回填 */
const editData = computed<AutoFormData | null>(() => {
    if (props.data?.id) {
        return deepClone(props.data) as unknown as AutoFormData;
    }
    return { ...basicFormData } as unknown as AutoFormData;
});

const btnOk = async (rawForm: AutoFormData) => {
    const reqForm = { ...(rawForm as unknown as FormData) };
    if (reqForm.id) {
        await dbMaskApi.updateMaskRule.request(reqForm);
    } else {
        await dbMaskApi.saveMaskRule.request(reqForm);
    }
    emit('val-change', reqForm);
};
</script>
<style lang="scss" scoped></style>
