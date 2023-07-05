<template>
    <div>
        <page-table :query="state.queryConfig" v-model:query-form="query" :show-selection="true"
            v-model:selection-data="selectionData" :data="data" :columns="state.columns" :total="total"
            v-model:page-size="query.pageSize" v-model:page-num="query.pageNum" @pageChange="search()">

            <template #queryRight>
                <el-button v-auth="'team:save'" type="primary" icon="plus" @click="showSaveTeamDialog(false)">添加</el-button>
                <el-button v-auth="'team:del'" :disabled="selectionData.length < 1" @click="deleteTeam()" type="danger"
                    icon="delete">删除</el-button>

            </template>

            <template #tagPath="{ data }">
                <tag-info :tag-path="data.tagPath" />
                <span class="ml5">
                    {{ data.tagPath }}
                </span>
            </template>

            <template #action="{ data }">
                <el-button @click.prevent="showMembers(data)" link type="primary">成员</el-button>

                <el-button @click.prevent="showTags(data)" link type="success">标签</el-button>

                <el-button v-auth="'team:save'" @click.prevent="showSaveTeamDialog(data)" link type="warning">编辑</el-button>
            </template>
        </page-table>

        <el-dialog width="400px" title="团队编辑" :before-close="cancelSaveTeam" v-model="addTeamDialog.visible">
            <el-form ref="teamForm" :model="addTeamDialog.form" label-width="auto">
                <el-form-item prop="name" label="团队名:" required>
                    <el-input v-model="addTeamDialog.form.name" auto-complete="off"></el-input>
                </el-form-item>
                <el-form-item label="备注:">
                    <el-input v-model="addTeamDialog.form.remark" auto-complete="off"></el-input>
                </el-form-item>
            </el-form>
            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="cancelSaveTeam()">取 消</el-button>
                    <el-button @click="saveTeam" type="primary">确 定</el-button>
                </div>
            </template>
        </el-dialog>

        <el-dialog width="500px" :title="showTagDialog.title" :before-close="closeTagDialog"
            v-model="showTagDialog.visible">
            <el-form label-width="auto">
                <el-form-item prop="tag" label="标签:">
                    <el-tree-select ref="tagTreeRef" style="width: 100%" v-model="showTagDialog.tagTreeTeams"
                        :data="showTagDialog.tags" :default-expanded-keys="showTagDialog.tagTreeTeams" multiple
                        :render-after-expand="true" show-checkbox check-strictly node-key="id" :props="showTagDialog.props"
                        @check="tagTreeNodeCheck">
                        <template #default="{ data }">
                            <span class="custom-tree-node">
                                <span style="font-size: 13px">
                                    {{ data.code }}
                                    <span style="color: #3c8dbc">【</span>
                                    {{ data.name }}
                                    <span style="color: #3c8dbc">】</span>
                                    <el-tag v-if="data.children !== null" size="small">{{ data.children.length }}
                                    </el-tag>
                                </span>
                            </span>
                        </template>
                    </el-tree-select>
                </el-form-item>
            </el-form>
            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="closeTagDialog()">取 消</el-button>
                    <el-button v-auth="'team:tag:save'" @click="saveTags()" type="primary">确 定</el-button>
                </div>
            </template>
        </el-dialog>

        <el-dialog width="700px" :title="showMemDialog.title" v-model="showMemDialog.visible">
            <div class="toolbar">
                <el-button v-auth="'team:member:save'" @click="showAddMemberDialog()" type="primary" icon="plus"
                    size="small">添加</el-button>
                <el-button v-auth="'team:member:del'" @click="deleteMember" :disabled="showMemDialog.chooseId == null"
                    type="danger" icon="delete" size="small">移除</el-button>
                <div style="float: right">
                    <el-input placeholder="请输入用户名" class="mr2" style="width: 150px" v-model="showMemDialog.query.username"
                        size="small" @clear="search" clearable></el-input>
                    <el-button @click="setMemebers" type="success" icon="search" size="small"></el-button>
                </div>
            </div>
            <el-table @current-change="chooseMember" border :data="showMemDialog.members.list" size="small">
                <el-table-column label="选择" width="50px">
                    <template #default="scope">
                        <el-radio v-model="showMemDialog.chooseId" :label="scope.row.id">
                            <i></i>
                        </el-radio>
                    </template>
                </el-table-column>
                <el-table-column property="name" label="姓名" width="115"></el-table-column>
                <el-table-column property="username" label="账号" width="135"></el-table-column>
                <el-table-column property="createTime" label="加入时间">
                    <template #default="scope">
                        {{ dateFormat(scope.row.createTime) }}
                    </template>
                </el-table-column>
                <el-table-column property="creator" label="分配者" width="135"></el-table-column>
            </el-table>
            <el-pagination size="small" @current-change="setMemebers" style="text-align: center" background
                layout="prev, pager, next, total, jumper" :total="showMemDialog.members.total"
                v-model:current-page="showMemDialog.query.pageNum" :page-size="showMemDialog.query.pageSize" />

            <el-dialog width="400px" title="添加成员" :before-close="cancelAddMember" v-model="showMemDialog.addVisible">
                <el-form :model="showMemDialog.memForm" label-width="auto">
                    <el-form-item label="账号:">
                        <el-select style="width: 100%" remote :remote-method="getAccount"
                            v-model="showMemDialog.memForm.accountIds" filterable multiple placeholder="请输入账号模糊搜索并选择">
                            <el-option v-for="item in showMemDialog.accounts" :key="item.id"
                                :label="`${item.username} [${item.name}]`" :value="item.id">
                            </el-option>
                        </el-select>
                    </el-form-item>
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
import { ref, toRefs, reactive, onMounted } from 'vue';
import { tagApi } from './api';
import { accountApi } from '../../system/api';
import { ElMessage, ElMessageBox } from 'element-plus';
import { dateFormat } from '@/common/utils/date';
import { notBlank } from '@/common/assert';
import PageTable from '@/components/pagetable/PageTable.vue'
import { TableColumn, TableQuery } from '@/components/pagetable';

