<template>
    <el-row class="es-op-header shrink-0">
        <el-col :span="20">
            <el-space>
                <el-link @click="emit('refreshData')" icon="refresh" underline="never" :title="t('common.refresh')" />
                <el-link @click="emit('basicSearch')" icon="Search" underline="never" :title="t('es.opSearch')" />
                <el-link v-auth="perms.saveData" @click="emit('addDoc')" icon="plus" underline="never" :title="t('common.create')" />
                <el-link v-auth="perms.delData" :disabled="selectKeysLen === 0" @click="emit('deleteDocs')" icon="Minus" underline="never" :title="t('common.delete')" />
                <el-link v-auth="perms.saveData" :disabled="selectKeysLen !== 1" @click="emit('editSelectDoc')" icon="EditPen" underline="never" :title="t('common.edit')" />
                <el-link :disabled="state.search.from === 0" @click="emit('firstPage')" icon="DArrowLeft" underline="never" :title="t('es.page.home')" />
                <el-link :disabled="state.search.from === 0" @click="emit('prevPage')" icon="ArrowLeft" underline="never" :title="t('es.page.prev')" />
                <el-dropdown placement="bottom" size="small" :teleported="false" :title="t('es.page.changeSize')">
                    <el-link underline="never" :style="{ fontSize: '12px' }">
                        {{ state.currentFrom + 1 }} - {{ Math.min(state.currentFrom + state.search.size, state.total) }}</el-link
                    >
                    <template #dropdown>
                        <el-dropdown-menu>
                            <el-dropdown-item @click="emit('changePageSize', 25)">25</el-dropdown-item>
                            <el-dropdown-item @click="emit('changePageSize', 50)">50</el-dropdown-item>
                            <el-dropdown-item @click="emit('changePageSize', 100)">100</el-dropdown-item>
                            <el-dropdown-item @click="emit('changePageSize', 200)">200</el-dropdown-item>
                            <el-dropdown-item @click="emit('changePageSize', 1000)">1000</el-dropdown-item>
                        </el-dropdown-menu>
                    </template>
                </el-dropdown>
                /
                <el-link
                    underline="never"
                    @click="emit('switchTrackTotal')"
                    :type="state.search.track_total_hits === true ? 'success' : 'info'"
                    :style="{ fontSize: '12px' }"
                    :title="t('es.page.total')"
                >
                    {{ state.searchRes.hits?.total?.value || 0 }}</el-link
                >
                <el-link
                    :disabled="state.search.from + state.search.size >= (state.total || 0)"
                    @click="emit('nextPage')"
                    icon="ArrowRight"
                    underline="never"
                    :title="t('es.page.next')"
                />

                <el-dropdown placement="bottom" size="small" :max-height="300" :hide-on-click="false" trigger="click" :teleported="false" :title="t('es.opViewColumns')">
                    <el-link icon="Operation" underline="never" />
                    <template #dropdown>
                        <el-dropdown-menu class="dropdown-menu">
                            <el-dropdown-item>
                                <el-space>
                                    <el-checkbox @change="emit('checkAllColumns')" v-model="state.checkAllColumns" />
                                    <el-input
                                        v-model="state.columnsFilterText"
                                        @input="emit('filterColumns')"
                                        :placeholder="t('es.filterColumn')"
                                        clearable
                                        size="small"
                                    />
                                </el-space>
                            </el-dropdown-item>
                            <template v-for="column in state.columns" :key="column.key">
                                <el-dropdown-item v-if="column._filterd" :command="column.key">
                                    <el-checkbox v-model="column._show" @change="emit('checkColumnFilter', column)">
                                        {{ column.title }}
                                    </el-checkbox>
                                </el-dropdown-item>
                            </template>
                        </el-dropdown-menu>
                    </template>
                </el-dropdown>

                <el-link @click="emit('openExportDialog')" icon="Download" underline="never" :title="t('es.export.title')" />
            </el-space>
        </el-col>
    </el-row>
</template>

<script lang="ts" setup>
import type { EsColumn } from '@/views/ops/es/types';
import { useI18n } from 'vue-i18n';

interface ToolbarState {
    search: { from: number; size: number; track_total_hits?: boolean; [key: string]: unknown };
    searchRes: { hits?: { total?: { value?: number } } };
    total: number;
    currentFrom: number;
    columns: EsColumn[];
    columnsFilterText: string;
    checkAllColumns: boolean;
}

defineProps<{
    state: ToolbarState;
    selectKeysLen: number;
    perms: { saveData: string; delData: string };
}>();

const emit = defineEmits<{
    refreshData: [];
    basicSearch: [];
    addDoc: [];
    deleteDocs: [];
    editSelectDoc: [];
    firstPage: [];
    prevPage: [];
    nextPage: [];
    changePageSize: [size: number];
    switchTrackTotal: [];
    checkAllColumns: [];
    filterColumns: [];
    checkColumnFilter: [column: EsColumn];
    openExportDialog: [];
}>();

const { t } = useI18n();
</script>
