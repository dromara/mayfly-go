<template>
    <div class="h-full">
        <el-row :gutter="12">
            <!-- 急停开关 + 一键清场 -->
            <el-col :span="8">
                <el-card shadow="never" class="h-full">
                    <template #header>{{ $t('agentops.governanceKillSwitch') }}</template>
                    <div class="agentops-gov-switch">
                        <el-switch :model-value="killSwitch" :loading="switching" @change="onKillSwitchChange" />
                        <el-tag :type="killSwitch ? 'danger' : 'success'" effect="dark">
                            {{ killSwitch ? $t('agentops.governanceKillSwitchOn') : $t('agentops.governanceKillSwitchOff') }}
                        </el-tag>
                    </div>
                    <div class="agentops-gov-tip">{{ $t('agentops.governanceKillSwitchTip') }}</div>
                    <el-button
                        v-auth="perms.kill"
                        type="danger"
                        plain
                        icon="CircleClose"
                        class="agentops-gov-cancel"
                        :disabled="!killSwitch"
                        @click="cancelAll"
                    >
                        {{ $t('agentops.governanceCancelAll') }}
                    </el-button>
                </el-card>
            </el-col>

            <!-- 运行 KPI -->
            <el-col :span="16">
                <el-card shadow="never" class="h-full">
                    <template #header>
                        <span>{{ $t('agentops.governanceRunTotal') }}</span>
                        <!-- 窗口天数读 stats.windowDays（服务端实际生效口径）而不是手抄请求入参：
                             后端兜底/改默认值时手抄那份会静默说谎，标题里的范围就不再是统计用的范围 -->
                        <span v-if="stats" class="agentops-gov-tip">{{ $t('agentops.governanceRunWindowTip', { days: stats.windowDays }) }}</span>
                    </template>
                    <el-descriptions v-if="stats" :column="3" border label-width="200">
                        <el-descriptions-item :label="$t('agentops.governanceRunTotal')">{{ stats.runTotal }}</el-descriptions-item>
                        <el-descriptions-item :label="$t('agentops.governanceSuccessRate')">{{ successRateText }}</el-descriptions-item>
                        <el-descriptions-item :label="$t('agentops.governanceActiveRuns')">{{ stats.activeRuns }}</el-descriptions-item>
                        <el-descriptions-item :label="$t('agentops.governanceAvgCost')">{{ formatDurationMs(stats.avgCostMs) }}</el-descriptions-item>
                        <el-descriptions-item :label="$t('agentops.governanceTotalTokens')">{{ formatTokenCount(stats.totalTokens, true) }}</el-descriptions-item>
                        <el-descriptions-item :label="$t('agentops.governancePendingProposals')">
                            <el-tag v-if="stats.pendingProposals > 0" type="warning" effect="plain">{{ stats.pendingProposals }}</el-tag>
                            <span v-else>0</span>
                        </el-descriptions-item>
                        <!-- 停滞提案：审批已领取但受控执行超过静默期仍未落库（执行前进程退出丢单）。
                             后端停滞清扫会自动重投，这里显示的是尚未被收走的瞬时值；长期不归零才是异常。
                             静默期分钟数取自 stats.stallAfterMinutes（服务端实际生效口径）而不是写死在文案里。
                             用 warning 而不是 danger：这一态系统自己会收、无需人工介入，打红会让人以为要立刻动手 -->
                        <el-descriptions-item>
                            <template #label>
                                <span>{{ $t('agentops.governanceStuckApproved') }}</span>
                                <el-tooltip placement="top">
                                    <template #content
                                        ><span style="white-space: pre-line">{{
                                            $t('agentops.governanceStuckApprovedTips', { minutes: stats.stallAfterMinutes })
                                        }}</span></template
                                    >
                                    <SvgIcon name="QuestionFilled" class="ml-1" />
                                </el-tooltip>
                            </template>
                            <el-tag v-if="stats.stuckApproved > 0" type="warning" effect="plain">{{ stats.stuckApproved }}</el-tag>
                            <span v-else>0</span>
                        </el-descriptions-item>
                    </el-descriptions>
                </el-card>
            </el-col>
        </el-row>

        <el-row :gutter="12" class="agentops-gov-row2">
            <!-- 运行状态分布 -->
            <el-col :span="12">
                <el-card shadow="never" class="h-full">
                    <template #header>{{ $t('agentops.governanceRunByStatus') }}</template>
                    <div class="agentops-gov-tags">
                        <el-tag v-for="item in statusEntries" :key="item.status" :type="enumTagType(RunStatusEnum, item.status)" effect="plain" class="agentops-gov-tag">
                            {{ enumLabel(RunStatusEnum, item.status, t) }}: {{ item.count }}
                        </el-tag>
                    </div>
                </el-card>
            </el-col>

            <!-- 资源盘点 -->
            <el-col :span="12">
                <el-card shadow="never" class="h-full">
                    <template #header>{{ $t('agentops.governanceResourceInventory') }}</template>
                    <el-descriptions v-if="stats" :column="1" border label-width="150">
                        <el-descriptions-item :label="$t('agentops.governanceEnabledTasks')">{{ stats.enabledTasks }}</el-descriptions-item>
                        <el-descriptions-item :label="$t('agentops.governanceEnabledTriggers')">{{ stats.enabledTriggers }}</el-descriptions-item>
                        <el-descriptions-item :label="$t('agentops.governancePendingProposals')">{{ stats.pendingProposals }}</el-descriptions-item>
                    </el-descriptions>
                </el-card>
            </el-col>
        </el-row>
    </div>
