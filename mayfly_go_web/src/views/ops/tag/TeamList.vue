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

            <template #tagPath="{ data }">
                <tag-info :tag-path="data.tagPath" />
                <span class="ml5">
                    {{ data.tagPath }}
                </span>
            </template>

            <template #action="{ data }">
                <el-button @click.prevent="showMembers(data)" link type="primary">成员</el-button>

                <el-button v-auth="'team:save'" @click.prevent="showSaveTeamDialog(data)" link type="warning">编辑</el-button>
            </template>
        </page-table>

        <el-drawer title="团队编辑" v-model="addTeamDialog.visible" :before-close="cancelSaveTeam" :destroy-on-close="true" :close-on-click-modal="false">
            <template #header>
                <DrawerHeader header="团队编辑" :back="cancelSaveTeam" />
            </template>

            <el-form ref="teamForm" :model="addTeamDialog.form" label-width="auto">
                <el-form-item prop="name" label="团队名" required>
                    <el-input :disabled="addTeamDialog.form.id" v-model="addTeamDialog.form.name" auto-complete="off"></el-input>
                </el-form-item>
                <el-form-item label="备注">
                    <el-input v-model="addTeamDialog.form.remark" auto-complete="off"></el-input>
                </el-form-item>

                <el-form-item prop="tag" label="标签">
                    <el-scrollbar style="height: calc(100vh - 300px); border: 1px solid var(--el-border-color)">
                        <el-tree
                            ref="tagTreeRef"
                            style="width: 100%"
                            :data="state.tags"
                            :default-expanded-keys="state.addTeamDialog.form.tags"
                            :default-checked-keys="state.addTeamDialog.form.tags"
                            multiple
                            :render-after-expand="true"
                            show-checkbox
                            check-strictly
                            node-key="id"
                            :props="{
                                value: 'id',
                                label: 'codePath',
                                children: 'children',
                            }"
                            @check="tagTreeNodeCheck"
                        >
                            <template #default="{ data }">
                                <span class="custom-tree-node">
                                    <SvgIcon :name="EnumValue.getEnumByValue(TagResourceTypeEnum, data.type)?.extra.icon" />

                                    <span class="font13 ml5">
                                        {{ data.code }}
                                        <span style="color: #3c8dbc">【</span>
                                        {{ data.name }}
                                        <span style="color: #3c8dbc">】</span>
                                        <el-tag v-if="data.children !== null" size="small">{{ data.children.length }} </el-tag>
                                    </span>
                                </span>
                            </template>
                        </el-tree>
                    </el-scrollbar>
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
import { TagResourceTypeEnum } from '@/common/commonEnum';
import EnumValue from '@/common/Enum';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';

const teamForm: any = ref(null);
const tagTreeRef: any = ref(null);
const pageTableRef: Ref<any> = ref(null);
const showMemPageTableRef: Ref<any> = ref(null);

const searchItems = [SearchItem.input('name', '团队名称')];
const columns = [
    TableColumn.new('name', '团队名称'),
    TableColumn.new('remark', '备注'),
    TableColumn.new('creator', '创建者'),
    TableColumn.new('createTime', '创建时间').isTime(),
    TableColumn.new('modifier', '修改者'),
    TableColumn.new('updateTime', '修改时间').isTime(),
    TableColumn.new('action', '操作').isSlot().setMinWidth(120).fixedRight().alignCenter(),
];

const state = reactive({
    currentEditPermissions: false,
    tags: [],
    addTeamDialog: {
        title: '新增团队',
        visible: false,
        form: { id: 0, name: '', remark: '', tags: [] },
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
    if (state.tags.length == 0) {
        state.tags = await tagApi.getTagTrees.request(null);
    }

    if (data) {
        state.addTeamDialog.form.id = data.id;
        state.addTeamDialog.form.name = data.name;
        state.addTeamDialog.form.remark = data.remark;
        state.addTeamDialog.title = `修改 [${data.codePath}] 信息`;
        state.addTeamDialog.form.tags = await tagApi.getTeamTagIds.request({ teamId: data.id });
    }

    state.addTeamDialog.visible = true;
};

const saveTeam = async () => {
    teamForm.value.validate(async (valid: any) => {
        if (valid) {
            const form = state.addTeamDialog.form;
            form.tags = tagTreeRef.value.getCheckedKeys(false);
            await tagApi.saveTeam.request(form);
            ElMessage.success('保存成功');
            search();
            cancelSaveTeam();
        }
    });
};

const cancelSaveTeam = () => {
    state.addTeamDialog.visible = false;
    state.addTeamDialog.form = {} as any;
    teamForm.value.resetFields();
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

const tagTreeNodeCheck = () => {
    // const node = tagTreeRef.value.getNode(data.id);
    // console.log(node);
    // // state.showTagDialog.tagTreeTeams = [16]
    // if (node.checked) {
    //     if (node.parent) {
    //         console.log(node.parent);
    //         // removeCheckedTagId(node.parent.key);
    //         tagTreeRef.value.setChecked(node.parent, false, false);
    //     }
    //     // // parentNode = node.parent
    //     // for (let parentNode of node.parent) {
    //     //     parentNode.setChecked(false);
    //     // }
    // }
    // console.log(data);
    // console.log(checkInfo);
};

// function removeCheckedTagId(id: any) {
//     console.log(state.showTagDialog.tagTreeTeams);
//     for (let i = 0; i < state.showTagDialog.tagTreeTeams.length; i++) {
//         if (state.showTagDialog.tagTreeTeams[i] == id) {
//             console.log('has id', id);
//             state.showTagDialog.tagTreeTeams.splice(i, 1);
//         }
//     }
//     console.log(state.showTagDialog.tagTreeTeams);
// }
</script>
<style lang="scss" scoped></style>
