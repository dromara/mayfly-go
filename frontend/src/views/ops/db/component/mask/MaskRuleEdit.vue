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
import { dbMaskApi } from '../../api';
import type { DbMaskRule } from '../../types';
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

const emit = defineEmits(['update:visible', 'cancel', 'val-change']);

const dialogVisible = defineModel<boolean>('visible', { default: false });

const algorithmOptions = [
    { label: t('db.maskAlgoFull'), value: 'full' },
    { label: t('db.maskAlgoPartial'), value: 'partial' },
    { label: t('db.maskAlgoHash'), value: 'hash' },
    { label: t('db.maskAlgoRegexReplace'), value: 'regexReplace' },
    { label: t('db.maskAlgoPhone'), value: 'phone' },
    { label: t('db.maskAlgoEmail'), value: 'email' },
    { label: t('db.maskAlgoIdcard'), value: 'idcard' },
    { label: t('db.maskAlgoBankCard'), value: 'bankCard' },
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
        props: { inlinePrompt: true, activeText: t('common.enable'), inactiveText: t('common.disable'), activeValue: 1, inactiveValue: 0 },
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
    status: 1,
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
