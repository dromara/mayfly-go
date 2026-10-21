<template>
    <div class="machine-param-input">
        <div class="machine-param-input__hint">
            {{ t('ai.interrupt.paramCompletion.selectMachineHint') }}
        </div>

        <!-- 使用 MachineSelectTree 组件：单向传值 + 事件写回（不在对象属性上直接 v-model） -->
        <MachineSelectTree
            :auth-cert-name="machineValue.authCertName"
            :machine-id="machineValue.machineId"
            :machine-name="machineValue.machineName"
            :machine-ip="machineValue.machineIp"
            :machine-port="machineValue.machinePort"
            :username="machineValue.username"
            :tag-path="machineValue.tagPath"
            :disabled="isConfirmed"
            @update:auth-cert-name="patch({ authCertName: $event })"
            @update:machine-id="patch({ machineId: $event })"
            @update:machine-name="patch({ machineName: $event })"
            @update:machine-ip="patch({ machineIp: $event })"
            @update:machine-port="patch({ machinePort: $event })"
            @update:username="patch({ username: $event })"
            @update:tag-path="patch({ tagPath: $event })"
        />

        <!-- 已选中的机器详细信息 -->
        <div v-if="machineValue.authCertName || machineValue.machineId" class="machine-param-input__detail">
            <div class="machine-param-input__detail-row">
                <SvgIcon name="Monitor" :size="20" />
                <div class="machine-param-input__detail-info">
                    <div class="machine-param-input__detail-name">{{ machineValue.machineName || t('ai.interrupt.paramCompletion.machineSelected') }}</div>
                    <div class="machine-param-input__detail-meta">
                        {{ t('ai.interrupt.paramCompletion.machineIp') }}: {{ machineValue.machineIp || '-' }}:{{ machineValue.machinePort || '-' }}
                    </div>
                    <div class="machine-param-input__detail-meta">
                        {{ t('ai.interrupt.paramCompletion.authCert') }}: {{ machineValue.authCertName }} ({{ machineValue.username || '-' }})
                    </div>
                </div>
                <SvgIcon v-if="isConfirmed" class="text-success" name="check" />
            </div>
        </div>
    </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';
import MachineSelectTree from '@/views/ops/machine/component/MachineSelectTree.vue';

interface MachineParamValue {
    authCertName: string;
    machineId: number;
    machineName: string;
    machineIp: string;
    machinePort: number;
    username: string;
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
    // modelValue 由下方 defineModel 拥有契约与默认值，不在 Props 里重复声明（两份默认值靠编译器合并顺序决定谁生效）
}

const props = withDefaults(defineProps<Props>(), {
    readonly: false,
    isConfirmed: false,
});

const { t } = useI18n();

// 使用 defineModel 实现双向绑定
const machineValue = defineModel<MachineParamValue>('modelValue', {
    default: () => ({
        authCertName: '',
        machineId: 0,
        machineName: '',
        machineIp: '',
        machinePort: 0,
        username: '',
        tagPath: '',
    }),
});

/**
 * 以「整体替换值对象」的方式写回 model（与 DbParamInput 同一约定）：
 * 原地 mutate 传入对象的属性不会触发 update:modelValue，父层失去变更事件与回滚点
 */
const patch = (partial: Partial<MachineParamValue>) => {
    machineValue.value = { ...machineValue.value, ...partial };
};

// 检查是否有效
const isValid = () => {
    return machineValue.value.machineId > 0;
};

// 获取参数值
const getValues = () => {
    return {
        id: machineValue.value.machineId,
        params: {
            authCertName: machineValue.value.authCertName,
            machineId: machineValue.value.machineId,
            machineName: machineValue.value.machineName,
            machineIp: machineValue.value.machineIp,
            machinePort: machineValue.value.machinePort,
            username: machineValue.value.username,
            tagPath: machineValue.value.tagPath,
        },
        displayName: `${machineValue.value.machineName} (${machineValue.value.machineIp})`,
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
.machine-param-input {
    padding: 8px;
}

.machine-param-input__hint {
    font-size: 12px;
    color: var(--el-text-color-secondary);
    margin-bottom: 8px;
}

.machine-param-input__detail {
    margin-top: 12px;
    padding: 12px;
    background: var(--el-color-primary-light-9);
    border-radius: 6px;
    border: 1px solid var(--el-color-primary-light-7);
}

.machine-param-input__detail-row {
    display: flex;
    align-items: center;
    gap: 8px;
}

.machine-param-input__detail-info {
    flex: 1;
}

.machine-param-input__detail-name {
    font-size: 13px;
    font-weight: 500;
}

.machine-param-input__detail-meta {
    font-size: 12px;
    color: var(--el-text-color-secondary);
    margin-top: 2px;
}
</style>
