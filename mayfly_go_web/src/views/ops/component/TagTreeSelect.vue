<template>
    <div>
        <el-tree-select
            v-bind="$attrs"
            v-model="selectTags"
            @change="changeTag"
            style="width: 100%"
            :data="tags"
            placeholder="请选择关联标签"
            :render-after-expand="true"
            :default-expanded-keys="[selectTags]"
            show-checkbox
            node-key="id"
            :props="{
                value: 'id',
                label: 'codePath',
                children: 'children',
            }"
        >
            <template #default="{ data }">
                <span class="custom-tree-node">
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
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, onMounted } from 'vue';
import { tagApi } from '../tag/api';

//定义事件
const emit = defineEmits(['update:modelValue', 'changeTag', 'input']);

const props = defineProps({
    resourceCode: {
        type: [String],
        required: true,
    },
    resourceType: {
        type: [Number],
        required: true,
    },
});

const state = reactive({
    tags: [],
    // 单选则为id，多选为id数组
    selectTags: [],
});

const { tags, selectTags } = toRefs(state);

onMounted(async () => {
    if (props.resourceCode) {
        const resourceTags = await tagApi.getTagResources.request({
            resourceCode: props.resourceCode,
            resourceType: props.resourceType,
        });
        state.selectTags = resourceTags.map((x: any) => x.tagId);
        changeTag();
    }

    state.tags = await tagApi.getTagTrees.request(null);
});

const changeTag = () => {
    emit('changeTag', state.selectTags);
};
</script>
<style lang="scss"></style>
