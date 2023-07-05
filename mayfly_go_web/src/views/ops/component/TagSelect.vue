<template>
    <div>
        <el-tree-select v-bind="$attrs" @check="changeTag" style="width: 100%" :data="tags" placeholder="请选择关联标签"
            :render-after-expand="true" :default-expanded-keys="[selectTags]" show-checkbox check-strictly node-key="id"
            :props="{
                value: 'id',
                label: 'codePath',
                children: 'children',
            }">
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
import { useAttrs, toRefs, reactive, onMounted } from 'vue';
import { tagApi } from '../tag/api';

const attrs = useAttrs()
//定义事件
const emit = defineEmits(['changeTag', 'update:tagPath'])

const state = reactive({
    tags: [],
    // 单选则为id，多选为id数组
    selectTags: null as any,
});

const {
    tags,
    selectTags,
} = toRefs(state)

onMounted(async () => {
    if (attrs.modelValue) {
        state.selectTags = attrs.modelValue;
    }
    state.tags = await tagApi.getTagTrees.request(null);
});

const changeTag = (tag: any, checkInfo: any) => {
    if (checkInfo.checkedNodes.length > 0) {
        emit('update:tagPath', tag.codePath);
        emit('changeTag', tag);
    } else {
        emit('update:tagPath', null);
    }
};
</script>
<style lang="scss">

</style>
