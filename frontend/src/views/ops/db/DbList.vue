<template>
    <div class="db-list">
        <el-drawer
            :append-to-body="false"
            :title="title"
            v-model="dialogVisible"
            @open="search"
            :before-close="cancel"
            :destroy-on-close="true"
            :close-on-click-modal="true"
            size="60%"
        >
            <template #header>
                <DrawerHeader :header="title" :back="cancel">
                    <template #extra>
                        <div class="mr-4.5">
                            <span>{{ $props.instance?.tags?.[0]?.codePath }}</span>
                            <el-divider direction="vertical" border-style="dashed" />
                            <SvgIcon :name="getDbDialect($props.instance?.type).getInfo()?.icon" :size="20" />
                            <el-divider direction="vertical" border-style="dashed" />
                            <span>{{ $props.instance?.host }}:{{ $props.instance?.port }}</span>
                        </div>
                    </template>
                </DrawerHeader>
            </template>

            <page-table
                ref="pageTableRef"
                :page-api="dbApi.dbs"
                v-model:query-form="query"
                :columns="columns"
                lazy
                show-selection
                v-model:selection-data="state.selectionData"
            >
                <template #tableHeader>
                    <el-button v-auth="perms.saveDb" type="primary" circle icon="Plus" @click="editDb(null)"> </el-button>
                    <el-button v-auth="perms.delDb" :disabled="state.selectionData.length < 1" @click="deleteDb" type="danger" circle icon="delete"></el-button>
                </template>

                <template #type="{ data }">
                    <el-tooltip :content="data.type" placement="top">
                        <SvgIcon :name="getDbDialect(data.type).getInfo().icon" :size="20" />
                    </el-tooltip>
                </template>

                <template #database="{ data }">
                    <el-popover placement="bottom" :width="200" trigger="click">
                        <template #reference>
                            <el-button @click="getDbNames(data)" type="primary" link>{{ $t('db.showDb') }}</el-button>
                        </template>
                        <el-table :data="filterDbs" v-loading="state.loadingDbNames" size="small">
                            <el-table-column prop="dbName" :label="$t('db.db')">
                                <template #header>
                                    <el-input v-model="state.dbNameSearch" size="small" :placeholder="$t('db.dbFilterPlaceholder')" clearable />
                                </template>
                            </el-table-column>
                        </el-table>
                    </el-popover>
                </template>

                <template #tagPath="{ data }">
                    <TagCodePath :path="data.tags" show-popover />
                </template>

                <template #action="{ data }">
                    <el-button v-auth="perms.saveDb" @click="editDb(data)" type="primary" link>{{ $t('common.edit') }}</el-button>

                    <el-divider direction="vertical" border-style="dashed" />

                    <el-button type="primary" @click="onShowSqlExec(data)" link>{{ $t('db.sqlRecord') }}</el-button>

                    <el-divider direction="vertical" border-style="dashed" />

                    <el-dropdown @command="handleMoreActionCommand">
                        <span class="el-dropdown-link-more">
                            {{ $t('common.more') }}
                            <el-icon class="el-icon--right">
                                <arrow-down />
                            </el-icon>
                        </span>
                        <template #dropdown>
                            <el-dropdown-menu>
                                <el-dropdown-item :command="{ type: 'dumpDb', data }"> {{ $t('db.dump') }} </el-dropdown-item>
                            </el-dropdown-menu>
                        </template>
                    </el-dropdown>
                </template>
            </page-table>
        </el-drawer>

        <el-dialog width="750px" :title="`${exportDialog.db} DB Dump`" v-model="exportDialog.visible">
            <el-row justify="space-between">
                <el-col :span="9">
                    <el-form-item :label="$t('db.dumpContent')">
                        <el-checkbox-group v-model="exportDialog.contents" :min="1">
                            <el-checkbox :label="$t('db.structure')" value="结构" />
                            <el-checkbox :label="$t('db.data')" value="数据" />
                        </el-checkbox-group>
                    </el-form-item>
                </el-col>
                <el-col :span="9">
                    <el-form-item :label="$t('db.extName')">
                        <el-radio-group v-model="exportDialog.extName">
                            <el-radio label="sql" value="sql" />
                            <el-radio label="gzip" value="gzip" />
                        </el-radio-group>
                    </el-form-item>
                </el-col>
            </el-row>

            <el-form-item>
                <el-transfer
                    v-model="exportDialog.value"
                    filterable
                    :filter-placeholder="$t('db.dbFilterPlacehoder')"
                    :titles="[$t('db.allDb'), $t('db.dumpDb')]"
                    :data="exportDialog.data"
                    max-height="300"
                    size="small"
                />
            </el-form-item>

            <template #footer>
                <el-button @click="exportDialog.visible = false">{{ $t('common.cancel') }}</el-button>
                <el-button @click="dumpDbs()" type="primary">{{ $t('common.confirm') }}</el-button>
            </template>
        </el-dialog>

        <el-dialog
            width="90%"
            :title="`${sqlExecLogDialog.title} - SQL`"
            :before-close="onBeforeCloseSqlExecDialog"
            :close-on-click-modal="false"
            v-model="sqlExecLogDialog.visible"
            :destroy-on-close="true"
            body-class="h-[65vh]"
        >
            <db-sql-exec-log :db-id="sqlExecLogDialog.dbId" :dbs="sqlExecLogDialog.dbs" />
        </el-dialog>

        <db-edit
            @confirm="confirmEditDb"
            @cancel="cancelEditDb"
            :title="dbEditDialog.title"
            v-model:visible="dbEditDialog.visible"
            :instance="props.instance"
            v-model:db="dbEditDialog.data"
        ></db-edit>
    </div>
