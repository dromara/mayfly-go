import type { SearchItem } from './SearchForm/index';

/**
 * 搜索项「按类型」的渲染元数据。
 *
 * SearchFormItem 过去把「哪些 type 要绑选项、选项子节点怎么渲染、要不要合并 fieldNames」散落在模板与
 * 计算属性里（select/cascader/tree-select 各写几处 `item.type === ...`），新增一个可带选项的类型要改多处。
 * 收敛成这张表后，SearchFormItem 只按 meta 驱动，新增类型 = 加一条登记 + 在 SearchForm/index.ts 的
 * SearchItemType 联合里登记该 type（开闭原则，两个登记点相邻、不会漏）。
 *
 * 说明：绝大多数类型（input、input-number、switch、slider、time-select…）无需任何特殊处理——
 * 组件名即 `el-${type}`、不绑选项、不渲染子节点，走 meta 全空的默认即可；自定义形态另有 SearchItem.render 逃生口。
 */
export interface SearchTypeMeta {
    /** 选项数组绑到组件的哪个 prop（select/cascader→'options'，tree-select→'data'）；不填=不绑选项 */
    optionsProp?: 'options' | 'data';
    /** 选项子节点渲染方式：'select'→el-option 列表；'cascader'→默认插槽按 fieldNames.label 取文本 */
    optionSlot?: 'select' | 'cascader';
    /** 是否把 fieldNames 合并进组件的 props.props（tree-select/cascader 需要，且两者键位不同） */
    fieldNames?: 'tree-select' | 'cascader';
}

const searchTypeMetas: Record<string, SearchTypeMeta> = {
    // select 的选项通过 el-option 子节点渲染（不绑 :options prop）
    select: { optionSlot: 'select' },
    'select-v2': { optionsProp: 'options' },
    'tree-select': { optionsProp: 'data', fieldNames: 'tree-select' },
    cascader: { optionsProp: 'options', optionSlot: 'cascader', fieldNames: 'cascader' },
};

/** 取某搜索项类型的渲染元数据；未登记（普通控件）返回空 meta，即「不绑选项、不渲染子节点、不合并 fieldNames」 */
export function searchTypeMeta(item: SearchItem): SearchTypeMeta {
    return (item.type && searchTypeMetas[item.type]) || {};
}
