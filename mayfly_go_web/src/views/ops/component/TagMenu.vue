<template>
    <div class="instances-box layout-aside">
        <el-row type="flex" justify="space-between">
            <el-col :span="24"
                :style="{ maxHeight: instanceMenuMaxHeight, height: instanceMenuMaxHeight, overflow: 'auto' }"
                class="el-scrollbar flex-auto">
                <el-menu background-color="transparent" :collapse-transition="false" ref="menuRef">
                    <!-- 第一级：tag -->
                    <el-sub-menu v-for="tag of tags" :index="tag.tagPath" :key="tag.tagPath"
                        @click.stop="clickTag(tag.tagPath)">
                        <template #title>
                            <tag-info :tag-path="tag.tagPath" />

                            <el-icon>
                                <FolderOpened v-if="opend[tag.tagPath]" color="#e6a23c" />
                                <Folder v-else />
                            </el-icon>
                            <span>{{ tag.tagPath }}</span>
                        </template>
                        <slot :tag="tag" name="submenu"></slot>
                    </el-sub-menu>
                </el-menu>
            </el-col>
        </el-row>
    </div>
</template>

<script lang="ts" setup>
import { reactive, ref, Ref, toRefs } from 'vue';
import TagInfo from './TagInfo.vue';

const props = defineProps({
    instanceMenuMaxHeight: {
        type: [Number, String],
    },
    tags: {
        type: Object, required: true
    },
})

const menuRef = ref(null) as Ref

const state = reactive({
    instanceMenuMaxHeight: props.instanceMenuMaxHeight,
    tags: props.tags,
    opend: {},
})

const {
    opend,
} = toRefs(state)

const clickTag = (tagPath: string) => {
    if (state.opend[tagPath] === undefined) {
        state.opend[tagPath] = true;
        return;
    }
    const opend = state.opend[tagPath]
    state.opend[tagPath] = !opend
}

const open = (index: string, isTag: boolean = false) => {
    if (!index) {
        return;
    }
    menuRef.value.open(index)
    if (isTag) {
        clickTag(index)
    }
}

defineExpose({
    open
})

</script>

<style lang="scss">
.instances-box {
    .el-menu {
        width: 100%;
    }

    .el-sub-menu {
        .checked {
            .checked-schema {
                color: var(--el-color-primary);
            }
        }
    }

    .el-sub-menu__title {
        padding-left: 0 !important;
        height: 30px !important;
        line-height: 30px !important;
    }

    .el-menu--vertical:not(.el-menu--collapse):not(.el-menu--popup-container) .el-sub-menu__title {
        padding-right: 10px;
    }

    .el-menu-item {
        padding-left: 0 !important;
        height: 20px !important;
        line-height: 20px !important;
    }

    .el-icon {
        margin: 0;
    }

    .el-sub-menu__icon-arrow {
        top: inherit;
        right: 10px;
    }

}

.instances-pop-form {
    .el-form-item {
        margin-bottom: unset;
    }
}
</style>