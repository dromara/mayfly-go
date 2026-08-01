<template>
    <Base
        :resource-label="$t('home.myMachines')"
        :resource-icon="ResourceTypeEnum.Machine.extra.icon"
        :resource-color="ResourceTypeEnum.Machine.extra.iconColor"
        :api-method="machineApi.list"
        ref="baseRef"
    >
        <!-- 自定义列表项 -->
        <template #item="{ resource }">
            <div class="flex flex-col gap-2 p-2.5 border border-[var(--el-border-color-lighter)] rounded-md bg-[var(--el-bg-color)] cursor-pointer transition-all duration-200 min-w-0 max-w-full box-border hover:border-[var(--el-color-primary-light-7)] hover:bg-[var(--el-fill-color-light)] hover:shadow-[0_2px_8px_rgba(0,0,0,0.08)]">
                <div class="flex items-center justify-between gap-2">
                    <div class="flex items-center gap-1.5 font-medium text-[13px] text-[var(--el-text-color-primary)] flex-1 min-w-0">
                        <SvgIcon :name="ResourceTypeEnum.Machine.extra.icon" :size="16" />
                        <span class="overflow-hidden text-ellipsis whitespace-nowrap">{{ resource.name || resource.code }}</span>
                    </div>
                    <el-tag v-if="resource.stat" size="small" type="success">{{ $t('common.online') }}</el-tag>
                    <el-tag v-else size="small" type="danger">{{ $t('common.offline') }}</el-tag>
                </div>
                <div class="flex flex-col gap-1.5 text-[12px] text-[var(--el-text-color-secondary)]">
                    <div class="font-['Courier_New',monospace]">{{ resource.ip }}:{{ resource.port }}</div>
                    <div v-if="resource.stat" class="flex flex-col gap-1">
                        <div class="flex gap-2">
                            <span class="text-[11px] font-medium" :class="getCpuUsageClass(100 - resource.stat.cpuIdle) === 'stat-danger' ? 'text-[var(--el-color-danger)]' : getCpuUsageClass(100 - resource.stat.cpuIdle) === 'stat-warning' ? 'text-[var(--el-color-warning)]' : 'text-[var(--el-color-success)]'">
                                CPU: {{ (100 - resource.stat.cpuIdle).toFixed(0) }}%
                            </span>
                        </div>
                        <div class="flex gap-2">
                            <span
                                class="text-[11px] font-medium"
                                :class="getMemUsageClass(resource.stat.memTotal - resource.stat.memAvailable, resource.stat.memTotal) === 'stat-danger' ? 'text-[var(--el-color-danger)]' : getMemUsageClass(resource.stat.memTotal - resource.stat.memAvailable, resource.stat.memTotal) === 'stat-warning' ? 'text-[var(--el-color-warning)]' : 'text-[var(--el-color-success)]'"
                            >
                                MEM: {{ formatByteSize(resource.stat.memTotal - resource.stat.memAvailable) }} /
                                {{ formatByteSize(resource.stat.memTotal) }} ({{
                                    (((resource.stat.memTotal - resource.stat.memAvailable) / resource.stat.memTotal) * 100).toFixed(0)
                                }}%)
                            </span>
                        </div>
                        <div v-if="resource.stat.fsInfos && resource.stat.fsInfos.length > 0" class="flex gap-2">
                            <span class="text-[11px] font-medium" :class="getDiskUsageClass(resource.stat.fsInfos) === 'stat-danger' ? 'text-[var(--el-color-danger)]' : getDiskUsageClass(resource.stat.fsInfos) === 'stat-warning' ? 'text-[var(--el-color-warning)]' : 'text-[var(--el-color-success)]'">
                                DISK: {{ getDiskUsed(resource.stat.fsInfos) }} / {{ getDiskTotal(resource.stat.fsInfos) }} ({{
                                    getDiskUsage(resource.stat.fsInfos)
                                }})
                            </span>
                        </div>
                        <div v-else-if="resource.stat" class="flex gap-2">
                            <span class="text-[11px] text-[var(--el-text-color-secondary)]"> DISK: N/A </span>
                        </div>
                    </div>
                    <div v-else class="text-[11px] text-[var(--el-text-color-secondary)]">{{ $t('common.offline') }}</div>
                </div>
            </div>
        </template>
    </Base>
</template>

<script lang="ts" setup>
import { ResourceTypeEnum } from '@/common/commonEnum';
import SvgIcon from '@/components/svg-icon/index.vue';
import { machineApi } from '@/views/ops/machine/api';
import Base from './Base.vue';

interface FsInfo {
    mountPoint?: string;
    used: number;
    free: number;
}

const formatByteSize = (bytes: number): string => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return (bytes / Math.pow(k, i)).toFixed(2) + ' ' + sizes[i];
};

// 计算磁盘使用率
const getDiskUsage = (fSInfos: FsInfo[]) => {
    if (!fSInfos || fSInfos.length === 0) return '0%';

    let totalUsed = 0;
    let totalFree = 0;

    fSInfos.forEach((fs: FsInfo) => {
        totalUsed += fs.used || 0;
        totalFree += fs.free || 0;
    });

    const total = totalUsed + totalFree;
    if (total === 0) return '0%';

    const usage = (totalUsed / total) * 100;
    return usage.toFixed(0) + '%';
};

// 获取CPU使用率颜色类
const getCpuUsageClass = (cpuUsage: number) => {
    if (cpuUsage > 90) return 'stat-danger';
    if (cpuUsage > 70) return 'stat-warning';
    return 'stat-success';
};

// 获取内存使用率颜色类
const getMemUsageClass = (memUsed: number, memTotal: number) => {
    if (memTotal === 0) return 'stat-success';
    const usage = (memUsed / memTotal) * 100;
    if (usage > 90) return 'stat-danger';
    if (usage > 70) return 'stat-warning';
    return 'stat-success';
};

// 获取磁盘使用率颜色类
const getDiskUsageClass = (fSInfos: FsInfo[]) => {
    if (!fSInfos || fSInfos.length === 0) return 'stat-success';

    let totalUsed = 0;
    let totalFree = 0;

    fSInfos.forEach((fs: FsInfo) => {
        totalUsed += fs.used || 0;
        totalFree += fs.free || 0;
    });

    const total = totalUsed + totalFree;
    if (total === 0) return 'stat-success';

    const usage = (totalUsed / total) * 100;
    if (usage > 90) return 'stat-danger';
    if (usage > 70) return 'stat-warning';
    return 'stat-success';
};

// 计算磁盘已用空间
const getDiskUsed = (fSInfos: FsInfo[]) => {
    if (!fSInfos || fSInfos.length === 0) return '0 B';

    let totalUsed = 0;
    fSInfos.forEach((fs: FsInfo) => {
        totalUsed += fs.used || 0;
    });

    return formatByteSize(totalUsed);
};

// 计算磁盘总空间
const getDiskTotal = (fSInfos: FsInfo[]) => {
    if (!fSInfos || fSInfos.length === 0) return '0 B';

    let totalUsed = 0;
    let totalFree = 0;

    fSInfos.forEach((fs: FsInfo) => {
        totalUsed += fs.used || 0;
        totalFree += fs.free || 0;
    });

    return formatByteSize(totalUsed + totalFree);
};
</script>
