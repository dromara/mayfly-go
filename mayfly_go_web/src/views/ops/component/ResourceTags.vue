<template>
    <div v-if="props.tags">
        <el-row v-for="(tagPath, idx) in tagPaths?.slice(0, 1)" :key="idx">
            <TagInfo :tag-path="tagPath" />
            <span class="ml3">{{ tagPath }}</span>

            <!-- 展示剩余的标签信息 -->
            <el-popover :show-after="300" v-if="tagPaths?.length > 1 && idx == 0" placement="top-start" width="230" trigger="hover">
                <template #reference>
                    <SvgIcon class="mt5 ml5" color="var(--el-color-primary)" name="MoreFilled" />
                </template>

                <el-row v-for="i in tagPaths.slice(1)" :key="i">
                    <TagInfo :tag-path="i" />
                    <span class="ml3">{{ i }}</span>
                </el-row>
            </el-popover>
        </el-row>
    </div>
</template>

<script lang="ts" setup>
import SvgIcon from '@/components/svgIcon/index.vue';
import TagInfo from './TagInfo.vue';
import { computed } from 'vue';
import { getTagPath } from './tag';
const props = defineProps({
    tags: {
        type: [Array<any>],
        required: true,
    },
});

const tagPaths = computed(() => {
    return props.tags?.map((item) => getTagPath(item.codePath)) as any;
});
</script>

<style lang="scss"></style>