</template>

<script lang="ts" setup>
import { Msg, useI18nConfirm } from '@/hooks/useI18n';
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { agentGovernanceApi } from './api';
import { RunStatusEnum } from './enums';
import { enumLabel, enumTagType } from '@/common/Enum';
import { formatDurationMs, formatTokenCount } from '@/common/utils/format';
import { AgentopsPerms as P } from './perms';
import type { GovernanceStats } from './types';

const { t } = useI18n();

// 权限码取自集中声明（与后端路由、迁移资源三方同源）
const perms = {
    kill: P.governanceKillSwitch,
};

const stats = ref<GovernanceStats | null>(null);
const killSwitch = ref(false);
const switching = ref(false);

const successRateText = computed(() => `${Math.round((stats.value?.successRate ?? 0) * 1000) / 10}%`);

const statusEntries = computed(() => Object.entries(stats.value?.runByStatus ?? {}).map(([status, count]) => ({ status, count })));

// 状态标签的文案与颜色均以 RunStatusEnum 上登记的元数据为真源（enumLabel / enumTagType）：
// 此前这里另写了一份 switch，新增运行状态时两处必然分叉，而分叉只表现为「颜色不对」，没有任何报错

// 统计窗口请求入参（前端主动选择的窗口；展示口径一律用响应里的 stats.windowDays，
// 那才是服务端实际生效并据此算出数字的那个值）
const STATS_WINDOW_DAYS = 30;

const loadStats = async () => {
    stats.value = await agentGovernanceApi.stats.request({ days: STATS_WINDOW_DAYS });
    killSwitch.value = stats.value.killSwitch;
};

onMounted(loadStats);

/** 切换急停开关：确认后写库并重载（受控 model-value，取消时自动回弹真实状态） */
const onKillSwitchChange = async (val: string | number | boolean) => {
    if (!(await useI18nConfirm('agentops.governanceKillSwitchConfirm'))) {
        return;
    }
    try {
        switching.value = true;
        await agentGovernanceApi.setKillSwitch.request({ engaged: Boolean(val) });
        Msg.success('agentops.governanceKillSwitchChanged');
    } finally {
        switching.value = false;
        await loadStats();
    }
};

/**
 * 一键取消全部进行中运行（急停清场附带动作，后端逐个走条件更新取消）。
 *
 * 提示必须用后端回报的实际取消数：后端专门把「已终态不可取消」排除在计数外
 * （见 run_service.ErrRunNotCancelable），丢弃返回值会让提示重新夸大清场结果
 */
const cancelAll = async () => {
    if (!(await useI18nConfirm('agentops.governanceCancelAllConfirm'))) {
        return;
    }
    const { canceled } = await agentGovernanceApi.cancelAll.request();
    Msg.success('agentops.governanceCancelAllDone', { canceled });
    await loadStats();
};
</script>

<style lang="scss" scoped>
.agentops-gov-switch {
    display: flex;
    align-items: center;
    gap: 12px;
}

.agentops-gov-tip {
    margin-top: 8px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    line-height: 1.6;
}

.agentops-gov-cancel {
    margin-top: 12px;
}

.agentops-gov-row2 {
    margin-top: 12px;
}

.agentops-gov-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
}
</style>
