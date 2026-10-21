<template>
    <div class="home-page overflow-x-hidden">
        <HomeHero
            :name="userInfo.name || userInfo.username || ''"
            :username="userInfo.username || ''"
            :photo="userInfo.photo"
            :role-text="rolesText"
            :last-login-ip="userInfo.lastLoginIp"
            :last-login-time="userInfo.lastLoginTime"
            :refreshing="refreshing"
            @refresh="refreshAll"
            @navigate-personal="navigateToPersonal"
        />

        <div class="home-stats">
            <StatTile
                icon="Tickets"
                tone="warning"
                :clickable="canOpenTasks"
                :label="$t('home.pendingTasks')"
                :hint="$t('home.pendingTasksHint')"
                :value="state.pendingTaskTotal"
                :loading="state.loading.tasks"
                :failed="state.failed.tasks"
                @select="go('ProcinstTaskList')"
            />
            <StatTile
                icon="Bell"
                tone="danger"
                :clickable="canOpenAlerts"
                :label="$t('home.activeAlerts')"
                :hint="$t('home.activeAlertsHint')"
                :value="state.firingTotal"
                :loading="state.loading.alerts"
                :failed="state.failed.alerts"
                @select="go('AlertEventList')"
            />
            <StatTile
                icon="Message"
                tone="primary"
                :label="$t('home.unreadMessages')"
                :hint="$t('home.unreadMessagesHint')"
                :value="state.unreadCount"
                :loading="state.loading.messages"
                :failed="state.failed.messages"
            />
            <StatTile
                icon="Menu"
                tone="success"
                :clickable="canOpenResources"
                :label="$t('home.resourceTotal')"
                :hint="$t('home.resourceTotalHint')"
                :value="resourceTotal"
                :loading="resourceTotal === null"
                @select="go('ResourceOp')"
            />
        </div>

        <div class="home-grid">
            <SectionCard :title="$t('home.resourceOverview')" icon="Coin" :count="resourceTotal">
                <template #actions>
                    <span class="home-panel-hint">{{ $t('home.resourceSubtitle') }}</span>
                </template>
                <div class="resource-grid">
                    <!-- 资源卡外层组件未声明 loaded：监听器经 attrs 透传到根 Base.vue，链路由 __tests__/resourceTotal.test.ts 钉住 -->
                    <component
                        :is="comp"
                        v-for="(comp, index) of resourceComponents"
                        :key="index"
                        :ref="(el) => setResourceRef(el, index)"
                        @loaded="(total: number) => onResourceLoaded(index, total)"
                    />
                </div>
            </SectionCard>

            <SectionCard :title="$t('home.quickActions')" icon="Position">
                <template #actions>
                    <span class="home-panel-hint">{{ $t('home.quickActionsDesc') }}</span>
                </template>
                <QuickActions @navigate="navigateToAction" />
            </SectionCard>
        </div>

        <!-- 待办与告警等高：面板高度足以容纳 5 条待办，不会因为当前只有 1 条就被压成一条的高度 -->
        <div class="home-split">
            <TodoPanel
                :tasks="state.pendingTasks"
                :total="state.pendingTaskTotal"
                :loading="state.loading.tasks"
                :failed="state.failed.tasks"
                :viewable="canOpenTasks"
                @view-all="go('ProcinstTaskList')"
                @retry="loadTasks"
            />

            <AlertPanel
                :alerts="state.firingAlerts"
                :total="state.firingTotal"
                :loading="state.loading.alerts"
                :failed="state.failed.alerts"
                :viewable="canOpenAlerts"
                @view-all="go('AlertEventList')"
                @retry="loadAlerts"
            />
        </div>

        <RecentOpsPanel
            :logs="state.recentOpLogs"
            :loading="state.loading.opLogs"
            :failed="state.failed.opLogs"
            :linkable="canOpenResources"
            @navigate="navigateToResource"
            @retry="loadOpLogs"
        />
    </div>
</template>

