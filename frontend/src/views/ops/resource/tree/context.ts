import type { InjectionKey } from 'vue';

import type { TreeApi } from './types';

/** 容器 provide 的树操作 API（节点渲染器/行组件消费） */
export const TreeApiKey: InjectionKey<TreeApi> = Symbol('TreeApi');
