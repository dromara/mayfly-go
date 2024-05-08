<template>
    <div class="db-list">
        <page-table
            ref="pageTableRef"
            :page-api="dbApi.dbs"
            :before-query-fn="checkRouteTagPath"
            :search-items="searchItems"
            v-model:query-form="query"
            :columns="columns"
            lazy
        >
            <template #instanceSelect>
                <el-select remote :remote-method="getInstances" v-model="query.instanceId" placeholder="输入并选择实例" filterable clearable>
                    <el-option v-for="item in state.instances" :key="item.id" :label="`${item.name}`" :value="item.id">
                        {{ item.name }}
                        <el-divider direction="vertical" border-style="dashed" />

                        {{ item.type }} / {{ item.host }}:{{ item.port }}
                        <el-divider direction="vertical" border-style="dashed" />
                        {{ item.username }}
                    </el-option>
                </el-select>
            </template>

            <template #type="{ data }">
                <el-tooltip :content="data.type" placement="top">
                    <SvgIcon :name="getDbDialect(data.type).getInfo().icon" :size="20" />
                </el-tooltip>
            </template>

            <template #host="{ data }">
                {{ `${data.host}:${data.port}` }}
            </template>

            <template #database="{ data }">
                <el-popover placement="bottom" :width="200" trigger="click">
                    <template #reference>
                        <el-button @click="state.currentDbs = data.database" type="primary" link>查看库</el-button>
                    </template>
                    <el-table :data="filterDbs" size="small">
                        <el-table-column prop="dbName" label="数据库">
                            <template #header>
                                <el-input v-model="state.dbNameSearch" size="small" placeholder="库名: 输入可过滤" clearable />
                            </template>
                        </el-table-column>
                    </el-table>
                </el-popover>
            </template>

            <template #tagPath="{ data }">
                <ResourceTags :tags="data.tags" />
            </template>

            <template #action="{ data }">
                <el-button type="primary" @click="onShowSqlExec(data)" link>SQL记录</el-button>
                <el-divider direction="vertical" border-style="dashed" />

                <el-dropdown @command="handleMoreActionCommand">
                    <span class="el-dropdown-link-more">
                        更多
                        <el-icon class="el-icon--right">
                            <arrow-down />
                        </el-icon>
                    </span>
                    <template #dropdown>
                        <el-dropdown-menu>
                            <el-dropdown-item :command="{ type: 'detail', data }"> 详情 </el-dropdown-item>
                            <el-dropdown-item :command="{ type: 'dumpDb', data }"> 导出 </el-dropdown-item>
                            <el-dropdown-item :command="{ type: 'backupDb', data }" v-if="actionBtns[perms.backupDb] && supportAction('backupDb', data.type)">
                                备份任务
                            </el-dropdown-item>
                            <el-dropdown-item
                                :command="{ type: 'backupHistory', data }"
                                v-if="actionBtns[perms.backupDb] && supportAction('backupDb', data.type)"
                            >
                                备份历史
                            </el-dropdown-item>
                            <el-dropdown-item
                                :command="{ type: 'restoreDb', data }"
                                v-if="actionBtns[perms.restoreDb] && supportAction('restoreDb', data.type)"
                            >
                                恢复任务
                            </el-dropdown-item>
                        </el-dropdown-menu>
                    </template>
                </el-dropdown>
            </template>
        </page-table>

        <el-dialog width="750px" :title="`${db} 数据库导出`" v-model="exportDialog.visible">
            <el-row justify="space-between">
                <el-col :span="9">
                    <el-form-item label="导出内容: ">
                        <el-checkbox-group v-model="exportDialog.contents" :min="1">
                            <el-checkbox label="结构" value="结构" />
                            <el-checkbox label="数据" value="数据" />
                        </el-checkbox-group>
                    </el-form-item>
                </el-col>
                <el-col :span="9">
                    <el-form-item label="扩展名: ">
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
                    filter-placeholder="按数据库名称筛选"
                    :titles="['全部数据库', '导出数据库']"
                    :data="exportDialog.data"
                    max-height="300"
                    size="small"
                />
            </el-form-item>

            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="exportDialog.visible = false">取消</el-button>
                    <el-button @click="dumpDbs()" type="primary">确定</el-button>
                </div>
            </template>
        </el-dialog>

        <el-dialog
            width="90%"
            :title="`${sqlExecLogDialog.title} - SQL执行记录`"
            :before-close="onBeforeCloseSqlExecDialog"
            :close-on-click-modal="false"
            v-model="sqlExecLogDialog.visible"
            :destroy-on-close="true"
        >
            <db-sql-exec-log :db-id="sqlExecLogDialog.dbId" :dbs="sqlExecLogDialog.dbs" />
        </el-dialog>

        <el-dialog
            width="80%"
            :title="`${dbBackupDialog.title} - 数据库备份`"
            :close-on-click-modal="false"
            :destroy-on-close="true"
            v-model="dbBackupDialog.visible"
        >
            <db-backup-list :dbId="dbBackupDialog.dbId" :dbNames="dbBackupDialog.dbs" />
        </el-dialog>

        <el-dialog
            width="80%"
            :title="`${dbBackupHistoryDialog.title} - 数据库备份历史`"
            :close-on-click-modal="false"
            :destroy-on-close="true"
            v-model="dbBackupHistoryDialog.visible"
        >
            <db-backup-history-list :dbId="dbBackupHistoryDialog.dbId" :dbNames="dbBackupHistoryDialog.dbs" />
        </el-dialog>

        <el-dialog
            width="80%"
            :title="`${dbRestoreDialog.title} - 数据库恢复`"
            :close-on-click-modal="false"
            :destroy-on-close="true"
            v-model="dbRestoreDialog.visible"
        >
            <db-restore-list :dbId="dbRestoreDialog.dbId" :dbNames="dbRestoreDialog.dbs" />
        </el-dialog>

        <el-dialog v-if="infoDialog.visible" v-model="infoDialog.visible" :before-close="onBeforeCloseInfoDialog">
            <el-descriptions title="详情" :column="3" border>
                <el-descriptions-item :span="2" label="名称">{{ infoDialog.data?.name }}</el-descriptions-item>
                <el-descriptions-item :span="1" label="id">{{ infoDialog.data?.id }}</el-descriptions-item>

                <el-descriptions-item :span="3" label="关联标签"><ResourceTags :tags="infoDialog.data.tags" /></el-descriptions-item>
                <el-descriptions-item :span="3" label="数据库实例名称">{{ infoDialog.instance?.name }}</el-descriptions-item>

                <el-descriptions-item :span="2" label="主机">{{ infoDialog.instance?.host }}</el-descriptions-item>
                <el-descriptions-item :span="1" label="端口">{{ infoDialog.instance?.port }}</el-descriptions-item>

                <el-descriptions-item :span="2" label="授权凭证">{{ infoDialog.instance.authCertName }}</el-descriptions-item>
                <el-descriptions-item :span="1" label="类型">
                    <SvgIcon :name="getDbDialect(infoDialog.instance?.type).getInfo().icon" :size="20" />{{ infoDialog.instance?.type }}
                </el-descriptions-item>

                <el-descriptions-item :span="3" label="数据库">{{ infoDialog.data?.database }}</el-descriptions-item>
                <el-descriptions-item :span="3" label="备注">{{ infoDialog.data?.remark }}</el-descriptions-item>

                <el-descriptions-item :span="2" label="创建时间">{{ dateFormat(infoDialog.data?.createTime) }} </el-descriptions-item>
                <el-descriptions-item :span="1" label="创建者">{{ infoDialog.data?.creator }}</el-descriptions-item>

                <el-descriptions-item :span="2" label="更新时间">{{ dateFormat(infoDialog.data?.updateTime) }} </el-descriptions-item>
                <el-descriptions-item :span="1" label="修改者">{{ infoDialog.data?.modifier }}</el-descriptions-item>
            </el-descriptions>
        </el-dialog>

        <db-edit @val-change="search()" :title="dbEditDialog.title" v-model:visible="dbEditDialog.visible" v-model:db="dbEditDialog.data"></db-edit>
    </div>
