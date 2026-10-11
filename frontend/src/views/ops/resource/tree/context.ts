import type { InjectionKey, Ref } from 'vue';

import type { TreeApi } from './types';

/** 容器 provide 的树操作 API（节点渲染器/行组件消费） */
export const TreeApiKey: InjectionKey<TreeApi> = Symbol('TreeApi');

/**
 * 容器 provide 的当前生效查找文本（行组件据此高亮命中片段）。
 *
 * 注：是「引擎已生效」的词而非输入框意图值（防抖由 VirtualTree 上报 filterApplied），
 * 否则会出现已标色但该行还没被过滤掉的分叉。
 *
 * 走注入而非逐层透传 props：行渲染器由贡献者提供、层级不定，透传会逼每个自定义渲染器都接一个
 * 与自身业务无关的 prop。未由容器 provide 时（如选择器场景）行组件按「无查找」渲染。
 */
export const TreeFilterQueryKey: InjectionKey<Ref<string>> = Symbol('TreeFilterQuery');

