import Api from '@/common/Api';
import type { PageResult } from '@/types/common';
import type {
    IDatabase,
    ICollection,
    IPartition,
    IUser,
    IPrivilegeGroup,
    IResourceGroup,
    IMilvusCollectionDetail,
    IMilvusDatabaseDetail,
    IMilvusHealthStatus,
    IMilvusInsertResult,
    IMilvusMockDataResult,
    IMilvusQueryResult,
    IMilvusRoleDetail,
    Milvus,
    MilvusListParam,
} from './types';
import { currentAcName } from './resource/authCert';
let db = '';

// 将当前选中的授权凭证名注入到请求参数中（useApiFetch 会提取 ac 并转为查询参数）
function withAc(params?: Record<string, unknown>): Record<string, unknown> | undefined {
    if (currentAcName) {
        return { ...(params || {}), ac: currentAcName };
    }
    return params;
}

export const milvusApi = {
    // 实例管理
    list: Api.newGet<PageResult<Milvus>, MilvusListParam>('/milvus'),
    testConn: Api.newPost<void>('/milvus/test-conn'),
    save: Api.newPost<number>('/milvus'),
    delete: Api.newDelete<void>(`/milvus/{ids}`),

    // 数据库操作
    listDatabases: (milvusId: number) => Api.newGet<IDatabase[]>(`/milvus/${milvusId}/databases`).request(withAc()),
    createDatabase: (milvusId: number, data: Record<string, unknown>) => Api.newPost<void>(`/milvus/${milvusId}/databases`).request(withAc(data)),
    dropDatabase: (milvusId: number, database: string) => Api.newDelete(`/milvus/${milvusId}/databases/${database}`).request(withAc()),
    describeDatabase: (milvusId: number, database: string) => Api.newGet<IMilvusDatabaseDetail>(`/milvus/${milvusId}/databases/${database}/describe`).request(withAc()),
    alterDatabase: (milvusId: number, database: string, data: Record<string, unknown>) => Api.newPost<void>(`/milvus/${milvusId}/databases/${database}/properties`).request(withAc(data)),
    useDatabase: (milvusId: number, database: string) => {
        db = database;
        // return Api.newPost(`/milvus/${milvusId}/databases/${database}/use`').request();
    },
    // Collection 操作
    listCollections: (milvusId: number, dbName?: string) => {
        return Api.newGet<ICollection[]>(`/milvus/${milvusId}/collections?db=${dbName || db}`).request(withAc());
    },
    createCollection: (milvusId: number, data: Record<string, unknown>) => Api.newPost<void>(`/milvus/${milvusId}/collections?db=${db}`).request(withAc(data)),
    alterCollection: (milvusId: number, collection: string, data: Record<string, unknown>) =>
        Api.newPost<void>(`/milvus/${milvusId}/collections/${collection}/alter?db=${db}`).request(withAc(data)),
    dropCollection: (milvusId: number, collection: string) => Api.newDelete(`/milvus/${milvusId}/collections/${collection}?db=${db}`).request(withAc()),
    describeCollection: (milvusId: number, collection: string) =>
        Api.newGet<IMilvusCollectionDetail>(`/milvus/${milvusId}/collections/${collection}/describe?db=${db}`).request(withAc()),
    getCollectionStatistics: (milvusId: number, collection: string) =>
        Api.newGet<Record<string, string>>(`/milvus/${milvusId}/collections/${collection}/statistics?db=${db}`).request(withAc()),
    loadCollection: (milvusId: number, collection: string, options?: Record<string, unknown>) =>
        Api.newPost<void>(`/milvus/${milvusId}/collections/${collection}/load?db=${db}`).request(withAc({}), options),
    releaseCollection: (milvusId: number, collection: string) =>
        Api.newPost(`/milvus/${milvusId}/collections/${collection}/release?db=${db}`).request(withAc()),
    hasCollection: (milvusId: number, collection: string) => Api.newGet<boolean>(`/milvus/${milvusId}/collections/${collection}/has?db=${db}`).request(withAc()),
    getLoadState: (milvusId: number, collection: string) =>
        Api.newGet<{ loaded: boolean }>(`/milvus/${milvusId}/collections/${collection}/load-state?db=${db}`).request(withAc()),

    // 别名操作
    listAliases: (milvusId: number, collection: string) =>
        Api.newGet<string[]>(`/milvus/${milvusId}/collections/${collection}/aliases?db=${db}`).request(withAc()),
    createAlias: (milvusId: number, collection: string, alias: string) =>
        Api.newPost(`/milvus/${milvusId}/collections/${collection}/aliases?db=${db}`).request(withAc({ alias })),
    dropAlias: (milvusId: number, alias: string) => Api.newDelete(`/milvus/${milvusId}/aliases/${alias}?db=${db}`).request(withAc()),

    // 分区操作
    listPartitions: (milvusId: number, collection: string) =>
        Api.newGet<IPartition[]>(`/milvus/${milvusId}/collections/${collection}/partitions?db=${db}`).request(withAc()),
    createPartition: (milvusId: number, collection: string, data: Record<string, unknown>) =>
        Api.newPost<void>(`/milvus/${milvusId}/collections/${collection}/partitions?db=${db}`).request(withAc(data)),
    dropPartition: (milvusId: number, collection: string, partition: string) =>
        Api.newDelete(`/milvus/${milvusId}/collections/${collection}/partitions/${partition}?db=${db}`).request(withAc()),
    hasPartition: (milvusId: number, collection: string, partition: string) =>
        Api.newGet(`/milvus/${milvusId}/collections/${collection}/partitions/${partition}/has?db=${db}`).request(withAc()),
    releasePartition: (milvusId: number, collection: string, partition: string) =>
        Api.newGet(`/milvus/${milvusId}/collections/${collection}/partitions/release?db=${db}`).request(withAc({ partitionNames: [partition] })),

    // 索引操作
    createIndex: (milvusId: number, collection: string, field: string, data: Record<string, unknown>) =>
        Api.newPost<void>(`/milvus/${milvusId}/collections/${collection}/fields/${field}/index?db=${db}`).request(withAc(data)),
    describeIndex: (milvusId: number, collection: string, field: string) =>
        Api.newGet(`/milvus/${milvusId}/collections/${collection}/fields/${field}/index?db=${db}`).request(withAc()),
    dropIndex: (milvusId: number, collection: string, field: string) =>
        Api.newDelete(`/milvus/${milvusId}/collections/${collection}/fields/${field}/index?db=${db}`).request(withAc()),

    // 数据操作
    insert: (milvusId: number, collection: string, data: Record<string, unknown>) =>
        Api.newPost<void>(`/milvus/${milvusId}/collections/${collection}/insert?db=${db}`).request(withAc(data)),
    deleteData: (milvusId: number, collection: string, data: Record<string, unknown>) =>
        Api.newPost<void>(`/milvus/${milvusId}/collections/${collection}/delete?db=${db}`).request(withAc(data)),
    query: (milvusId: number, collection: string, params: Record<string, unknown>) =>
        Api.newPost<IMilvusQueryResult>(`/milvus/${milvusId}/collections/${collection}/query?db=${db}`).request(withAc(params)),
    search: (milvusId: number, collection: string, params: Record<string, unknown>) =>
        Api.newPost<IMilvusQueryResult>(`/milvus/${milvusId}/collections/${collection}/search?db=${db}`).request(withAc(params)),
    generateMockData: (milvusId: number, collection: string, data: Record<string, unknown>) =>
        Api.newPost<IMilvusMockDataResult>(`/milvus/${milvusId}/collections/${collection}/generate-mock-data?db=${db}`).request(withAc(data)),
    insertSampleData: (milvusId: number, collection: string, data: Record<string, unknown>) =>
        Api.newPost<IMilvusInsertResult>(`/milvus/${milvusId}/collections/${collection}/insert-sample-data?db=${db}`).request(withAc(data)),
    importFile: (milvusId: number, collection: string, formData: FormData) =>
        Api.newPost<IMilvusInsertResult>(`/milvus/${milvusId}/collections/${collection}/import-file?db=${db}&ac=${currentAcName}`).request(formData),

    // 用户权限
    listUsers: (milvusId: number) => Api.newGet<IUser[]>(`/milvus/${milvusId}/users`).request(withAc()),
    createUser: (milvusId: number, data: Record<string, unknown>) => Api.newPost<void>(`/milvus/${milvusId}/users`).request(withAc(data)),
    deleteUser: (milvusId: number, username: string) => Api.newDelete(`/milvus/${milvusId}/users/${username}`).request(withAc()),
    updatePassword: (milvusId: number, username: string, data: Record<string, unknown>) => Api.newPost<void>(`/milvus/${milvusId}/users/${username}/password`).request(withAc(data)),
    grantRole: (milvusId: number, username: string, roleName: string) =>
        Api.newPost(`/milvus/${milvusId}/users/${username}/grantRole`).request(withAc({ roleName })),
    revokeRole: (milvusId: number, username: string, roleName: string) =>
        Api.newPost(`/milvus/${milvusId}/users/${username}/revokeRole`).request(withAc({ roleName })),

    // 角色管理
    listRoles: (milvusId: number) => Api.newGet<string[]>(`/milvus/${milvusId}/roles`).request(withAc()),
    createRole: (milvusId: number, data: Record<string, unknown>) => Api.newPost<void>(`/milvus/${milvusId}/roles`).request(withAc(data)),
    dropRole: (milvusId: number, role: string) => Api.newDelete(`/milvus/${milvusId}/roles/${role}`).request(withAc()),
    describeRole: (milvusId: number, role: string) => Api.newGet<IMilvusRoleDetail>(`/milvus/${milvusId}/roles/${role}`).request(withAc()),
    updateRole: (milvusId: number, data: Record<string, unknown>) => Api.newPost<void>(`/milvus/${milvusId}/roles`).request(withAc(data)),
    getPrivilegeGroups: (milvusId: number) => Api.newGet<IPrivilegeGroup[]>(`/milvus/${milvusId}/privilege-group`).request(withAc()),
    savePrivilegeGroup: (milvusId: number, data: Record<string, unknown>) => Api.newPost<void>(`/milvus/${milvusId}/privilege-group`).request(withAc(data)),
    dropPrivilegeGroup: (milvusId: number, name: string) => Api.newDelete(`/milvus/${milvusId}/privilege-group/${name}`).request(withAc()),

    // 资源组
    listResourceGroups: (milvusId: number) => Api.newGet<string[]>(`/milvus/${milvusId}/resource-groups`).request(withAc()),
    createResourceGroup: (milvusId: number, data: Record<string, unknown>) => Api.newPost<void>(`/milvus/${milvusId}/resource-groups`).request(withAc(data)),
    dropResourceGroup: (milvusId: number, name: string) => Api.newDelete(`/milvus/${milvusId}/resource-groups/${name}`).request(withAc()),
    describeResourceGroup: (milvusId: number, name: string) => Api.newGet<IResourceGroup>(`/milvus/${milvusId}/resource-groups/${name}/describe`).request(withAc()),

    // 系统信息
    getVersion: (milvusId: number) => Api.newGet<string>(`/milvus/${milvusId}/version`).request(withAc()),
    checkHealth: (milvusId: number) => Api.newGet<IMilvusHealthStatus>(`/milvus/${milvusId}/health`).request(withAc()),
};

