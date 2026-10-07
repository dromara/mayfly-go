<template>
    <div class="h-full">
        <page-table ref="pageTableRef" :page-api="hostKeyApi.list" :search-items="searchItems" :columns="columns">
            <template #fingerprint="{ data }">
                <span class="font-mono text-xs">{{ data.fingerprint }}</span>
            </template>

            <template #action="{ data }">
                <el-button v-auth="perms.revoke" type="danger" link @click="onRevoke(data)">{{ $t('machine.revokeTrust') }}</el-button>
            </template>
        </page-table>
    </div>
</template>

<script lang="ts" setup>
import { TableColumn } from '@/components/page-table';
import PageTable from '@/components/page-table/PageTable.vue';
import { SearchItem } from '@/components/page-table/SearchForm';
import { Msg, useI18nConfirm } from '@/hooks/useI18n';
import { useTemplateRef } from 'vue';
import { hostKeyApi } from '../api';
import type { MachineHostKeyVO } from '../types';

// 信任库属安全管理面：查看与撤销信任的门槛与后端一致（机器管理权限）
const perms = {
    revoke: 'machine:update',
};

const searchItems = [SearchItem.input('hostAddr', 'machine.hostAddr')];

const columns = [
    TableColumn.new('hostAddr', 'machine.hostAddr').setAddWidth(15),
    TableColumn.new('keyType', 'machine.keyType'),
    TableColumn.new('fingerprint', 'machine.fingerprint').isSlot().setAddWidth(40),
    TableColumn.new('remark', 'common.remark'),
    TableColumn.new('createTime', 'machine.firstTrustTime').isTime(),
    TableColumn.new('creator', 'common.creator'),
    TableColumn.new('action', 'common.operation').isSlot().setMinWidth(110).fixedRight().noShowOverflowTooltip().alignCenter(),
];

const pageTableRef = useTemplateRef<InstanceType<typeof PageTable>>('pageTableRef');

/**
 * 撤销信任：下次连接该主机将重新走首次信任（TOFU）流程，用于主机重装/换钥后的指纹更新
 */
const onRevoke = async (data: MachineHostKeyVO) => {
    if (!(await useI18nConfirm('machine.revokeTrustConfirm', { addr: data.hostAddr }))) {
        return;
    }
    await hostKeyApi.del.request({ ids: data.id });
    Msg.deleteSuccess();
    pageTableRef.value?.search();
};
</script>

<style></style>
