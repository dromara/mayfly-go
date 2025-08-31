import { defineAsyncComponent } from 'vue';
import { ResourceTypeEnum, TagResourceTypeEnum } from '@/common/commonEnum';
import { sleep } from '@/common/utils/loading';
import { ContextmenuItem } from '@/components/contextmenu';
import { esApi } from '@/views/ops/es/api';
import { i18n } from '@/i18n';
import { NodeType, TagTreeNode, ResourceComponentConfig } from '@/views/ops/component/tag';
import { ResourceConfig } from '../../component/tag';

const Icon = {
    name: ResourceTypeEnum.Es.extra.icon,
    color: ResourceTypeEnum.Es.extra.iconColor,
};

const EsInstanceList = defineAsyncComponent(() => import('../EsInstanceList.vue'));
const EsDataOp = defineAsyncComponent(() => import('./EsDataOp.vue'));

const NodeEs = defineAsyncComponent(() => import('./NodeEs.vue'));
const NodeEsIndex = defineAsyncComponent(() => import('./NodeEsIndex.vue'));

export const EsOpComp: ResourceComponentConfig = {
    name: 'tag.esDataOp',
    component: EsDataOp,
    icon: Icon,
};

// tagpath 节点类型
const NodeTypeEsTag = new NodeType(TagTreeNode.TagPath)
    .withContextMenuItems([
        new ContextmenuItem('refresh', 'common.refresh')
            .withIcon('refresh')
            .withOnClick(async (nodeData: TagTreeNode) => (await nodeData.ctx?.addResourceComponent(EsOpComp)).reloadNode(nodeData.key)),
    ])
    .withLoadNodesFunc(async (parentNode: TagTreeNode) => {
        parentNode.ctx?.addResourceComponent(EsOpComp);
        // 加载es实例列表
        const res = await esApi.instances.request({ tagPath: parentNode.params.tagPath });
        if (!res.total) {
            return [];
        }
        const insts = res.list;
        await sleep(100);
        return insts?.map((x: any) => {
            x.tagPath = parentNode.key;
            return TagTreeNode.new(parentNode, `es.inst.${x.code}`, x.name, NodeTypeInst).withNodeComponent(NodeEs).withParams(x);
        });
    });

// 加载实例列表
const NodeTypeInst = new NodeType(1)
    .withLoadNodesFunc(async (parentNode: TagTreeNode) => {
        const params = parentNode.params;

        let oiKey = `es.${params.id}.opIndex`;
        let bsKey = `es.${params.id}.opBasicSearch`;
        let ssKey = `es.${params.id}.opSeniorSearch`;
        let dbKey = `es.${params.id}.opDashboard`;
        let stKey = `es.${params.id}.opSettings`;
        let tpKey = `es.${params.id}.optemplates`;

        let nodeParams = { inst: params, instId: params.id };

        return [
            TagTreeNode.new(parentNode, oiKey, i18n.global.t('es.opIndex'), NodeTypeIndexs).withParams(nodeParams).withIcon({ name: 'Document' }),
            // new TagTreeNode(ssKey, t('es.opSeniorSearch'), NodeTypeSeniorSearch).withParams(nodeParams).withIsLeaf(true),
            // new TagTreeNode(dbKey, t('es.opDashboard'), NodeTypeDashboard).withParams(nodeParams).withIsLeaf(true),
            // new TagTreeNode(stKey, t('es.opSettings'), NodeTypeSettings).withParams(nodeParams),
        ];
    })
    .withNodeClickFunc(async (nodeData: TagTreeNode) => {
        // 添加一个dashboard tab
        (await nodeData.ctx?.addResourceComponent(EsOpComp)).onInstClick(nodeData);
    });

const NodeTypeIndexs = new NodeType(2)
    .withContextMenuItems([
        new ContextmenuItem('refresh', 'common.refresh')
            .withIcon('refresh')
            .withOnClick(async (nodeData: TagTreeNode) =>
                (await nodeData.ctx?.addResourceComponent(EsOpComp)).onRefreshIndices(nodeData.params.instId, nodeData.key)
            ),
        new ContextmenuItem('addIndex', 'es.contextmenu.index.addIndex')
            .withIcon('plus')
            .withOnClick(async (nodeData: TagTreeNode) => (await nodeData.ctx?.addResourceComponent(EsOpComp)).onAddIndex(nodeData)),
        new ContextmenuItem('showSys', 'es.contextmenu.index.showSys')
            .withIcon('View')
            .withOnClick(async (nodeData: TagTreeNode) => (await nodeData.ctx?.addResourceComponent(EsOpComp)).onShowSysIndex(nodeData)),
        new ContextmenuItem('idxTemplate', 'es.templates')
            .withIcon('DocumentCopy')
            .withOnClick(async (nodeData: TagTreeNode) => (await nodeData.ctx?.addResourceComponent(EsOpComp)).onShowTemplate(nodeData)),
    ])
    .withLoadNodesFunc(async (parentNode: TagTreeNode) => {
        const params = parentNode.params;
        console.log(params);
        // 展示索引列表，显示索引名，文档总数
        // 加载索引列表
        let indicesRes = await (await parentNode.ctx?.addResourceComponent(EsOpComp)).loadIdxs(params);

        let idxNodes = [];
        for (let idx of indicesRes) {
            idxNodes.push(
                TagTreeNode.new(parentNode, idx.key, idx.idxName, NodeTypeIndex)
                    .withIsLeaf(true)
                    .withParams({
                        parentKey: parentNode.key,
                        ...idx,
                    })
                    .withNodeComponent(NodeEsIndex)
            );
        }
        return idxNodes;
    });

