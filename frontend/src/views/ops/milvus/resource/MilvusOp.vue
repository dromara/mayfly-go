<template>
    <el-tabs v-model="activeTab" type="border-card" class="milvus-tabs">
        <!-- 数据库管理 -->
        <el-tab-pane :label="$t('milvus.databaseManagement')" name="databases">
            <Databases :milvus-id="milvusId" :tab-key="tabKey" v-if="activeTab === 'databases'" @use="onUseDb" />
        </el-tab-pane>

        <!-- Collection 管理 -->
        <el-tab-pane :label="$t('milvus.collectionManagement')" name="collections">
            <Collections :milvus-id="milvusId" :tab-key="tabKey" v-if="activeTab === 'collections'" @change-tab="(name) => (activeTab = name)" />
        </el-tab-pane>

        <!-- 数据操作 -->
        <el-tab-pane :label="$t('milvus.dataOperation')" name="data">
            <DataOperation :milvus-id="milvusId" :tab-key="tabKey" v-if="activeTab === 'data'" />
        </el-tab-pane>

        <!-- 分区管理 -->
        <el-tab-pane :label="$t('milvus.partitionManagement')" name="partitions">
            <Partitions :milvus-id="milvusId" :tab-key="tabKey" v-if="activeTab === 'partitions'" />
        </el-tab-pane>
        <!-- 用户权限 -->
        <el-tab-pane :label="$t('milvus.userPermission')" name="users">
            <Users :milvus-id="milvusId" :tab-key="tabKey" v-if="activeTab === 'users'" />
        </el-tab-pane>

        <!-- 角色管理 -->
        <el-tab-pane :label="$t('milvus.roleManagement')" name="roles">
            <Roles :milvus-id="milvusId" :tab-key="tabKey" v-if="activeTab === 'roles'" />
        </el-tab-pane>

        <!-- 资源组 -->
        <el-tab-pane :label="$t('milvus.resourceGroup')" name="resourceGroups">
            <ResourceGroups :milvus-id="milvusId" :tab-key="tabKey" v-if="activeTab === 'resourceGroups'" />
        </el-tab-pane>

        <!-- 系统信息 -->
        <el-tab-pane :label="$t('milvus.systemInfo')" name="system">
            <SystemInfo :milvus-id="milvusId" :tab-key="tabKey" v-if="activeTab === 'system'" />
        </el-tab-pane>
    </el-tabs>
</template>

<script setup lang="ts">
import { setCurrentAcName } from '@/views/ops/milvus/resource/authCert';
import type { MilvusNodeParams } from '@/views/ops/milvus/resource';
import type { MilvusOpTabApi } from './index';
import { useMilvusStore } from '@/views/ops/milvus/resource/store';
import { onBeforeMount, ref } from 'vue';
import Collections from '../components/Collections.vue';
import Databases from '../components/Databases.vue';
import DataOperation from '../components/DataOperation.vue';
import Partitions from '../components/Partitions.vue';
import ResourceGroups from '../components/ResourceGroups.vue';
import Roles from '../components/Roles.vue';
import SystemInfo from '../components/SystemInfo.vue';
import Users from '../components/Users.vue';

const props = defineProps<{
    milvusId: number;
    acName: string;
    tabKey: string;
}>();

// 使用 per-tab 独立 store，实现多标签页状态隔离
const milvusStore = useMilvusStore(props.tabKey || 'milvusStore');

const emits = defineEmits(['init']);

const activeTab = ref('databases');

// 在子组件挂载前同步全局 ac，确保子组件 watcher / onMounted 发出的 API 请求使用正确的凭证
onBeforeMount(() => {
    if (props.acName) {
        setCurrentAcName(props.acName);
    }
});

const initMilvus = (params: MilvusNodeParams) => {
    // 设置当前选中的授权凭证名（同步全局 ac，确保 API 调用使用正确的凭证）
    const newAcName = params.selectAuthCert?.name || '';
    milvusStore.setAuthCertName(newAcName);
};

// 标签页激活时同步全局 ac（确保 API 调用使用正确的凭证）
const onActivate = () => {
    if (milvusStore.authCertName) {
        milvusStore.setAuthCertName(milvusStore.authCertName);
    }
};

const onUseDb = (_db: string) => {
    activeTab.value = 'collections';
};

defineExpose({
    initMilvus,
    onActivate,
} satisfies MilvusOpTabApi);
</script>

<style scoped lang="scss">
.milvus-tabs {
    height: 100%;
    display: flex;
    flex-direction: column;

    :deep(.el-tabs__content) {
        flex: 1;
        min-height: 0;
        overflow: visible;
        padding: 8px;
    }

    :deep(.el-tab-pane) {
        height: 100%;
    }
}
</style>
