<template>
    <div class="h-full">
        <data-table ref="dataTableRef" :page-api="hostKeyApi.list" :search-items="searchItems" :columns="columns">
            <template #fingerprint="{ data }">
                <span class="font-mono text-xs">{{ data.fingerprint }}</span>
            </template>

            <!-- 行内操作由表格统一渲染（见 columns 里的 setActions），列宽与按钮同源 -->
        </data-table>
    </div>
</template>

<script lang="ts" setup>
import { TableColumn } from '@/components/data-table';
import DataTable from '@/components/data-table/DataTable.vue';
import { SearchItem } from '@/components/data-table/SearchForm';
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
    TableColumn.new('hostAddr', 'machine.hostAddr'),
    TableColumn.new('keyType', 'machine.keyType'),
    TableColumn.new('fingerprint', 'machine.fingerprint').isSlot(),
    TableColumn.new('remark', 'common.remark'),
    TableColumn.new('createTime', 'machine.firstTrustTime').isTime(),
    TableColumn.new('creator', 'common.creator'),
    TableColumn.new('action', 'common.operation')
        .setActions<MachineHostKeyVO>([{ key: 'revoke', label: 'machine.revokeTrust', type: 'danger', perm: perms.revoke, onClick: (row) => onRevoke(row) }])
        .fixedRight()
        .noShowOverflowTooltip()
        .alignCenter(),
];

const dataTableRef = useTemplateRef<InstanceType<typeof DataTable>>('dataTableRef');

/**
 * 撤销信任：下次连接该主机将重新走首次信任（TOFU）流程，用于主机重装/换钥后的指纹更新
 */
const onRevoke = async (data: MachineHostKeyVO) => {
    if (!(await useI18nConfirm('machine.revokeTrustConfirm', { addr: data.hostAddr }))) {
        return;
    }
    await hostKeyApi.del.request({ ids: data.id });
    Msg.deleteSuccess();
    dataTableRef.value?.search();
};
</script>

<style></style>
