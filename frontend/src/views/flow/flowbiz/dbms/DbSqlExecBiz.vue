<template>
    <div>
        <el-descriptions :column="3" border>
            <el-descriptions-item :span="3" :label="$t('common.tag')"><TagCodePath :path="db.codePaths" /></el-descriptions-item>

            <el-descriptions-item :span="1" :label="$t('common.name')">{{ db?.name }}</el-descriptions-item>
            <el-descriptions-item :span="1" label="Host">
                <SvgIcon :name="getDbDialect(db?.type).getInfo().icon" :size="20" />
                {{ `${db?.host}:${db?.port}` }}
            </el-descriptions-item>
            <el-descriptions-item :span="1" :label="$t('tag.db')">{{ dbName }}</el-descriptions-item>

            <el-descriptions-item :label="$t('flow.runSql')">
                <monaco-editor height="300px" language="sql" v-model="sql" :options="{ readOnly: true }" />
            </el-descriptions-item>
        </el-descriptions>

        <div v-if="runRes && runRes.length > 0">
            <el-divider content-position="left">{{ $t('flow.handleResult') }}</el-divider>
            <el-table :data="runRes" :max-height="400">
                <el-table-column prop="sql" label="SQL" show-overflow-tooltip />

                <el-table-column prop="res" :label="$t('flow.runResult')" :min-width="30" show-overflow-tooltip>
                    <template #default="scope">
                        <el-popover placement="top" width="50%" trigger="hover">
                            <template #reference>
                                <el-link icon="view" :type="scope.row.errorMsg ? 'danger' : 'success'" underline="never"> </el-link>
                            </template>

                            <el-text v-if="scope.row.errorMsg">{{ scope.row.errorMsg }}</el-text>
                            <el-table max-height="600px" v-else :data="scope.row.res" size="small">
                                <el-table-column
                                    :width="DbInst.flexColumnWidth(col.name, scope.row.res)"
                                    v-for="col in scope.row.columns"
                                    :key="col.name"
                                    :label="col.name"
                                    :prop="col.name"
                                    show-overflow-tooltip
                                />
                            </el-table>
                        </el-popover>
                    </template>
                </el-table-column>

                <!-- <el-table-column prop="errorMsg" label="错误信息" :min-width="60" show-overflow-tooltip /> -->
            </el-table>
        </div>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, watch, onMounted } from 'vue';
import { dbApi } from '@/views/ops/db/api';
import MonacoEditor from '@/components/monaco/MonacoEditor.vue';
import { getDbDialect } from '@/views/ops/db/dialect';
import { tagApi } from '@/views/ops/tag/api';
import { TagResourceTypeEnum } from '@/common/commonEnum';
import TagCodePath from '@/views/ops/component/TagCodePath.vue';
import SvgIcon from '@/components/svgIcon/index.vue';
import { DbInst } from '@/views/ops/db/db';

const props = defineProps({
    procinst: {
        type: [Object],
        default: () => {},
    },
});

const state = reactive({
    // sqlExec: {
    //     sql: '',
    // } as any,
    db: {} as any,
    dbName: '',
    sql: '',
    runRes: [],
});

const { db, dbName, sql, runRes } = toRefs(state);

onMounted(() => {
    parseBizForm(props.procinst.bizForm);
});

watch(
    () => props.procinst.bizForm,
    (newValue: any) => {
        parseBizForm(newValue);
    }
);

const parseBizForm = async (bizFormStr: string) => {
    if (props.procinst.bizHandleRes) {
        state.runRes = JSON.parse(props.procinst.bizHandleRes);
    } else {
        state.runRes = [];
    }

    const bizForm = JSON.parse(bizFormStr);
    state.sql = bizForm.sql;
    state.dbName = bizForm.dbName;

    const dbRes = await dbApi.dbs.request({ id: bizForm.dbId });
    state.db = dbRes.list?.[0];

    tagApi.listByQuery.request({ type: TagResourceTypeEnum.Db.value, codes: state.db.code }).then((res) => {
        state.db.codePaths = res.map((item: any) => item.codePath);
    });
};
</script>
<style lang="scss"></style>
