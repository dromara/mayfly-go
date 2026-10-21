<template>
    <el-dialog v-model="visible" :title="$t('ai.integration.newPlugin')" width="480px" append-to-body @closed="onDialogClosed">
        <div class="type-picker">
            <div v-for="item in types" :key="item.type" class="type-card" @click="pick(item.type)">
                <el-icon :size="26" :color="item.color">
                    <component :is="item.icon" />
                </el-icon>
                <div class="type-info">
                    <div class="type-name">{{ item.name }}</div>
                    <div class="type-desc">{{ item.desc }}</div>
                </div>
                <el-icon class="type-arrow"><ArrowRight /></el-icon>
            </div>
        </div>
    </el-dialog>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref, type Component } from 'vue';
import { useI18n } from 'vue-i18n';
import { ArrowRight } from '@element-plus/icons-vue';
import { pluginApi, type PluginTypeInfo } from './api';
import type { PluginType } from './types';
import { pluginTypeMeta } from './types';
import { pluginTypeIcon } from './pluginTypeIcons';

const { t } = useI18n();

const props = defineProps<{ modelValue: boolean }>();
const emit = defineEmits<{
    (e: 'update:modelValue', v: boolean): void;
    (e: 'pick', type: PluginType): void;
}>();

const visible = computed({
    get: () => props.modelValue,
    set: (v: boolean) => emit('update:modelValue', v),
});

// 类型展示映射不再本组件自写一份：图标/主色/名称/副标题全部取自 PLUGIN_TYPE_META（与卡片同源），
// 新增插件类型只需在 types 表里登记一行，本组件零改动

interface PickerType {
    type: PluginType;
    icon: Component;
    color: string;
    name: string;
    desc: string;
}

const types = ref<PickerType[]>([]);

onMounted(async () => {
    try {
        const list: PluginTypeInfo[] = (await pluginApi.listTypes.request()) || [];
        types.value = list.map((it) => {
            const meta = pluginTypeMeta(it.code);
            return {
                type: it.code,
                icon: pluginTypeIcon(it.code),
                color: meta.accentColor,
                // 未登记类型直接显示后端 code 且副标题为空，不伪装成已知类型的文案
                name: meta.labelKey ? t(meta.labelKey) : it.code,
                desc: meta.descKey ? t(meta.descKey) : '',
            };
        });
    } catch {
        // 拉取失败（请求层已 toast），选择器为空
    }
});

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
