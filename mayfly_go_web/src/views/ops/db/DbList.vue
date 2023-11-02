<template>
    <div class="db-list">
        <page-table
            ref="pageTableRef"
            :query="queryConfig"
            v-model:query-form="query"
            :show-selection="true"
            v-model:selection-data="state.selectionData"
            :data="datas"
            :columns="columns"
            :total="total"
            v-model:page-size="query.pageSize"
            v-model:page-num="query.pageNum"
            @pageChange="search()"
        >
            <template #tagPathSelect>
                <el-select @focus="getTags" v-model="query.tagPath" placeholder="请选择标签" filterable clearable style="width: 200px">
                    <el-option v-for="item in tags" :key="item" :label="item" :value="item"> </el-option>
                </el-select>
            </template>

            <template #instanceSelect>
                <el-select
                    remote
                    :remote-method="getInstances"
                    v-model="query.instanceId"
                    placeholder="输入并选择实例"
                    filterable
                    clearable
                    style="width: 200px"
                >
                    <el-option v-for="item in state.instances" :key="item.id" :label="`${item.name}`" :value="item.id">
                        {{ item.name }}
                        <el-divider direction="vertical" border-style="dashed" />

                        {{ item.type }} / {{ item.host }}:{{ item.port }}
                        <el-divider direction="vertical" border-style="dashed" />
                        {{ item.username }}
                    </el-option>
                </el-select>
            </template>

            <template #queryRight>
                <el-button v-auth="perms.saveDb" type="primary" icon="plus" @click="editDb(false)">添加</el-button>
                <el-button v-auth="perms.delDb" :disabled="selectionData.length < 1" @click="deleteDb()" type="danger" icon="delete">删除</el-button>
            </template>

            <template #tagPath="{ data }">
                <tag-info :tag-path="data.tagPath" />
                <span class="ml5">
                    {{ data.tagPath }}
                </span>
            </template>

            <template #host="{ data }">
                {{ `${data.host}:${data.port}` }}
            </template>

            <template #database="{ data }">
                <el-popover placement="right" trigger="click" :width="300">
                    <template #reference>
                        <el-link type="primary" :underline="false" plain @click="selectDb(data.dbs)">查看 </el-link>
                    </template>
                    <el-input v-model="filterDb.param" @keyup="filterSchema" class="w-50 m-2" placeholder="搜索" size="small">
                        <template #prefix>
                            <el-icon class="el-input__icon">
                                <search-icon />
                            </el-icon>
                        </template>
                    </el-input>
                    <div
                        class="el-tag--plain el-tag--success"
                        v-for="db in filterDb.list"
                        :key="db"
                        style="border: 1px var(--color-success-light-3) solid; margin-top: 3px; border-radius: 5px; padding: 2px; position: relative"
                    >
                        <el-link type="success" plain size="small" :underline="false">{{ db }}</el-link>
                        <el-link type="primary" plain size="small" :underline="false" @click="showTableInfo(data, db)" style="position: absolute; right: 4px"
                            >操作
                        </el-link>
                    </div>
                </el-popover>
            </template>

            <template #more="{ data }">
                <el-button @click="showInfo(data)" link>详情</el-button>
                <el-button class="ml5" type="primary" @click="onShowSqlExec(data)" link>SQL执行记录</el-button>
            </template>

            <template #action="{ data }">
                <el-button v-if="actionBtns[perms.saveDb]" @click="editDb(data)" type="primary" link>编辑</el-button>
                <el-button v-if="data.type == 'mysql'" class="ml5" type="primary" @click="onDumpDbs(data)" link>导出</el-button>
            </template>
        </page-table>

        <el-dialog width="80%" :title="`${db} 表信息`" :before-close="closeTableInfo" v-model="tableInfoDialog.visible">
            <db-table-list :db-id="dbId" :db="db" :db-type="state.row.type" />
        </el-dialog>

        <el-dialog width="620" :title="`${db} 数据库导出`" v-model="exportDialog.visible">
            <el-row justify="space-between">
                <el-col :span="9">
                    <el-form-item label="导出内容: ">
                        <el-checkbox-group v-model="exportDialog.contents" :min="1">
                            <el-checkbox label="结构" />
                            <el-checkbox label="数据" />
                        </el-checkbox-group>
                    </el-form-item>
                </el-col>
                <el-col :span="9">
                    <el-form-item label="扩展名: ">
                        <el-radio-group v-model="exportDialog.extName">
                            <el-radio label="sql" />
                            <el-radio label="gzip" />
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
        >
            <db-sql-exec-log :db-id="sqlExecLogDialog.dbId" :dbs="sqlExecLogDialog.dbs" />
        </el-dialog>

        <el-dialog v-model="infoDialog.visible" :before-close="onBeforeCloseInfoDialog" :close-on-click-modal="false">
            <el-descriptions title="详情" :column="3" border>
                <el-descriptions-item :span="3" label="标签路径">{{ infoDialog.data?.tagPath }}</el-descriptions-item>
                <el-descriptions-item :span="2" label="名称">{{ infoDialog.data?.name }}</el-descriptions-item>
                <el-descriptions-item :span="1" label="id">{{ infoDialog.data?.id }}</el-descriptions-item>
                <el-descriptions-item :span="3" label="数据库">{{ infoDialog.data?.database }}</el-descriptions-item>
                <el-descriptions-item :span="3" label="备注">{{ infoDialog.data?.remark }}</el-descriptions-item>
                <el-descriptions-item :span="2" label="创建时间">{{ dateFormat(infoDialog.data?.createTime) }} </el-descriptions-item>
                <el-descriptions-item :span="1" label="创建者">{{ infoDialog.data?.creator }}</el-descriptions-item>
                <el-descriptions-item :span="2" label="更新时间">{{ dateFormat(infoDialog.data?.updateTime) }} </el-descriptions-item>
                <el-descriptions-item :span="1" label="修改者">{{ infoDialog.data?.modifier }}</el-descriptions-item>

                <el-descriptions-item :span="3" label="数据库实例名称">{{ infoDialog.instance?.name }}</el-descriptions-item>
                <el-descriptions-item :span="2" label="主机">{{ infoDialog.instance?.host }}</el-descriptions-item>
                <el-descriptions-item :span="1" label="端口">{{ infoDialog.instance?.port }}</el-descriptions-item>
                <el-descriptions-item :span="2" label="用户名">{{ infoDialog.instance?.username }}</el-descriptions-item>
                <el-descriptions-item :span="1" label="类型">{{ infoDialog.instance?.type }}</el-descriptions-item>
            </el-descriptions>
        </el-dialog>

        <db-edit @val-change="valChange" :title="dbEditDialog.title" v-model:visible="dbEditDialog.visible" v-model:db="dbEditDialog.data"></db-edit>
    </div>
