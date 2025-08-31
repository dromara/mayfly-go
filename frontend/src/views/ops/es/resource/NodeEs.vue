<template>
    <BaseTreeNode v-bind="$attrs">
        <template #prefix="{ data }">
            <el-popover :show-after="500" placement="right-start" :title="$t('common.detail')" trigger="hover" :width="250">
                <template #reference>
                    <SvgIcon name="icon es/es-color" />
                </template>
                <template #default>
                    <el-descriptions :column="1" size="small">
                        <el-descriptions-item :label="$t('common.name')">
                            {{ data.params.name }}
                        </el-descriptions-item>
                        <el-descriptions-item label="version">
                            {{ data.params.version }}
                        </el-descriptions-item>
                        <el-descriptions-item label="uri">
                            {{ `${data.params.host}:${data.params.port}` }}
                        </el-descriptions-item>
                    </el-descriptions>
                </template>
            </el-popover>
        </template>
    </BaseTreeNode>
</template>

<script lang="ts" setup>
import BaseTreeNode from '@/views/ops/resource/BaseTreeNode.vue';
</script>

<style lang="scss"></style>
