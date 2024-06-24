<template>
    <div>
        <el-descriptions :column="3" border>
            <el-descriptions-item :span="3" label="标签"><TagCodePath :path="db.codePaths" /></el-descriptions-item>

            <el-descriptions-item :span="1" label="名称">{{ db?.name }}</el-descriptions-item>
            <el-descriptions-item :span="1" label="主机">{{ `${db?.host}:${db?.port}` }}</el-descriptions-item>
            <el-descriptions-item :span="1" label="类型">
                <SvgIcon :name="getDbDialect(db?.type).getInfo().icon" :size="20" />{{ db?.type }}
            </el-descriptions-item>

            <el-descriptions-item label="数据库">{{ sqlExec.db }}</el-descriptions-item>
            <el-descriptions-item label="表">
                {{ sqlExec.table }}
            </el-descriptions-item>
            <el-descriptions-item label="类型">
                <el-tag size="small">{{ EnumValue.getLabelByValue(DbSqlExecTypeEnum, sqlExec.type) }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="执行SQL">
                <monaco-editor height="300px" language="sql" v-model="sqlExec.sql" :options="{ readOnly: true }" />
            </el-descriptions-item>
        </el-descriptions>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, watch, onMounted } from 'vue';
import EnumValue from '@/common/Enum';
import { dbApi } from '@/views/ops/db/api';
import { DbSqlExecTypeEnum } from '@/views/ops/db/enums';
import MonacoEditor from '@/components/monaco/MonacoEditor.vue';
import { getDbDialect } from '@/views/ops/db/dialect';
import { tagApi } from '@/views/ops/tag/api';
import { TagResourceTypeEnum } from '@/common/commonEnum';
import TagCodePath from '@/views/ops/component/TagCodePath.vue';

const props = defineProps({
    // 业务key
    bizKey: {
        type: [String],
        default: '',
    },
});

const state = reactive({
    sqlExec: {
        sql: '',
    } as any,
    db: {} as any,
});

const { sqlExec, db } = toRefs(state);

onMounted(() => {
    getDbSqlExec(props.bizKey);
});

watch(
    () => props.bizKey,
    (newValue: any) => {
        getDbSqlExec(newValue);
    }
);

const getDbSqlExec = async (bizKey: string) => {
    if (!bizKey) {
        return;
    }
    const res = await dbApi.getSqlExecs.request({ flowBizKey: bizKey });
    if (!res.list) {
        return;
    }
    state.sqlExec = res.list?.[0];
    const dbRes = await dbApi.dbs.request({ id: state.sqlExec.dbId });
    state.db = dbRes.list?.[0];

    tagApi.listByQuery.request({ type: TagResourceTypeEnum.DbName.value, codes: state.db.code }).then((res) => {
        state.db.codePaths = res.map((item: any) => item.codePath);
    });
};
</script>
<style lang="scss"></style>