</template>

<script lang="ts" setup>
import { ref, toRefs, reactive, onMounted, defineAsyncComponent } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import { dbApi } from './api';
import config from '@/common/config';
import { joinClientParams } from '@/common/request';
import { isTrue } from '@/common/assert';
import { Search as SearchIcon } from '@element-plus/icons-vue';
import { dateFormat } from '@/common/utils/date';
import TagInfo from '../component/TagInfo.vue';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn, TableQuery } from '@/components/pagetable';
import { hasPerms } from '@/components/auth/auth';
import DbSqlExecLog from './DbSqlExecLog.vue';

const DbEdit = defineAsyncComponent(() => import('./DbEdit.vue'));
const DbTableList = defineAsyncComponent(() => import('./table/DbTableList.vue'));

const perms = {
    base: 'db',
    saveDb: 'db:save',
    delDb: 'db:del',
};

const queryConfig = [TableQuery.slot('tagPath', '标签', 'tagPathSelect'), TableQuery.slot('instanceId', '实例', 'instanceSelect')];

const columns = ref([
    TableColumn.new('tagPath', '标签路径').isSlot().setAddWidth(20),
    TableColumn.new('instanceName', '实例名'),
    TableColumn.new('type', '类型'),
    TableColumn.new('host', 'ip:port').isSlot().setAddWidth(40),
    TableColumn.new('username', 'username'),
    TableColumn.new('name', '名称'),
    TableColumn.new('database', '数据库').isSlot().setMinWidth(70),
    TableColumn.new('remark', '备注'),
    TableColumn.new('more', '更多').isSlot().setMinWidth(180).fixedRight(),
]);

