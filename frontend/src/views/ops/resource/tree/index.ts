/**
 * 资源树 v2 核心统一出口：新增资产类型只需在各资源模块（ops 下各自 resource 模块）注册 Contributor。
 */
export { registerContributor, getContributor, getRootContributor, resolveHasChildren, isNodeSelectable, toNodeMatcher, type NodeMatcher, type TreeContributor } from './registry';
export { registerCommand, registerMenu, resolveNodeMenu, findTriggerCommand, validateMenuCommands, type TreeCommand, type MenuEntry, type CommandTrigger } from './commands';
export { treeEvents, DEFAULT_TREE_SCOPE } from './events';
export { TreeApiKey } from './context';
export { useLazyTree } from './useLazyTree';
export { LOADING_KIND, ERROR_KIND, RES_GROUP_KIND, type NodeKind, type TreeNode, type TreeNodeData, type TreeApi, type TreeCommandCtx, type TreeEngineExpose } from './types';
