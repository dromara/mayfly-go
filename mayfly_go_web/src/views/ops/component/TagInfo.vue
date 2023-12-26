<template>
    <div style="display: inline-flex; justify-content: center; align-items: center; cursor: pointer; vertical-align: middle">
        <el-popover :show-after="500" @show="showTagInfo" placement="top-start" title="标签信息" :width="300" trigger="hover">
            <template #reference>
                <el-icon>
                    <InfoFilled />
                </el-icon>
            </template>
            <span v-for="(v, i) in tags" :key="i">
                <el-tooltip :content="v.remark" placement="top">
                    <span class="color-success">{{ v.name }}</span>
                </el-tooltip>
                <span v-if="i != state.tags.length - 1" class="color-primary"> / </span>
            </span>
        </el-popover>
    </div>
</template>

<script lang="ts" setup>
import { reactive, toRefs, onMounted } from 'vue';
import { tagApi } from '../tag/api';
const props = defineProps({
    tagPath: {
        type: [String],
        required: true,
    },
});

const state = reactive({
    tagPath: '',
    tags: [] as any,
});

const { tags } = toRefs(state);

onMounted(async () => {
    state.tagPath = props.tagPath;
});

const showTagInfo = async () => {
    if (state.tags && state.tags.length > 0) {
        return;
    }
    const tagStrs = state.tagPath.split('/');
    const tagPaths = [];
    let nowTag = '';
    for (let tagStr of tagStrs) {
        if (!tagStr) {
            continue;
        }
        if (nowTag) {
            nowTag = `${nowTag}${tagStr}/`;
        } else {
            nowTag = tagStr + '/';
        }
        tagPaths.push(nowTag);
    }
    state.tags = await tagApi.listByQuery.request({ tagPaths: tagPaths.join(',') });
};
</script>

<style lang="scss"></style>