// 该用户拥有的的操作列按钮权限
const actionBtns = hasPerms([perms.base, perms.saveDb]);
const actionColumn = TableColumn.new('action', '操作').isSlot().setMinWidth(150).fixedRight().alignCenter();

const pageTableRef: any = ref(null);

const state = reactive({
    row: {} as any,
    dbId: 0,
    db: '',
    tags: [],
    instances: [] as any,
    /**
     * 选中的数据
     */
    selectionData: [],
    /**
     * 查询条件
     */
    query: {
        tagPath: null,
        instanceId: null,
        pageNum: 1,
        pageSize: 10,
    },
    datas: [],
    total: 0,
    infoDialog: {
        visible: false,
        data: null as any,
        instance: null as any,
        query: {
            instanceId: 0,
        },
    },
    showDumpInfo: false,
    dumpInfo: {
        id: 0,
        db: '',
        type: 3,
        tables: [],
    },
    // sql执行记录弹框
    sqlExecLogDialog: {
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

const { dbId, db, tags, selectionData, query, datas, total, infoDialog, sqlExecLogDialog, tableInfoDialog, exportDialog, dbEditDialog, filterDb } =
    toRefs(state);

onMounted(async () => {
    if (Object.keys(actionBtns).length > 0) {
        columns.value.push(actionColumn);
    }
    search();
});

const search = async () => {
    try {
        pageTableRef.value.loading(true);
        let res: any = await dbApi.dbs.request(state.query);
        // 切割数据库
        res.list?.forEach((e: any) => {
            e.popoverSelectDbVisible = false;
            e.dbs = e.database.split(' ');
        });
        state.datas = res.list;
        state.total = res.total;
    } finally {
        pageTableRef.value.loading(false);
    }
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

const getTags = async () => {
    state.tags = await dbApi.dbTags.request(null);
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

const editDb = async (data: any) => {
    if (!data) {
        state.dbEditDialog.data = null;
        state.dbEditDialog.title = '新增数据库资源';
    } else {
        state.dbEditDialog.data = data;
        state.dbEditDialog.title = '修改数据库资源';
    }
    state.dbEditDialog.visible = true;
};

const valChange = () => {
    search();
};

const deleteDb = async () => {
    try {
        await ElMessageBox.confirm(`确定删除【${state.selectionData.map((x: any) => x.name).join(', ')}】库?`, '提示', {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning',
        });
        await dbApi.deleteDb.request({ id: state.selectionData.map((x: any) => x.id).join(',') });
        ElMessage.success('删除成功');
        search();
    } catch (err) {
        //
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
const dumpDbs = () => {
    isTrue(state.exportDialog.value.length > 0, '请添加要导出的数据库');
    const a = document.createElement('a');
    let type = 0;
    for (let c of state.exportDialog.contents) {
        if (c == '结构') {
            type += 1;
        } else if (c == '数据') {
            type += 2;
        }
    }
    a.setAttribute(
        'href',
        `${config.baseApiUrl}/dbs/${state.exportDialog.dbId}/dump?db=${state.exportDialog.value.join(',')}&type=${type}&extName=${
            state.exportDialog.extName
        }&${joinClientParams()}`
    );
    a.click();
    state.exportDialog.visible = false;
};

const showTableInfo = async (row: any, db: string) => {
    state.dbId = row.id;
    state.row = row;
    state.db = db;
    state.tableInfoDialog.visible = true;
};

const closeTableInfo = () => {
    state.showDumpInfo = false;
    state.tableInfoDialog.visible = false;
};

// 点击查看时初始化数据
const selectDb = (row: any) => {
    state.filterDb.param = '';
    state.filterDb.cache = row;
    state.filterDb.list = row;
};

// 输入字符过滤schema
const filterSchema = () => {
    if (state.filterDb.param) {
        state.filterDb.list = state.filterDb.cache.filter((a) => {
            return String(a).toLowerCase().indexOf(state.filterDb.param) > -1;
        });
    } else {
        state.filterDb.list = state.filterDb.cache;
    }
};
</script>
<style lang="scss"></style>
