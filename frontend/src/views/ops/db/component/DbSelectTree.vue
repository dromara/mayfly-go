<template>
    <ResourceSelect
        v-bind="$attrs"
        v-model="selectNode"
        @change="changeNode"
        :resource-type="ResourceTypeEnum.Db.value"
        :leaf-kinds="[DbSchemaKind]"
        :transform-node="transformNode"
    >
        <template #iconPrefix>
            <SvgIcon v-if="dbType && getDbDialect(dbType)" :name="getDbDialect(dbType).getInfo().icon" :size="16" />
            <TagCodePath :code="displayCode" />
        </template>
    </ResourceSelect>
</template>

<script setup lang="ts">
import { ResourceTypeEnum } from '@/common/commonEnum';
import { getDbDialect, schemaDbTypes } from '@/views/ops/db/dialect';
import { DbKind, DbSchemaKind } from '@/views/ops/db/resource';
import type { TreeNodeData } from '@/views/ops/resource/tree/types';
import ResourceSelect from '@/views/ops/resource/ResourceSelect.vue';
import { computed, ref, watch } from 'vue';
import TagCodePath from '../../component/TagCodePath.vue';
import { dbApi } from '@/views/ops/db/api';

const dbId = defineModel<number>('dbId');
const instName = defineModel<string>('instName');
const dbName = defineModel<string>('dbName');
const tagPath = defineModel<string>('tagPath');
const dbType = defineModel<string>('dbType');
// 可选：选中库节点后由 db 记录回填的实例id（如脱敏列标签按实例维度存储）
const instanceId = defineModel<number | undefined>('instanceId');
// 可选：外部传入的资源code，用于编辑态无 dbId 时回显标签路径
const outerCode = defineModel<string>('code');

const dbCode = ref('');
// 展示code优先取外部传入值（编辑回显场景），否则按 dbId 查询获取
const displayCode = computed(() => outerCode.value || dbCode.value);

const emits = defineEmits(['selectDb']);

/** 数据库树节点参数 (db 资源模块 kind 节点 params 动态结构) */
interface DbNodeParams {
    db?: string;
    name?: string;
    id?: number;
    tagPath?: string;
    type?: string;
    [key: string]: unknown;
}

const selectNode = computed({
    get: () => {
        return dbName.value;
    },
    set: () => {
        //
    },
});

watch(
    () => dbId.value,
    async (id) => {
        if (!id || id <= 0) {
            return;
        }

        const dbRes = await dbApi.dbs.request({ id: dbId.value });
        const db = dbRes.list?.[0];
        if (!db) {
            return '';
        }
        dbCode.value = db.code;
        instanceId.value = db.instanceId;
    },
    { immediate: true }
);

// 节点转换函数：动态判断数据库节点是否为叶子节点
const transformNode = (node: TreeNodeData): TreeNodeData => {
    // 如果是数据库节点，根据数据库类型动态设置 isLeaf
    if (node.kind === DbKind) {
        const params = node.params as DbNodeParams;
        const hasSchema = schemaDbTypes.includes(params.type ?? '');
        // 没有 schema 的数据库（如 MySQL），标记为叶子节点
        if (!hasSchema) {
            (node as TreeNodeData & { isLeaf?: boolean }).isLeaf = true;
        }
    }
    return node;
};

const changeNode = (nodeData: TreeNodeData) => {
    // 有 db/schema 粒度才进入此回调（el-tree-select 任意节点点击的 change 守卫已收口到 ResourceSelect）
    const params = nodeData.params as DbNodeParams;
    dbName.value = params.db;
    instName.value = params.name;
    dbId.value = params.id;
    tagPath.value = params.tagPath;
    dbType.value = params.type;

    emits('selectDb', params);
};
</script>

<style lang="scss"></style>
