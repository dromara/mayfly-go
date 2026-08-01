<template>
    <el-tree-select
        v-bind="$attrs"
        v-model="modelValue"
        :data="tags"
        :placeholder="$t('tag.selectTagPlaceholder')"
        :default-expanded-keys="defaultExpandedKeys"
        show-checkbox
        node-key="codePath"
        :props="{
            value: 'codePath',
            label: 'name',
            children: 'children',
        }"
    >
        <template #default="{ data }">
            <span class="custom-tree-node">
                <SvgIcon :name="EnumValue.getEnumByValue(TagResourceTypeEnum, data.type)?.extra.icon" class="mr-0.5" />
                <span style="font-size: 13px">
                    {{ data.name }}
                    <el-tag v-if="data.children !== null" size="small">{{ data.children.length }}</el-tag>
                </span>
            </span>
        </template>
    </el-tree-select>
</template>

<script lang="ts" setup>
import { TagResourceTypeEnum } from '@/common/commonEnum';
import EnumValue from '@/common/Enum';
import { computed, onMounted, reactive, toRefs, watch } from 'vue';
import { tagApi } from '../tag/api';
import type { TagTree } from '../tag/types';

const props = defineProps({
    tagType: {
        type: [Number, String],
        default: TagResourceTypeEnum.Tag.value,
    },
    // 资源编号
    code: {
        type: String,
        default: '',
    },
});

const modelValue = defineModel<string[] | string>('modelValue');

const state = reactive({
    tags: [] as TagTree[],
});

const { tags } = toRefs(state);

const defaultExpandedKeys = computed(() => {
    if (Array.isArray(modelValue.value)) {
        // 如果 modelValue 是数组，直接返回
        return modelValue.value;
    }

    // 如果 modelValue 不是数组，转换为包含 state.selectTags 的数组
    return [modelValue.value];
});

// 加载标签路径
const loadTagPaths = async () => {
    if (!props.code) {
        modelValue.value = [];
        return;
    }

    try {
        const res = await tagApi.listResourceTags.request({ resourceCode: props.code });
        // 去重
        const uniquePaths = [...new Set(res.map((t) => t.codePath))];
        modelValue.value = uniquePaths || [];
    } catch (error) {
        console.error('Failed to load tag paths:', error);
        modelValue.value = [];
    }
};

// 监听 code 变化
watch(
    () => props.code,
    () => {
        loadTagPaths();
    },
    { immediate: true }
);

onMounted(async () => {
    state.tags = await tagApi.getTagTrees.request({ type: props.tagType });
});
</script>
<style lang="scss"></style>
