<template>
    <BaseTreeNode v-bind="$attrs">
        <template #prefix="{ data }">
            <el-popover placement="right-start" :title="$t('common.detail')" trigger="hover" :width="250">
                <template #reference>
                    <SvgIcon name="Document" />
                </template>
                <el-descriptions :column="1" size="small">
                    <el-descriptions-item :label="$t('common.name')">
                        {{ data.params.idxName }}
                    </el-descriptions-item>
                    <el-descriptions-item :label="$t('es.size')">
                        {{ data.params.size }}
                    </el-descriptions-item>
                    <el-descriptions-item :label="$t('es.docs')">
                        {{ data.params.idx['docs.count'] }}
                    </el-descriptions-item>
                </el-descriptions>
            </el-popover>
        </template>
    </BaseTreeNode>
</template>

<script lang="ts" setup>
import BaseTreeNode from '@/views/ops/resource/BaseTreeNode.vue';
</script>

<style lang="scss"></style>
