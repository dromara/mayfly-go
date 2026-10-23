/**
 * Milvus 固定权限组按作用域分三类（cluster/database/collection），其余归 custom。
 * 分类是纯数据事实，集中在此单点维护，供权限弹窗的 Tab 过滤与归属判定共用。
 */
export const PRIVILEGE_GROUP_CATEGORIES = {
    CLUSTER: {
        name: 'cluster',
        label: '集群权限',
        groupNames: ['ClusterReadOnly', 'ClusterReadWrite', 'ClusterAdmin'],
    },
    DATABASE: {
        name: 'database',
        label: '数据库权限',
        groupNames: ['DatabaseReadOnly', 'DatabaseReadWrite', 'DatabaseAdmin'],
    },
    COLLECTION: {
        name: 'collection',
        label: 'Collection权限',
        groupNames: ['CollectionReadOnly', 'CollectionReadWrite', 'CollectionAdmin'],
    },
    CUSTOM: {
        name: 'custom',
        label: '自定义权限组',
        groupNames: [], // 其余所有权限组
    },
};

/** 全部固定权限组名（cluster/database/collection 三类之和） */
export const FIXED_PRIVILEGE_GROUP_NAMES: string[] = [
    ...PRIVILEGE_GROUP_CATEGORIES.CLUSTER.groupNames,
    ...PRIVILEGE_GROUP_CATEGORIES.DATABASE.groupNames,
    ...PRIVILEGE_GROUP_CATEGORIES.COLLECTION.groupNames,
];

/** 检查权限组是否属于当前 Tab 类型（custom 为排除所有固定权限组后的其余组） */
export function isPrivilegeGroupBelongsToTab(groupName: string, tabType: string): boolean {
    if (tabType === 'cluster') {
        return PRIVILEGE_GROUP_CATEGORIES.CLUSTER.groupNames.includes(groupName);
    } else if (tabType === 'database') {
        return PRIVILEGE_GROUP_CATEGORIES.DATABASE.groupNames.includes(groupName);
    } else if (tabType === 'collection') {
        return PRIVILEGE_GROUP_CATEGORIES.COLLECTION.groupNames.includes(groupName);
    }
    // custom: 排除所有固定权限组
    return !FIXED_PRIVILEGE_GROUP_NAMES.includes(groupName);
}

/** 权限组所属分类（未命中任何固定组即 custom） */
export function privilegeGroupCategory(groupName: string): 'cluster' | 'database' | 'collection' | 'custom' {
    if (PRIVILEGE_GROUP_CATEGORIES.CLUSTER.groupNames.includes(groupName)) return 'cluster';
    if (PRIVILEGE_GROUP_CATEGORIES.DATABASE.groupNames.includes(groupName)) return 'database';
    if (PRIVILEGE_GROUP_CATEGORIES.COLLECTION.groupNames.includes(groupName)) return 'collection';
    return 'custom';
}
