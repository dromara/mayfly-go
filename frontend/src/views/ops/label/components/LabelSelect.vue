<template>
    <div class="label-selector">
        <!-- 已选标签 -->
        <div v-if="modelValue && modelValue.length > 0" class="selected-labels">
            <el-tooltip
                v-for="(label, index) in modelValue"
                :key="index"
                :content="getLabelDescription(label.key, label.value)"
                :disabled="!getLabelDescription(label.key, label.value)"
                placement="top"
            >
                <el-tag
                    closable
                    @close="removeLabel(index)"
                    class="label-tag"
                    :style="getTagStyle(label)"
                    size="small"
                >
                    {{ label.key }}:{{ label.value }}
                </el-tag>
            </el-tooltip>
        </div>

        <!-- 添加按钮 -->
        <el-popover
            v-model:visible="popoverVisible"
            :width="420"
            trigger="click"
            placement="bottom-start"
            :show-arrow="false"
            @show="onPopoverShow"
        >
            <template #reference>
                <el-button size="small" :icon="Plus" class="add-btn">
                    {{ $t('label.addLabel') }}
                </el-button>
            </template>

            <div class="label-popover">
                <!-- 搜索框 -->
                <div class="search-box">
                    <el-input
                        v-model="searchText"
                        :placeholder="$t('label.searchLabel')"
                        prefix-icon="Search"
                        clearable
                        size="small"
                        ref="searchInputRef"
                    />
                </div>

                <!-- 统计信息 -->
                <div v-if="!searchText" class="stats-bar">
                    <span class="stats-text">
                        {{ $t('label.totalLabels', { count: totalCount }) }}
                    </span>
                    <span v-if="selectedCount > 0" class="stats-selected">
                        {{ $t('label.selectedCount', { count: selectedCount }) }}
                    </span>
                </div>

                <!-- 标签列表 -->
                <div class="label-list" ref="labelListRef">
                    <template v-if="displayedLabels.length > 0">
                        <div
                            v-for="item in displayedLabels"
                            :key="item.key"
                            class="label-group"
                        >
                            <div class="group-header">
                                <span class="group-key">{{ item.key }}</span>
                            </div>
                            <div class="group-values">
                                <div
                                    v-for="val in getFilteredValues(item)"
                                    :key="val.value"
                                    class="label-item"
                                    :class="{ selected: isSelected(item.key, val.value) }"
                                    @click="toggleLabel(item.key, val.value)"
                                >
                                    <span
                                        v-if="val.color"
                                        class="value-color"
                                        :style="{ backgroundColor: val.color }"
                                    />
                                    <div class="value-info">
                                        <span class="value-name">{{ val.value }}</span>
                                        <span v-if="val.description" class="value-desc">{{ val.description }}</span>
                                    </div>
                                    <div class="value-action">
                                        <el-icon v-if="isSelected(item.key, val.value)" class="check-icon">
                                            <Check />
                                        </el-icon>
                                        <span v-else class="add-hint">{{ $t('label.add') }}</span>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </template>

                    <div v-else class="empty-state">
                        <el-icon class="empty-icon"><Search /></el-icon>
                        <span>{{ $t('label.noLabels') }}</span>
                    </div>
                </div>

                <!-- 加载更多 -->
                <div v-if="hasMore" class="load-more">
                    <el-button text size="small" @click="loadMore">
                        {{ $t('label.loadMore', { count: pageSize }) }}
                    </el-button>
                </div>
            </div>
        </el-popover>
    </div>
</template>

<script lang="ts" setup>
import { ref, computed, onMounted, nextTick } from 'vue';
import { Plus, Check, Search } from '@element-plus/icons-vue';
import { labelApi } from '../api';
import type { LabelSelectValue, LabelAutocompleteItem, LabelValueDetail } from '../types';

interface Props {
    modelValue?: LabelSelectValue[];
}

const props = withDefaults(defineProps<Props>(), {
    modelValue: () => [],
});

const emit = defineEmits<{
    (e: 'update:modelValue', value: LabelSelectValue[]): void;
}>();

const popoverVisible = ref(false);
const searchText = ref('');
const autocompleteData = ref<LabelAutocompleteItem[]>([]);
const searchInputRef = ref();
const labelListRef = ref();
const displayLimit = ref(10); // 初始显示 10 个分组
const pageSize = 10;

// 加载自动补全数据
const loadAutocomplete = async () => {
    try {
        const data = await labelApi.autocomplete.request({ key: '' });
        autocompleteData.value = data ?? [];
    } catch {
        // 静默处理
    }
};

onMounted(() => {
    loadAutocomplete();
});

// 弹窗打开时聚焦搜索框
const onPopoverShow = () => {
    nextTick(() => {
        searchInputRef.value?.focus();
    });
};

// 过滤后的标签列表
const filteredLabels = computed(() => {
    if (!searchText.value) {
        return autocompleteData.value;
    }
    const search = searchText.value.toLowerCase();
    return autocompleteData.value.filter(
        (item) =>
            item.key.toLowerCase().includes(search) ||
            item.values.some((v) => v.toLowerCase().includes(search)) ||
            item.valueDetails?.some((d) => d.value.toLowerCase().includes(search) || d.description?.toLowerCase().includes(search)) ||
            item.description?.toLowerCase().includes(search)
    );
});

// 显示的标签列表（分页）
const displayedLabels = computed(() => {
    return filteredLabels.value.slice(0, displayLimit.value);
});

