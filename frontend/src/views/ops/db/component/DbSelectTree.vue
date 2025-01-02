<template>
    <TagTreeResourceSelect
        v-bind="$attrs"
        v-model="selectNode"
        @change="changeNode"
        :resource-type="TagResourceTypePath.Db"
        :tag-path-node-type="NodeTypeTagPath"
    >
        <template #iconPrefix>
            <SvgIcon v-if="dbType && getDbDialect(dbType)" :name="getDbDialect(dbType).getInfo().icon" :size="18" />
        </template>
        <template #prefix="{ data }">
            <SvgIcon v-if="data.type.value == SqlExecNodeType.DbInst" :name="getDbDialect(data.params.type).getInfo().icon" :size="18" />
            <SvgIcon v-if="data.icon" :name="data.icon.name" :color="data.icon.color" />
        </template>
    </TagTreeResourceSelect>
</template>

<script setup lang="ts">
import { TagResourceTypeEnum, TagResourceTypePath } from '@/common/commonEnum';
import { NodeType, TagTreeNode } from '@/views/ops/component/tag';
import { dbApi } from '@/views/ops/db/api';
import { sleep } from '@/common/utils/loading';
import SvgIcon from '@/components/svgIcon/index.vue';
import { getDbDialect, noSchemaTypes } from '@/views/ops/db/dialect';
import TagTreeResourceSelect from '../../component/TagTreeResourceSelect.vue';
import { computed } from 'vue';
import { DbInst } from '../db';

const dbId = defineModel<number>('dbId');
const instName = defineModel<string>('instName');
const dbName = defineModel<string>('dbName');
const tagPath = defineModel<string>('tagPath');
const dbType = defineModel<string>('dbType');

const emits = defineEmits(['selectDb']);

/**
 * 树节点类型
 */
class SqlExecNodeType {
    static DbInst = 1;
    static Db = 2;
    static TableMenu = 3;
    static SqlMenu = 4;
    static Table = 5;
    static Sql = 6;
    static PgSchemaMenu = 7;
    static PgSchema = 8;
}

const selectNode = computed({
    get: () => {
        return dbName.value ? `${tagPath.value} > ${instName.value} > ${dbName.value}` : '';
    },
    set: () => {
        //
    },
});

const DbIcon = {
    name: 'Coin',
    color: '#67c23a',
};

// pgsql schema icon
const SchemaIcon = {
    name: 'List',
    color: '#67c23a',
};

const NodeTypeTagPath = new NodeType(TagTreeNode.TagPath).withLoadNodesFunc(async (parentNode: TagTreeNode) => {
    const dbInfoRes = await dbApi.dbs.request({ tagPath: parentNode.key });
    const dbInfos = dbInfoRes.list;
    if (!dbInfos) {
        return [];
    }

    // 防止过快加载会出现一闪而过，对眼睛不好
    await sleep(100);
    return dbInfos?.map((x: any) => {
        x.tagPath = parentNode.key;
        return new TagTreeNode(`${parentNode.key}.${x.id}`, x.name, NodeTypeDbInst).withParams(x);
    });
});

/**  mysql类型的数据库，没有schema层 */
const noSchemaType = (type: string) => {
    return noSchemaTypes.includes(type);
};

// 数据库实例节点类型
const NodeTypeDbInst = new NodeType(SqlExecNodeType.DbInst).withLoadNodesFunc(async (parentNode: TagTreeNode) => {
    const params = parentNode.params;
    const dbs = (await DbInst.getDbNames(params))?.sort();
    let fn: NodeType;
    if (noSchemaType(params.type)) {
        fn = MysqlNodeTypes;
    } else {
        fn = PgNodeTypes;
    }
    return dbs.map((x: any) => {
        let tagTreeNode = new TagTreeNode(`${parentNode.key}.${x}`, `${x}`, fn)
            .withParams({
                tagPath: params.tagPath,
                id: params.id,
                code: params.code,
                instanceId: params.instanceId,
                name: params.name,
                type: params.type,
                host: `${params.host}:${params.port}`,
                dbs: dbs,
                db: x,
            })
            .withIcon(DbIcon);
        if (noSchemaType(params.type)) {
            tagTreeNode.isLeaf = true;
        }
        return tagTreeNode;
    });
});

// 数据库节点
const PgNodeTypes = new NodeType(SqlExecNodeType.Db).withLoadNodesFunc(async (parentNode: TagTreeNode) => {
    // pg类数据库会多一层schema
    const params = parentNode.params;
    const { id, db } = params;
    const schemaNames = await dbApi.pgSchemas.request({ id, db });
    return schemaNames.map((sn: any) => {
        // 将db变更为  db/schema;
        const nParams = { ...params };
        nParams.schema = sn;
        nParams.db = nParams.db + '/' + sn;
        nParams.dbs = schemaNames;
        let tagTreeNode = new TagTreeNode(`${params.id}.${params.db}.schema.${sn}`, sn, NodeTypePostgresSchema).withParams(nParams).withIcon(SchemaIcon);
        tagTreeNode.isLeaf = true;
        return tagTreeNode;
    });
});

const MysqlNodeTypes = new NodeType(SqlExecNodeType.Db);

// postgres schema模式
const NodeTypePostgresSchema = new NodeType(SqlExecNodeType.PgSchema);

const changeNode = (nodeData: TagTreeNode) => {
    const params = nodeData.params;
    dbName.value = params.db;
    instName.value = params.name;
    dbId.value = params.id;
    tagPath.value = params.tagPath;
    dbType.value = params.type;

    emits('selectDb', params);
};
</script>

<style lang="scss"></style>