</template>

<script lang="ts" setup>
import { computed, defineAsyncComponent, onMounted, reactive, ref, Ref, toRefs } from 'vue';
import { dbApi } from './api';
import config from '@/common/config';
import { joinClientParams } from '@/common/request';
import { isTrue } from '@/common/assert';
import { dateFormat } from '@/common/utils/date';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { hasPerms } from '@/components/auth/auth';
import DbSqlExecLog from './DbSqlExecLog.vue';
import { DbType } from './dialect';
import { TagResourceTypeEnum } from '@/common/commonEnum';
import { useRoute } from 'vue-router';
import { getDbDialect } from './dialect/index';
import { getTagPathSearchItem } from '../component/tag';
import { SearchItem } from '@/components/SearchForm';
import DbBackupList from './DbBackupList.vue';
import DbBackupHistoryList from './DbBackupHistoryList.vue';
import DbRestoreList from './DbRestoreList.vue';
import ResourceTags from '../component/ResourceTags.vue';
import { sleep } from '@/common/utils/loading';

const DbEdit = defineAsyncComponent(() => import('./DbEdit.vue'));

const searchItems = [
    getTagPathSearchItem(TagResourceTypeEnum.DbName.value),
    SearchItem.slot('instanceId', '实例', 'instanceSelect'),
    SearchItem.input('code', '编号'),
];