</template>

<script lang="ts" setup>
import { isTrue } from '@/common/assert';
import config from '@/common/config';
import { joinClientParams } from '@/common/request';
import { sleep } from '@/common/utils/loading';
import { hasPerms } from '@/components/auth/auth';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';
import { TableColumn } from '@/components/page-table';
import PageTable from '@/components/page-table/PageTable.vue';
import { Msg, useI18nCreateTitle, useI18nDeleteConfirm, useI18nEditTitle } from '@/hooks/useI18n';
import { computed, defineAsyncComponent, reactive, ref, Ref, toRefs, useTemplateRef } from 'vue';
import { useI18n } from 'vue-i18n';
import TagCodePath from '../component/TagCodePath.vue';
import { dbApi } from './api';
import { DbInst } from './db';
import DbSqlExecLog from './DbSqlExecLog.vue';
import { DbType } from './dialect';
import { getDbDialect } from './dialect/index';
import { DbGetDbNamesMode } from './enums';
import type { Db, DbInstance } from './types';

const DbEdit = defineAsyncComponent(() => import('./DbEdit.vue'));

const { t } = useI18n();

const props = defineProps({
    instance: {
        type: [Object],
        required: true,
    },
    title: {
        type: String,
    },
});

const dialogVisible = defineModel<boolean>('visible');

const emit = defineEmits(['cancel']);

const columns = ref([
    TableColumn.new('name', 'common.name'),
    TableColumn.new('authCertName', 'db.acName'),
    TableColumn.new('getDatabaseMode', 'db.getDbMode').typeTag(DbGetDbNamesMode),
    TableColumn.new('database', 'DB').isSlot().setMinWidth(90),
    TableColumn.new('remark', 'common.remark'),
    TableColumn.new('code', 'common.code'),
    TableColumn.new('action', 'common.operation').isSlot().setMinWidth(210).fixedRight().noShowOverflowTooltip().alignCenter(),
]);

const perms = {
    base: 'db',
    saveDb: 'db:save',
    delDb: 'db:del',
    backupDb: 'db:backup',
    restoreDb: 'db:restore',
};

const actionBtns: Record<string, boolean> = hasPerms(Object.values(perms));

const pageTableRef = useTemplateRef<InstanceType<typeof PageTable>>('pageTableRef');

const state = reactive({
    loadingDbNames: false,
    currentDbNames: [] as string[],
    dbNameSearch: '',
    instances: [] as DbInstance[],
    /**
     * 选中的数据
     */
    selectionData: [] as Db[],
    /**
     * 查询条件
     */
    query: {
        instanceId: 0,
        pageNum: 1,
        pageSize: 0,
    },
    // sql执行记录弹框
    sqlExecLogDialog: {
        title: '',
        visible: false,
        dbs: [] as string[],
        dbId: 0,
    },
    chooseTableName: '',
    tableInfoDialog: {
        visible: false,
    },
    exportDialog: {
        visible: false,
        dbId: 0,
        db: '',
        type: 3,
        data: [] as { key: string; label: string }[],
        value: [],
        contents: [] as string[],
        extName: '',
    },
    dbEditDialog: {
        visible: false,
        data: null as Partial<Db> | null,
        title: '',
    },
    filterDb: {
        param: '',
        cache: [],
        list: [],
    },
});

const { query, sqlExecLogDialog, dbEditDialog, exportDialog } = toRefs(state);

