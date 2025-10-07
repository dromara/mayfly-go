<template>
    <ResourceSelect v-bind="$attrs" v-model="selectNode" @change="changeNode" :resource-type="TagResourceTypePath.Db" :tag-path-node-type="NodeTypeDbInst">
        <template #iconPrefix>
            <SvgIcon v-if="dbType && getDbDialect(dbType)" :name="getDbDialect(dbType).getInfo().icon" :size="18" />
        </template>
    </ResourceSelect>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { TagResourceTypeEnum, TagResourceTypePath } from '@/common/commonEnum';
import { NodeType, TagTreeNode } from '@/views/ops/component/tag';
import { dbApi } from '@/views/ops/db/api';
import { sleep } from '@/common/utils/loading';
import { getDbDialect, schemaDbTypes } from '@/views/ops/db/dialect';
import ResourceSelect from '@/views/ops/resource/ResourceSelect.vue';
import NodeDbInst from '@/views/ops/db/resource/NodeDbInst.vue';
import NodeDb from '@/views/ops/db/resource/NodeDb.vue';
import { DbIcon, SchemaIcon } from '@/views/ops/db/resource';
import { DbInst } from '@/views/ops/db/db';

const dbId = defineModel<number>('dbId');
const instName = defineModel<string>('instName');
const dbName = defineModel<string>('dbName');
const tagPath = defineModel<string>('tagPath');
const dbType = defineModel<string>('dbType');

const emits = defineEmits(['selectDb']);

const selectNode = computed({
    get: () => {
        return dbName.value ? `${tagPath.value} > ${instName.value} > ${dbName.value}` : '';
    },
    set: () => {
        //
    },
});

const NodeTypeDbInst = new NodeType(TagResourceTypeEnum.DbInstance.value).withLoadNodesFunc(async (parentNode: TagTreeNode) => {
    const tagPath = parentNode.key;

    const dbInstancesRes = await dbApi.instances.request({ tagPath, pageSize: 100 });
    const dbInstances = dbInstancesRes.list;
    if (!dbInstances) {
        return [];
    }

    // 防止过快加载会出现一闪而过，对眼睛不好
    await sleep(100);
    return dbInstances?.map((x: any) => {
        x.tagPath = tagPath;
        return TagTreeNode.new(parentNode, `${x.code}`, x.name, NodeTypeDbConf).withParams(x).withNodeComponent(NodeDbInst);
    });
});

const NodeTypeDbConf = new NodeType(TagResourceTypeEnum.Db.value).withLoadNodesFunc(async (parentNode: TagTreeNode) => {
    const params = parentNode.params;

    const tagPath = params.tagPath;
    const authCerts = {} as any;
    for (let authCert of params.authCerts) {
        authCerts[authCert.name] = authCert;
    }

    const dbInfoRes = await dbApi.dbs.request({
        tagPath: `${tagPath}${TagResourceTypeEnum.DbInstance.value}|${params.code}`,
    });
    const dbInfos = dbInfoRes.list;
    if (!dbInfos) {
        return [];
    }

    return dbInfos?.map((x: any) => {
        x.tagPath = tagPath;
        x.username = authCerts[x.authCertName]?.username;
        return TagTreeNode.new(parentNode, `${x.code}`, x.name, NodeTypeDbs).withParams(x).withIcon(DbIcon).withNodeComponent(NodeDb);
    });
});

// 数据库列表名类型
const NodeTypeDbs = new NodeType(222).withLoadNodesFunc(async (parentNode: TagTreeNode) => {
    const params = parentNode.params;
    const dbs = (await DbInst.getDbNames(params))?.sort();
    const hasSchema = schemaDbTypes.includes(params.type);
    const nodeType = hasSchema ? NodeTypeDbSchema : NodeTypeNoSchemaDb;

    return dbs.map((x: any) => {
        return TagTreeNode.new(parentNode, `${parentNode.key}.${x}`, x, nodeType)
            .withParams({
                tagPath: params.tagPath,
                id: params.id,
                name: params.name,
                type: params.type,
                host: `${params.host}:${params.port}`,
                dbs: dbs,
                db: x,
                code: params.code,
            })
            .withIcon(DbIcon)
            .withIsLeaf(!hasSchema);
    });
});

// 数据库节点
const NodeTypeDbSchema = new NodeType(2).withLoadNodesFunc(async (parentNode: TagTreeNode) => {
    const params = parentNode.params;
    params.parentKey = parentNode.key;
    const { id, db } = params;
    const schemaNames = await dbApi.pgSchemas.request({ id, db });
    const dbs = schemaNames.map((x: any) => `${db}/${x}`);
    return schemaNames.map((sn: any) => {
        // 将db变更为  db/schema;
        const nParams = { ...params };
        nParams.schema = sn;
        nParams.db = nParams.db + '/' + sn;
        nParams.dbs = dbs;
        return TagTreeNode.new(parentNode, `${params.id}.${params.db}.schema.${sn}`, sn, NodeTypePostgresSchema)
            .withParams(nParams)
            .withIcon(SchemaIcon)
            .withIsLeaf(true);
    });
});

// postgres schema模式
const NodeTypePostgresSchema = new NodeType(99);
const NodeTypeNoSchemaDb = new NodeType(99);

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