const columns = ref([
    TableColumn.new('tags[0].tagPath', '关联标签').isSlot('tagPath').setAddWidth(20),
    TableColumn.new('name', '名称'),
    TableColumn.new('type', '类型').isSlot().setAddWidth(-15).alignCenter(),
    TableColumn.new('instanceName', '实例名'),
    TableColumn.new('host', 'ip:port').isSlot().setAddWidth(40),
    TableColumn.new('authCertName', '授权凭证'),
    TableColumn.new('database', '库').isSlot().setMinWidth(80),
    TableColumn.new('remark', '备注'),
    TableColumn.new('code', '编号'),
]);

const perms = {
    backupDb: 'db:backup',
    restoreDb: 'db:restore',
};

// 该用户拥有的的操作列按钮权限
// const actionBtns = hasPerms([perms.base, perms.saveDb]);
const actionBtns = hasPerms(Object.values(perms));
const actionColumn = TableColumn.new('action', '操作').isSlot().setMinWidth(180).fixedRight().alignCenter();

const route = useRoute();
const pageTableRef: Ref<any> = ref(null);
const state = reactive({
    row: {} as any,
    dbId: 0,
    db: '',
    currentDbs: '',
    dbNameSearch: '',
    instances: [] as any,
    /**
     * 选中的数据
     */
    selectionData: [],
    /**
     * 查询条件
     */
    query: {
        tagPath: '',
        instanceId: null,
        pageNum: 1,
        pageSize: 0,
    },
    infoDialog: {
        visible: false,
        data: null as any,
        instance: null as any,
        query: {
            instanceId: 0,
        },
    },
    // sql执行记录弹框
    sqlExecLogDialog: {
        title: '',
        visible: false,
        dbs: [],
        dbId: 0,
    },
    // 数据库备份弹框
    dbBackupDialog: {
        title: '',
        visible: false,
        dbs: [],
        dbId: 0,
    },
    // 数据库备份历史弹框
    dbBackupHistoryDialog: {
        title: '',
        visible: false,
        dbs: [],
        dbId: 0,
    },
    // 数据库恢复弹框
    dbRestoreDialog: {
        title: '',
        visible: false,
        dbs: [],
        dbId: 0,
    },
    chooseTableName: '',
    tableInfoDialog: {
        visible: false,
    },
    exportDialog: {
        visible: false,
        dbId: 0,
        type: 3,
        data: [] as any,
        value: [],
        contents: [] as any,
        extName: '',
    },
    dbEditDialog: {
        visible: false,
        data: null as any,
        title: '新增数据库',
    },
    filterDb: {
        param: '',
        cache: [],
        list: [],
    },
});

const { db, query, infoDialog, sqlExecLogDialog, exportDialog, dbEditDialog, dbBackupDialog, dbBackupHistoryDialog, dbRestoreDialog } = toRefs(state);

