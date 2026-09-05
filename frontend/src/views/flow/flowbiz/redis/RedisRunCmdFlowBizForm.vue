<template>
    <auto-form ref="formRef" v-model="bizForm" :items="bizItems" label-width="auto">
        <template #id>
            <ResourceSelect
                v-bind="$attrs"
                v-model="selectRedis"
                @change="changeRedis"
                :resource-type="TagResourceTypeEnum.Redis.value"
                :placeholder="$t('flow.selectRedisPlaceholder')"
            >
                <template #iconPrefix>
                    <TagCodePath v-if="bizForm.redisCode" :code="bizForm.redisCode" />
                </template>
            </ResourceSelect>
        </template>
    </auto-form>
</template>

<script lang="ts" setup>
import { TagResourceTypeEnum } from '@/common/commonEnum';
import type { AutoFormItem } from '@/components/auto-form';
import { Rules } from '@/common/rule';
import { TagTreeNode } from '@/views/ops/component/tag';
import TagCodePath from '@/views/ops/component/TagCodePath.vue';
import ResourceSelect from '@/views/ops/resource/ResourceSelect.vue';
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import type { FlowBizForm } from '@/views/flow/types';

const { t } = useI18n();

/** Redis 执行命令流程业务表单 */
interface RedisRunCmdForm extends FlowBizForm {
    id?: number;
    db?: number;
    cmd?: string;
    tagPath?: string;
    redisName?: string;
    redisCode?: string;
}

/** Redis 资源节点 params 结构（对应 redis/resource 构造的 db 节点参数） */
interface RedisNodeParams {
    tagPath: string;
    id: number;
    redisName: string;
    code: string;
    db: string;
    [key: string]: unknown;
}

/** Redis 命令执行业务表单声明（资源选择为 custom 插槽；id 校验保留原 message） */
const bizItems: AutoFormItem[] = [
    { prop: 'id', label: 'DB', type: 'custom', rules: [{ required: true, message: t('flow.selectRedisPlaceholder'), trigger: ['change', 'blur'] }] },
    { prop: 'cmd', label: 'CMD', type: 'textarea', required: true, rules: Rules.requiredInput('flow.runCmd'), props: { rows: 5 }, placeholder: 'flow.cmdPlaceholder' },
];

const emit = defineEmits(['changeResourceCode']);

const formRef = ref<{ validate: (...args: unknown[]) => unknown; resetFields?: () => void } | null>(null);

const bizForm = defineModel<RedisRunCmdForm>('bizForm', {
    default: {
        id: 0,
        db: 0,
        cmd: '',
        tagPath: '',
        redisName: '',
    },
});

const selectRedis = computed({
    get: () => {
        return `db${bizForm.value.db}`;
    },
    set: () => {
        //
    },
});

const changeRedis = (nodeData: TagTreeNode) => {
    const params = nodeData.params as RedisNodeParams;
    bizForm.value.tagPath = params.tagPath;
    bizForm.value.redisName = params.redisName;
    bizForm.value.id = params.id;
    bizForm.value.db = parseInt(params.db);
    bizForm.value.redisCode = params.code;

    changeResourceCode(params.code);
};

const changeResourceCode = async (redisCode: string) => {
    emit('changeResourceCode', TagResourceTypeEnum.Redis.value, redisCode);
};

const validateBizForm = async () => {
    return formRef.value?.validate();
};

const resetBizForm = () => {
    //重置表单域
    formRef.value?.resetFields?.();
    bizForm.value.id = 0;
    bizForm.value.db = 0;
    bizForm.value.cmd = '';
};

defineExpose({ validateBizForm, resetBizForm });
</script>
<style lang="scss"></style>
