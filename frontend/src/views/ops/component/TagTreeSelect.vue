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
            label: 'codePath',
            children: 'children',
        }"
    >
        <template #default="{ data }">
            <span class="custom-tree-node">
                <SvgIcon :name="EnumValue.getEnumByValue(TagResourceTypeEnum, data.type)?.extra.icon" class="mr-0.5" />
                <span style="font-size: 13px">
                    {{ data.code }}
                    <span style="color: #3c8dbc">【</span>
                    {{ data.name }}
                    <span style="color: #3c8dbc">】</span>
                    <el-tag v-if="data.children !== null" size="small">{{ data.children.length }}</el-tag>
                </span>
            </span>
        </template>
    </el-tree-select>
</template>

<script lang="ts" setup>
import { toRefs, reactive, onMounted, computed } from 'vue';
import { tagApi } from '../tag/api';
import { TagResourceTypeEnum } from '@/common/commonEnum';
import EnumValue from '@/common/Enum';

const props = defineProps({
    tagType: {
        type: Number,
        default: TagResourceTypeEnum.Tag.value,
    },
});

const modelValue = defineModel<Array<any> | Object>('modelValue');

const state = reactive({
    tags: [],
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

onMounted(async () => {
    state.tags = await tagApi.getTagTrees.request({ type: props.tagType });
});
</script>
<style lang="scss"></style>
