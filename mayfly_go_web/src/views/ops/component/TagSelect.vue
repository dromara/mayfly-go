<template>
    <div>
        <el-tree-select
            @check="changeTag"
            style="width: 100%"
            v-model="selectTags"
            :data="tags"
            :render-after-expand="true"
            :default-expanded-keys="[selectTags]"
            show-checkbox
            check-strictly
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

<script lang="ts">
import { toRefs, reactive, defineComponent, onMounted } from 'vue';
import { tagApi } from '../tag/api';

export default defineComponent({
    name: 'TagSelect',
    props: {
        tagId: {
            type: Number,
        },
        tagPath: {
            type: String,
        },
    },
    setup(props: any, { emit }) {
        const state = reactive({
            tags: [],
            // 单选则为id，多选为id数组
            selectTags: null as any,
        });

        onMounted(async () => {
            if (props.tagId) {
                state.selectTags = props.tagId;
            }
            state.tags = await tagApi.getTagTrees.request(null);
        });

        const changeTag = (tag: any, checkInfo: any) => {
            if (checkInfo.checkedNodes.length > 0) {
                emit('update:tagId', tag.id);
                emit('update:tagPath', tag.codePath);
                emit('changeTag', tag);
            } else {
                emit('update:tagId', null);
                emit('update:tagPath', null);
            }
        };

        return {
            ...toRefs(state),
            changeTag,
        };
    },
});
</script>
<style lang="scss">
</style>