const teamForm: any = ref(null);
const tagTreeRef: any = ref(null);
const state = reactive({
    currentEditPermissions: false,
    addTeamDialog: {
        title: '新增团队',
        visible: false,
        form: { id: 0, name: '', remark: '' },
    },
    query: {
        pageNum: 1,
        pageSize: 10,
        name: null,
    },
    queryConfig: [
        TableQuery.text("name", "团队名称"),
    ],
    columns: [
        TableColumn.new("name", "团队名称"),
        TableColumn.new("remark", "备注"),
        TableColumn.new("createTime", "创建时间").isTime(),
        TableColumn.new("creator", "创建人"),
        TableColumn.new("action", "操作").isSlot().setMinWidth(100).fixedRight().alignCenter(),
    ],
    total: 0,
    data: [],
    selectionData: [],
    showMemDialog: {
        visible: false,
        chooseId: 0,
        chooseData: null,
        query: {
            pageSize: 10,
            pageNum: 1,
            teamId: null,
            username: null,
        },
        members: {
            list: [],
            total: null,
        },
        title: '',
        addVisible: false,
        memForm: {
            accountIds: [] as any,
            teamId: 0 as any,
        },
        accounts: Array(),
    },
    showTagDialog: {
        title: '项目信息',
        visible: false,
        tags: [],
        teamId: 0,
        tagTreeTeams: [] as any,
        props: {
            value: 'id',
            label: 'codePath',
            children: 'children',
        },
    },
});

const {
    query,
    addTeamDialog,
    total,
    data,
    selectionData,
    showMemDialog,
    showTagDialog,
} = toRefs(state)

onMounted(() => {
    search();
});

const search = async () => {
    let res = await tagApi.getTeams.request(state.query);
    state.data = res.list;
    state.total = res.total;
};

const showSaveTeamDialog = (data: any) => {
    if (data) {
        state.addTeamDialog.form.id = data.id;
        state.addTeamDialog.form.name = data.name;
        state.addTeamDialog.form.remark = data.remark;
        state.addTeamDialog.title = `修改 [${data.codePath}] 信息`;
    }
    state.addTeamDialog.visible = true;
};

const saveTeam = async () => {
    teamForm.value.validate(async (valid: any) => {
        if (valid) {
            const form = state.addTeamDialog.form;
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
    ElMessageBox.confirm(`此操作将删除【${state.selectionData.map((x: any) => x.name).join(", ")}】团队信息, 是否继续?`, '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
    }).then(async () => {
        await tagApi.delTeam.request({ id: state.selectionData.map((x: any) => x.id).join(",") });
        ElMessage.success('删除成功！');
        search();
    });
};

/********** 团队成员相关 ***********/

const showMembers = async (team: any) => {
    state.showMemDialog.query.teamId = team.id;
    await setMemebers();
    state.showMemDialog.title = `[${team.name}] 成员信息`;
    state.showMemDialog.visible = true;
};

const getAccount = (username: any) => {
    if (username) {
        accountApi.list.request({ username }).then((res) => {
            state.showMemDialog.accounts = res.list;
        });
    }
};

/**
 * 选中成员
 */
const chooseMember = (item: any) => {
    if (!item) {
        return;
    }
    state.showMemDialog.chooseData = item;
    state.showMemDialog.chooseId = item.id;
};

const deleteMember = async () => {
    await tagApi.delTeamMem.request(state.showMemDialog.chooseData);
    ElMessage.success('移除成功');
    // 重新赋值成员列表
    setMemebers();
};

/**
 * 设置成员列表信息
 */
const setMemebers = async () => {
    const res = await tagApi.getTeamMem.request(state.showMemDialog.query);
    state.showMemDialog.members.list = res.list;
    state.showMemDialog.members.total = res.total;
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
    state.showMemDialog.chooseData = null;
    state.showMemDialog.chooseId = 0;
};

/********** 标签相关 ***********/

const showTags = async (team: any) => {
    state.showTagDialog.tags = await tagApi.getTagTrees.request(null);
    state.showTagDialog.tagTreeTeams = await tagApi.getTeamTagIds.request({ teamId: team.id });
    state.showTagDialog.title = `[${team.name}] 团队标签信息`;
    state.showTagDialog.teamId = team.id;
    state.showTagDialog.visible = true;
};

const closeTagDialog = () => {
    state.showTagDialog.visible = false;
    setTimeout(() => {
        state.showTagDialog.tagTreeTeams = [];
    }, 500);
};

const saveTags = async () => {
    await tagApi.saveTeamTags.request({
        teamId: state.showTagDialog.teamId,
        tagIds: state.showTagDialog.tagTreeTeams,
    });
    ElMessage.success('保存成功');
    closeTagDialog();
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
<style lang="scss"></style>
