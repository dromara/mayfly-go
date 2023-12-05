<template>
    <div style="display: inline-flex; justify-content: center; align-items: center; cursor: pointer; vertical-align: middle">
        <el-popover :show-after="500" @show="getTags" placement="top-start" width="230" trigger="hover">
            <template #reference>
                <div>
                    <!-- <el-button type="primary" link size="small">标签</el-button> -->
                    <SvgIcon name="view" :size="16" color="var(--el-color-primary)" />
                </div>
            </template>

            <el-tag effect="plain" v-for="tag in tags" :key="tag" class="ml5" type="success" size="small">{{ tag.tagPath }}</el-tag>
        </el-popover>
    </div>
</template>

<script lang="ts" setup>
import { reactive, toRefs } from 'vue';
import { tagApi } from '../tag/api';
import SvgIcon from '@/components/svgIcon/index.vue';
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
    tags: [] as any,
});

const { tags } = toRefs(state);

const getTags = async () => {
    state.tags = await tagApi.getTagResources.request({
        resourceCode: props.resourceCode,
        resourceType: props.resourceType,
    });
};
</script>

<style lang="scss"></style>
