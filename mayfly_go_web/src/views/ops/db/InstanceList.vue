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
            <template #queryRight>
                <el-button v-auth="perms.saveInstance" type="primary" icon="plus" @click="editInstance(false)">添加</el-button>
                <el-button v-auth="perms.delInstance" :disabled="selectionData.length < 1" @click="deleteInstance()" type="danger" icon="delete"
                    >删除</el-button
                >
            </template>

            <template #type="{ data }">
                <el-tooltip :content="data.type" placement="top">
                    <SvgIcon :name="getDbDialect(data.type).getInfo().icon" :size="20" />
                </el-tooltip>
            </template>

            <template #action="{ data }">
                <el-button @click="showInfo(data)" link>详情</el-button>
                <el-button v-if="actionBtns[perms.saveInstance]" @click="editInstance(data)" type="primary" link>编辑</el-button>
            </template>
        </page-table>

        <el-dialog v-model="infoDialog.visible">
            <el-descriptions title="详情" :column="3" border>
                <el-descriptions-item :span="2" label="名称">{{ infoDialog.data.name }}</el-descriptions-item>
                <el-descriptions-item :span="1" label="id">{{ infoDialog.data.id }}</el-descriptions-item>
                <el-descriptions-item :span="2" label="主机">{{ infoDialog.data.host }}</el-descriptions-item>
                <el-descriptions-item :span="1" label="端口">{{ infoDialog.data.port }}</el-descriptions-item>

                <el-descriptions-item :span="2" label="用户名">{{ infoDialog.data.username }}</el-descriptions-item>
                <el-descriptions-item :span="1" label="类型">{{ infoDialog.data.type }}</el-descriptions-item>

                <el-descriptions-item :span="3" label="连接参数">{{ infoDialog.data.params }}</el-descriptions-item>
                <el-descriptions-item :span="3" label="备注">{{ infoDialog.data.remark }}</el-descriptions-item>

                <el-descriptions-item :span="3" label="SSH隧道">{{ infoDialog.data.sshTunnelMachineId > 0 ? '是' : '否' }} </el-descriptions-item>

                <el-descriptions-item :span="2" label="创建时间">{{ dateFormat(infoDialog.data.createTime) }} </el-descriptions-item>
                <el-descriptions-item :span="1" label="创建者">{{ infoDialog.data.creator }}</el-descriptions-item>

                <el-descriptions-item :span="2" label="更新时间">{{ dateFormat(infoDialog.data.updateTime) }} </el-descriptions-item>
                <el-descriptions-item :span="1" label="修改者">{{ infoDialog.data.modifier }}</el-descriptions-item>
            </el-descriptions>
        </el-dialog>

        <instance-edit
            @val-change="valChange"
            :title="instanceEditDialog.title"
            v-model:visible="instanceEditDialog.visible"
            v-model:data="instanceEditDialog.data"
        ></instance-edit>
    </div>
</template>

<script lang="ts" setup>
import { ref, toRefs, reactive, onMounted, defineAsyncComponent } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import { dbApi } from './api';
import { dateFormat } from '@/common/utils/date';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn, TableQuery } from '@/components/pagetable';
import { hasPerms } from '@/components/auth/auth';
import SvgIcon from '@/components/svgIcon/index.vue';
import { getDbDialect } from './dialect';

const InstanceEdit = defineAsyncComponent(() => import('./InstanceEdit.vue'));

const perms = {
    saveInstance: 'db:instance:save',
    delInstance: 'db:instance:del',
};

const queryConfig = [TableQuery.text('name', '名称')];

const columns = ref([
    TableColumn.new('name', '名称'),
    TableColumn.new('type', '类型').isSlot().setAddWidth(-15).alignCenter(),
    TableColumn.new('host', 'host:port').setFormatFunc((data: any) => `${data.host}:${data.port}`),
    TableColumn.new('username', '用户名'),
    TableColumn.new('params', '连接参数'),
    TableColumn.new('remark', '备注'),
]);

// 该用户拥有的的操作列按钮权限
const actionBtns = hasPerms([perms.saveInstance]);
const actionColumn = TableColumn.new('action', '操作').isSlot().setMinWidth(110).fixedRight().alignCenter();

const pageTableRef: any = ref(null);

const state = reactive({
    row: {},
    dbId: 0,
    db: '',
    /**
     * 选中的数据
     */
    selectionData: [],
    /**
     * 查询条件
     */
    query: {
        name: null,
        pageNum: 1,
        pageSize: 0,
    },
    datas: [],
    total: 0,
    infoDialog: {
        visible: false,
        data: null as any,
    },
    instanceEditDialog: {
        visible: false,
        data: null as any,
        title: '新增数据库实例',
    },
});

const { selectionData, query, datas, total, infoDialog, instanceEditDialog } = toRefs(state);

onMounted(async () => {
    if (Object.keys(actionBtns).length > 0) {
        columns.value.push(actionColumn);
    }
    search();
});

const search = async () => {
    try {
        pageTableRef.value.loading(true);
        let res: any = await dbApi.instances.request(state.query);
        state.datas = res.list;
        state.total = res.total;
    } finally {
        pageTableRef.value.loading(false);
    }
};

const showInfo = (info: any) => {
    state.infoDialog.data = info;
    state.infoDialog.visible = true;
};

const editInstance = async (data: any) => {
    if (!data) {
        state.instanceEditDialog.data = null;
        state.instanceEditDialog.title = '新增数据库实例';
    } else {
        state.instanceEditDialog.data = data;
        state.instanceEditDialog.title = '修改数据库实例';
    }
    state.instanceEditDialog.visible = true;
};

const valChange = () => {
    search();
};

const deleteInstance = async () => {
    try {
        await ElMessageBox.confirm(`确定删除数据库实例【${state.selectionData.map((x: any) => x.name).join(', ')}】?`, '提示', {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning',
        });
        await dbApi.deleteInstance.request({ id: state.selectionData.map((x: any) => x.id).join(',') });
        ElMessage.success('删除成功');
        search();
    } catch (err) {
        //
    }
};
</script>
<style lang="scss"></style>