const search = async () => {
    state.query.instanceId = props.instance?.id || 0;
    pageTableRef.value?.search();
};

const getDbNames = async (db: Db) => {
    try {
        state.loadingDbNames = true;
        state.currentDbNames = await DbInst.getDbNames(db);
    } finally {
        state.loadingDbNames = false;
    }
};

const filterDbs = computed(() => {
    const dbNames = state.currentDbNames;
    if (!dbNames) {
        return [];
    }
    const dbNameObjs = dbNames.map((x) => {
        return {
            dbName: x,
        };
    });
    return dbNameObjs.filter((db: { dbName: string }) => {
        return db.dbName.includes(state.dbNameSearch);
    });
});

const editDb = (data: Db | null) => {
    if (data) {
        state.dbEditDialog.data = { ...data };
    } else {
        state.dbEditDialog.data = {
            instanceId: props.instance?.id,
        };
    }
    state.dbEditDialog.title = data ? useI18nEditTitle('db.db') : useI18nCreateTitle('db.db');
    state.dbEditDialog.visible = true;
};

const confirmEditDb = async (db: Db) => {
    db.instanceId = props.instance?.id || 0;
    await dbApi.saveDb.request(db);
    Msg.saveSuccess();
    search();
    cancelEditDb();
};

const cancelEditDb = () => {
    state.dbEditDialog.visible = false;
    state.dbEditDialog.data = {};
};

const deleteDb = async () => {
    try {
        await useI18nDeleteConfirm(state.selectionData.map((x: Db) => x.name).join('、'));
        for (let db of state.selectionData) {
            await dbApi.deleteDb.request({ id: db.id });
        }
        Msg.deleteSuccess();
    } catch (err) {
        //
    } finally {
        search();
    }
};

const handleMoreActionCommand = (commond: { data: unknown; type: string }) => {
    const data = commond.data;
    const type = commond.type;
    switch (type) {
        case 'dumpDb': {
            onDumpDbs(data as Db);
            return;
        }
    }
};

const onShowSqlExec = async (row: Db) => {
    state.sqlExecLogDialog.title = `${row.name}`;
    state.sqlExecLogDialog.dbId = row.id;
    DbInst.getDbNames(row).then((res) => {
        state.sqlExecLogDialog.dbs = res;
    });
    state.sqlExecLogDialog.visible = true;
};

const onBeforeCloseSqlExecDialog = () => {
    state.sqlExecLogDialog.visible = false;
    state.sqlExecLogDialog.dbs = [];
    state.sqlExecLogDialog.dbId = 0;
};

const onDumpDbs = async (row: Db) => {
    const dbs = await DbInst.getDbNames(row);
    const data = [];
    for (let name of dbs) {
        data.push({
            key: name,
            label: name,
        });
    }
    state.exportDialog.db = row.name;
    state.exportDialog.value = [];
    state.exportDialog.data = data;
    state.exportDialog.dbId = row.id;
    state.exportDialog.contents = [t('db.structure'), t('db.data')];
    state.exportDialog.extName = 'sql';
    state.exportDialog.visible = true;
};

/**
 * 数据库信息导出
 */
const dumpDbs = async () => {
    isTrue(state.exportDialog.value.length > 0, 'db.noDumpDbMsg');
    let type = 0;
    for (let c of state.exportDialog.contents) {
        if (c == '结构') {
            type += 1;
        } else if (c == '数据') {
            type += 2;
        }
    }
    for (let db of state.exportDialog.value) {
        const a = document.createElement('a');
        a.setAttribute(
            'href',
            `${config.baseApiUrl}/dbs/${state.exportDialog.dbId}/dump?db=${db}&type=${type}&extName=${state.exportDialog.extName}&${joinClientParams()}`
        );
        a.click();
        await sleep(500);
    }
    state.exportDialog.visible = false;
};

const supportAction = (action: string, dbType: string): boolean => {
    let actions: string[] = [];
    switch (dbType) {
        case DbType.mysql:
        case DbType.mariadb:
            actions = ['dumpDb', 'backupDb', 'restoreDb'];
    }
    return actions.includes(action);
};

const cancel = () => {
    dialogVisible.value = false;
    emit('cancel');
};
</script>
<style lang="scss">
.db-list {
    .el-transfer-panel {
        width: 250px;
    }
}
.el-dropdown-link-more {
    cursor: pointer;
    color: var(--el-color-primary);
    display: flex;
    align-items: center;
    margin-top: 6px;
}
</style>
