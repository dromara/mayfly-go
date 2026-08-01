import { ResourceTypeEnum, TagResourceTypeEnum } from '@/common/commonEnum';
import type { PageParam } from '@/types/common';
import { sleep } from '@/common/utils/loading';
import { milvusApi, perms } from '@/views/ops/milvus/api';
import type { ResourceConfig } from '@/views/ops/resource/resource';
import { createResourceOpTab } from '@/views/ops/resource/resourceOp';
import { defineAsyncComponent } from 'vue';
import { NodeType, TagTreeNode } from '../../component/tag';
import type { Milvus } from '../types';
import type { MachineAuthCert } from '@/views/ops/machine/types';

export interface MilvusNodeParams {
    id: number;
    code: string;
    name: string;
    host: string;
    selectAuthCert?: MachineAuthCert;
    authCerts?: MachineAuthCert[];
    [key: string]: unknown;
}

export const MilvusIcon = {
    name: ResourceTypeEnum.Milvus.extra.icon,
    color: ResourceTypeEnum.Milvus.extra.iconColor,
};

const MilvusList = defineAsyncComponent(() => import('../MilvusList.vue'));
const MilvusOp = defineAsyncComponent(() => import('./MilvusOp.vue'));

const NodeMilvus = defineAsyncComponent(() => import('./NodeMilvus.vue'));
const NodeMilvusAc = defineAsyncComponent(() => import('./NodeMilvusAc.vue'));

const getMilvusOpTab = async (milvus: MilvusNodeParams, acName: string) => {
    const tabKey = `milvus.${milvus.id}.${acName}`;
    return await createResourceOpTab({
        key: tabKey,
        name: milvus.acUsername ? `${milvus.name} (${milvus.acUsername})` : milvus.name,
        component: MilvusOp,
        componentProps: {
            milvusId: milvus.id,
            acName,
            tabKey,
        },
        tabComponentProps: { icon: MilvusIcon },
    });
};

const getMilvusOpTabCompInst = async (milvus: MilvusNodeParams, acName: string) => {
    return (await getMilvusOpTab(milvus, acName)).componentInstance;
};

// milvus 授权凭证节点类型：点击后打开独立标签页
const NodeTypeMilvusAc = new NodeType(Number(TagResourceTypeEnum.Milvus.value) * 10 + 1).withNodeClickFunc(async (node: TagTreeNode) => {
    const milvus = node.params as unknown as MilvusNodeParams;
    const acName = milvus.selectAuthCert?.name || '';
    // 仅在首次创建时初始化（已存在的标签页只是激活，不重置状态）
    const compRef = await getMilvusOpTabCompInst(milvus, acName);
    compRef?.initMilvus?.(milvus);
});

const NodeTypeMilvus = new NodeType(TagResourceTypeEnum.Milvus.value).withLoadNodesFunc(async (node: TagTreeNode) => {
    const milvus = node.params as unknown as MilvusNodeParams;
    const authCerts = milvus.authCerts || [];
    return authCerts.map((x: MachineAuthCert) =>
        TagTreeNode.new(node, `milvus.${milvus.id}.${x.name}`, x.username, NodeTypeMilvusAc)
            .withNodeComponent(NodeMilvusAc)
            .withParams({ ...milvus, selectAuthCert: x })
            .withIsLeaf(true)
            .withIcon({ name: 'Ticket', color: '#409eff' })
    );
});

// tagpath 节点类型
const NodeTypeMilvusTag = new NodeType(TagTreeNode.TagPath).withLoadNodesFunc(async (parentNode: TagTreeNode) => {
    const tagPath = parentNode.params.tagPath;
    const res = await milvusApi.list.request({ tagPath: tagPath as string });
    if (!res.total) {
        return [];
    }
    const milvusInfos = res.list;
    await sleep(100);
    return milvusInfos.map((x: Milvus) => {
        return TagTreeNode.new(parentNode, `milvus.${x.id}`, x.name, NodeTypeMilvus).withParams(x as unknown as Record<string, unknown>).withNodeComponent(NodeMilvus);
    });
});

export default {
    order: 7,
    resourceType: TagResourceTypeEnum.Milvus.value,
    rootNodeType: NodeTypeMilvusTag,
    manager: {
        componentConf: {
            component: MilvusList,
            icon: MilvusIcon,
            name: 'milvus',
        },
        countKey: 'milvus',
        permCode: perms.base,
    },
} as ResourceConfig;