export const timezones = [
    { label: 'UTC - London, Dublin, Lisbon', value: 'UTC' },
    { label: 'UTC+0/+1 - London (BST)', value: 'Europe/London' },
    { label: 'UTC+1 - Paris, Berlin, Rome, Madrid, Amsterdam, Brussels', value: 'Europe/Paris' },
    { label: 'UTC+2 - Athens, Cairo, Helsinki, Kyiv', value: 'Europe/Athens' },
    { label: 'UTC+3 - Moscow, Istanbul, Nairobi, Riyadh', value: 'Europe/Moscow' },
    { label: 'UTC+3 - Istanbul', value: 'Europe/Istanbul' },
    { label: 'UTC+4 - Dubai, Abu Dhabi, Baku', value: 'Asia/Dubai' },
    { label: 'UTC+5 - Karachi, Islamabad, Tashkent', value: 'Asia/Karachi' },
    { label: 'UTC+5:30 - New Delhi, Mumbai, Chennai, Kolkata, Colombo', value: 'Asia/Kolkata' },
    { label: 'UTC+6 - Dhaka, Almaty', value: 'Asia/Dhaka' },
    { label: 'UTC+7 - Bangkok, Jakarta, Hanoi', value: 'Asia/Bangkok' },
    { label: 'UTC+8 - Beijing, Shanghai, Singapore, Manila, Taipei, Hong Kong', value: 'Asia/Shanghai' },
    { label: 'UTC+9 - Tokyo, Seoul, Pyongyang', value: 'Asia/Tokyo' },
    { label: 'UTC+10 - Sydney, Melbourne, Guam', value: 'Australia/Sydney' },
    { label: 'UTC+11 - Noumea, Solomon Islands', value: 'Pacific/Noumea' },
    { label: 'UTC+12 - Auckland, Fiji, Marshall Islands', value: 'Pacific/Auckland' },
    { label: 'UTC-1 - Azores, Cape Verde', value: 'Atlantic/Azores' },
    { label: 'UTC-2 - South Georgia, South Sandwich Islands', value: 'Atlantic/South_Georgia' },
    { label: 'UTC-3 - Buenos Aires, Brasilia, Montevideo', value: 'America/Buenos_Aires' },
    { label: 'UTC-4 - Santiago, La Paz, Manaus', value: 'America/Santiago' },
    { label: 'UTC-5 - New York, Bogota, Lima', value: 'America/New_York' },
    { label: 'UTC-6 - Chicago, Mexico City, San Jose', value: 'America/Chicago' },
    { label: 'UTC-7 - Denver, Phoenix, Chihuahua', value: 'America/Denver' },
    { label: 'UTC-8 - Los Angeles, Tijuana, Vancouver', value: 'America/Los_Angeles' },
    { label: 'UTC-9 - Anchorage, Juneau', value: 'America/Anchorage' },
    { label: 'UTC-10 - Honolulu', value: 'Pacific/Honolulu' },
];

export const perms = {
    base: 'milvus:base', // base
    inst_save: 'milvus:save', // 保存实例
    inst_del: 'milvus:del', // 删除实例
    data_save: 'milvus:data:save', // 保存数据
    data_del: 'milvus:data:del', // 删除数据
};