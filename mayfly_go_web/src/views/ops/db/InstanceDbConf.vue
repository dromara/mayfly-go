<template>
    <div>
        <el-drawer :title="title" v-model="dialogVisible" :before-close="cancel" :destroy-on-close="true" :close-on-click-modal="false" size="50%">
            <template #header>
                <DrawerHeader :header="title" :back="cancel" />
            </template>

            <el-table :data="state.dbs" stripe>
                <el-table-column prop="name" label="名称" show-overflow-tooltip min-width="100"> </el-table-column>
                <el-table-column prop="authCertName" label="授权凭证" min-width="120" show-overflow-tooltip> </el-table-column>
                <el-table-column prop="database" label="库" min-width="80">
                    <template #default="scope">
                        <el-popover placement="bottom" :width="200" trigger="click">
                            <template #reference>
                                <el-button @click="state.currentDbs = scope.row.database" type="primary" link>查看库</el-button>
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
                </el-table-column>

                <el-table-column prop="remark" label="备注" show-overflow-tooltip min-width="120"> </el-table-column>
                <el-table-column prop="code" label="编号" show-overflow-tooltip min-width="120"> </el-table-column>
                <el-table-column min-wdith="120px">
                    <template #header>
                        操作
                        <el-button v-auth="perms.saveDb" type="primary" circle size="small" icon="Plus" @click="editDb(null)"> </el-button>
                    </template>
                    <template #default="scope">
                        <el-button v-auth="perms.saveDb" @click="editDb(scope.row)" type="primary" icon="edit" link></el-button>
                        <el-button class="ml1" v-auth="perms.delDb" type="danger" @click="deleteDb(scope.row)" icon="delete" link></el-button>
                    </template>
                </el-table-column>
            </el-table>

            <db-edit
                @confirm="confirmEditDb"
                @cancel="cancelEditDb"
                :title="dbEditDialog.title"
                v-model:visible="dbEditDialog.visible"
                :instance="props.instance"
                v-model:db="dbEditDialog.data"
            ></db-edit>
        </el-drawer>
    </div>
</template>

<script lang="ts" setup>
import { computed, reactive, toRefs, watchEffect } from 'vue';
import { dbApi } from './api';
import { ElMessage, ElMessageBox } from 'element-plus';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';
import DbEdit from './DbEdit.vue';

const props = defineProps({
    visible: {
        type: Boolean,
    },
    instance: {
        type: [Object],
        required: true,
    },
    title: {
        type: String,
    },
});

const perms = {
    base: 'db',
    saveDb: 'db:save',
    delDb: 'db:del',
};

//定义事件
const emit = defineEmits(['update:visible', 'cancel', 'val-change']);

const state = reactive({
    dialogVisible: false,
    dbs: [] as any,
    currentDbs: '', // 当前数据库名，空格分割库名
    dbNameSearch: '',
    dbEditDialog: {
        visible: false,
        data: null as any,
        title: '新增数据库',
    },
});

const { dialogVisible, dbEditDialog } = toRefs(state);

watchEffect(() => {
    state.dialogVisible = props.visible;
    if (!state.dialogVisible) {
        return;
    }

    getDbs();
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

const cancel = () => {
    emit('update:visible', false);
    emit('cancel');
};

const getDbs = () => {
    dbApi.dbs.request({ pageSize: 200, instanceId: props.instance.id }).then((res: any) => {
        state.dbs = res.list || [];
    });
};

const editDb = (data: any) => {
    if (data) {
        state.dbEditDialog.data = { ...data };
    } else {
        state.dbEditDialog.data = {
            instanceId: props.instance.id,
        };
    }
    state.dbEditDialog.title = data ? '编辑数据库' : '新增数据库';
    state.dbEditDialog.visible = true;
};

const deleteDb = async (db: any) => {
    try {
        await ElMessageBox.confirm(`确定删除【${db.name}】库?`, '提示', {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning',
        });
        await dbApi.deleteDb.request({ id: db.id });
        ElMessage.success('删除成功');
        getDbs();
    } catch (err) {
        //
    }
};

const confirmEditDb = async (db: any) => {
    db.instanceId = props.instance.id;
    await dbApi.saveDb.request(db);
    ElMessage.success('保存成功');
    getDbs();
    cancelEditDb();
};

const cancelEditDb = () => {
    state.dbEditDialog.visible = false;
    state.dbEditDialog.data = {};
};
</script>
<style lang="scss"></style>