// 索引操作
const NodeTypeIndex = new NodeType(3)
    .withContextMenuItems([
        // 右键菜单支持：复制名字、新增别名、迁移索引、关闭、启用、删除、数据浏览、跳转基础查询、跳转高级查询
        new ContextmenuItem('copyName', 'es.contextmenu.index.copyName')
            .withIcon('copyDocument')
            .withOnClick(async (nodeData: TagTreeNode) => (await nodeData.ctx?.addResourceComponent(EsOpComp)).onIdxCopyName(nodeData)),
        new ContextmenuItem('refresh', 'es.contextmenu.index.refresh')
            .withIcon('refresh')
            .withOnClick(async (nodeData: TagTreeNode) => (await nodeData.ctx?.addResourceComponent(EsOpComp)).onRefreshIdx(nodeData)),
        new ContextmenuItem('clearCache', 'es.contextmenu.index.clearCache')
            .withIcon('refresh')
            .withOnClick(async (nodeData: TagTreeNode) => (await nodeData.ctx?.addResourceComponent(EsOpComp)).onClearIdxCache(nodeData)),
        new ContextmenuItem('flush', 'es.contextmenu.index.flush')
            .withIcon('refresh')
            .withOnClick(async (nodeData: TagTreeNode) => (await nodeData.ctx?.addResourceComponent(EsOpComp)).onFlushIdx(nodeData)),
        new ContextmenuItem('Reindex', 'es.Reindex')
            .withIcon('Switch')
            .withOnClick(async (nodeData: TagTreeNode) => (await nodeData.ctx?.addResourceComponent(EsOpComp)).onIdxReindex(nodeData)),
        new ContextmenuItem('Close', 'es.contextmenu.index.Close')
            .withIcon('Close')
            .withOnClick(async (nodeData: TagTreeNode) => (await nodeData.ctx?.addResourceComponent(EsOpComp)).onIdxClose(nodeData))
            .withHideFunc((data: any) => {
                return data.params.idx.status !== 'open';
            }),
        new ContextmenuItem('Open', 'es.contextmenu.index.Open')
            .withIcon('Select')
            .withOnClick(async (nodeData: TagTreeNode) => (await nodeData.ctx?.addResourceComponent(EsOpComp)).onIdxOpen(nodeData))
            .withHideFunc((data: any) => {
                return data.params.idx.status === 'open';
            }),
        new ContextmenuItem('Delete', 'es.contextmenu.index.Delete')
            .withIcon('Delete')
            .withOnClick(async (nodeData: TagTreeNode) => (await nodeData.ctx?.addResourceComponent(EsOpComp)).onIdxDelete(nodeData)),
        new ContextmenuItem('BaseSearch', 'es.contextmenu.index.BaseSearch')
            .withIcon('Search')
            .withOnClick(async (nodeData: TagTreeNode) => (await nodeData.ctx?.addResourceComponent(EsOpComp)).onIdxBaseSearch(nodeData)),
        // new ContextmenuItem('SeniorSearch', 'es.contextmenu.index.SeniorSearch').withIcon('Search').withOnClick((data: any) => onIdxSeniorSearch(data)),
        new ContextmenuItem('IndexDetail', 'es.indexDetail')
            .withIcon('InfoFilled')
            .withOnClick(async (nodeData: TagTreeNode) => (await nodeData.ctx?.addResourceComponent(EsOpComp)).onIndexDetail(nodeData)),
    ])

    .withNodeClickFunc(async (nodeData: TagTreeNode) => {
        const params = nodeData.params;
        (await nodeData.ctx?.addResourceComponent(EsOpComp)).loadIndexData(params.params.inst.id, params);
    });

export default {
    order: 5,
    resourceType: TagResourceTypeEnum.EsInstance.value,
    rootNodeType: NodeTypeEsTag,
    manager: {
        componentConf: {
            component: EsInstanceList,
            icon: Icon,
            name: 'tag.es',
        },
        countKey: 'es',
        permCode: 'es:instance:save',
    },
} as ResourceConfig;
