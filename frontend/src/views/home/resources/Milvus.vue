<template>
    <Base
        :resource-label="$t('home.myMilvus')"
        :resource-icon="ResourceTypeEnum.Milvus.extra.icon"
        :resource-color="ResourceTypeEnum.Milvus.extra.iconColor"
        :api-method="milvusApi.list"
    >
        <!-- 自定义列表项 -->
        <template #item="{ resource }">
            <div
                class="flex flex-col gap-2 p-2.5 border border-(--el-border-color-lighter) rounded-md bg-(--el-bg-color) cursor-pointer transition-all duration-200 min-w-0 max-w-full box-border hover:border-[var(--el-color-primary-light-7)] hover:bg-[var(--el-fill-color-light)] hover:shadow-[0_2px_8px_rgba(0,0,0,0.08)]"
            >
                <div class="flex items-center justify-between gap-2">
                    <div class="flex items-center gap-1.5 font-medium text-[13px] text-(--el-text-color-primary) flex-1 min-w-0">
                        <SvgIcon :name="ResourceTypeEnum.Milvus.extra.icon" :size="16" />
                        <span class="overflow-hidden text-ellipsis whitespace-nowrap">{{ resource.name || resource.code }}</span>
                    </div>
                </div>
                <div class="flex flex-col gap-1.5 text-[12px] text-(--el-text-color-secondary)">
                    <div class="font-['Courier_New',monospace]">{{ resource.host }}</div>
                    <div v-if="resource.remark" class="overflow-hidden text-ellipsis whitespace-nowrap text-[11px]">{{ resource.remark }}</div>
                </div>
            </div>
        </template>
    </Base>
</template>

<script lang="ts" setup>
import { ResourceTypeEnum } from '@/common/commonEnum';
import SvgIcon from '@/components/svg-icon/index.vue';
import { milvusApi } from '@/views/ops/milvus/api';
import Base from './Base.vue';
</script>
