<template>
    <div class="db-param-input">
        <div class="db-param-input__hint">
            {{ t('ai.interrupt.paramCompletion.selectDbHint') }}
        </div>

        <!-- 使用 DbSelectTree 组件 -->
        <DbSelectTree
            v-model:db-id="dbValue.dbId"
            v-model:inst-name="dbValue.instanceName"
            v-model:db-name="dbValue.dbName"
            v-model:tag-path="dbValue.tagPath"
            v-model:db-type="dbValue.dbType"
            :disabled="isConfirmed"
            @select-db="onSelectDb"
        />

        <!-- 已选中的数据库详细信息 -->
        <div v-if="dbValue.dbId" class="db-param-input__detail">
            <div class="db-param-input__detail-row">
                <SvgIcon :name="getDbDialect(dbValue.dbType)?.getInfo().icon || 'DataLine'" :size="20" />
                <div class="db-param-input__detail-info">
                    <div class="db-param-input__detail-name">{{ dbValue.instanceName }} - {{ dbValue.dbName }}</div>
                    <div class="db-param-input__detail-meta">{{ t('ai.interrupt.paramCompletion.dbType') }}: {{ dbValue.dbType }}</div>
                </div>
                <SvgIcon v-if="isConfirmed" name="check" class="text-success" :size="20" />
            </div>
        </div>
    </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';
import SvgIcon from '@/components/svg-icon/index.vue';
import DbSelectTree from '@/views/ops/db/component/DbSelectTree.vue';
import { getDbDialect } from '@/views/ops/db/dialect';

interface DbParamValue {
    dbId: number;
    dbName: string;
    dbType: string;
    instanceName: string;
    tagPath: string;
}

interface ParamDef {
    param: string;
    cacheable?: boolean;
}

interface Props {
    params: ParamDef[];
    readonly?: boolean;
    isConfirmed?: boolean;
    modelValue?: DbParamValue;
}

const props = withDefaults(defineProps<Props>(), {
    readonly: false,
    isConfirmed: false,
    modelValue: () => ({
        dbId: 0,
        dbName: '',
        dbType: '',
        instanceName: '',
        tagPath: '',
    }),
});

const { t } = useI18n();

// 使用 defineModel 实现双向绑定
const dbValue = defineModel<DbParamValue>('modelValue', {
    default: () => ({
        dbId: 0,
        dbName: '',
        dbType: '',
        instanceName: '',
        tagPath: '',
    }),
});

// 处理数据库选择
const onSelectDb = (_params: Record<string, unknown>) => {
    // Database selected, handled by DbSelectTree
};

// 检查是否有效
const isValid = () => {
    return dbValue.value.dbId > 0;
};

// 获取参数值
const getValues = () => {
    return {
        id: dbValue.value.dbId,
        params: { ...dbValue.value },
        displayName: `${dbValue.value.instanceName} - ${dbValue.value.dbName}`,
    };
};

// 获取需要缓存的参数名
const getCacheableParams = () => {
    return props.params.filter((p) => p.cacheable === true).map((p) => p.param);
};

defineExpose({
    isValid,
    getValues,
    getCacheableParams,
});
</script>

<style scoped>
.db-param-input {
    padding: 4px;
}

.db-param-input__hint {
    font-size: 12px;
    color: var(--el-text-color-secondary);
    margin-bottom: 8px;
}

.db-param-input__detail {
    margin-top: 8px;
    padding: 8px;
    background: var(--el-color-primary-light-9);
    border-radius: 6px;
    border: 1px solid var(--el-color-primary-light-7);
}

.db-param-input__detail-row {
    display: flex;
    align-items: center;
    gap: 6px;
}

.db-param-input__detail-info {
    flex: 1;
}

.db-param-input__detail-name {
    font-size: 13px;
    font-weight: 500;
}

.db-param-input__detail-meta {
    font-size: 12px;
    color: var(--el-text-color-secondary);
    margin-top: 2px;
}
</style>
