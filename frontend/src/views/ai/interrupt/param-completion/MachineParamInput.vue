<template>
    <div class="machine-param-input">
        <div class="machine-param-input__hint">
            {{ t('ai.interrupt.paramCompletion.selectMachineHint') }}
        </div>

        <!-- 使用 MachineSelectTree 组件 -->
        <MachineSelectTree
            v-model:auth-cert-name="machineValue.authCertName"
            v-model:machine-id="machineValue.machineId"
            v-model:machine-name="machineValue.machineName"
            v-model:machine-ip="machineValue.machineIp"
            v-model:machine-port="machineValue.machinePort"
            v-model:username="machineValue.username"
            v-model:tag-path="machineValue.tagPath"
            :disabled="isConfirmed"
            @select-machine="onSelectMachine"
        />

        <!-- 已选中的机器详细信息 -->
        <div
            v-if="machineValue.authCertName || machineValue.machineId"
            class="machine-param-input__detail"
        >
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
    modelValue?: MachineParamValue;
}

const props = withDefaults(defineProps<Props>(), {
    readonly: false,
    isConfirmed: false,
    modelValue: () => ({
        authCertName: '',
        machineId: 0,
        machineName: '',
        machineIp: '',
        machinePort: 0,
        username: '',
        tagPath: '',
    }),
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

// 处理机器选择
const onSelectMachine = (_params: Record<string, unknown>) => {
    // Machine selected
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
