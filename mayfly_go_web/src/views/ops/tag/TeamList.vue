<template>
    <div>
        <page-table
            ref="pageTableRef"
            :page-api="tagApi.getTeams"
            :search-items="searchItems"
            v-model:query-form="query"
            :show-selection="true"
            v-model:selection-data="selectionData"
            :columns="columns"
        >
            <template #tableHeader>
                <el-button v-auth="'team:save'" type="primary" icon="plus" @click="showSaveTeamDialog(false)">添加</el-button>
                <el-button v-auth="'team:del'" :disabled="selectionData.length < 1" @click="deleteTeam()" type="danger" icon="delete">删除</el-button>
            </template>

            <template #tags="{ data }">
                <TagCodePath :path="data.tags?.map((tag: any) => tag.codePath)" />
            </template>

            <template #validityDate="{ data }"> {{ data.validityStartDate }} ~ {{ data.validityEndDate }} </template>

            <template #action="{ data }">
                <el-button @click.prevent="showMembers(data)" link type="primary">成员</el-button>

                <el-button v-auth="'team:save'" @click.prevent="showSaveTeamDialog(data)" link type="warning">编辑</el-button>
            </template>
        </page-table>

        <el-drawer
            :title="addTeamDialog.form.id ? '编辑团队' : '添加团队'"
            v-model="addTeamDialog.visible"
            :before-close="cancelSaveTeam"
            :destroy-on-close="true"
            :close-on-click-modal="false"
        >
            <template #header>
                <DrawerHeader :header="addTeamDialog.form.id ? '编辑团队' : '添加团队'" :back="cancelSaveTeam" />
            </template>

            <el-form ref="teamForm" :model="addTeamDialog.form" :rules="teamFormRules" label-width="auto">
                <el-form-item prop="name" label="团队名" required>
                    <el-input :disabled="addTeamDialog.form.id" v-model="addTeamDialog.form.name" auto-complete="off"></el-input>
                </el-form-item>

                <el-form-item prop="validityDate" label="生效时间" required>
                    <el-date-picker
                        v-model="addTeamDialog.form.validityDate"
                        type="datetimerange"
                        start-placeholder="生效开始时间"
                        end-placeholder="生效结束时间"
                        format="YYYY-MM-DD HH:mm:ss"
                        value-format="YYYY-MM-DD HH:mm:ss"
                        date-format="YYYY-MM-DD"
                        time-format="HH:mm:ss"
                    />
                </el-form-item>

                <el-form-item label="备注">
                    <el-input v-model="addTeamDialog.form.remark" auto-complete="off"></el-input>
                </el-form-item>

                <el-form-item prop="tag" label="标签">
                    <TagTreeCheck height="calc(100vh - 390px)" v-model="state.addTeamDialog.form.codePaths" :tag-type="0" />
                </el-form-item>
            </el-form>
            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="cancelSaveTeam()">取 消</el-button>
                    <el-button @click="saveTeam" type="primary">确 定</el-button>
                </div>
            </template>
        </el-drawer>

        <el-dialog @open="setMemebers" width="50%" :title="showMemDialog.title" v-model="showMemDialog.visible">
            <page-table
                ref="showMemPageTableRef"
                :page-api="tagApi.getTeamMem"
                :lazy="true"
                :search-items="showMemDialog.searchItems"
                v-model:query-form="showMemDialog.query"
                :columns="showMemDialog.columns"
            >
                <template #tableHeader>
                    <el-button v-auth="'team:member:save'" @click="showAddMemberDialog()" type="primary" icon="plus">添加</el-button>
                </template>

                <template #action="{ data }">
                    <el-button v-auth="'team:member:del'" @click="deleteMember(data)" type="danger" link icon="delete"></el-button>
                </template>
            </page-table>

            <el-dialog width="400px" title="添加成员" :before-close="cancelAddMember" v-model="showMemDialog.addVisible">
                <el-form :model="showMemDialog.memForm" label-width="auto">
                    <AccountSelectFormItem v-model="showMemDialog.memForm.accountIds" multiple focus />
                </el-form>
                <template #footer>
                    <div class="dialog-footer">
                        <el-button @click="cancelAddMember()">取 消</el-button>
                        <el-button @click="addMember" type="primary">确 定</el-button>
                    </div>
                </template>
            </el-dialog>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { ref, toRefs, reactive, onMounted, Ref } from 'vue';
import { tagApi } from './api';
import { ElMessage, ElMessageBox } from 'element-plus';
import { notBlank } from '@/common/assert';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { SearchItem } from '@/components/SearchForm';
import AccountSelectFormItem from '@/views/system/account/components/AccountSelectFormItem.vue';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';
import TagTreeCheck from '../component/TagTreeCheck.vue';
import TagCodePath from '../component/TagCodePath.vue';
import { formatDate } from '@/common/utils/format';

const teamForm: any = ref(null);
const pageTableRef: Ref<any> = ref(null);
const showMemPageTableRef: Ref<any> = ref(null);

