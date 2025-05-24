<template>
    <div class="h-full">
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
                <el-button v-auth="'team:save'" type="primary" icon="plus" @click="onShowSaveTeamDialog(false)">{{ $t('common.create') }}</el-button>
                <el-button v-auth="'team:del'" :disabled="selectionData.length < 1" @click="onDeleteTeam()" type="danger" icon="delete">
                    {{ $t('common.delete') }}
                </el-button>
            </template>

            <template #tags="{ data }">
                <TagCodePath :path="data.tags" />
            </template>

            <template #validityDate="{ data }"> {{ formatDate(data.validityStartDate) }} ~ {{ formatDate(data.validityEndDate) }} </template>

            <template #action="{ data }">
                <el-button @click.prevent="onShowMembers(data)" link type="primary">{{ $t('team.member') }}</el-button>

                <el-button v-auth="'team:save'" @click.prevent="onShowSaveTeamDialog(data)" link type="warning">{{ $t('common.edit') }}</el-button>
            </template>
        </page-table>

        <el-drawer
            :title="addTeamDialog.title"
            v-model="addTeamDialog.visible"
            :before-close="onCancelSaveTeam"
            :destroy-on-close="true"
            :close-on-click-modal="false"
            size="40%"
        >
            <template #header>
                <DrawerHeader :header="addTeamDialog.title" :back="onCancelSaveTeam" />
            </template>

            <el-form ref="teamForm" :model="addTeamDialog.form" :rules="teamFormRules" label-width="auto">
                <el-form-item prop="name" :label="$t('common.name')" required>
                    <el-input :disabled="addTeamDialog.form.id" v-model="addTeamDialog.form.name" auto-complete="off"></el-input>
                </el-form-item>

                <el-form-item prop="validityDate" :label="$t('team.validity')" required>
                    <el-date-picker
                        v-model="addTeamDialog.form.validityDate"
                        type="datetimerange"
                        :start-placeholder="$t('team.effectiveStartTime')"
                        :end-placeholder="$t('team.effectiveEndTime')"
                        format="YYYY-MM-DD HH:mm:ss"
                        value-format="YYYY-MM-DD HH:mm:ss"
                        date-format="YYYY-MM-DD"
                        time-format="HH:mm:ss"
                    />
                </el-form-item>

                <el-form-item :label="$t('common.remark')">
                    <el-input v-model="addTeamDialog.form.remark" auto-complete="off"></el-input>
                </el-form-item>

                <el-form-item prop="tag" :label="$t('common.tag')">
                    <TagTreeCheck height="calc(100vh - 390px)" v-model="state.addTeamDialog.form.codePaths" :tag-type="0" />
                </el-form-item>
            </el-form>
            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="onCancelSaveTeam()">{{ $t('common.cancel') }}</el-button>
                    <el-button @click="onSaveTeam" type="primary">{{ $t('common.confirm') }}</el-button>
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
                    <el-button v-auth="'team:member:save'" @click="onShowAddMemberDialog()" type="primary" icon="plus">{{ $t('common.add') }}</el-button>
                </template>

                <template #action="{ data }">
                    <el-button v-auth="'team:member:del'" @click="onDeleteMember(data)" type="danger" link icon="delete"></el-button>
                </template>
            </page-table>

            <el-dialog width="400px" :title="$t('team.addMember')" :before-close="onCancelAddMember" v-model="showMemDialog.addVisible">
                <el-form :model="showMemDialog.memForm" label-width="auto">
                    <AccountSelectFormItem v-model="showMemDialog.memForm.accountIds" multiple focus />
                </el-form>
                <template #footer>
                    <div class="dialog-footer">
                        <el-button @click="onCancelAddMember()">{{ $t('common.cancel') }}</el-button>
                        <el-button @click="onAddMember" type="primary">{{ $t('common.confirm') }}</el-button>
                    </div>
                </template>
            </el-dialog>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { ref, toRefs, reactive, onMounted, Ref } from 'vue';
import { tagApi } from './api';
import { notBlank } from '@/common/assert';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { SearchItem } from '@/components/SearchForm';
import AccountSelectFormItem from '@/views/system/account/components/AccountSelectFormItem.vue';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';
import TagTreeCheck from '../component/TagTreeCheck.vue';
import TagCodePath from '../component/TagCodePath.vue';
import { formatDate } from '@/common/utils/format';
import { useI18n } from 'vue-i18n';
import {
    useI18nCreateTitle,
    useI18nDeleteConfirm,
    useI18nDeleteSuccessMsg,
    useI18nEditTitle,
    useI18nFormValidate,
    useI18nOperateSuccessMsg,
    useI18nSaveSuccessMsg,
} from '@/hooks/useI18n';
import { Rules } from '@/common/rule';

