<template>
    <el-tree-select
        v-bind="$attrs"
        v-model="state.selectTags"
        @change="changeTag"
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
                <SvgIcon :name="EnumValue.getEnumByValue(TagResourceTypeEnum, data.type)?.extra.icon" class="mr2" />
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

//定义事件
const emit = defineEmits(['update:modelValue', 'changeTag', 'input']);

const props = defineProps({
    selectTags: {
        type: [Array<any>, Object],
    },
    tagType: {
        type: Number,
        default: TagResourceTypeEnum.Tag.value,
    },
});

const state = reactive({
    tags: [],
    // 单选则为codePath，多选为codePath数组
    selectTags: [] as any,
});

const { tags } = toRefs(state);

const defaultExpandedKeys = computed(() => {
    if (Array.isArray(state.selectTags)) {
        // 如果 state.selectTags 是数组，直接返回
        return state.selectTags;
    }

    // 如果 state.selectTags 不是数组，转换为包含 state.selectTags 的数组
    return [state.selectTags];
});

onMounted(async () => {
    state.selectTags = props.selectTags;
    state.tags = await tagApi.getTagTrees.request({ type: props.tagType });
});

const changeTag = () => {
    emit('changeTag', state.selectTags);
};
</script>
<style lang="scss"></style>