<script lang="ts" setup>
import { Msg } from '@/hooks/useI18n';
import { hasPerm } from '@/components/auth/auth';
import { useAutoOpenResource } from '@/store/autoOpenResource';
import { useUserInfo } from '@/store/userInfo';
import { alertEventApi } from '@/views/ops/alert/api';
import { AlertEventStatusEnum } from '@/views/ops/alert/enums';
import type { AlertEventVO } from '@/views/ops/alert/types';
import { procinstTaskApi } from '@/views/flow/api';
import { ProcinstTaskStatus } from '@/views/flow/enums';
import type { ProcinstTask } from '@/views/flow/types';
import type { ResourceOpLog } from '@/views/ops/tag/types';
import { resourceOpLogApi } from '@/views/ops/tag/api';
import { personApi } from '@/views/personal/api';
import type { AccountRoleVO } from '@/views/personal/api';
import { storeToRefs } from 'pinia';
import { computed, onActivated, onMounted, reactive, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';
import AlertPanel from './components/AlertPanel.vue';
import HomeHero from './components/HomeHero.vue';
import QuickActions from './components/QuickActions.vue';
import RecentOpsPanel from './components/RecentOpsPanel.vue';
import SectionCard from '@/components/section-card/SectionCard.vue';
import StatTile from './components/StatTile.vue';
import TodoPanel from './components/TodoPanel.vue';
import { resourceComponents } from './resources';

const router = useRouter();
const { t } = useI18n();
const { userInfo } = storeToRefs(useUserInfo());

/**
 * 入口可见性只看菜单基础权限码（菜单授权时会连带下发对应基础码）：
 * 没有认证码说明连对应菜单都没授权，直接不展示，避免点了跳不过去
 */
const canOpenTasks = hasPerm('flow:task:base');
const canOpenAlerts = hasPerm('alert:event');
const canOpenResources = hasPerm('resource:base');

/** 首页每个面板最多展示的条目数：够看清最近动态，又不至于把首屏撑成一张长表 */
const FEED_SIZE = 5;
const OP_LOG_SIZE = 8;
/** 页面被 keep-alive 缓存，切回来超过该时长才重取，避免每次切页都打一轮接口 */
const STALE_AFTER_MS = 60 * 1000;

const state = reactive({
    roles: [] as AccountRoleVO[],
    pendingTasks: [] as ProcinstTask[],
    pendingTaskTotal: null as number | null,
    firingAlerts: [] as AlertEventVO[],
    firingTotal: null as number | null,
    unreadCount: null as number | null,
    recentOpLogs: [] as ResourceOpLog[],
    loading: {
        tasks: true,
        alerts: true,
        messages: true,
        opLogs: true,
    },
    failed: {
        tasks: false,
        alerts: false,
        messages: false,
        opLogs: false,
    },
});

const refreshing = ref(false);

const rolesText = computed(() => state.roles.map((role) => role.roleName).join('、'));

// 资源卡总数由各卡自行上报（见 resources/Base.vue 的 loaded 事件），避免再发一轮只取 total 的请求
const resourceTotals = ref<Array<number | undefined>>([]);
const resourceLoadedCount = ref(0);
const resourceTotal = computed<number | null>(() => {
    if (resourceLoadedCount.value < resourceComponents.length) {
        return null;
    }
    return resourceTotals.value.reduce<number>((sum, total) => sum + (total ?? 0), 0);
});

const onResourceLoaded = (index: number, total: number) => {
    if (resourceTotals.value[index] === undefined) {
        resourceLoadedCount.value += 1;
    }
    resourceTotals.value[index] = total;
};

type ReloadableResource = { reload: () => Promise<void> };
const resourceRefs: Array<ReloadableResource | null> = [];
const setResourceRef = (el: unknown, index: number) => {
    resourceRefs[index] = (el as ReloadableResource | null) ?? null;
};

let lastLoadedAt = 0;

const loadAll = () => {
    lastLoadedAt = Date.now();
    loadAccountInfo();
    loadTasks();
    loadAlerts();
    loadMessages();
    loadOpLogs();
};

onMounted(loadAll);

// keep-alive 下再次进入不重建组件：数据放旧了就重取，60s 内的来回切页不重复请求
onActivated(() => {
    if (Date.now() - lastLoadedAt > STALE_AFTER_MS) {
        loadAll();
    }
});

const loadAccountInfo = async () => {
    try {
        const info = await personApi.accountInfo.request();
        state.roles = info?.roles ?? [];
    } catch (error) {
        console.error('Failed to load account info:', error);
    }
};

const loadTasks = async () => {
    state.loading.tasks = true;
    state.failed.tasks = false;
    try {
        const res = await procinstTaskApi.tasks.request({
            status: ProcinstTaskStatus.Process.value as number,
            pageNum: 1,
            pageSize: FEED_SIZE,
        });
        state.pendingTasks = res?.list ?? [];
        state.pendingTaskTotal = res?.total ?? 0;
    } catch (error) {
        console.error('Failed to load pending tasks:', error);
        state.failed.tasks = true;
        state.pendingTaskTotal = null;
    } finally {
        state.loading.tasks = false;
    }
};

const loadAlerts = async () => {
    state.loading.alerts = true;
    state.failed.alerts = false;
    try {
        const res = await alertEventApi.list.request({
            status: AlertEventStatusEnum.Firing.value as number,
            pageNum: 1,
            pageSize: FEED_SIZE,
        });
        // 高优先级优先：同一批里先看到 critical，再按最近触发时间排序
        state.firingAlerts = [...(res?.list ?? [])].sort((a, b) => a.priority - b.priority || Date.parse(b.lastTriggerTime) - Date.parse(a.lastTriggerTime));
        state.firingTotal = res?.total ?? 0;
    } catch (error) {
        console.error('Failed to load firing alerts:', error);
        state.failed.alerts = true;
        state.firingTotal = null;
    } finally {
        state.loading.alerts = false;
    }
};

const loadMessages = async () => {
    state.loading.messages = true;
    state.failed.messages = false;
    try {
        state.unreadCount = (await personApi.getUnreadMsgCount.request()) ?? 0;
    } catch (error) {
        console.error('Failed to load unread message count:', error);
        state.failed.messages = true;
        state.unreadCount = null;
    } finally {
        state.loading.messages = false;
    }
};

const loadOpLogs = async () => {
    state.loading.opLogs = true;
    state.failed.opLogs = false;
    try {
        const res = await resourceOpLogApi.getAccountResourceOpLogs.request({ pageNum: 1, pageSize: OP_LOG_SIZE });
        state.recentOpLogs = res?.list ?? [];
    } catch (error) {
        console.error('Failed to load recent operation logs:', error);
        state.failed.opLogs = true;
    } finally {
        state.loading.opLogs = false;
    }
};

/** 顶栏刷新：面板数据与资源卡一起重取（资源卡自己持有分页状态，走其暴露的 reload） */
const refreshAll = async () => {
    if (refreshing.value) {
        return;
    }
    refreshing.value = true;
    try {
        lastLoadedAt = Date.now();
        await Promise.allSettled([...resourceRefs.map((resource) => resource?.reload()), loadTasks(), loadAlerts(), loadMessages(), loadOpLogs()]);
    } finally {
        refreshing.value = false;
    }
};

/**
 * 按路由名跳转：菜单路径由后端 ui_path 生成，首页不持有具体 URL。
 * 菜单被下线（前端没有对应组件）时 push 会抛错，这里给出明确提示而不是静默无反应
 */
const go = async (routeName: string) => {
    if (!router.hasRoute(routeName)) {
        Msg.error(t('home.navigateFailed'));
        return;
    }
    try {
        await router.push({ name: routeName });
    } catch (error) {
        console.error(`Failed to navigate to route "${routeName}":`, error);
        Msg.error(t('home.navigateFailed'));
    }
};

const navigateToAction = (action: { routeName: string }) => {
    go(action.routeName);
};

const navigateToPersonal = () => {
    router.push('/personal').catch((error) => {
        console.error('Failed to open personal page:', error);
        Msg.error(t('home.navigateFailed'));
    });
};

// 资源导航：与资源树同一套「待打开标签」机制，进资源页后直接打开对应节点
const navigateToResource = (codePath: string) => {
    useAutoOpenResource().setCodePath(codePath);
    router.push({ path: '/my-resource' });
};
</script>

<style scoped lang="scss">
.home-page {
    display: flex;
    flex-direction: column;
    gap: 16px;
    padding-bottom: 4px;
}

.home-stats {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 16px;
}

/* 上两行统一 6/6：列分界线贯通，资源概览与快捷入口、待办与告警各自等高 */
.home-grid,
.home-split {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 16px;
    align-items: stretch;
}

.resource-grid {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 10px;
    padding: 16px 18px 18px;
}

.home-panel-hint {
    font-size: 11px;
    color: var(--el-text-color-placeholder);
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
}

/* 窄屏优先留给标题与操作按钮，副标题是纯辅助信息 */
@media (max-width: 960px) {
    .home-panel-hint {
        display: none;
    }
}

/* 入场只做一次轻微上浮：避免首页每次切回来都大动效，也照顾 prefers-reduced-motion */
.home-page > * {
    animation: home-rise 0.4s cubic-bezier(0.22, 1, 0.36, 1) both;
}

.home-page > *:nth-child(2) {
    animation-delay: 0.04s;
}

.home-page > *:nth-child(3) {
    animation-delay: 0.08s;
}

.home-page > *:nth-child(4) {
    animation-delay: 0.12s;
}

.home-page > *:nth-child(5) {
    animation-delay: 0.16s;
}

@keyframes home-rise {
    from {
        opacity: 0;
        transform: translateY(8px);
    }
    to {
        opacity: 1;
        transform: none;
    }
}

@media (max-width: 1280px) {
    .home-stats {
        grid-template-columns: repeat(2, minmax(0, 1fr));
    }

    .home-grid,
    .home-split {
        grid-template-columns: minmax(0, 1fr);
    }
}

@media (max-width: 640px) {
    .home-stats {
        grid-template-columns: minmax(0, 1fr);
    }

    .resource-grid {
        grid-template-columns: repeat(2, minmax(0, 1fr));
    }
}

@media (prefers-reduced-motion: reduce) {
    .home-page > * {
        animation: none;
    }
}
</style>