const { t } = useI18n();

const teamForm: any = ref(null);
const pageTableRef: Ref<any> = ref(null);
const showMemPageTableRef: Ref<any> = ref(null);

const teamFormRules = {
    name: [Rules.requiredInput('common.name')],
    validityDate: [Rules.requiredSelect('team.validity')],
};

const searchItems = [SearchItem.input('name', 'common.name')];
const columns = [
    TableColumn.new('name', 'common.name'),
    TableColumn.new('tags', 'team.allocateTag').isSlot().setAddWidth(40),
    TableColumn.new('validityDate', 'team.validity').isSlot('validityDate').setMinWidth(310),
    TableColumn.new('remark', 'common.remark'),
    TableColumn.new('creator', 'common.creator'),
    TableColumn.new('createTime', 'common.createTime').isTime(),
    TableColumn.new('modifier', 'common.modifier'),
    TableColumn.new('updateTime', 'common.updateTime').isTime(),
    TableColumn.new('action', 'common.operation').isSlot().setMinWidth(130).fixedRight().noShowOverflowTooltip().alignCenter(),
];

const state = reactive({
    currentEditPermissions: false,
    addTeamDialog: {
        title: '',
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
        searchItems: [SearchItem.input('username', 'common.username').withSpan(2)],
        columns: [
            TableColumn.new('name', 'team.accountName'),
            TableColumn.new('username', 'common.username'),
            TableColumn.new('createTime', 'team.joinTime').isTime(),
            TableColumn.new('creator', 'team.assigner'),
            TableColumn.new('action', 'common.operation').isSlot().setMinWidth(80).fixedRight().alignCenter(),
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

const onShowSaveTeamDialog = async (data: any) => {
    if (data) {
        state.addTeamDialog.title = useI18nEditTitle('team.team');
        state.addTeamDialog.form.id = data.id;
        state.addTeamDialog.form.name = data.name;
        state.addTeamDialog.form.validityDate = [data.validityStartDate, data.validityEndDate];
        state.addTeamDialog.form.remark = data.remark;
        state.addTeamDialog.form.codePaths = data.tags?.map((tag: any) => tag.codePath);
    } else {
        state.addTeamDialog.title = useI18nCreateTitle('team.team');
        let end = new Date();
        end.setFullYear(end.getFullYear() + 10);
        state.addTeamDialog.form.validityDate = [formatDate(new Date()), formatDate(end)];
    }

    state.addTeamDialog.visible = true;
};

const onSaveTeam = async () => {
    await useI18nFormValidate(teamForm);
    const form = state.addTeamDialog.form;
    form.validityStartDate = formatDate(form.validityDate[0]);
    form.validityEndDate = formatDate(form.validityDate[1]);
    await tagApi.saveTeam.request(form);
    useI18nSaveSuccessMsg();
    search();
    onCancelSaveTeam();
};

const onCancelSaveTeam = () => {
    state.addTeamDialog.visible = false;
    teamForm.value.resetFields();
    setTimeout(() => {
        state.addTeamDialog.form = {} as any;
    }, 500);
};

const onDeleteTeam = async () => {
    await useI18nDeleteConfirm(state.selectionData.map((x: any) => x.name).join('、'));
    await tagApi.delTeam.request({ id: state.selectionData.map((x: any) => x.id).join(',') });
    useI18nDeleteSuccessMsg();
    search();
};

/********** 团队成员相关 ***********/

const onShowMembers = async (team: any) => {
    state.showMemDialog.query.teamId = team.id;
    state.showMemDialog.visible = true;
    state.showMemDialog.title = t('team.teamMember', { teamName: team.name });
};

const onDeleteMember = async (data: any) => {
    await tagApi.delTeamMem.request(data);
    useI18nOperateSuccessMsg();
    // 重新赋值成员列表
    setMemebers();
};

/**
 * 设置成员列表信息
 */
const setMemebers = async () => {
    showMemPageTableRef.value.search();
};

const onShowAddMemberDialog = () => {
    state.showMemDialog.addVisible = true;
};

const onAddMember = async () => {
    const memForm = state.showMemDialog.memForm;
    memForm.teamId = state.showMemDialog.query.teamId;
    notBlank(memForm.accountIds, t('team.selectAccountTips'));

    await tagApi.saveTeamMem.request(memForm);
    useI18nSaveSuccessMsg();
    setMemebers();
    onCancelAddMember();
};

const onCancelAddMember = () => {
    state.showMemDialog.memForm = {} as any;
    state.showMemDialog.addVisible = false;
};
</script>
<style lang="scss" scoped></style>
