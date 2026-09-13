<template>
    <div
        v-for="(column, i) in columns"
        :key="i"
        :style="{
            width: `${column.width}px`,
            height: '100%',
            textAlign: 'center',
            borderRight: 'var(--el-table-border)',
            borderTop: 'var(--el-table-border)',
        }"
    >
            <!-- 行号列 -->
            <div v-if="column.key === rowNoColumnKey" class="header-column-title">
                <b class="el-text" tag="b"> {{ column.title }} </b>
            </div>

            <!-- 字段名列 -->
            <div v-else style="position: relative" @mouseenter="showColumnAction(column)" @mouseleave="hideColumnAction">
                <!-- 字段列的数据类型 -->
                <div class="column-type">
                    <span v-if="column.dataTypeSubscript === 'icon-clock'">
                        <SvgIcon :size="9" name="Clock" style="cursor: unset" />
                    </span>
                    <span class="text-[8px]!" v-else>{{ column.dataTypeSubscript }}</span>
                </div>

                <div v-if="showColumnTip">
                    <div class="header-column-title">
                        <b :title="column.remark" class="el-text cursor-pointer">
                            {{ column.title }}
                        </b>
                    </div>

                    <!-- 字段备注信息 -->
                    <div
                        v-if="showColumnComment"
                        style="color: var(--el-color-info-light-3)"
                        class="text-[10px]! el-text el-text--small is-truncated"
                    >
                        {{ column.columnComment }}
                    </div>
                </div>

                <div v-else class="header-column-title">
                    <b class="el-text"> {{ column.title }} </b>
                </div>

                <!-- 字段列右部分内容 -->
                <div class="column-right">
                    <el-dropdown
                        @command="handleColumnCommand(column, $event)"
                        @visibleChange="onColumnActionVisibleChange(column, $event)"
                        trigger="click"
                        v-if="column.key !== rowNoColumnKey"
                        size="small"
                        placement="bottom-start"
                    >
                        <span class="column-actions-trigger">
                            <!-- 排序箭头图标 -->
                            <SvgIcon
                                v-if="column.key === nowSortColumn?.key && !showColumnActions[column.key] && !columnActionVisible[column.key]"
                                :color="'var(--el-color-primary)'"
                                :name="nowSortColumn?.order == 'asc' ? 'top' : 'bottom'"
                                :size="14"
                            />
                            <!-- 更多操作图标 -->
                            <SvgIcon
                                v-if="columnActionVisible[column.key] || showColumnActions[column.key]"
                                name="MoreFilled"
                                :size="14"
                                :color="'var(--el-color-primary)'"
                                class="column-more-icon"
                                :class="{ 'column-more-icon-visible': columnActionVisible[column.key] || showColumnActions[column.key] }"
                            />
                        </span>
                        <template #dropdown>
                            <el-dropdown-menu>
                                <el-dropdown-item v-if="showColumnActionSort" command="sort-asc">
                                    <SvgIcon name="top" class="mr-1" />
                                    {{ $t('db.asc') }}
                                </el-dropdown-item>
                                <el-dropdown-item v-if="showColumnActionSort" command="sort-desc">
                                    <SvgIcon name="bottom" class="mr-1" />
                                    {{ $t('db.desc') }}
                                </el-dropdown-item>
                                <el-dropdown-item v-if="showColumnActionFixed && !column.fixed" command="fix">
                                    <SvgIcon name="Paperclip" class="mr-1" />
                                    {{ $t('db.fixed') }}
                                </el-dropdown-item>
                                <el-dropdown-item v-if="showColumnActionFixed && column.fixed" command="unfix">
                                    <SvgIcon name="Minus" class="mr-1" />
                                    {{ $t('db.cancelFiexd') }}
                                </el-dropdown-item>
                            </el-dropdown-menu>
                        </template>
                    </el-dropdown>
                </div>
            </div>
        </div>
</template>

<script lang="ts" setup>
import SvgIcon from '@/components/svg-icon/index.vue';
import type { VirtualTableColumn } from '@/components/virtual-table/adapters';
import { ref } from 'vue';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();

const props = defineProps<{
    columns: VirtualTableColumn[];
    rowNoColumnKey: string;
    showColumnTip: boolean;
    showColumnComment: boolean;
    nowSortColumn: { key: string; order: string } | null;
    showColumnActionSort: boolean;
    showColumnActionFixed: boolean;
}>();

const emit = defineEmits<{
    sortChange: [sort: { key: string; order: string }];
    fixChange: [column: VirtualTableColumn, fixed: boolean];
}>();

// 用于控制列操作按钮的显示
const showColumnActions = ref<Record<string, boolean>>({});
const columnActionVisible = ref<Record<string, boolean>>({});

const showColumnAction = (column: VirtualTableColumn) => {
    showColumnActions.value[column.key] = true;
};

const hideColumnAction = () => {
    showColumnActions.value = {};
};

const handleColumnCommand = (column: VirtualTableColumn, command: string) => {
    switch (command) {
        case 'sort-asc':
            emit('sortChange', { key: column.key, order: 'asc' });
            break;
        case 'sort-desc':
            emit('sortChange', { key: column.key, order: 'desc' });
            break;
        case 'fix':
            emit('fixChange', column, true);
            break;
        case 'unfix':
            emit('fixChange', column, false);
            break;
    }
    // 点击了取消固定等操作后，可能更多的icon还是显示在列上，所以需要重新置为空对象
    columnActionVisible.value = {};
};

const onColumnActionVisibleChange = (column: VirtualTableColumn, visible: boolean) => {
    columnActionVisible.value = {}; // 只显示一个列的更多icon
    columnActionVisible.value[column.key] = visible;
};
</script>
