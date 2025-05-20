import EnumValue from '@/common/Enum';
import LogicFlow from '@logicflow/core';

const allNodes: Record<string, any> = import.meta.glob('./**/index.ts', { eager: true });

const nodeMap = new Map<string, CustomNode>();

export interface CustomNode {
    order?: number; // 节点排序（影响拖拽面板显示顺序）
    type: string; // 节点类型
    registerConf: any; // 节点注册信息
    dndPanelConf: any; // 节点拖拽面板配置信息
    propSettingComp?: any; // 属性设置组件
}

/**
 * 获取所有自定义节点
 *
 * @returns 自定义节点配置
 */
export const getCustomNodes = () => {
    const nodes = [];
    for (const path in allNodes) {
        // path => ./start/index.ts
        // 获取默认导出的部件
        const node = allNodes[path].default;
        nodes.push(node);
        nodeMap.set(node.type, node);
    }

    return nodes.sort((a, b) => {
        if (a.order !== undefined && b.order !== undefined) {
            return a.order - b.order; // 按order字段排序
        } else if (a.order !== undefined) {
            return -1; // a有order字段，排在前面
        } else if (b.order !== undefined) {
            return 1; // b有order字段，排在前面
        } else {
            return 0; // 两个都没有order字段，保持原顺序
        }
    });
};

/**
 * 根据节点类型获取自定义节点
 *
 * @param type 节点类型
 * @returns 节点信息
 */
export const getCustomNode = (type: string): CustomNode | undefined => {
    return nodeMap.get(type);
};

/**
 * 注册自定义节点
 *
 * @param lf LogicFlow 实例
 */
export const initCustomNodes = (lf: LogicFlow, disable: boolean = false) => {
    const customNodes = getCustomNodes();
    const dndPanelItmes = [];

    // 注册自定义节点
    for (const node of customNodes) {
        if (!node.registerConf) {
            continue;
        }
        lf.register(node.registerConf);
        if (node.dndPanelConf) {
            dndPanelItmes.push(node.dndPanelConf);
        }
    }

    if (disable) {
        return;
    }

    const extension: any = lf.extension;
    //  注册自定义节点面板
    extension?.dndPanel?.setPatternItems(dndPanelItmes);
};
