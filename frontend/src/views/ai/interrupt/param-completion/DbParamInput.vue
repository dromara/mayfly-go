<template>
    <div class="db-param-input">
        <div class="db-param-input__hint">
            {{ t('ai.interrupt.paramCompletion.selectDbHint') }}
        </div>

        <!-- 使用 DbSelectTree 组件：单向传值 + 事件写回（不在对象属性上直接 v-model） -->
        <DbSelectTree
            :db-id="dbValue.dbId"
            :inst-name="dbValue.instanceName"
            :db-name="dbValue.dbName"
            :tag-path="dbValue.tagPath"
            :db-type="dbValue.dbType"
            :disabled="isConfirmed"
            @update:db-id="patch({ dbId: $event })"
            @update:inst-name="patch({ instanceName: $event })"
            @update:db-name="patch({ dbName: $event })"
            @update:tag-path="patch({ tagPath: $event })"
            @update:db-type="patch({ dbType: $event })"
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
import DbSelectTree from '@/views/ops/db/widgets/DbSelectTree.vue';
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
    // modelValue 不在此声明：它由下面的 defineModel 拥有契约与默认值。
    // 两边各声明一次就是两份默认值，最终以哪份生效取决于编译器合并顺序（未定义行为面）
}

const props = withDefaults(defineProps<Props>(), {
    readonly: false,
    isConfirmed: false,
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

/**
 * 以「整体替换值对象」的方式写回 model。
 *
 * 不用 `v-model:db-id="dbValue.dbId"`：那是在原地 mutate 传入的对象属性，
 * update:modelValue 永不触发——父层的 paramInputValues 被就地改写、没有变更事件也没有
 * 回滚点，而它同时是另一个子组件的 v-model 目标（两个子组件共享同一个可变对象互写）
 */
const patch = (partial: Partial<DbParamValue>) => {
    dbValue.value = { ...dbValue.value, ...partial };
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
