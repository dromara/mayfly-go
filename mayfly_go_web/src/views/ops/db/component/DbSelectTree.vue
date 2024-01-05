<template>
    <div class="db-select-tree">
        <div style="color: gray">{{ (tagPath || '') + ' - ' + (dbName || '请选择数据源schema') }}</div>
        <tag-tree :resource-type="TagResourceTypeEnum.Db.value" :tag-path-node-type="NodeTypeTagPath" ref="tagTreeRef">
            <template #prefix="{ data }">
                <SvgIcon v-if="data.type.value == SqlExecNodeType.DbInst" :name="getDbDialect(data.params.type).getInfo().icon" :size="18" />
                <SvgIcon v-if="data.icon" :name="data.icon.name" :color="data.icon.color" />
            </template>
        </tag-tree>
    </div>
</template>

<script setup lang="ts">
import { TagResourceTypeEnum } from '@/common/commonEnum';
import TagTree from '@/views/ops/component/TagTree.vue';
import { NodeType, TagTreeNode } from '@/views/ops/component/tag';
import { dbApi } from '@/views/ops/db/api';
import { sleep } from '@/common/utils/loading';
import SvgIcon from '@/components/svgIcon/index.vue';
import { DbType, getDbDialect } from '@/views/ops/db/dialect';

defineProps({
    dbId: {
        type: Number,
    },
    dbName: {
        type: String,
    },
    tagPath: {
        type: String,
    },
});

const emits = defineEmits(['update:dbName', 'update:tagPath', 'update:dbId', 'selectDb']);

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
const mysqlType = (type: string) => {
    return type === DbType.mysql;
};

// 数据库实例节点类型
const NodeTypeDbInst = new NodeType(SqlExecNodeType.DbInst).withLoadNodesFunc((parentNode: TagTreeNode) => {
    const params = parentNode.params;
    const dbs = params.database.split(' ')?.sort();
    let fn: NodeType;
    if (mysqlType(params.type)) {
        fn = MysqlNodeTypes;
    } else {
        fn = PgNodeTypes;
    }
    return dbs.map((x: any) => {
        let tagTreeNode = new TagTreeNode(`${parentNode.key}.${x}`, x, fn)
            .withParams({
                tagPath: params.tagPath,
                id: params.id,
                instanceId: params.instanceId,
                name: params.name,
                type: params.type,
                host: `${params.host}:${params.port}`,
                dbs: dbs,
                db: x,
            })
            .withIcon(DbIcon);
        if (mysqlType(params.type)) {
            tagTreeNode.isLeaf = true;
        }
        return tagTreeNode;
    });
});

const nodeClickChangeDb = (nodeData: TagTreeNode) => {
    const params = nodeData.params;
    // postgres
    emits('update:dbName', params.db);
    emits('update:dbId', params.id);
    emits('update:tagPath', params.tagPath);
    emits('selectDb', params);

    return true;
};

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

const MysqlNodeTypes = new NodeType(SqlExecNodeType.Db).withNodeClickFunc(nodeClickChangeDb);

// postgres schema模式
const NodeTypePostgresSchema = new NodeType(SqlExecNodeType.PgSchema).withNodeClickFunc(nodeClickChangeDb);
</script>

<style lang="scss">
.db-select-tree {
    .tag-tree {
        height: auto !important;
        overflow-x: hidden;
        width: 560px;
        .el-tree {
            height: 200px;
            overflow-y: auto;
            overflow-x: hidden;
        }
    }
}
</style>