const teamFormRules = {
    name: [
        {
            required: true,
            message: '请输入团队名',
            trigger: ['change', 'blur'],
        },
    ],
    validityDate: [
        {
            required: true,
            message: '请选择生效时间',
            trigger: ['change', 'blur'],
        },
    ],
};

const searchItems = [SearchItem.input('name', '团队名称')];
const columns = [
    TableColumn.new('name', '团队名称'),
    TableColumn.new('tags', '分配标签').isSlot().setAddWidth(40),
    TableColumn.new('validityDate', '有效期').isSlot('validityDate').setMinWidth(310),
    TableColumn.new('remark', '备注'),
    TableColumn.new('creator', '创建者'),
    TableColumn.new('createTime', '创建时间').isTime(),
    TableColumn.new('modifier', '修改者'),
    TableColumn.new('updateTime', '修改时间').isTime(),
    TableColumn.new('action', '操作').isSlot().setMinWidth(120).fixedRight().alignCenter(),
];

const state = reactive({
    currentEditPermissions: false,
    addTeamDialog: {
        visible: false,
        form: { id: 0, name: '', validityDate: ['', ''], validityStartDate: '', validityEndDate: '', remark: '', codePaths: [] },
    },
    query: {
        pageNum: 1,
        pageSize: 0,
        name: null,
    },
    selectionData: [],
    showMemDialog: {
        searchItems: [SearchItem.input('username', '用户名').withSpan(2)],
        columns: [
            TableColumn.new('name', '姓名'),
            TableColumn.new('username', '账号'),
            TableColumn.new('createTime', '加入时间').isTime(),
            TableColumn.new('creator', '分配者'),
            TableColumn.new('action', '操作').isSlot().setMinWidth(80).fixedRight().alignCenter(),
        ],
        visible: false,
        query: {
            pageSize: 10,
            pageNum: 1,
            teamId: null,
            username: null,
        },
        members: {
            list: [],
            total: 0,
        },
        title: '',
        addVisible: false,
        memForm: {
            accountIds: [] as any,
            teamId: 0 as any,
        },
        accounts: Array(),
    },
});

const { query, addTeamDialog, selectionData, showMemDialog } = toRefs(state);

onMounted(() => {});

const search = async () => {
    pageTableRef.value.search();
};

const showSaveTeamDialog = async (data: any) => {
    if (data) {
        state.addTeamDialog.form.id = data.id;
        state.addTeamDialog.form.name = data.name;
        state.addTeamDialog.form.validityDate = [data.validityStartDate, data.validityEndDate];
        state.addTeamDialog.form.remark = data.remark;
        state.addTeamDialog.form.codePaths = data.tags?.map((tag: any) => tag.codePath);
    } else {
        let end = new Date();
        end.setFullYear(end.getFullYear() + 10);
        state.addTeamDialog.form.validityDate = [formatDate(new Date()), formatDate(end)];
    }

    state.addTeamDialog.visible = true;
};

const saveTeam = async () => {
    try {
        await teamForm.value.validate();
    } catch (e: any) {
        ElMessage.error('请正确填写信息');
        return false;
    }

    const form = state.addTeamDialog.form;
    form.validityStartDate = form.validityDate[0];
    form.validityEndDate = form.validityDate[1];
    await tagApi.saveTeam.request(form);
    ElMessage.success('保存成功');
    search();
    cancelSaveTeam();
};

const cancelSaveTeam = () => {
    state.addTeamDialog.visible = false;
    teamForm.value.resetFields();
    setTimeout(() => {
        state.addTeamDialog.form = {} as any;
    }, 500);
};

const deleteTeam = () => {
    ElMessageBox.confirm(`此操作将删除【${state.selectionData.map((x: any) => x.name).join(', ')}】团队信息, 是否继续?`, '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
    }).then(async () => {
        await tagApi.delTeam.request({ id: state.selectionData.map((x: any) => x.id).join(',') });
        ElMessage.success('删除成功！');
        search();
    });
};

/********** 团队成员相关 ***********/

const showMembers = async (team: any) => {
    state.showMemDialog.query.teamId = team.id;
    state.showMemDialog.visible = true;
    state.showMemDialog.title = `[${team.name}] 成员信息`;
};

const deleteMember = async (data: any) => {
    await tagApi.delTeamMem.request(data);
    ElMessage.success('移除成功');
    // 重新赋值成员列表
    setMemebers();
};

/**
 * 设置成员列表信息
 */
const setMemebers = async () => {
    showMemPageTableRef.value.search();
};

const showAddMemberDialog = () => {
    state.showMemDialog.addVisible = true;
};

const addMember = async () => {
    const memForm = state.showMemDialog.memForm;
    memForm.teamId = state.showMemDialog.query.teamId;
    notBlank(memForm.accountIds, '请先选择账号');

    await tagApi.saveTeamMem.request(memForm);
    ElMessage.success('保存成功');
    setMemebers();
    cancelAddMember();
};

const cancelAddMember = () => {
    state.showMemDialog.memForm = {} as any;
    state.showMemDialog.addVisible = false;
};
</script>
<style lang="scss" scoped></style>
