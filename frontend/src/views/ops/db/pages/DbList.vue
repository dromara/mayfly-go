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
                            <!-- 实例列表接口不返回关联标签（vo.InstanceListVO 无 tags），故不渲染标签路径 -->
                            <SvgIcon :name="getDbDialect($props.instance.type).getInfo()?.icon" :size="20" />
                            <el-divider direction="vertical" border-style="dashed" />
                            <span>{{ $props.instance.host }}:{{ $props.instance.port }}</span>
                        </div>
                    </template>
                </DrawerHeader>
            </template>

            <data-table
                ref="dataTableRef"
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

                <!-- 行内操作由表格统一渲染（见 columns 里的 setActions） -->
            </data-table>
        </el-drawer>

        <el-dialog width="750px" :title="`${exportDialog.db} DB Dump`" v-model="exportDialog.visible">
            <el-row justify="space-between">
                <el-col :span="9">
                    <el-form-item :label="$t('db.dumpContent')">
                        <el-checkbox-group v-model="exportDialog.contents" :min="1">
                            <el-checkbox :label="$t('db.structure')" value="structure" />
                            <el-checkbox :label="$t('db.data')" value="data" />
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

        <!-- 结构对比：两侧都靠库记录 id + 真实库名定位，同一取法不拿记录名猜库名 -->
        <el-dialog
            width="80%"
            :title="$t('db.schemaDiff')"
            :close-on-click-modal="false"
            v-model="schemaDiffDialog.visible"
            :destroy-on-close="true"
            body-class="h-[65vh]"
        >
            <db-schema-diff v-if="schemaDiffDialog.visible" :src-db-id="schemaDiffDialog.dbId" :src-dbs="schemaDiffDialog.dbs" />
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
import { TableColumn } from '@/components/data-table';
import DataTable from '@/components/data-table/DataTable.vue';
import { Msg, useI18nCreateTitle, useI18nDeleteConfirm, useI18nEditTitle } from '@/hooks/useI18n';
import { i18n } from '@/i18n';
import { computed, defineAsyncComponent, reactive, ref, Ref, toRefs, useTemplateRef, type PropType } from 'vue';
import TagCodePath from '../../component/TagCodePath.vue';
import { dbApi } from '../api';
import { DbInst } from '../db';
import DbSqlExecLog from './DbSqlExecLog.vue';
import DbSchemaDiff from './DbSchemaDiff.vue';
import { getDbDialect } from '../dialect/index';
import { DbGetDbNamesMode } from '../enums';
import type { Db, DbInstance, DbForm } from '../types';

/**
 * 导出内容 → 后端 dump 接口的 type 位掩码（见 server/internal/db/api/db.go 的 DumpSql）。
 *
 * 勾选值必须用与语言无关的稳定 token：此前直接拿译文当值，且默认勾选用 t() 赋值、
 * 位掩码计算又去比中文字面量，三处一旦语言不是中文就互相失配，弹窗打开时默认勾选整体丢失。
 */
const DUMP_CONTENT_BIT = {
    structure: 1,
    data: 2,
} as const;

type DumpContent = keyof typeof DUMP_CONTENT_BIT;

const DbEdit = defineAsyncComponent(() => import('./DbEdit.vue'));

const props = defineProps({
    instance: {
        // 当前实例（由 InstanceList 的行数据传入），DbEdit 需据此拉凭证与库名列表
        type: Object as PropType<DbInstance>,
        required: true,
    },
    title: {
        type: String,
    },
});

const dialogVisible = defineModel<boolean>('visible');

const emit = defineEmits<{
    /** 取消（新建库入口的取消由本组件内部处理，向父级冒泡关闭抽屉） */
    cancel: [];
}>();

const perms = {
    base: 'db',
    saveDb: 'db:save',
    delDb: 'db:del',
    backupDb: 'db:backup',
    restoreDb: 'db:restore',
};

const columns = ref([
    TableColumn.new('name', 'common.name'),
    TableColumn.new('authCertName', 'db.acName'),
    TableColumn.new('getDatabaseMode', 'db.getDbMode').typeTag(DbGetDbNamesMode),
    TableColumn.new('database', 'DB').isSlot().setMinWidth(90),
    TableColumn.new('remark', 'common.remark'),
    TableColumn.new('code', 'common.code'),
    TableColumn.new('action', 'common.operation')
        .setActions<Db>([
            { key: 'edit', label: 'common.edit', type: 'primary', perm: perms.saveDb, onClick: (row) => editDb(row) },
            { key: 'sqlRecord', label: 'db.sqlRecord', type: 'primary', onClick: (row) => onShowSqlExec(row) },
            { key: 'schemaDiff', label: 'db.schemaDiff', type: 'primary', onClick: (row) => onShowSchemaDiff(row) },
            // dump 不按方言门控：服务端 DumpDbScript 走通用 dbi 方言接口，全部库类型均可导出，且支持指定目标方言做异构导出
            { key: 'dump', label: 'db.dump', onClick: (row) => onDumpDbs(row) },
        ])
        .fixedRight()
        .noShowOverflowTooltip()
        .alignCenter(),
]);

const actionBtns: Record<string, boolean> = hasPerms(Object.values(perms));

const dataTableRef = useTemplateRef<InstanceType<typeof DataTable>>('dataTableRef');

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
    // 结构对比弹框
    schemaDiffDialog: {
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
        data: [] as { key: string; label: string }[],
        value: [],
        contents: [] as DumpContent[],
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

const { query, sqlExecLogDialog, schemaDiffDialog, dbEditDialog, exportDialog } = toRefs(state);

const search = async () => {
    state.query.instanceId = props.instance?.id || 0;
    dataTableRef.value?.search();
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
    state.dbEditDialog.title = data ? useI18nEditTitle('db.db') : i18n.global.t('db.addDbTitle');
    state.dbEditDialog.visible = true;
};

const confirmEditDb = async (form: DbForm) => {
    form.instanceId = props.instance?.id || 0;
    await dbApi.saveDb.request(form);
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
        if (!(await useI18nDeleteConfirm(state.selectionData.map((x: Db) => x.name).join('、')))) {
            return;
        }
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

const onShowSqlExec = async (row: Db) => {
    state.sqlExecLogDialog.title = `${row.name}`;
    state.sqlExecLogDialog.dbId = row.id;
    DbInst.getDbNames(row).then((res) => {
        state.sqlExecLogDialog.dbs = res;
    });
    state.sqlExecLogDialog.visible = true;
};

// 结构对比弹窗：表清单与目标库候选由面板自己取，这里只递给它的源库定位
const onShowSchemaDiff = async (row: Db) => {
    state.schemaDiffDialog.dbId = row.id;
    state.schemaDiffDialog.dbs = (await DbInst.getDbNames(row)) ?? [];
    state.schemaDiffDialog.visible = true;
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
    state.exportDialog.contents = ['structure', 'data'];
    state.exportDialog.extName = 'sql';
    state.exportDialog.visible = true;
};

/**
 * 数据库信息导出
 */
const dumpDbs = async () => {
    isTrue(state.exportDialog.value.length > 0, 'db.noDumpDbMsg');
    const type = state.exportDialog.contents.reduce((acc, c) => acc + DUMP_CONTENT_BIT[c], 0);
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
</style>
