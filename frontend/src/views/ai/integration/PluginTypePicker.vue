<template>
    <el-dialog v-model="visible" :title="$t('ai.integration.newPlugin')" width="480px" append-to-body @closed="onDialogClosed">
        <div class="type-picker">
            <div v-for="item in types" :key="item.type" class="type-card" @click="pick(item.type)">
                <el-icon :size="26" :color="item.color">
                    <component :is="item.icon" />
                </el-icon>
                <div class="type-info">
                    <div class="type-name">{{ $t(item.nameKey) }}</div>
                    <div class="type-desc">{{ $t(item.descKey) }}</div>
                </div>
                <el-icon class="type-arrow"><ArrowRight /></el-icon>
            </div>
        </div>
    </el-dialog>
</template>

<script lang="ts" setup>
import { computed, ref } from 'vue';
import { ArrowRight, FolderOpened, Connection } from '@element-plus/icons-vue';
import type { PluginType } from './types';

const props = defineProps<{ modelValue: boolean }>();
const emit = defineEmits<{
    (e: 'update:modelValue', v: boolean): void;
    (e: 'pick', type: PluginType): void;
}>();

const visible = computed({
    get: () => props.modelValue,
    set: (v: boolean) => emit('update:modelValue', v),
});

const types = [
    { type: 'skill' as const, icon: FolderOpened, color: 'var(--el-color-primary)', nameKey: 'ai.integration.skillPlugin', descKey: 'ai.integration.skillPluginDesc' },
    { type: 'mcp' as const, icon: Connection, color: 'var(--el-color-success)', nameKey: 'ai.integration.mcpPlugin', descKey: 'ai.integration.mcpPluginDesc' },
];

// 选中类型后暂存，待 dialog 完全关闭（@closed，overlay 销毁完毕）再派发，
// 避免同步「关 dialog → 开 drawer」时 overlay 管理器冲突导致抽屉打不开
const pendingType = ref<PluginType | null>(null);

const pick = (type: PluginType) => {
    pendingType.value = type;
    visible.value = false;
};

const onDialogClosed = () => {
    if (pendingType.value) {
        emit('pick', pendingType.value);
        pendingType.value = null;
    }
};
</script>

<style lang="scss" scoped>
.type-picker {
    display: flex;
    flex-direction: column;
    gap: 12px;

    .type-card {
        display: flex;
        align-items: center;
        gap: 14px;
        padding: 14px 16px;
        border: 1px solid var(--el-border-color-light);
        border-radius: 8px;
        cursor: pointer;
        transition: all 0.2s;

        &:hover {
            border-color: var(--el-color-primary);
            background-color: var(--el-color-primary-light-9);
        }

        .type-info {
            flex: 1;

            .type-name {
                font-size: 14px;
                font-weight: 600;
                color: var(--el-text-color-primary);
            }

            .type-desc {
                margin-top: 2px;
                font-size: 12px;
                color: var(--el-text-color-secondary);
            }
        }

        .type-arrow {
            color: var(--el-text-color-placeholder);
        }
    }
}
</style>