onMounted(async () => {
    if (Object.keys(actionBtns).length > 0) {
        columns.value.push(actionColumn);
    }
    search();
});

const filterDbs = computed(() => {
    const dbsStr = state.currentDbs;
    if (!dbsStr) {
        return [];
    }
    const dbs = dbsStr.split(' ').map((db: any) => {
        return { dbName: db };
    });
    return dbs.filter((db: any) => {
        return db.dbName.includes(state.dbNameSearch);
    });
});

const checkRouteTagPath = (query: any) => {
    if (route.query.tagPath) {
        query.tagPath = route.query.tagPath as string;
    }
    return query;
};

const search = async (tagPath: string = '') => {
    if (tagPath) {
        state.query.tagPath = tagPath;
    }
    pageTableRef.value.search();
};

const showInfo = async (info: any) => {
    state.infoDialog.data = info;
    state.infoDialog.query.instanceId = info.instanceId;
    const res = await dbApi.getInstance.request(state.infoDialog.query);
    state.infoDialog.instance = res;
    state.infoDialog.visible = true;
};

const onBeforeCloseInfoDialog = () => {
    state.infoDialog.visible = false;
    state.infoDialog.data = null;
    state.infoDialog.instance = null;
};

const getInstances = async (instanceName = '') => {
    if (!instanceName) {
        state.instances = [];
        return;
    }
    const data = await dbApi.instances.request({ name: instanceName });
    if (data) {
        state.instances = data.list;
    }
};

const handleMoreActionCommand = (commond: any) => {
    const data = commond.data;
    const type = commond.type;
    switch (type) {
        case 'detail': {
            showInfo(data);
            return;
        }
        case 'dumpDb': {
            onDumpDbs(data);
            return;
        }
        case 'backupDb': {
            onShowDbBackupDialog(data);
            return;
        }
        case 'backupHistory': {
            onShowDbBackupHistoryDialog(data);
            return;
        }
        case 'restoreDb': {
            onShowDbRestoreDialog(data);
            return;
        }
    }
};

const onShowSqlExec = async (row: any) => {
    state.sqlExecLogDialog.title = `${row.name}`;
    state.sqlExecLogDialog.dbId = row.id;
    state.sqlExecLogDialog.dbs = row.database.split(' ');
    state.sqlExecLogDialog.visible = true;
};

const onBeforeCloseSqlExecDialog = () => {
    state.sqlExecLogDialog.visible = false;
    state.sqlExecLogDialog.dbs = [];
    state.sqlExecLogDialog.dbId = 0;
};

const onShowDbBackupDialog = async (row: any) => {
    state.dbBackupDialog.title = `${row.name}`;
    state.dbBackupDialog.dbId = row.id;
    state.dbBackupDialog.dbs = row.database.split(' ');
    state.dbBackupDialog.visible = true;
};

const onShowDbBackupHistoryDialog = async (row: any) => {
    state.dbBackupHistoryDialog.title = `${row.name}`;
    state.dbBackupHistoryDialog.dbId = row.id;
    state.dbBackupHistoryDialog.dbs = row.database.split(' ');
    state.dbBackupHistoryDialog.visible = true;
};

const onShowDbRestoreDialog = async (row: any) => {
    state.dbRestoreDialog.title = `${row.name}`;
    state.dbRestoreDialog.dbId = row.id;
    state.dbRestoreDialog.dbs = row.database.split(' ');
    state.dbRestoreDialog.visible = true;
};

const onDumpDbs = async (row: any) => {
    const dbs = row.database.split(' ');
    const data = [];
    for (let name of dbs) {
        data.push({
            key: name,
            label: name,
        });
    }
    state.exportDialog.value = [];
    state.exportDialog.data = data;
    state.exportDialog.dbId = row.id;
    state.exportDialog.contents = ['结构', '数据'];
    state.exportDialog.extName = 'sql';
    state.exportDialog.visible = true;
};

/**
 * 数据库信息导出
 */
const dumpDbs = async () => {
    isTrue(state.exportDialog.value.length > 0, '请添加要导出的数据库');
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

defineExpose({ search });
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
