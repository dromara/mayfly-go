<template>
    <TreeNodeRow :data="data" :show-actions="showActions">
        <template #prefix="{ data: d }">
            <el-popover @show="showDbInfo(d.params)" :show-after="500" placement="right-start" :title="$t('db.dbInstInfo')" trigger="hover" :width="250">
                <template #reference>
                    <SvgIcon :name="getDbDialect(d.params?.type as string)?.getInfo().icon" :size="18" />
                </template>
                <template #default>
                    <el-descriptions :column="1" size="small">
                        <el-descriptions-item :label="$t('common.name')">
                            {{ d.params?.name }}
                        </el-descriptions-item>
                        <el-descriptions-item label="Host">
                            {{ `${d.params?.host}:${d.params?.port}` }}
                        </el-descriptions-item>
                        <el-descriptions-item label="version">
                            <span v-loading="loadingServerInfo"> {{ `${dbServerInfo?.version}` }}</span>
                        </el-descriptions-item>
                        <el-descriptions-item :label="$t('common.remark')">
                            {{ d.params?.remark }}
                        </el-descriptions-item>
                    </el-descriptions>
                </template>
            </el-popover>
        </template>
    </TreeNodeRow>
</template>

<script lang="ts" setup>
import { ref } from 'vue';

import SvgIcon from '@/components/svg-icon/index.vue';
import TreeNodeRow from '@/views/ops/resource/tree/TreeNodeRow.vue';
import type { TreeNode } from '@/views/ops/resource/tree/types';
import { dbApi } from '../api';
import { getDbDialect } from '../dialect/index';

defineProps<{
    data: TreeNode;
    showActions?: boolean;
}>();

const serverInfoReqParam = ref({
    instanceId: 0,
});

const { execute: getDbServerInfo, isFetching: loadingServerInfo, data: dbServerInfo } = dbApi.getInstanceServerInfo.useApi(serverInfoReqParam);

const showDbInfo = async (db: Record<string, unknown>) => {
    if (dbServerInfo.value) {
        dbServerInfo.value.version = '';
    }
    serverInfoReqParam.value.instanceId = db.id as number;
    await getDbServerInfo();
};
</script>

<style lang="scss"></style>
