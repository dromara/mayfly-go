<template>
    <div class="db-session-panel h-full flex flex-col gap-1">
        <div class="flex items-center gap-2 flex-shrink-0">
            <el-button size="small" :loading="state.loading" icon="Refresh" @click="load()">{{ t('common.refresh') }}</el-button>
            <el-switch v-model="state.autoRefresh" size="small" :active-text="t('db.sessionAutoRefresh')" />
            <span class="text-[12px] text-gray-400">{{ t('db.sessionUpdatedAt', { time: state.updatedAt }) }}</span>
        </div>

        <!-- 不支持的方言：明确说明原因，而不是给一张看起来「没有会话」的空表 -->
        <el-alert v-if="state.unsupported" :title="t('db.sessionUnsupported')" type="info" :closable="false" show-icon class="flex-shrink-0" />

        <el-table v-loading="state.loading && !state.sessions.length" :data="state.sessions" size="small" height="100%" class="flex-1 min-h-0">
            <el-table-column prop="id" :label="t('db.sessionId')" width="90" />
            <el-table-column prop="user" :label="t('db.sessionUser')" width="130" show-overflow-tooltip />
            <el-table-column prop="db" :label="t('db.dbName')" width="140" show-overflow-tooltip />
            <el-table-column :label="t('db.sessionElapsed')" width="100">
                <template #default="{ row }">{{ formatElapsed(row.elapsedMs) }}</template>
            </el-table-column>
            <el-table-column :label="t('db.sessionLockWait')" width="90">
                <template #default="{ row }">
                    <el-tag v-if="row.lockWait" type="danger" size="small">{{ t('common.yes') }}</el-tag>
                    <span v-else>-</span>
                </template>
            </el-table-column>
            <el-table-column prop="state" :label="t('db.sessionState')" width="150" show-overflow-tooltip />
            <el-table-column prop="query" :label="t('db.sessionQuery')" min-width="260" show-overflow-tooltip />
            <el-table-column :label="t('common.operation')" width="90" fixed="right">
                <template #default="{ row }">
                    <el-button v-auth="'db:session:cancel'" type="warning" link size="small" :disabled="state.canceling" @click="cancelSession(row)">
                        {{ t('db.sessionCancel') }}
                    </el-button>
                </template>
            </el-table-column>
            <template #empty>
                <span class="text-[12px] text-gray-400">{{ state.unsupported ? t('db.sessionUnsupported') : t('db.sessionEmpty') }}</span>
            </template>
        </el-table>
    </div>
</template>

<script lang="ts" setup>
/**
 * 活动会话面板（库工作区的一个页签）。
 *
 * 存在理由：前端 abort 只断 HTTP，长查询在数据库侧仍在跑；没有会话视图就既看不到「谁在跑什么」，
 * 也无法停掉它。列表数据全部来自库侧系统表（由方言的 SessionInspector 给出），本组件不做任何推断。
 */
import { onBeforeUnmount, reactive, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useIntervalFn } from '@vueuse/core';
import { dbApi } from '../api';
import type { DbSessionInfo } from '../types';
import { Msg, useI18nConfirm } from '@/hooks/useI18n';
import { formatTime } from '@/common/utils/format';

const props = defineProps<{
    dbId: number;
    dbName: string;
}>();

const { t } = useI18n();

const state = reactive({
    sessions: [] as DbSessionInfo[],
    loading: false,
    // 失败原因是否为「该方言不支持会话内省」：为 true 时给出说明，而不是留一张看起来「没有会话」的空表
    unsupported: false,
    autoRefresh: true,
    updatedAt: '',
    // 已有取消流程在走（包括确认框未关）：不允许再开第二个，否则会叠出多个确认框而旧框已陈旧
    canceling: false,
});

// 自动刷新：面板卸载即停（onBeforeUnmount），不会在后台反复查库侧系统表。
// load 内有 loading 互斥，首轮加载未完成时轮询到的一轮会被直接跳过，不会并发打库
const { pause, resume } = useIntervalFn(() => void load(true), 3000, { immediate: true });

watch(
    () => [props.dbId, props.dbName],
    () => void load(),
    { immediate: true }
);

watch(
    () => state.autoRefresh,
    (on) => (on ? resume() : pause())
);

/**
 * 拉取会话列表
 *
 * @param silent 自动刷新时不铺 loading、不弹错误提示（否则每 3 秒闪一次）
 */
async function load(silent = false) {
    if (state.loading) {
        return;
    }
    state.loading = true;
    try {
        const sessions = await dbApi.dbSessions.request({ dbId: props.dbId, db: props.dbName });
        state.sessions = sessions ?? [];
        state.unsupported = false;
        state.updatedAt = new Date().toLocaleTimeString();
    } catch (e: unknown) {
        // 能力缺失与真实查询失败要分开呈现：前者是「这个库看不了」，后者必须让人看到错误本身
        state.unsupported = isUnsupportedErr(e);
        state.sessions = [];
        if (!silent && !state.unsupported) {
            Msg.error(e instanceof Error ? e.message : String(e));
        }
    } finally {
        state.loading = false;
    }
}

function isUnsupportedErr(e: unknown): boolean {
    return (e instanceof Error ? e.message : String(e)).toLowerCase().includes('unsupported');
}

/** 已执行时长：ms 交给既有的秒级人类可读格式化（8s / 1m20s），不在此重写单位换算 */
function formatElapsed(ms: number): string {
    return formatTime(Math.round((ms ?? 0) / 1000), 's').trim() || '-';
}

async function cancelSession(row: DbSessionInfo) {
    // 同时只允许一个取消确认框：列表在自动刷新，连着点两行会叠出多个确认框，
    // 而旧框里的会话可能已经结束，误点就是把错误的语句被中断
    if (state.canceling) {
        return;
    }
    state.canceling = true;
    try {
        // 取消是危险动作（会中断别人的业务语句）：确认框里必须看清是哪条语句
        if (!(await useI18nConfirm('db.sessionCancelConfirm', { id: row.id, query: truncate(row.query) }))) {
            return;
        }
        await dbApi.cancelDbSession.request({ id: props.dbId, db: props.dbName, sessionId: row.id });
        Msg.success('db.sessionCancelSuccess');
    } catch {
        // 失败文案由请求中心层 toast（服务端已把库侧拒绝原因包成人话），这里不叠第二条；
        // 不 catch 则成了未处理的 rejection，界面看起来就是点了没反应
    } finally {
        state.canceling = false;
        // 无论成败都重取一次列表：失败往往正因为该行已是陈旧数据
        void load(true);
    }
}

function truncate(text?: string): string {
    const value = (text ?? '').trim();
    return value.length > 80 ? `${value.slice(0, 80)}...` : value;
}

onBeforeUnmount(pause);
</script>

<style lang="scss" scoped>
.db-session-panel {
    padding: 4px;
}
</style>