// 是否还有更多
const hasMore = computed(() => {
    return displayLimit.value < filteredLabels.value.length;
});

// 加载更多
const loadMore = () => {
    displayLimit.value += pageSize;
};

// 总数
const totalCount = computed(() => {
    return autocompleteData.value.reduce((sum, item) => sum + (item.valueDetails?.length || item.values.length), 0);
});

// 已选数量
const selectedCount = computed(() => {
    return props.modelValue?.length || 0;
});

// 获取过滤后的值
const getFilteredValues = (item: LabelAutocompleteItem): LabelValueDetail[] => {
    const details = item.valueDetails?.length ? item.valueDetails : item.values.map((v) => ({ value: v }));
    return details;
};

// 检查是否已选
const isSelected = (key: string, value: string): boolean => {
    return (props.modelValue || []).some((l) => l.key === key && l.value === value);
};

// 根据 key+value 查找颜色
const getLabelColor = (key: string, value?: string): string => {
    const item = autocompleteData.value.find((i) => i.key === key);
    if (!item) return '';
    const detail = item.valueDetails?.find((d) => d.value === value);
    return detail?.color || item.color || '';
};

// 获取标签样式
const getTagStyle = (label: LabelSelectValue): Record<string, string> => {
    const color = getLabelColor(label.key, label.value);
    if (!color) {
        // 标签不存在或无颜色定义，使用默认灰色
        return {
            backgroundColor: '#f5f7fa',
            borderColor: '#dcdfe6',
            color: '#909399',
        };
    }
    return {
        backgroundColor: color + '15',
        borderColor: color + '50',
        color: color,
    };
};

// 根据 key+value 查找描述
const getLabelDescription = (key: string, value?: string): string => {
    const item = autocompleteData.value.find((i) => i.key === key);
    if (!item) return '';
    const detail = item.valueDetails?.find((d) => d.value === value);
    return detail?.description || item.description || '';
};

// 切换标签选择
const toggleLabel = (key: string, value: string) => {
    const currentLabels = [...(props.modelValue || [])];
    const index = currentLabels.findIndex((l) => l.key === key);

    if (index >= 0) {
        // 同一 key 只能有一个 value，替换
        currentLabels[index] = { key, value };
    } else {
        currentLabels.push({ key, value });
    }

    emit('update:modelValue', currentLabels);
};

// 移除标签
const removeLabel = (index: number) => {
    const currentLabels = [...(props.modelValue || [])];
    currentLabels.splice(index, 1);
    emit('update:modelValue', currentLabels);
};
</script>

<style lang="scss" scoped>
.label-selector {
    display: flex;
    flex-direction: column;
    gap: 8px;
}

.selected-labels {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    min-height: 24px;
}

.label-tag {
    margin: 0;
    font-weight: 500;
    transition: all 0.2s ease-out;

    &:hover {
        box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
    }

    :deep(.el-tag__close) {
        margin-left: 4px;
        opacity: 0.7;

        &:hover {
            opacity: 1;
        }
    }
}

.add-btn {
    align-self: flex-start;
}

// Popover 样式
.label-popover {
    display: flex;
    flex-direction: column;
    max-height: 480px;
}

.search-box {
    padding: 12px;
    border-bottom: 1px solid var(--el-border-color-lighter);

    :deep(.el-input__wrapper) {
        box-shadow: none;
        background-color: var(--el-fill-color-light);
    }
}

.stats-bar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 8px 12px;
    background-color: var(--el-fill-color-lighter);
    font-size: 12px;
}

.stats-text {
    color: var(--el-text-color-secondary);
}

.stats-selected {
    color: var(--el-color-primary);
    font-weight: 500;
}

.label-list {
    overflow-y: auto;
    padding: 8px 0;
    flex: 1;
}

.label-group {
    margin-bottom: 12px;

    &:last-child {
        margin-bottom: 0;
    }
}

.group-header {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 12px 4px;
}

.group-key {
    font-size: 12px;
    font-weight: 600;
    color: var(--el-text-color-primary);
    text-transform: uppercase;
    letter-spacing: 0.5px;
}

.group-values {
    display: flex;
    flex-direction: column;
}

.label-item {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 8px 12px 8px 20px;
    cursor: pointer;
    transition: all 0.15s ease-out;
    border-left: 2px solid transparent;

    &:hover {
        background-color: var(--el-fill-color-light);
        border-left-color: var(--el-color-primary);
    }

    &.selected {
        background-color: var(--el-color-primary-light-9);
        border-left-color: var(--el-color-primary);

        .value-name {
            color: var(--el-color-primary);
            font-weight: 600;
        }
    }
}

.value-color {
    flex-shrink: 0;
    width: 10px;
    height: 10px;
    border-radius: 50%;
}

.value-info {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
}

.value-name {
    font-weight: 500;
    font-size: 13px;
}

.value-desc {
    color: var(--el-text-color-secondary);
    font-size: 11px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.value-action {
    flex-shrink: 0;
    width: 40px;
    text-align: right;
}

.check-icon {
    color: var(--el-color-primary);
    font-size: 14px;
}

.add-hint {
    color: var(--el-text-color-placeholder);
    font-size: 12px;
    opacity: 0;
    transition: opacity 0.15s ease-out;

    .label-item:hover & {
        opacity: 1;
    }
}

.empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
    padding: 32px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
}

.empty-icon {
    font-size: 24px;
    opacity: 0.5;
}

.load-more {
    padding: 8px;
    border-top: 1px solid var(--el-border-color-lighter);
    text-align: center;
}
</style>
